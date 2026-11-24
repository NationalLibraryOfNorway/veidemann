# Recorderproxy / browser-controller ownership review

Reviewed on 2026-10-12 against `d772a3dc` (`browser-controller: baseline`).
Scope: the working-tree changes, including untracked implementation and test
files, in browser-controller, recorderproxy, the shared recorder API, and the
robots/build/deployment changes required by this refactor.

**Current verdict: R1–R5 are addressed; R1's live Kubernetes restart validation
remains a staging check.** R5 is closed under the explicitly accepted recorder-only
inference rules and documented ambiguity, not a guarantee of complete browser
ancestry. The standard suites, new ancestry regressions, and original review
reproductions pass. No production rollout was performed.

The original review found five gaps. Four had real Chromium reproductions;
the recovery gap had a recorder lifecycle reproduction and a deployment
consequence inferred from the manifest and Kubernetes lifecycle documentation.
The original review did not change service implementation files; the follow-ups
below describe subsequent fixes. Finding descriptions retain the original evidence.

## Findings

**2026-10-14 follow-up — R5 addressed:** recorderproxy now assigns immutable
paths from an attempt-local index of admitted transfers and pending HTTP redirect
targets. Initialization establishes ancestry before cancellation can snapshot a
crawl log, and redirect headers are observed before they reach Chromium. Children
can inherit a streaming parent's path without waiting for recording completion.
Finalization discards the index and ignores late response observations.

Network URL normalization ignores fragments and resolves relative redirects.
Unique compatible 301/302/303/307/308 targets append `R`, including repeated-URL
redirects and method rewriting. Exact referrer matches inherit the earlier path
when all matching admissions agree. Ambiguous redirect candidates are discarded;
missing or ambiguous parents fall back to the latest admitted main path, or the
queued path before the first main admission. A new main URL without a matched
HTTP redirect adds `X`; same-URL reloads and History/fragment changes add no hop.
For subresources, `Sec-Fetch-Dest: empty` adds `X`; all other or missing fetch
metadata adds `E`. Browser-controller still appends `L` for outlinks. Main outcome
selection, recording ownership, and protobuf contracts are unchanged.

