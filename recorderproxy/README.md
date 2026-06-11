# veidemann-recorderproxy

## Overview

`recorderproxy` is the HTTP/HTTPS capture proxy used by Veidemann to fetch traffic, record request and response material, and coordinate side effects with BrowserController, DNS resolver, and ContentWriter.

At the current stage, the proxy is built on the newer `github.com/getlantern/proxy/v3` request loop, but it no longer relies on upstream for CONNECT/MITM handling. Upstream removed the old built-in MITM constructor path, so this module now carries a small local compatibility layer in `proxycompat` that restores:

- CONNECT handling
- TLS MITM interception
- tunneling back into HTTP request processing after MITM succeeds
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
2. `TracingInitFilter`
3. `ContextInitFilter`
4. `DnsLookupFilter`
5. `RecorderFilter`
6. `ErrorHandlerFilter`
7. `ChainedProxyFilter` when `nextProxy` is configured

The important consequence is that recorderproxy state exists before DNS lookup and body wrapping, and chained-proxy rewriting happens after recorder logic has already identified the real target URI.

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
	|  proxycompat establishes upstream TCP connection
	|  proxycompat MITMs TLS locally
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
