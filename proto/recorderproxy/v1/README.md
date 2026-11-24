# Recorder attempt contract

Browser-controller calls `BeginHarvest → browse/capture → FinishHarvest`, writes
its screenshot, and reports to Frontier. Recorderproxy owns transfer outcomes,
crawl-log persistence, page-log assembly, and recording totals. Browser events
guide browsing and identify root document requests; they never create or remove
recorder outcomes.

Begin supplies a never-reused attempt UUID, collection and crawl identifiers,
requested URI, discovery path, referrer, user agent, and robots policy. An
identical Begin for the current attempt returns the original reply, including
its next token. Each slot worker starts with the random admission token from
RecoverSlot, presents it on Begin, and retains the returned token for the next
session. Only successful admission rotates the token. Conflicting metadata,
stale tokens and premature replacements fail. After an ambiguous Begin reply,
the worker reconciles the slot and retrieves its token before taking more work. Slots are routing addresses, numbered `1..PROXY_COUNT`, mapped to
`base + slot - 1`. An undrained attempt cannot be replaced.

At process startup, before readiness or requesting Frontier work, the exclusive
controller calls `RecoverSlot` for every configured slot. This closes an orphaned
attempt without a page capture and waits for finalization and all remaining
handlers and terminal persistence. Each call is bounded by the recorder's
finalization timeout and the caller's deadline. An undrained slot fails recovery;
the controller exits without consuming crawl work and can retry on restart.
Availability is derived from the published Finish result and a closed lifecycle
with no connections or records remaining. After late work drains, a new Begin can
succeed without another recovery call. Startup recovery is still needed to close
abandoned attempts that never received Finish. The original Finish reply remains
unchanged and replayable until replacement.
Recovery requires exclusive ownership at startup and after an ambiguous Begin.
Deploy exactly one controller per recorder slot set.

Every HTTP request, including an inner HTTPS request, must carry
`X-Veidemann-Attempt-Id` matching Begin's attempt UUID. Browser-controller sets it
before navigation and before resuming newly attached frames and workers. The
recorder strips it before admission, recording, and forwarding. Missing or old
tokens receive 403 without a transfer or crawl log. Thus a surviving old browser
cannot reconnect into a replacement attempt. CONNECT remains setup; errors before
an identified inner request are diagnostic only and cannot poison another attempt.
The token provides attempt isolation, not authentication of untrusted clients;
keep the recorder and its control API private to their controller.

Main-document identity comes exclusively from `X-Veidemann-Main-Document`
matching the attempt UUID. Browser-controller intercepts HTTP/HTTPS document
requests on the root page target, marking only those with the root frame ID.
Each redirect hop is marked independently; iframe requests remain unmarked.
The recorder consumes and strips the marker before recording or forwarding.
Missing or mismatched markers mean subresource, regardless of URL, Accept,
Upgrade-Insecure-Requests, or Sec-Fetch headers. The attempt token remains
mandatory for every request. These internal headers are not exposed to origins.
A same-URL iframe's robots exclusion is a resource outcome, not a main failure.

Accepted connections are pinned to their attempt. Each HTTP transfer is admitted
before policy checks and receives an internal admission sequence. CONNECT is
setup, not a crawl resource. Transfer completion finalizes ContentWriter and
writes its crawl log directly to log-service, including errors without a WARC ID.
Persistence uses a bounded context independent of browser cancellation. A failed
log write fails the harvest; confirmed recording references are never invented
or removed to disguise a failure.

## Discovery ancestry

Discovery paths are inferred entirely by the recorder from admitted requests and
HTTP responses. Each transfer receives an immutable path during initialization,
before cancellation or terminal persistence can snapshot it. The attempt keeps
an index of earlier admissions and pending redirect targets, independent of
recording completion. A child can therefore inherit a streaming parent's path.
The index is discarded during finalization; late responses cannot revive it.
No browser request registry, extra header, or new RPC supplies ancestry.

Matching uses the same WHATWG network URL normalization as main selection:
fragments do not distinguish requests, while meaningful paths and queries do.
Original request URLs and HTTP referrers remain unchanged in crawl logs.
Malformed or non-HTTP(S) URLs cannot establish a parent.

The inference rules, in precedence order, are:

| Evidence | Path |
| --- | --- |
| One compatible pending HTTP redirect | Redirect parent's path + `R` |
| First marked main request | Queued discovery path |
| Main request repeats the latest main's network URL, without a matched redirect | Preserve that main's path |
| New main URL without a matched redirect | Referrer parent's path, or fallback, + `X` |
| Other request with `Sec-Fetch-Dest: empty` | Referrer parent's path, or fallback, + `X` |
| Other request, including absent fetch metadata | Referrer parent's path, or fallback, + `E` |

