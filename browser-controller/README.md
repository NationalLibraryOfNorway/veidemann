# veidemann-browser-controller

## Build container

Need to use the context of the monorepo root when building the container:

    docker build -f Dockerfile ..

## Test

    go test ./...

## Recorder certificate trust

When `--proxy-host`/`PROXY_HOST` is configured, browser-controller also requires
`--recorder-cert-file`/`RECORDER_CERT_FILE`. It reads the recorderproxy leaf at
startup and scopes Chromium's certificate exception to that public key. Startup
fails if the certificate is missing, expired, a CA, or unsuitable for TLS
server authentication.

## Browser scripts

Runtime-invoked browser scripts may return either their final JSON value or a Promise resolving to that value.

- `ON_LOAD` scripts are executed via the Chrome runtime `CallFunctionOn` API and may resolve asynchronously to the usual `{waitForData, next, data}` object.
- `EXTRACT_OUTLINKS` scripts are executed via the Chrome runtime `Evaluate` API and may resolve asynchronously to the usual `[]string` outlink list.
- Plain functions that return `Promise.resolve(...)` are supported in addition to `async` functions.

`ON_NEW_DOCUMENT` scripts are different: they are injected with `page.AddScriptToEvaluateOnNewDocument` and run fire-and-forget, so their return values are not observed by browser-controller.

## Run integration test

The integration test is useful to validate compatibility between the `browserless` container image and the `chromedp` library.

### Using docker

    go test --tags=integration -run TestSession_Fetch ./server

### Diagnose lazy images on a URL

The browser-only diagnostic runs without recorderproxy or the other Veidemann
services. It compares DOM image state with Chrome network events and prints a
bounded JSON report. It is skipped unless `-diagnostic-url` is provided.

    go test --tags=integration -run '^TestURLDiagnostics$' -v ./session -args \
      -diagnostic-url=https://www.nb.no/ \
      -diagnostic-target='20260624GKG116' \
      -diagnostic-modes=isolated-scroll,main-scroll,scroll-into-view,eager

Add `foreground-scroll` to compare the normal hidden browserless target with a
target activated through CDP `Page.bringToFront`.

Use the full `TestSession_Fetch` harness when Chrome reports an image request
but the corresponding crawl log or WARC record is missing.

### Using podman

> Note that you have to manually clean up containers after every run with using podman as container provider because the container reaper (ryuk) is disabled.

    # Setup podman socket
    systemctl --user start podman

    # Run integration test
    DOCKER_HOST="unix:///run/user/$UID/podman/podman.sock" go test -run TestSession_Fetch --tags=integration ./server -provider=podman

## Chromium background traffic

Chromium may send browser-owned requests, for example to Google account and
check-in services, even when the usual background-networking launch flags are
enabled. Do not classify these requests by hostname: a crawl may legitimately
visit the same service.

The stable distinction is request correlation. Page traffic observed by
browser-controller receives a `veidemann_reqid` through CDP Fetch interception;
browser-owned traffic outside the page target does not. Recorderproxy must still
allow an uncorrelated CONNECT because Chromium creates CONNECT before the inner
request is visible. Once decrypted, a browser-proxy request without
`veidemann_reqid` is cancelled by BrowserController (with the existing OPTIONS
exception) and must not produce a WARC record or crawl log.

Possible follow-up work:

- classify this explicitly as unattributed browser traffic and log it at debug
  level with a metric;
- verify the cancellation path closes its recorder context exactly once without
  starting ContentWriter or sending `CompleteResource`; and
- if avoiding the upstream HTTPS handshake is important, investigate lazy
  upstream dialing after the downstream TLS handshake and inner request
  classification. Hostname blocking is not an acceptable substitute.
