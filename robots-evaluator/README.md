# Veidemann robots evaluator

The robots evaluator fetches, caches, parses, and evaluates `robots.txt` on
behalf of Frontier and recorderproxy. Responsibility is split between the
two callers: Frontier checks queued pages before starting a browser session,
while recorderproxy checks browser requests only for the `*_CLASSIC`
policies.

## Robots policies

The table describes the current implementation. An unavailable `robots.txt` is
a final 3xx or 4xx response. An unreachable `robots.txt` is a 5xx response,
timeout, network error, or response-body error.

| Value | Policy | Frontier check before browser start | Recorderproxy check | Rules used | Site `robots.txt` unavailable or unreachable |
|---:|---|---|---|---|---|
| 0 | `OBEY_ROBOTS` | Evaluates the queued page | None | Site `robots.txt` | Unavailable allows; unreachable uses stale rules or returns `Unavailable` if none exist |
| 1 | `IGNORE_ROBOTS` | None | None | No robots rules | Not applicable |
| 2 | `CUSTOM_ROBOTS` | Evaluates the queued page | None | Configured custom rules | Not applicable; site `robots.txt` is not fetched |
| 3 | `OBEY_ROBOTS_CLASSIC` | Evaluates the queued page | Evaluates requests registered during the browser session | Site `robots.txt` | Unavailable allows; unreachable uses stale rules or returns `Unavailable` if none exist |
| 4 | `CUSTOM_ROBOTS_CLASSIC` | Evaluates the queued page | Evaluates requests registered during the browser session | Configured custom rules | Not applicable; site `robots.txt` is not fetched |
| 5 | `CUSTOM_IF_MISSING` | Evaluates the queued page | None | Site rules, stale site rules when unreachable, otherwise configured custom rules | Configured custom rules are used when no site rules are available |
| 6 | `CUSTOM_IF_MISSING_CLASSIC` | Evaluates the queued page | Evaluates requests registered during the browser session | Site rules, stale site rules when unreachable, otherwise configured custom rules | Configured custom rules are used when no site rules are available |

For the three `*_CLASSIC` policies, recorderproxy translates the policy
to its corresponding non-classic policy before calling robots-evaluator. For
all other policies, recorderproxy performs no robots-evaluator RPC.

An internal robots-evaluator RPC failure remains fail-open in Frontier and
recorderproxy. An unreachable origin without cached or custom rules is
returned as gRPC `Unavailable`; it is not reported as a robots-rule denial.
Frontier therefore allows the request and logs the evaluator failure instead
of completing the seed with `PRECLUDED_BY_ROBOTS` (`-9998`).

## Caching

Olric is the authoritative robots cache. Entries are partitioned by collection
and origin:

```text
robots|<collection-id>|<scheme>|<host>|<port>
```

Jobs in the same collection share an entry. Different collections fetch and
archive their own `robots.txt` and never share stale rules. Requests without a
collection ID use a separate `unscoped` partition.

Each entry records the rules, whether site rules are available, when they were
fetched, when they become stale, and when an unreachable origin may be retried.
Normal freshness is fixed by the service rather than derived from origin
response headers. The two relevant service settings are:

| Flag | Environment variable | Default | Validation | Meaning |
|---|---|---:|---|---|
| `--robots-cache-freshness` | `ROBOTS_CACHE_FRESHNESS` | `24h` | Greater than zero and no more than 24 hours | Maximum normal age of a cached result |
| `--robots-unreachable-retry-interval` | `ROBOTS_UNREACHABLE_RETRY_INTERVAL` | `1h` | Greater than zero | Delay between attempts while the origin is unreachable |

Refresh outcomes are handled as follows:

| Situation | Cached result |
|---|---|
| Successful 2xx | Store the site rules for the configured freshness interval |
| Final 3xx or 4xx | Store an unavailable result for the configured freshness interval |
| 5xx, timeout, network failure, or body-read failure with cached rules | Keep using the stale rules and postpone refresh for the retry interval |
| Unreachable without cached rules in this collection | Store the retry timestamp; obey policies return `Unavailable` and `CUSTOM_IF_MISSING` policies use their custom rules |

Logical freshness does not delete an Olric entry. When a fresh result becomes
stale, robots-evaluator attempts to refresh it. If the origin is unreachable,
stale rules remain usable and the retry timestamp prevents a request storm.
Without stale rules, `OBEY_ROBOTS` returns `Unavailable` until retry so callers
can apply their service-failure policy; a `CUSTOM_IF_MISSING` policy evaluates
its custom rules.