This is intentionally best-effort inference. HTTP cannot disambiguate every
concurrent same-URL request, distinguish an actual root referrer from a truncated
origin, or identify all script-created resources. The accepted rules and limits
are documented in the [recorder contract](../../proto/recorderproxy/v1/README.md#discovery-ancestry).

Permanent recorder tests cover redirect methods and compatibility, same-URL
redirects, nested embeddings, missing/malformed/ambiguous referrers, client
navigation, reloads, immediate cancellation, out-of-order completion, duplicate
observations, late observations, and attempt isolation. HTTP/HTTPS transport tests
also assert redirect paths. The ten-case `TestDiscoveryAncestry` Chromium suite
checks persisted crawl logs, page resources, and outlinks for fragments, redirect
chains, nested resources, redirected stylesheets, History changes, explicit
no-referrer responses, client navigation, reloads, and HTTP/HTTPS script requests.
It runs in the harvest-boundary CI job alongside the existing regressions.

Validation on the corrected tree:

| Check | Result |
| --- | --- |
| Recorderproxy and browser-controller `go test ./... -count=1` | Passed |
| Both modules `go test -race ./... -count=1 -timeout=90s` | Passed |
| Focused ancestry race tests, including final duplicate/late-observation assertions | Passed |
| Combined Chromium `TestSession_Fetch`, `TestDiscoveryAncestry`, `TestHistoryDocumentArtifacts`, `TestSameURLIframeArtifacts`, and `TestFailedSubresourcePreservesArtifacts` | Passed, 232.420 seconds including harness setup |
| Recorder review runner | All probes passed |
| Chromium review runner | All five cases passed, 30.398 seconds including harness setup |
| `git diff --check` | Passed |

The initial restricted run could not bind local sockets; authorized reruns with
socket/Docker access passed, using a writable Go cache under `/tmp`. The first
new Chromium fixture run exposed a test assumption: a child document can send
its own referrer even when the parent's requests omit it. The no-referrer fixture
now explicitly suppresses referrers on every response; the History fixture
correctly expects the image to inherit its known child-frame ancestry. No
production change was needed for that fixture correction.

R1's live Kubernetes restart/termination validation and the existing real-Squid,
backend-service, image-build, and rollout limits remain unchanged.

**2026-10-14 follow-up — R2 addressed:** browser-controller now intercepts only
HTTP/HTTPS document requests on the root page target. It sets
`X-Veidemann-Main-Document` to the attempt UUID only for the known root frame,
independently on every redirect hop. Recorderproxy recognizes only a matching
marker and strips it before recording or forwarding. Neither HTML/upgrade
headers nor Sec-Fetch headers determine main identity. Child-target and worker
setup is unchanged; recording admission and outcomes remain recorder-owned.
Robots exclusions now use that identity even for an iframe at the queued URL.
Continuation errors cancel browsing with their cause; expired interception IDs
and already-cancelled contexts do not introduce new harvest failures.

The permanent `TestSameURLIframeArtifacts` Chromium regression passes for HTTP,
HTTPS, and a cross-site redirect (20.272 seconds including harness setup). Parent
and child bodies have distinct mock WARC IDs; page logs and valid PNG screenshots
refer to the parent, and both same-URL resources remain recorded. Unit and
transport tests cover marker matching and stripping, header preservation,
same-URL robots exclusions, fatal continuation errors, and cancellation.
Both module suites and race checks pass. Recorder review probes for R1, R3, and
R4 pass; R5's fragment probe still fails as expected. The combined Chromium
`TestSession_Fetch`, `TestHistoryDocumentArtifacts`, and
`TestFailedSubresourcePreservesArtifacts` suite also passes (161.904 seconds),
including existing iframe, worker, redirect, reload, and History API fixtures.
The original Chromium review probes confirm that R2's `http-iframe-main`, R3's
`history`, and R4's `failed-subresource` cases pass; only R5's `fragment` and
`redirect-chain` cases still fail (30.452 seconds including harness setup).

At this R2 follow-up R5 remained open; it is now addressed above. R1's live
Kubernetes restart validation remains a staging check;
no cluster rollout was performed for R2.

**2026-10-14 follow-up — R1 addressed:** browser-controller now calls the bounded
`RecoverSlot` startup barrier before readiness or Frontier admission. Recovery
closes orphaned attempts and waits for finalization and late terminal work.
Availability is derived from a published reply and a closed, empty lifecycle; late drainage permits Begin without another
recovery call. Finish and Recover share one finalization entry point, preserving
the first caller’s capture data. Published Finish replies remain unchanged.
An attempt token fences HTTP/HTTPS requests from surviving old browsers; it is
removed before recording or forwarding. Child frames and workers receive the
token before resuming. The deployment still requires one exclusive controller
per recorder slot set; permanently stuck work remains fail-closed.
An in-process gRPC reconnect test invokes the actual controller `RecoverSlots`
function against the surviving recorder, covering disconnect before Finish and
reconnect after a finalization timeout. Focused regressions cover undrained
rejection, late log persistence, stale HTTP/HTTPS traffic, token removal, and
unchanged Finish replay. The R1 review probe checks Begin after late drainage
without another recovery call. These tests establish client reconnection behavior,
not process or Kubernetes container recovery. Recorderproxy and browser-controller
module suites and race checks pass, as do the R1 probe and full Chromium
`TestSession_Fetch` suite (including iframe and worker fixtures).
Live Kubernetes container restart validation remains a staging check documented
in the harvester README; no cluster rollout was performed.

**2026-10-14 follow-up — R4 addressed:** recorder terminal completion now carries
the resource log and an independent attempt failure. Confirmed subresource fetch
failures preserve page success; recording and log persistence failures remain
fatal, including ContentWriter cancellation failure after an origin error.
The R4 recorder and Chromium review probes pass. Permanent regressions cover
HTTP/HTTPS resource failures, failed recording operations, and screenshot,
outlink, and recording-total preservation with a failed image. Recorderproxy
and browser-controller module suites and race checks pass. The report below
describes the original review; see the R3 follow-up below for its resolution. The existing
`TestSession_Fetch` Chromium integration suite also passes (112.105 seconds).

**2026-10-14 follow-up — R3 addressed:** browser-controller now retains the
committed root document URL, frame ID, and loader ID from `Page.frameNavigated`.
Child-frame events and same-document History API/fragment events cannot replace
that association. Capture checks the committed identity against the captured
frame before extraction and again afterward, and retries the entire artifact
bundle on navigation or current-URL changes. Missing or mismatched identities
fail capture rather than substituting the queued URL or an older document.
`PageCapture.uri` carries the committed URL; outlink referrers and diagnostics
retain the current URL, including its fragment. The existing attempt contract
is clarified without protobuf, RPC, recorder-selection, or deployment changes.
Recorder matching still uses normalized URLs and admission order, including
cached and failed later outcomes; there is no older-success fallback.

Root-document tests cover History API/fragment events, child frames, redirects,
reloads, and missing or mismatched identities (including a Navigate reply that
cannot repair a missing commit identity). Deterministic capture tests check both
URLs and whole-bundle retry across document and same-document changes. Eight
Chromium regressions cover pushState/replaceState path and query changes,
redirect-then-history, reload-then-history, fragments, and a later failed
navigation. Successful cases check page logs, PNG screenshots attached to the
committed recording, recording totals, and current-URL outlink referrers; the
failed navigation must not reuse the old page or screenshot.

R3 validation on this tree:

| Check | Result |
| --- | --- |
| `go test ./... -count=1` in browser-controller and recorderproxy | Passed |
| `go test -race ./... -count=1 -timeout=90s` in both modules | Passed |
| Recorder `TestReviewHistoryURLRetainsMainDocument` overlay probe | Passed with corrected caller URL and unmatched-URL rejection |
| Unchanged Chromium `TestReviewBrowserCases/history` overlay probe | Passed (8.748 seconds including harness setup) |
| Chromium `TestHistoryDocumentArtifacts` | All eight cases passed (47.346 seconds including harness setup) |
| Combined Chromium `TestSession_Fetch`, `TestHistoryDocumentArtifacts`, and `TestFailedSubresourcePreservesArtifacts` | Passed (161.942 seconds) |
| `git diff --check` | Passed |

At the time of the R3 follow-up, R2 iframe classification and R5 discovery
ancestry remained open. The fragment
regression here checks document association and referrers, not R5's discovery
path semantics. R1's live Kubernetes restart validation remains a staging check;
no deployment or cluster validation was performed for R3.

The findings below retain the original review evidence.

| ID | Priority | Finding | Primary location |
| --- | --- | --- | --- |
| R1 | P1 | Controller restart cannot recover an orphaned or poisoned recorder slot | `recorderproxy/recorderproxy/harvest.go:101`, `browser-controller/controller/browsercontroller.go:82` |
| R2 | P2 | An HTTP iframe can replace the top-level document as the main recording | `recorderproxy/context/recordcontext.go:245` |
| R3 | P1 | History API URL changes lose a successfully recorded main document | `recorderproxy/recorderproxy/harvest.go:187` |
| R4 | P2 | Ordinary subresource fetch failures now fail the entire page harvest | `recorderproxy/recorderproxy/harvest.go:266` |
| R5 | P2 | Discovery paths lose redirect ancestry and count fragments as redirects | `recorderproxy/context/recordcontext.go:147` |

P1 means address before rollout; P2 means a reproducible correctness regression
that needs a fix or an explicit, accepted behavior change.

### R1 — Restart does not restore recorder slot availability

`BeginHarvest` rejects another attempt while `a.drained` is false. A finalization
timeout permanently sets that field to false; late terminal work can persist
logs and empty the lifecycle tracker, but does not update slot availability.
Browser-controller responds to `reusable=false` by reporting the harvest and
exiting its worker group. `main` then exits the process.

The new [harvester deployment](../../deploy/k8s/base/harvester/deployment.yaml)
runs recorderproxy as a native sidecar and browser-controller as an application
container. Restarting the controller container does not reset the recorder
process or its attempt state. Native sidecars have independent container
lifecycles; their ordered shutdown applies to Pod termination.
[Kubernetes sidecar lifecycle documentation](https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/#sidecar-containers-and-pod-lifecycle).

The new controller generates a fresh UUID, gets `FailedPrecondition` from Begin,
then gets another `FailedPrecondition` when it tries to Finish that unknown UUID.
It poisons the session and exits again. A controller crash before Finish has the
same problem: even with no remaining connections, the old attempt never becomes
drained. There is no lease, startup reconciliation, or coordinated reset path.
This can leave the Pod repeatedly restarting its controller until the recorder
or the whole Pod is replaced.

**Reproduction:** admit a transfer, force Finish to time out, complete and remove
the late transfer, verify the lifecycle is empty, then issue Begin with a fresh
controller attempt ID against the unchanged recorder. The reply remains:

```text
FailedPrecondition: previous attempt has not drained
```

**Required completion:** define recovery of recorder-owned state when its browser
owner disappears. Provide a bounded ownership/reconciliation mechanism or a
supported recovery mechanism that resets the affected processes together. Keep
the rejection of genuinely undrained work. Verify both controller termination
before Finish and a finalization timeout followed by an application-container
restart. The Kubernetes restart loop itself was not exercised on a live cluster.

### R2 — HTTP iframe navigation is mistaken for main navigation

The fallback in [recordcontext.go](../../recorderproxy/context/recordcontext.go)
uses `Upgrade-Insecure-Requests: 1` plus an HTML Accept header when
`Sec-Fetch-Dest` is absent. Actual Chromium requests to the ordinary HTTP fixture
have those headers for both the root document and an iframe; neither request
contains `Sec-Fetch-Dest`.

With a parent page that sets a cookie and embeds its own URL, the later iframe
response contains different HTML. Both transfers are marked `MainDocument` and
both match the captured root URL. Finish therefore selects the iframe by its
later admission sequence. The page log is attached to the child WARC, and a
requested screenshot would receive that child crawl log as its metadata.
The same misclassification also affects the main/subresource distinction used
for robots exclusions.

**Reproduction:** `TestReviewBrowserCases/http-iframe-main` loads this case in
Browserless Chromium v2.57.0. The mock assigns distinct WARC IDs to different
response bodies. The selected page WARC is the child response's ID, while the
first root response has a different ID.

**Required completion:** identify top-level navigation reliably on ordinary HTTP
as well as HTTPS. The fallback headers cannot establish frame identity. Keep
recording admission and outcomes recorder-owned, and add the same-URL iframe
case with distinct parent/child recordings to the integration fixtures.

### R3 — History changes invalidate recorder main-page selection

[captureBundle](../../browser-controller/session/harvest.go) sends the current
frame URL. [FinishHarvest](../../recorderproxy/recorderproxy/harvest.go) requires
that URL to equal a main transfer's requested URI after fragment normalization.
`history.pushState` and `history.replaceState` can change the path or query while
keeping the same loaded document and making no new HTTP request.

A page recorded as `/index.html` that calls
`history.replaceState({}, '', '/new-route')` captures successfully, but Finish
returns recorded totals with no Main. No page log is written. Browser-controller
then fails with `no recorded main capture for .../new-route`, skips screenshot
persistence, and reports a failure instead of its extracted outlinks.

**Reproduction:** both `TestReviewHistoryURLRetainsMainDocument` and the real
Chromium case `TestReviewBrowserCases/history` reproduce this. The recorder-only
reply contains `uri_count:1 reusable:true` with no main recording.

**Required completion:** distinguish the loaded document's network identity from
its current History API URL. Preserve their relationship across capture without
using an arbitrary older successful recording to conceal a later failed
navigation. Cover path and query changes as well as ordinary navigation,
redirects, reloads, and fragments.

### R4 — A failed image turns a successful page into a failed harvest

The terminal callback in [harvest.go](../../recorderproxy/recorderproxy/harvest.go)
promotes every crawl-log error except browser cancellation and certain robots
exclusions into `a.failure`. It does not distinguish an ordinary origin failure
on a subresource from failure of the main document or archival infrastructure.

The baseline controller retained failed subresources in the page resources;
they did not independently set the returned fetch error. The new implementation
fails a recorded 200 main document when an image connection is refused.
`finishHarvest` returns that resource error, and `Session.Fetch` returns before
writing the screenshot or returning outlinks and recording totals. The page log
and successful crawl logs may already have been written. This creates an
unannounced change in page success semantics and can cause repeated attempts for
pages with permanently unavailable dependencies.

**Reproduction:** `TestReviewBrowserCases/failed-subresource` serves a complete
HTML page with a link and an image pointing to a closed, ordinary local TCP port.
The main response is recorded successfully and all three admitted transfers
finish, but the harvest returns `-2 CONNECT_FAILED: connection refused`.
`TestReviewOriginSubresourceFailureDoesNotFailCapturedPage` also reproduces the
failure with a selected, recorded main and successfully persisted resource logs.

**Required completion:** classify attempt-fatal failures separately from resource
outcomes. Preserve failed resource logs, while retaining page success for
ordinary subresource origin failures. Continue failing the attempt for storage
or log persistence failures and an unusable main document. If strict failure on
any dependency is intended, make that policy explicit and review its Frontier
retry and artifact consequences.

### R5 — Discovery ancestry was replaced by a one-character guess

The new [RecordContext initialization](../../recorderproxy/context/recordcontext.go)
starts every resource at the queued URI's discovery path and appends at most one
character based on URL inequality and fetch destination. The deleted registry
followed redirect parents and recorded referrers. That responsibility has not
been fully transferred.

Two concrete failures occur:

- Queuing `/index.html#section` with path `L` records the initial network request
  as `LR`, even though there is no redirect. Its discovered links become `LRL`.
- Queuing `/index.html` with path `L`, then redirecting through `/middle` to
  `/final`, records the final main as `LR` instead of `LRR`. Its outlinks become
  `LRL` instead of `LRRL`. Embedded descendants also start from the queued path
  instead of their actual recorded parent's path.

This affects more than log presentation. Browser-controller directly uses
`main.DiscoveryPath + "L"` for queued outlinks, and
[scope hop matching](../../scopeservice/internal/script/matcher_helpers.go)
counts those path characters, optionally including redirects. Incorrect ancestry
can change crawl scope decisions.

**Reproduction:** the real Chromium `fragment` and `redirect-chain` cases fail
with the paths above. `TestReviewFragmentIsNotRedirect` isolates the raw URL
comparison in initialization.

**Required completion:** normalize network URLs consistently and derive discovery
paths from actual redirect/referrer ancestry. Add multi-hop main redirects,
redirected subresources, nested embeddings, and fragments to the regression
coverage. Define how script-initiated and History API changes fit that ancestry.

## Ownership migration audit

| Responsibility | Current owner and evidence | Assessment |
| --- | --- | --- |
| Attempt identity and slot admission | Recorder Begin/Finish API; immutable Begin metadata, duplicate Begin checks, retired IDs | Implemented; owner-loss recovery missing (R1) |
| Connection and handler isolation | Connections pin an attempt at accept; per-attempt lifecycle closes and drains connections/records | Implemented; timeout correctly refuses immediate reuse |
| Transfer admission and terminal outcome | Admission precedes policy/writer setup; independent sequence per request; terminal coordinator guards completion | Implemented; resource/attempt failure policy needs correction (R4) |
| Crawl-log persistence | Recorder terminal callback writes directly to log-service with an independent bounded context | Implemented, including failures without WARC IDs |
| Page-log assembly and main selection | Recorder builds resources and selects by admission order | Implemented; frame/document identity incomplete (R2, R3) |
| Crawl ancestry and outlink discovery paths | Recorder infers immutable paths from redirect/referrer evidence; browser appends `L` | Addressed (R5), with explicit fallback rules and accepted ambiguity |
| Recording totals and confirmed WARC references | Recorder totals confirmed response crawl logs; terminal coordinator requires response WARC metadata | Implemented in tested cases |
| Robots enforcement | Recorder converts classic policies and preserves allow-on-evaluator-error behavior | Moved; main/subresource labeling affects exclusions (R2) |
| Robots recording outside browser slots | Robots evaluator resolves/captures exchanges and calls ContentWriter/log-service directly | Implemented; recording failures remain separate from decisions |
| Browser readiness and artifact capture | Root load signal, target-scoped activity tracker, shared media allowance, whole-bundle capture retry | Implemented in current tests; tracker no longer owns recording outcomes |
| Screenshot persistence and Frontier reporting | Browser uses Finish reply, persists screenshot, reports through existing Frontier API | Implemented; early failure propagation exposes R3/R4 |
| Builds and coordinated deployment | At review time, local module replacements and root-context builds (superseded by published Go modules) | Wiring present; renders pass; recovery integration incomplete (R1) |

The removed registration/completion callback server and browser request registry
are replaced by working recorder paths. Their removal alone is not evidence of
unfinished work. The replacement-script selector/fields were already unused in
the baseline request flow, so their removal is not listed as a new regression.

## Validation and reproducibility

The following existing checks passed on the reviewed tree:

| Check | Result |
| --- | --- |
| `go test ./... -count=1`, separately in recorderproxy, browser-controller, robots-evaluator | All passed |
| `go test -race ./... -count=1 -timeout=90s`, separately in those three modules | All passed |
| `go test -tags=integration ./server -run '^TestSession_Fetch$' -count=1 -timeout=5m`, in browser-controller | Passed in 112.683 seconds, using real Browserless/nginx and mocked backend services |
| Kustomize renders: harvester base, robots base, dev harvester overlay, complete dev overlay | All passed |
| `git diff --check` | Passed |

The initial sandbox attempts could not open local test sockets; reruns with local
socket/cache access passed. Those environmental failures are not code findings.

Review probes are retained as documentation, outside the service test packages:
[context probes](context_probes.go.txt), [harvest probes](harvest_probes.go.txt),
and [Chromium probes](browser_probes.go.txt). The
[runner](run_reproductions.py) inserts them through Go overlays in a temporary
directory. It makes no changes to service source files. The shared ContentWriter mock now
uses payload digests in WARC IDs so parent and child recordings can be
distinguished without a mock overlay.

Run from the repository root:

```sh
python3 docs/review/run_reproductions.py
python3 docs/review/run_reproductions.py --browser
```

Both commands now exit successfully: their assertions describe the corrected
behavior. Recorder probes cover R1, R3, R4 and the fragment portion of R5; the five
Chromium cases cover R2–R5. All pass. The recorder-only R3 probe now models the
corrected caller contract and separately rejects an unmatched URL. The Chromium
history probe is unchanged. Run R1 alone with
`python3 docs/review/run_reproductions.py --run '^TestReviewControllerRestartCannotRecoverPoisonedSlot$'`.
Run R3 alone with:

```sh
python3 docs/review/run_reproductions.py --run '^TestReviewHistoryURLRetainsMainDocument$'
python3 docs/review/run_reproductions.py --browser --run '^TestReviewBrowserCases$/history$'
```

Run R2 alone with:

```sh
python3 docs/review/run_reproductions.py --browser --run '^TestReviewBrowserCases$/http-iframe-main$'
(cd browser-controller && go test -tags=integration ./server -run '^TestSameURLIframeArtifacts$' -count=1 -timeout=2m)
```

Browser probes require Docker and the existing integration harness images. Run browser
integration commands sequentially because the harness uses fixed local ports.

Remaining validation limits: no live Kubernetes restart/termination test, no
production rollout, no new image build in this review, and no full browser
harvest through real Squid with real DNS/storage/log services. The existing
recorder tests exercise chained-proxy behavior, while the browser fixtures use
the constructor's direct transport with backend mocks. Those limits remain
relevant even after the five findings are fixed.
