# veidemann-browser-controller

## Build container

From the repository root:

    docker build ./browser-controller

## Test

    go test ./...

## Service clients

`main.go` creates and owns the gRPC connections to ContentWriter, Frontier, and
RecorderControl. Startup configuration uses accessors in `options.go`.
The Frontier adapter and screenshot writer receive generated service clients;
they do not manage connections. Connections close after active harvests have
finished and reported to Frontier. The Frontier connection retains its
OpenTracing unary interceptor.

## Chrome DevTools client

The controller uses the released `chromedp` module with typed actions and
protocol commands. Its `chromedp/remote` package is part of the core module;
do not require the former standalone module, which causes an ambiguous import.
There is no local fork or module replacement for chromedp.
See the upstream [migration guide](https://github.com/chromedp/chromedp/blob/main/docs/MIGRATION.md)
and [API guide](https://github.com/chromedp/chromedp/blob/main/docs/API.md).

`session/remote.go` wraps the remote package's WebSocket transport to dispatch
events in their original order. Browser activity and the main-document load signal use
that order across different event methods. Recording completion does not depend
on Chrome events. Event handlers must send protocol commands from a
goroutine so they do not block the transport reader.

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

The browser's default profile uses a closed loopback proxy. Each harvest gets a
fresh browser context configured with its recorder slot. Browser-wide requests using the global HTTP proxy therefore cannot reach the
recorder or fall back to direct connections through that proxy configuration.
The launch configuration also disables network-time and AI eligibility requests.
This prevents the observed traffic from contaminating captures; it does not
promise that Chromium stops internally initiating every background request.

## Bounded readiness and recorder completion

Render readiness and recorder completion have separate deadlines. Documents,
images, scripts, stylesheets, fonts, text tracks, and unknown CDP resource types
are required for readiness. Existing XHR/Fetch and other background exclusions
remain unchanged. Media is classified by CDP resource type.

`--media-settle-timeout` / `MEDIA_SETTLE_TIMEOUT` defaults to `5s`. This is one
allowance starting after the main document loads, shared by ON_LOAD script waits
and final settling. Activity can extend network quiet during that allowance;
an idle media transfer does not require EOF. Retries cannot restart the
allowance. Set it to `0s` to disable media waiting. Required progress continues
to postpone readiness, subject to the page-load deadline. An unfinished required
request ends collection after a bounded stall. A browsing deadline also ends
collection normally, then capture and recorder finalization determine success.

The controller emits one `Harvest finished` summary after browser teardown,
including the outcome, collection stopping reason, duration in milliseconds,
recorded resources/bytes, outlinks, screenshot status and slot reusability.
Recorderproxy emits `Recording finished` with its authoritative
accepted/finished transfers and persistence outcomes. Browser observations and
recorder totals are distinct; there is no expected-versus-received request count
in browser-controller.

The browser/recorder integration suite includes locally generated finite PCM
media, range transfers, repeated media cancellation/reload, a slow image, and
EventSource. It checks recorded outcomes and bounded duration; media request
counts may vary between browser versions. Run with Docker or Podman available:

    go test -tags=integration ./server ./session

Child targets use the flattened session supplied by `Target.attachedToTarget`.
The transport shares one wire command ID allocator with chromedp and restores
root command IDs when forwarding replies. Events retain wire order. Child setup
registers listeners first, enables network observation, disables cache, sets the
attempt header, and enables recursive paused auto-attachment. Only pages and
iframes enable page events. Setup executes no JavaScript and never attaches a
second time; each paused session resumes once configuration succeeds.

Setup has a ten-second deadline bounded by its parent context. Pending setup
blocks readiness, drains inside the capture budget, and quiesces before recorder
finalization. Detachment removes the session and its descendants, listeners,
pending commands, and readiness requests. A target can reattach with a new
session ID. A live setup failure cancels the harvest and includes target ID,
session ID, target type, failing command, and the original error; recorder
finalization still runs. Capture errors name the failing operation.

The CI acceptance gate includes mock CDP lifecycle/routing tests and
`TestChildTargetLifecycleArtifacts`, which uses distinct local sites to require
separate iframe sessions. It checks recursive attachment, worker initialization,
child removal, recorded resources, stripped internal headers, and main-page
artifacts. Run the focused checks with:

    go test -race ./session -run 'Test(Child|SessionTransport|Remote)'
    go test -tags=integration ./server -run '^TestChildTargetLifecycleArtifacts$'

For supplementary live harvests, run the full harvesting stack against
`https://ghg.no/` and `https://aquamanias.no/`. Check that the recorder
accepted/finished counts agree and stream completion confirms log persistence,
and canceled transfers retain their specific outcomes without independently
failing the attempt. Success does not establish that the complete video was saved.

The integration harness can also exercise these URLs using its real browser and
recorderproxy with mocked storage and service dependencies:

    go test -tags=integration ./server -run '^TestSession_Fetch/live-' -v -args \
      -harvest-urls=https://ghg.no/,https://aquamanias.no/,https://www.nb.no/

Live scenarios are constructed independently with the normal 60-second browsing
budget. They require a recorded main document, page log, screenshot, and extracted
links. Live-site content can change, so these optional checks supplement the
deterministic fixtures rather than acting as the CI acceptance gate.

## Harvest ownership

See [the attempt contract](../proto/recorderproxy/v1/README.md). Each fixed worker
owns one recorder slot and creates a fresh browser session per harvest. Chrome
events control readiness: a root-document load signal gates ON_LOAD scripts,
and one activity tracker handles required progress and bounded media. Root-page
document interception adds an attempt-scoped main-document header using the
existing root frame ID. The recorder strips it and owns all recording outcomes.
There is no session registry, transfer ledger, or request-correlation RPC.

Recorderproxy writes crawl and page logs. Browser-controller passes the captured
URI and outlinks through client-streaming FinishHarvest on port 7780, receives the selected main
recording and totals, writes the screenshot, and reports to Frontier. Log-service
configuration belongs to recorderproxy. Browsing cutoff is normal when a recorded
main document and complete artifact bundle exist. Capture defaults to 30 seconds,
recorder finalization to a 45-second caller budget, and screenshot persistence to
30 seconds. A non-reusable recorder reply or browser cleanup timeout stops the
workers after the affected harvest is reported.

Startup reconciles every recorder slot with `RecoverSlot` before becoming ready
or taking work from Frontier. It closes orphaned attempts and waits for remaining
terminal work; failed recovery exits without consuming another queued page.
The worker retains the admission token returned by each successful Begin for
its next session. After an ambiguous Begin, it reconciles the slot before taking
another Frontier item. The attempt UUID on HTTP/HTTPS requests prevents a lingering browser from entering
a recovered slot. New frames and workers are paused until their token is set.
Each recorder slot set must have one exclusive browser-controller process.

The recording proxy is assigned to a fresh browser context with
`Target.createBrowserContext`. Workers and child frames inherit that context.
Chromium's default profile instead receives the closed loopback proxy endpoint
`127.0.0.1:1`; the observed account/push HTTP requests are kept outside the attempt. This isolates browser-internal traffic without a Chrome request
allowlist. Each attempt disposes its browser context before the slot is reused.

`PROXY_COUNT` is the actual simultaneous harvest capacity (default 1). Slots are
numbered `1..N` and use `PROXY_PORT + slot - 1`. The controller no longer serves
registration/completion callbacks or opens a callback listening port. Robots
recording is independent of the browser-controller process. Deploy this component
with recorderproxy, robots-evaluator and their configuration as one coordinated
release; see [cutover and rollback](../deploy/k8s/base/harvester/README.md).

## Reading harvest logs

Both services emit JSON. The harvester base uses INFO for browser-controller and
recorderproxy; the development overlay enables DEBUG for both. Recorderproxy
uses `LOG_FORMATTER=json` (also its executable default). Existing format options
remain available for standalone recorderproxy use.

INFO follows each harvest through configuration, browser setup, main-document
load, page scripts, readiness, artifact capture, recorder finalization, screenshot
persistence, and browser teardown. A phase lasting more than ten seconds emits
`Harvest still in progress` with its elapsed time and remaining deadline. One
`Harvest finished` message follows teardown, including failed or canceled
harvests. `Screenshot stored` means the persistence operation succeeded;
`screenshot=disabled` means the crawl did not request one.

Use `harvestId` to join both services. It identifies one page collection, without
implying a retry. The existing protocol still calls this identifier `AttemptId`;
only the logging vocabulary changes. `component` is always the service name.
`slot` is the reusable worker/proxy slot; `pageUrl` is the original requested page
and `url` is an individual resource. `cdpSessionId` and `browserRequestId` identify
browser observations; `recorderRequestId` identifies recorder work. These are
separate identifier spaces. Harvest connections use a compact 22-character
fingerprint of the harvest UUID as Browserless's `trackingId`.
`Browser ready` logs both the full `harvestId` and `browserTrackingId` at INFO so
Browserless can be correlated with the other services. The compact ID fits
Browserless v2.57.3's length limit and avoids collisions when a slot is reused
before its previous provider session finishes closing. Standalone browser
diagnostics without a harvest retain a slot-based ID.
Times such as `durationMs`, `elapsedMs`, `remainingMs`, and request `ageMs` /
`idleMs` are numeric milliseconds. `durationMs` measures the operation named by
the message. Recorderproxy's `Recording finished` includes its total lifecycle
`durationMs` and the separate `finalizationDurationMs`; resource recordings retain
the fetch measurement as `fetchDurationMs`. Scripts use `scriptPhase` names such
as `ON_LOAD`. `httpStatus` is an HTTP status; `errorCode` is a recorder error code.

`outcome` describes the final harvest result; `stoppingReason` explains why
collection ended; `collectionPhase` identifies the operation that was active.
A successful harvest may end because of `quiet`,
`media_allowance_elapsed`, `required_request_stalled`, or `page_load_deadline`.
The last two also log required-request/setup counts and the five oldest pending
required requests. Their URLs, types, ages, and last-progress times explain what
was still outstanding. `canceled`, `harvest_deadline`, and `operational_failure`
cover interruption or failure before normal collection ends. Failure summaries
include `phase`, `errorCode`, and the original error. A later capture/persistence
failure retains the collection stopping reason and identifies its failing phase.

DEBUG emits one `Browser request finished` per observed terminal browser event
and one `Resource recording finished` per admitted recorder transfer. Browser
results include method, URL, resource type, HTTP status when available, outcome,
and timing. `browserRequestsFailed` counts CDP loading failures;
`browserHttpErrors` counts observed HTTP error responses. An HTTP 503 can arrive
as a normally completed browser transfer while recorderproxy records an upstream
connection failure. These are different observations, and neither count alone
is a harvest failure. Recorder outcomes follow ContentWriter termination and
crawl-log persistence, with a confirmed WARC ID when available. An archived HTTP
404 can have `outcome=recorded`; that differs from `outcome=persistence_failed`.
The recorder summary reports resource failures/cancellations separately from
finalization failure and states whether the page log was stored. Only the
controller's `Harvest finished` describes the whole harvest, including artifacts
and browser teardown.

TRACE adds starts, redirects, responses, admission, DNS, HTTPS tunnel setup,
connection open/close, body closure, and routine child session transitions. Set
`LOG_LEVEL=TRACE` in either service for that detail. Connection messages identify
the browser or upstream leg through `phase` and include local/remote addresses.
After child configuration `paused=false` describes current state; `resumed=true`
confirms that the controller resumed a target initially waiting for the debugger.
Unused CDP events are silent; malformed consumed events remain contextual WARNs.
Diagnostic URLs are limited to 512 UTF-8 bytes, embedded `data:` content is omitted,
and the encoded browser launch endpoint is not logged. Request diagnostics do
not dump headers or bodies or emit progress for individual payload chunks.

Follow either container, adding your namespace with `-n` if needed:

```sh
kubectl logs -f deployment/harvester -c browser-controller | jq -c .
kubectl logs -f deployment/harvester -c recorderproxy | jq -c .
```

Filter both containers for one harvest (replace the example UUID):

```sh
kubectl logs deployment/harvester --all-containers=true --prefix=true --since=10m |
  jq -Rc --arg harvestId '1f6fc607-0051-4785-a52a-200070e5e2ae' '
    sub("^[^{]*"; "") | fromjson? | select(.harvestId == $harvestId)'
```

For an INFO-only view of DEBUG output, append
`jq -c 'select(.level != "DEBUG" and .level != "TRACE")'`. To investigate one
resource, filter `.url`; browser failures and recorder outcomes share the
harvest ID and resource URL without introducing a request-correlation protocol.