Olric storage retention is configured separately with a 30-day
`maxIdleDuration`. Reads reset idle time, so actively used stale rules can
survive a longer outage. LRU eviction may still remove entries under memory
pressure. No Olric `ttlDuration` or per-entry expiry is used.

Origin requests include:

```http
Cache-Control: no-cache, no-store
```

This makes Squid validate any previously stored response and prevents it from
storing the request or new response. Squid remains the explicit outbound proxy. ContentWriter and log-service receive
recordings directly from robots-evaluator.

The deprecated `minimum_robots_validity_duration_s` politeness field is
accepted for wire compatibility but ignored. Freshness is controlled at the
service level.

## Redirects and rule matching

The evaluator follows up to five redirects when fetching robots.txt, including
redirects to another origin. It applies the resulting rules to the original
origin. The exact normalized target `/robots.txt` is implicitly allowed.
In rules, `%2A` matches a literal asterisk and `%24` matches a literal dollar
sign, whether the URL contains the character itself or its percent escape.

## Build and test

Run commands from this directory:

```sh
go test ./...
```

## Direct HTTP recording

A refresh carries collection, crawl execution and job execution identity in Go
request context. Each target is resolved through DnsResolver with collection and
execution attribution; DNS answers remain archived by that service. Negative DNS
answers fail the fetch. The HTTP client connects through the explicit Squid
endpoint, accepts existing upstream TLS certificates, requests identity encoding,
and preserves the 10-second fetch timeout and five-hop redirect limit. Squid job
partition headers are added only after capturing the request and Squid strips
them before forwarding to origins. Environment proxy settings are ignored.

Each completed GET exchange, including redirects back to the same URL and HTTP
error responses, produces one ContentWriter stream containing a request/response
pair. Sizes and SHA-1 digests cover the captured HTTP protocol headers and body.
Only confirmed record metadata and WARC references are used for response crawl
logs sent directly to log-service. Squid cache hits are filtered. Fresh Olric
hits perform no HTTP recording; shared refreshes belong to the caller that starts
the fetch. Requests without a collection continue evaluation and explicitly
report that recording was skipped.

Recording runs synchronously as part of a refresh: fetch and cache the rules,
then write each completed request/response pair and its crawl log before replying.
Storage latency provides backpressure to the refreshing caller. Subsequent cache
hits use the cached result without waiting for recording or recording it again.
Earlier caller cancellation or deadlines also stop recording.

The evaluator gives all WARC and crawl-log writes for one refresh a shared
30-second budget (`--recording-timeout` or `RECORDING_TIMEOUT`). If storage
times out, it returns the computed decision. Frontier allows 60 seconds per
evaluator call (`robotsEvaluatorTimeout` or `ROBOTS_EVALUATOR_TIMEOUT`) and
allows the harvest on RPC failure. When tuning, leave room in the Frontier
budget for fetching robots.txt, cache access and returning the decision.

A storage or log failure is reported without replacing the robots decision or
invalidating cached rules. Each exchange is limited to 4 MiB of archived headers
and body. Oversized and incomplete exchanges are skipped; completed earlier
redirects remain eligible for recording. One `Robots recording summary` reports
identities, recorded/skipped counts, failure causes and duration.

| Flag | Environment | Default |
|---|---|---|
| `--dns-resolver-address` | `DNS_RESOLVER_ADDRESS` | `dns-resolver:8053` |
| `--content-writer-address` | `CONTENT_WRITER_ADDRESS` | `contentwriter:8082` |
| `--log-service-address` | `LOG_SERVICE_ADDRESS` | `log-service-writer:8080` |
| `--cache-address` | `CACHE_ADDRESS` | `cache-balancer:3128` |
| `--recording-timeout` | `RECORDING_TIMEOUT` | `30s` |

Shutdown uses gRPC `GracefulStop`, which waits for active requests, including
their recording. Robots fetching and recording work with browser-controller stopped.

Build from the repository root:

```sh
docker build -t robots-recording:test ./robots-evaluator
```

Run `go test -race ./...` from this module. Deterministic HTTP/HTTPS origins and
gRPC fixtures cover recording bytes/digests/references, cache behavior, DNS and
persistence failures, caller cancellation, storage backpressure and concurrent refreshes.
See [the coordinated cutover](../deploy/k8s/base/harvester/README.md).

Unread redirect bodies are not drained for recording: closing them retains the
HTTP client's redirect behavior and reports an incomplete exchange. Completed
redirects are retained if a later DNS lookup or HTTP transfer fails.
