# Recorderproxy Agent Guide

## Scope

These instructions apply to the `recorderproxy` Go module. They supplement the repository-level `AGENTS.md`.

## What This Module Does

Recorderproxy is the crawler's HTTP/HTTPS interception proxy. Chromium sends its traffic here. For each recordable inner HTTP request, recorderproxy:

1. correlates the request with BrowserController;
2. resolves and records the target address through DnsResolver;
3. streams request and response records to ContentWriter;
4. sends one authoritative `CrawlLog` to BrowserController; and
5. returns the origin response, or a canonical proxy error response, to Chromium.

Recorderproxy owns the crawl result. BrowserController may observe Chromium network events such as `Network.loadingFailed`, but it does not derive the recorder error or manufacture the crawl log from those events.

This is an independent Go module. Run Go commands from this directory.

## Start Here

The main assembly point is `recorderproxy/recorderproxy.go`. Read it before changing request flow. It constructs the filter chain and configures the local `proxycompat` CONNECT/MITM loop.

The effective filter order, outermost to innermost, is:

1. `NonproxyFilter`
2. `ContextInitFilter`
3. `ErrorHandlerFilter`
4. `DnsLookupFilter`
5. `RecorderFilter`
6. `ChainedProxyFilter`, when a next proxy is configured
7. the proxycompat transport

Order is behavior. In particular:

- `ContextInitFilter` must run before any filter that needs a `RecordContext`.
- `ErrorHandlerFilter` must wrap DNS, recording, chained-proxy, and transport failures.
- `RecorderFilter` must see the real crawler target before `ChainedProxyFilter` rewrites a request for the next proxy.

Do not reorder filters as cleanup without proving HTTP, HTTPS, error, cancellation, and chained-proxy behavior.

## Directory Map

- `main.go`, `options.go`: process startup, flags/environment, gRPC connections, and the pool of listeners. Proxy `i` listens on base port plus `i`.
- `recorderproxy/`: recorder-specific filters, dialing, request/response recording, error classification, and listener lifecycle.
- `proxycompat/`: locally owned compatibility layer around `github.com/getlantern/proxy/v3` and `github.com/getlantern/mitm`. It owns CONNECT acknowledgement, MITM, tunneled HTTP recursion, and phase-aware connection errors.
- `context/`: connection/request state plus BrowserController and ContentWriter client lifecycles. This package is recorderproxy state, not ordinary immutable Go context values.
- `errors/`: canonical recorder error codes and `ProxyError` constructors.
- `serviceconnections/`: long-lived gRPC connections to BrowserController, DnsResolver, and ContentWriter.
- `constants/`: crawler headers and record content-type constants.
- `testutil/`: in-memory gRPC mocks, origin servers, and the secondary proxy used by integration tests.
- `proxyutil/`: small proxy client utility, separate from the runtime proxy implementation.
- `logger/`, `tracing/`: logging and gRPC tracing helpers.

## HTTP Request Flow

For plain HTTP, Chromium sends an absolute-form proxy request. `ContextInitFilter` creates a request-scoped `RecordContext` and registers it with BrowserController. DNS resolution, request recording, the round trip, response recording, ContentWriter finalization, and BrowserController completion then happen through the normal filter chain.

Record numbers are stable protocol:

- `0`: HTTP request
- `1`: HTTP response

The body wrappers stream the same bytes that travel through the proxy. EOF finalizes sizes and digests. Closing a response body before EOF is treated as a browser/client cancellation. Close and terminal RPC paths must remain idempotent.

## HTTPS And CONNECT Flow

CONNECT is setup, not the recorded HTTPS resource.

The intended flow is:

1. `ContextInitFilter` registers the browser's CONNECT with BrowserController and stores connection metadata.
2. With `OKWaitsForUpstream: false`, proxycompat returns CONNECT 200 immediately. This prevents Chromium from blocking on upstream setup. Proxycompat is the sole owner of this acknowledgement.
3. Proxycompat dials the origin or next proxy and performs local MITM.
4. The decrypted inner request, such as `GET /`, re-enters the normal filter chain and gets its own `RecordContext`.

Do not add another CONNECT 200 in `RecorderFilter` or recorder-specific error handling.

### Failed Upstream During CONNECT

An upstream TCP, upstream-proxy CONNECT, or upstream TLS failure must not be hidden in context state and must not be replaced by `EOF` or `closed pipe`.

The current model is deliberate:

- `Dial` returns the real error.
- For TCP dial and upstream-proxy CONNECT failures, `proxycompat.unavailableUpstreamConn` adapts the failure to getlantern/mitm's eager `net.Conn` API. It owns no socket, buffer, or goroutine, and every I/O operation returns the original error. It is not a simulated successful connection.
- For an upstream TLS failure, MITM returns the failure together with the usable downstream TLS connection.
- MITM completes the downstream/browser TLS handshake when possible.
- Once the inner HTTPS request is available, proxycompat sends it through the normal filter chain with a deterministic failed transport carrying the original phased error.
- The normal request path then registers the resource, starts/terminates ContentWriter consistently, completes BrowserController once, and returns a TLS-wrapped canonical error response to Chromium.