The fallback is the latest **admitted** marked main request's path, regardless of
its eventual outcome, or the queued path before the first main admission.
This ancestry fallback never substitutes an older successful recording for the
main outcome selected by Finish.

Redirect evidence is registered before response headers reach the browser. Only
301, 302, 303, 307 and 308 with a valid HTTP(S) `Location` qualify. Resolve relative
locations against the responding request. Match target network URL, main versus
subresource identity, expected method, and fetch destination when both requests
supply it. For 301/302, POST becomes GET; for 303, methods other than GET/HEAD
become GET; 307/308 preserve the method. A unique match is consumed once. Multiple
compatible candidates are discarded together and ordinary fallback applies,
without adding `R`. Nonmatching candidates remain pending until a compatible
request or attempt finalization.

Referrer parents must be earlier admissions at the exact normalized referrer URL.
Repeated matches are usable only when all their paths agree. Missing, malformed,
unmatched, or conflicting referrers use fallback; an origin-only referrer does
not trigger a search for other URLs on that origin. An exact root-URL match is
still usable because HTTP cannot distinguish it from an origin-truncated value.

History API and fragment-only changes make no new transfer and add no hop.
Requests referring to an unrecorded History URL use fallback. New main URLs from
JavaScript navigation or meta refresh add `X`; reloads preserve the latest main
path unless an observed HTTP redirect matches. Subresource `X` is a convention
based on fetch destination, not exact script attribution: a script-created image
is `E`, and absent metadata (notably ordinary HTTP) also defaults to `E`.

These rules deliberately provide best-effort ancestry. A unique pending redirect
can still coincide with an unrelated request, and missing/truncated referrers,
repeated URLs, or History changes can hide the actual parent. The recorder does
not change page success or reject requests because inference is incomplete.
For queued path `L`, a fragment-only initial URL remains `L` with outlinks `LL`;
two HTTP redirects produce `LRR` with outlinks `LRRL`. Nested known embeddings
produce `LE`, `LEE`, and so on. Browser-controller continues to append `L` to the
selected main's path for outlinks; scope hop-count semantics are unchanged.

## Capture and finalization

Finish is a client stream with one reply. The first message contains the slot,
attempt UUID and optional committed page URI. Each following message contains
one outlink URL, in extraction order, including duplicates. A present URI and
immediate EOF capture a page with zero links. An absent URI indicates capture
failure and permits no outlinks. Cancellation before EOF discards the upload.
Browser-controller sends scheduling metadata and cookies only to Frontier. It retains the
committed URL from root
`Page.frameNavigated` events and requires its frame and loader IDs to match the
captured document. History API and fragment changes do not replace it. The
current captured URL remains the outlink referrer and diagnostic URL. Capture
fails without a matching committed document; neither the queued URL nor an older
document is substituted. On clean EOF the recorder rechecks the attempt under
the slot lock. Browser-controller retries an unavailable Finish once within the same deadline.
The first completed upload closes admission and connections,
drains handlers and transfer persistence, and writes the page log when a captured, recorded main document exists. Page resource
labels come from HTTP fetch metadata; unknown types are empty. All terminal
resource outcomes, including failures and cache hits, remain in that page log.

The latest matching main-document transfer by admission order determines the
main outcome, even when it is cached or failed. URL fragments are ignored for
network matching. A previously cached navigation cannot override a later
recording, and an old recording cannot substitute for a later failed navigation.

The reply contains the selected main crawl log, its cache status, recorded URI
and byte totals, a terminal commons failure, and explicit slot reusability. It
contains no transfer ledger. Finalization runs independently of the calling RPC;
repeated Finish calls replay the completed reply without repeating persistence.
The first call's capture data is authoritative. Replay lasts until slot replacement.

Robots exclusions of subresources and deliberate browser cancellation do not
independently fail the attempt. Confirmed subresource fetch failures also remain
resource outcomes: a usable recorded main page can still produce a screenshot,
outlinks, and recording totals. The failed resource stays in the crawl and page
logs. This decision follows the failure site, not its numeric error code.
Main-document failures, unknown-source errors, and recording or persistence
failures still fail the attempt. A ContentWriter cancellation failure remains
fatal even when the resource already has an origin error; its original error
stays in the resource log and the recording failure is returned by Finish.
Finalization timeouts are retryable `-5` and return `reusable=false`; the worker
must stop rather than reuse that slot. Late terminal work may still persist its
crawl log but cannot change the published reply. Browser-controller alone uses
`-5004` when its browsing allowance expires without a usable main capture and no
more specific failure. Screenshot failure still fails the harvest, while logs
already written remain available.

There is no durable recovery journal or atomic transaction spanning recording,
logs, screenshots, and Frontier. Deploy browser-controller and recorderproxy
from the same contract version. Frontier, ContentWriter, log-service and dashboard
APIs are unchanged. Robots recording is independent of browser slots.
