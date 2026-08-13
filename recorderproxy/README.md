# veidemann-recorderproxy

## Overview

`recorderproxy` is the HTTP/HTTPS capture proxy used by Veidemann to fetch traffic, record request and response material, and coordinate side effects with BrowserController, DNS resolver, and ContentWriter.

At the current stage, the proxy is built on the newer `github.com/getlantern/proxy/v3` request loop, but it no longer relies on upstream for CONNECT/MITM handling. Upstream removed the old built-in MITM constructor path, so this module now carries a small local compatibility layer in `proxycompat` that restores:

- CONNECT handling
- TLS MITM interception
- tunneling back into HTTP request processing after downstream interception succeeds, including preserved upstream failures
- request-aware upstream connection reuse

That local compatibility layer is not just plumbing. It is now part of recorderproxy's behavior surface and must be treated as owned code.

## Main Pieces

- `recorderproxy/recorderproxy.go`: assembles the filter chain, configures MITM, accepts downstream connections, and seeds each connection with a base recorderproxy context.
- `recorderproxy/context_init_filter.go`: creates request-scoped state, derives the effective target URI, and registers requests with BrowserController. CONNECT is handled here before any tunneled HTTP request exists.
- `recorderproxy/dns_lookup_filter.go`: resolves the target host once host and port are known.
- `recorderproxy/recorder_filter.go`: wraps request and response bodies so they stream through ContentWriter while crawl-log state is built up.
- `recorderproxy/error_handler_filter.go`: normalizes transport and proxy-layer failures into recorderproxy error codes and short-circuit responses.
- `recorderproxy/chained_proxy_filter.go`: rewrites non-CONNECT requests into absolute-form requests when a second proxy is configured.
- `proxycompat`: local compatibility layer over newer getlantern proxy/v3 behavior, especially for CONNECT and MITM.

## Filter Order

The runtime filter chain is assembled in this order:

1. `NonproxyFilter`
2. `ContextInitFilter`
3. `ErrorHandlerFilter`
4. `DnsLookupFilter`
5. `RecorderFilter`
6. `ChainedProxyFilter` when `nextProxy` is configured
7. proxycompat's upstream transport

The important consequences are that recorderproxy state exists before error handling, DNS lookup, and body wrapping; `ErrorHandlerFilter` wraps failures from the rest of the request path; and chained-proxy rewriting happens after recorder logic has identified the real target URI.

## Flow Without A Second Proxy

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
	|  proxycompat immediately returns CONNECT 200
	|  proxycompat establishes upstream TCP and MITMs TLS locally
	v
Tunneled HTTP request loop inside proxycompat
	|
	|  GET /... over decrypted MITM connection
	v
Same filter chain as normal HTTP
	|
	v
Origin server
```

The CONNECT request itself is not the recorded fetch. It is the setup step that gives recorderproxy enough state to process the tunneled HTTPS request that follows.

The immediate CONNECT 200 is intentional. Chromium must be allowed to start its side of TLS even when upstream setup will fail. Proxycompat is the sole owner of this acknowledgement; recorder filters must not send a second CONNECT response.

If upstream TCP dialing, upstream-proxy CONNECT, or upstream TLS fails, recorderproxy preserves the original phased failure while allowing the downstream TLS handshake to complete. The inner HTTPS request then enters the same filter chain with a failed transport. This creates the normal request-scoped record, terminates BrowserController and ContentWriter consistently, and returns a canonical HTTPS error response such as 503 to Chromium.

For failures that occur before MITM has an upstream connection, the small `unavailableUpstreamConn` in `proxycompat` adapts the failure to getlantern/mitm's eager `net.Conn` API. It owns no socket, buffer, or goroutine and returns the preserved failure from all I/O. It must not be treated as a successful or fake upstream connection. Upstream TLS failures arrive from MITM itself and reuse the successfully established downstream TLS connection.

If no downstream TLS connection or inner HTTP request exists, recorderproxy logs the phased connection failure and closes without manufacturing a crawl resource. It must never write a plaintext HTTP error after CONNECT 200.

## Flow With A Second Proxy

When `nextProxy` is configured, recorderproxy still owns request state, DNS bookkeeping, ContentWriter streaming, cancellation behavior, and BrowserController interaction. The second proxy only becomes the upstream transport hop.

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

The recent migration made this especially important because newer getlantern request handling reuses connection state aggressively. Recorderproxy now has to distinguish clearly between:

- connection-scoped state that survives across requests on the same downstream connection
- request-scoped state that must remain attached to the specific tunneled request being recorded

## Implications Of The Local CONNECT/MITM Layer

Adding CONNECT/MITM support back on top of newer `proxy/v3` has a few concrete implications:

- Recorderproxy now owns behavior that used to be hidden inside upstream getlantern proxy releases.
- Bugs in CONNECT setup, TLS wrapping, and tunneled HTTP recursion are now local recorderproxy bugs, not just dependency quirks.
- MITM wrappers must preserve access to the recorderproxy base context when connections are rewrapped.
- Response writes for MITM and tunneled traffic must go directly to the downstream connection. Buffering that write path can hide client disconnects until too late and break cancel semantics.
- Chained-proxy behavior has to be maintained explicitly: recorderproxy must rewrite requests into absolute form for the second proxy while still recording the original target URI.

In practice, `proxycompat` should be treated as part of recorderproxy's core logic, not as a throwaway shim.

## Error Handling

Operational errors carry an explicit `proxycompat.ErrorPhase` and retain their original cause. Recorder-specific classification happens once in `recorderproxy/errors.go`, producing a canonical code, message, detail, phase, and connection/resource scope.

With a request-scoped `RecordContext`, the terminal path completes BrowserController at most once, terminates ContentWriter at most once, and returns a canonical HTTP response when the protocol still permits it. Without a `RecordContext`, the proxy logs and closes rather than inventing a resource result.

BrowserController remains the correlation partner, not the source of recorder errors. Chromium's inner request ID is registered normally, and recorderproxy's later `CompleteResource` supplies the authoritative crawl log even if Chromium reports `Network.loadingFailed` first.

## Runtime Metrics And Profiling

Recorderproxy exposes the standard Prometheus Go runtime and process metrics on
port `9302` at `/metrics`. The monitoring `ServiceMonitor` scrapes this as a
second endpoint on the harvester Service. The **Go Runtime / Profiling Signals**
Grafana dashboard can select recorderproxy by choosing the harvester job and the
instance whose address ends in `:9302`. The same dashboard works with any target
that exports the standard `client_golang` metrics.

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
go test ./proxycompat ./recorderproxy -count=1
go test ./... -count=1
```

The integration tests use local TCP listeners and in-memory gRPC service mocks. See `AGENTS.md` for the module map, invariants, and focused validation guidance.
