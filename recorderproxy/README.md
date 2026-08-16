# veidemann-recorderproxy

## Overview

`recorderproxy` is the HTTP/HTTPS capture proxy used by Veidemann to fetch traffic, record request and response material, and coordinate side effects with BrowserController, DNS resolver, and ContentWriter.

The proxy protocol loop is a small, purpose-built engine in `internal/proxy`. It owns only the behavior recorderproxy uses:

- CONNECT handling
- TLS MITM interception
- tunneling back into HTTP request processing after downstream interception succeeds, including preserved upstream failures
- HTTP keep-alive and chained-proxy transport

The internal engine is part of recorderproxy's behavior surface and must be treated as owned operational code.

## Main Pieces

- `recorderproxy/recorderproxy.go`: assembles the filter chain, configures MITM, accepts downstream connections, and seeds each connection with a base recorderproxy context.
- `recorderproxy/context_init_filter.go`: creates request-scoped state, derives the effective target URI, and registers requests with BrowserController. CONNECT is handled here before any tunneled HTTP request exists.
- `recorderproxy/dns_lookup_filter.go`: resolves the target host once host and port are known.
- `recorderproxy/recorder_filter.go`: wraps request and response bodies so they stream through ContentWriter while crawl-log state is built up.
- `recorderproxy/error_handler_filter.go`: normalizes transport and proxy-layer failures into recorderproxy error codes and short-circuit responses.
- `recorderproxy/chained_proxy_filter.go`: rewrites non-CONNECT requests into absolute-form requests when a second proxy is configured.
- `internal/proxy`: owned request loop, filter chain, CONNECT handling, fixed-certificate MITM, tunneling, and phased errors.
- `mitmcert`: generation, loading, and validation of the immutable interception identity.

## Filter Order

The runtime filter chain is assembled in this order:

1. `NonproxyFilter`
2. `ContextInitFilter`
3. `ErrorHandlerFilter`
4. `DnsLookupFilter`
5. `RecorderFilter`
6. `ChainedProxyFilter` when `nextProxy` is configured
7. the internal proxy transport

The important consequences are that recorderproxy state exists before error handling, DNS lookup, and body wrapping; `ErrorHandlerFilter` wraps failures from the rest of the request path; and chained-proxy rewriting happens after recorder logic has identified the real target URI.

## Direct Transport Internals

The recorderproxy executable requires both `--cache-host`/`CACHE_HOST` and
`--cache-port`/`CACHE_PORT`; Squid is part of the supported runtime topology.
`NewRecorderProxy` retains an empty-`nextProxy` path for focused integration
tests, but that path is not a supported deployment mode.

Basic HTTP flow:

```text
Browser/client
	|
	|  absolute-form HTTP request
	v
RecorderProxy listener
	|
	|  base connection context
	v
Filter chain
	|
	+--> ContextInitFilter
	|      - create request-scoped RecordContext
	|      - derive target URI
	|      - register request with BrowserController
	|
	+--> DnsLookupFilter
	|      - resolve host -> IP through DnsResolver
	|
	+--> RecorderFilter
	|      - write request prolog to ContentWriter
	|      - wrap request body
	|      - round-trip upstream
	|      - wrap response body
	|      - stream response prolog/body/meta to ContentWriter
	|      - save CrawlLog through BrowserController
	|
	v
Origin server
	|
	v
Browser/client
```

Basic HTTPS flow:

```text
Browser/client
	|
	|  CONNECT target:443
	v
RecorderProxy
	|
	|  ContextInitFilter registers CONNECT with BrowserController
	|  internal proxy engine immediately returns CONNECT 200
	|  internal proxy engine establishes upstream TCP and MITMs TLS locally
	v
Tunneled HTTP request loop inside internal/proxy
	|
	|  GET /... over decrypted MITM connection
	v
Same filter chain as normal HTTP
	|
	v
Origin server
```

The CONNECT request itself is not the recorded fetch. It is the setup step that gives recorderproxy enough state to process the tunneled HTTPS request that follows.

The immediate CONNECT 200 is intentional. Chromium must be allowed to start its side of TLS even when upstream setup will fail. The internal engine is the sole owner of this acknowledgement; recorder filters must not send a second CONNECT response.

If upstream TCP dialing, upstream-proxy CONNECT, or upstream TLS fails, recorderproxy preserves the original phased failure while allowing the downstream TLS handshake to complete. The inner HTTPS request then enters the same filter chain with a failed transport. This creates the normal request-scoped record, terminates BrowserController and ContentWriter consistently, and returns a canonical HTTPS error response such as 503 to Chromium.

For failures that occur before an upstream connection exists, the engine completes downstream TLS when possible and passes the original phased failure into the tunneled HTTP transport. Upstream TLS failures use the same request path and reuse the successfully established downstream TLS connection.

If no downstream TLS connection or inner HTTP request exists, recorderproxy logs the phased connection failure and closes without manufacturing a crawl resource. It must never write a plaintext HTTP error after CONNECT 200.

## Runtime Flow With Squid

Recorderproxy still owns request state, DNS bookkeeping, ContentWriter streaming, cancellation behavior, and BrowserController interaction. Squid only becomes the upstream transport hop.

```text
Browser/client
	|
	|  HTTP request or CONNECT
	v
RecorderProxy
	|
	+--> ContextInitFilter / DnsLookupFilter / RecorderFilter
	|
	+--> ChainedProxyFilter
	|      - rewrite non-CONNECT requests to absolute-form
	|      - keep recorderproxy's target URI as the source of truth
	|
	+--> Dial()
	|      - connect to nextProxy instead of origin
	|      - for CONNECT, send an explicit upstream CONNECT request
	|
	v
Second proxy
	|
	v
Origin server
```