If downstream TLS or inner-request parsing fails before a `RecordContext` exists, log the phased connection failure and close. Do not invent a crawl resource.

Never write a plaintext HTTP error after CONNECT 200; Chromium is speaking TLS at that point.

## Error Model

`proxycompat.ErrorPhase` identifies where an operational error occurred. Current phases cover request reading/filtering, upstream dial, upstream-proxy CONNECT, downstream and upstream TLS, inner HTTP parsing, HTTP round trip, response writing, and raw tunneling.

`recorderproxy.RecorderFailure` is the canonical internal classification and contains:

- recorder error code;
- canonical message;
- stable detail;
- phase;
- connection or resource scope; and
- original cause.

Classification belongs in `recorderproxy/errors.go`. Prefer typed errors and `errors.Is`/`errors.As`. String matching is a fallback for untyped dependency/protocol text, notably Squid error headers. Preserve the original error through wrapping.

There are two terminal policies:

- With a `RecordContext`, complete BrowserController at most once, terminate ContentWriter at most once, and return the canonical HTTP response when legal.
- Without a `RecordContext`, log and close; do not send `CompleteResource` for a request that was never established.

`handleRequestError` implements the request-scoped terminal path. `RecordContext.finalizeCrawlLog` and the ContentWriter session contain the idempotency guards. Avoid adding parallel completion paths.

## Context And Correlation

Connections are repeatedly wrapped by proxy, TLS, and MITM code. `filter_context.go` walks wrappers to recover the base proxy context. The `context` package stores mutable state behind a state handle so request state survives those wrappers.

Keep these scopes distinct:

- connection metadata established by CONNECT and reused by the tunneled request;
- request metadata and `RecordContext`, which must be fresh per inner HTTP request.

Crawler headers include the Chromium request ID, crawl execution ID, job execution ID, and collection ID. `RecordContext.Init` captures them and removes them before forwarding upstream. The request ID is how BrowserController correlates recorderproxy's later `CompleteResource` with Chromium's request.

Do not restore a context-based error side channel. Transport errors travel as errors with phases; context is for request/session metadata.

## External Services

- BrowserController: `RegisterResource` decides whether a request proceeds and provides crawl metadata; `CompleteResource` receives the authoritative crawl log.
- DnsResolver: resolves the target and records the IP associated with the crawl result.
- ContentWriter: receives one client-streamed record consisting of protocol headers, payload chunks, and final metadata, or one terminal cancel.

No protobuf or service API change should be needed for ordinary recorderproxy error handling.

## Chained Proxy Behavior

The runtime calls the configured next proxy the cache in flags (`cache-host` and `cache-port`). When configured:

- `Dial` connects to the next proxy instead of the origin.
- HTTPS sends an explicit upstream CONNECT before local MITM.
- Plain HTTP is rewritten to absolute form by `ChainedProxyFilter`.
- The original crawler URI remains the source of truth for recording and crawl logs.

Squid's `X-Squid-Error` is an untyped protocol boundary. Normalize it once into recorder errors; do not spread Squid-specific parsing through filters.

## Certificates And TLS

Recorderproxy MITMs every CONNECT (`ShouldMITM` returns true). Chromium must trust or ignore the generated interception certificate. The compatibility layer currently uses `/tmp/rpcert.pem` and intentionally skips verification of upstream certificates. The `--ca` and `--ca-key` options exist, but startup does not currently apply them; do not document them as active without implementing and testing that path.

## Tests And Validation

Use the narrowest package first, then the whole module:

```sh
go test ./proxycompat ./recorderproxy -count=1
go test ./... -count=1
```

Some integration tests bind local TCP listeners. In restricted environments they may require permission to open local sockets.

Important coverage lives in:

- `proxycompat/errors_test.go`: phase/cause preservation and the unavailable-upstream adapter.
- `recorderproxy/errors_test.go`: canonical classification.
- `recorderproxy/recorderproxy_test.go`: HTTP/HTTPS behavior plus BrowserController, DnsResolver, and ContentWriter interactions.
- `recorderproxy/dial_test.go`: dial timeout behavior.

When changing CONNECT or terminal handling, verify at least:

- successful HTTP and HTTPS;
- direct connection refusal for HTTP and HTTPS;
- upstream TLS failure;
- chained-proxy CONNECT failure if touched;
- client cancellation/response streaming failure;
- exactly one BrowserController completion and one ContentWriter terminal action; and
- no plaintext response after CONNECT 200.

The large integration test intentionally checks full RPC sequences. A changed expected sequence should be justified by the lifecycle, not merely updated to match output.

## Change Guidelines

- Keep `proxycompat` small, explicit, and dependency-focused. Recorder policy belongs in `recorderproxy/`; protocol-loop mechanics belong in `proxycompat/`.
- Preserve error causes and phases at package boundaries.
- Keep terminal operations idempotent.
- Do not create goroutines or in-memory connections just to turn an upstream error into later control flow.
- Do not fabricate crawl logs before an inner request and `RecordContext` exist.
- Be careful with shared downstream connections: request-scoped state must not leak between HTTP keep-alive requests or between CONNECT and the inner request.
- Avoid broad formatting changes in the large integration test.
- Run Go commands from this module, not from the monorepo root.
