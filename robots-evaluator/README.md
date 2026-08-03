# Veidemann robots evaluator

The robots evaluator fetches, caches, parses, and evaluates `robots.txt` on
behalf of Frontier and browser-controller. Responsibility is split between the
two callers: Frontier checks queued pages before starting a browser session,
while browser-controller checks browser requests only for the `*_CLASSIC`
policies.

## Robots policies

The table describes the current implementation. An unavailable `robots.txt` is
a final 3xx or 4xx response. An unreachable `robots.txt` is a 5xx response,
timeout, network error, or response-body error.

| Value | Policy | Frontier check before browser start | Browser-controller check | Rules used | Site `robots.txt` unavailable or unreachable |
|---:|---|---|---|---|---|
| 0 | `OBEY_ROBOTS` | Evaluates the queued page | None | Site `robots.txt` | Unavailable allows; unreachable uses stale rules or denies if none exist |
| 1 | `IGNORE_ROBOTS` | None | None | No robots rules | Not applicable |
| 2 | `CUSTOM_ROBOTS` | Evaluates the queued page | None | Configured custom rules | Not applicable; site `robots.txt` is not fetched |
| 3 | `OBEY_ROBOTS_CLASSIC` | Evaluates the queued page | Evaluates requests registered during the browser session | Site `robots.txt` | Unavailable allows; unreachable uses stale rules or denies if none exist |
| 4 | `CUSTOM_ROBOTS_CLASSIC` | Evaluates the queued page | Evaluates requests registered during the browser session | Configured custom rules | Not applicable; site `robots.txt` is not fetched |
| 5 | `CUSTOM_IF_MISSING` | Evaluates the queued page | None | Site rules, stale site rules when unreachable, otherwise configured custom rules | Configured custom rules are used when no site rules are available |
| 6 | `CUSTOM_IF_MISSING_CLASSIC` | Evaluates the queued page | Evaluates requests registered during the browser session | Site rules, stale site rules when unreachable, otherwise configured custom rules | Configured custom rules are used when no site rules are available |

For the three `*_CLASSIC` policies, browser-controller translates the policy
to its corresponding non-classic policy before calling robots-evaluator. For
all other policies, browser-controller performs no robots-evaluator RPC.

An internal robots-evaluator RPC failure remains fail-open in Frontier and
browser-controller. Expected origin outcomes are converted into explicit
decisions by robots-evaluator: an unreachable origin without cached or custom
rules is denied instead of being returned as an RPC failure.

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
| Unreachable without cached rules in this collection | Store the retry timestamp; obey policies deny and `CUSTOM_IF_MISSING` policies use their custom rules |

Logical freshness does not delete an Olric entry. When a fresh result becomes
stale, robots-evaluator attempts to refresh it. If the origin is unreachable,
stale rules remain usable and the retry timestamp prevents a request storm.
Without stale rules, `OBEY_ROBOTS` denies until retry; a
`CUSTOM_IF_MISSING` policy evaluates its custom rules.

Olric storage retention is configured separately with a 30-day
`maxIdleDuration`. Reads reset idle time, so actively used stale rules can
survive a longer outage. LRU eviction may still remove entries under memory
pressure. No Olric `ttlDuration` or per-entry expiry is used.

Origin requests include:

```http
Cache-Control: no-cache, no-store
```

This makes Squid validate any previously stored response and prevents it from
storing the request or new response. Squid remains in the request path for
archiving but is not a second robots cache.

The deprecated `minimum_robots_validity_duration_s` politeness field is
accepted for wire compatibility but ignored. Freshness is controlled at the
service level.

## Build and test

Run commands from this directory:

```sh
go test ./...
```