For HTTPS through a second proxy, there are effectively two CONNECT stages:

1. Browser to recorderproxy
2. Recorderproxy to second proxy

Recorderproxy still performs local MITM after the tunnel is established, so the internal request handling after CONNECT still looks like normal HTTP inside the proxy.

## Request State And Recording

- The downstream connection carries a base recorderproxy context.
- `ContextInitFilter` creates a child request context for each non-CONNECT request.
- `filterContext()` must preserve an existing request-scoped state handle instead of rebinding everything back to the connection-level context.
- `RecorderFilter` creates request and response wrappers that stream protocol headers, payload chunks, and final metadata to ContentWriter.
- `wrappedResponseBody.Close()` is where recorderproxy detects that the browser/client disappeared before response EOF and converts that into a cancel path.

The owned request loop makes it especially important to distinguish clearly between:

- connection-scoped state that survives across requests on the same downstream connection
- request-scoped state that must remain attached to the specific tunneled request being recorded

## Implications Of The Internal Proxy Engine

Owning the CONNECT/MITM loop has a few concrete implications:

- Bugs in CONNECT setup, TLS wrapping, and tunneled HTTP recursion are recorderproxy bugs.
- MITM wrappers must preserve access to the recorderproxy base context when connections are rewrapped.
- Response writes for MITM and tunneled traffic must go directly to the downstream connection. Buffering that write path can hide client disconnects until too late and break cancel semantics.
- Chained-proxy behavior has to be maintained explicitly: recorderproxy must rewrite requests into absolute form for the second proxy while still recording the original target URI.

In practice, `internal/proxy` is part of recorderproxy's core logic, not a general-purpose proxy library.

## Error Handling

Operational errors carry an explicit `proxy.ErrorPhase` and retain their original cause. Recorder-specific classification happens once in `recorderproxy/errors.go`, producing a canonical code, message, detail, phase, and connection/resource scope.

With a request-scoped `RecordContext`, the terminal path completes BrowserController at most once, terminates ContentWriter at most once, and returns a canonical HTTP response when the protocol still permits it. Without a `RecordContext`, the proxy logs and closes rather than inventing a resource result.

BrowserController remains the correlation partner, not the source of recorder errors. Chromium's inner request ID is registered normally, and recorderproxy's later `CompleteResource` supplies the authoritative crawl log even if Chromium reports `Network.loadingFailed` first.

## Runtime Metrics And Profiling

Recorderproxy exposes the standard Prometheus Go runtime and process metrics on
port `9302` at `/metrics`. The monitoring `ServiceMonitor` scrapes this as a
second endpoint on the harvester Service. The **Go Runtime / Profiling Signals**
Grafana dashboard can select recorderproxy by choosing the harvester job and the
instance whose address ends in `:9302`. The same dashboard works with any target
that exports the standard `client_golang` metrics.

The proxy also exports `recorderproxy_active_connections`,
`recorderproxy_open_sessions`, and
`recorderproxy_contentwriter_terminal_timeouts_total` for lifecycle and
finalization monitoring.

## Fixed MITM Identity

Runtime proxying requires a matching leaf certificate and private key through
`--mitm-cert-file`/`MITM_CERT_FILE` and
`--mitm-key-file`/`MITM_KEY_FILE`. The identity is validated before listeners
open and is shared by every listener without generating hostname certificates.
The same identity is presented when a TLS ClientHello omits SNI. BrowserController
limits Chromium's certificate exception to this leaf's SPKI, while recorderproxy
intentionally does not verify the certificate presented by its upstream Squid
connection.
The container includes `/generate-mitm-cert` for creating the pod-scoped leaf
used by the Kubernetes harvester deployment. ContentWriter terminal operations
are bounded by `--finalization-timeout`, which defaults to 30 seconds.

Prometheus metrics show trends such as live heap, process RSS, allocation rate,
GC frequency, goroutines, CPU, and file descriptors. Prometheus does not store
pprof profiles. Heap, CPU, and goroutine profiles must be fetched from the
process while the problem is occurring.

The pprof server is disabled by default and listens only on
`127.0.0.1:6060` when enabled. Enable it temporarily in the deployment overlay:

```yaml
env:
  - name: PROFILING_ENABLED
    value: "true"
```

Forward the profiling port without exposing it through a Kubernetes Service:

```sh
kubectl -n <namespace> port-forward pod/<harvester-pod> 6060:6060
```

Capture a heap baseline early in the crawl and another profile as memory grows:

```sh
curl -o recorderproxy-baseline.heap http://127.0.0.1:6060/debug/pprof/heap
curl -o recorderproxy-high.heap http://127.0.0.1:6060/debug/pprof/heap
go tool pprof -top recorderproxy-high.heap
go tool pprof -http=127.0.0.1:0 -diff_base=recorderproxy-baseline.heap recorderproxy-high.heap
```

Capture CPU and goroutine profiles when the dashboard shows pressure:

```sh
curl -o recorderproxy.cpu 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
curl -o recorderproxy.goroutine http://127.0.0.1:6060/debug/pprof/goroutine
go tool pprof -http=127.0.0.1:0 recorderproxy.cpu
go tool pprof -top recorderproxy.goroutine
```

Profiles can contain URLs, headers, and other process data. Keep the pprof port
out of the Service, store captures as sensitive operational artifacts, and turn
profiling off after the investigation.

## Development

This directory is an independent Go module. Run tests from here:

```sh
go test ./internal/proxy ./mitmcert ./recorderproxy -count=1
go test ./... -count=1
```

The integration tests use local TCP listeners and in-memory gRPC service mocks. See `AGENTS.md` for the module map, invariants, and focused validation guidance.
