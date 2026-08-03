# veidemann-dns-resolver

The resolver is a CoreDNS build with two Veidemann plugins:

- `resolve` exposes the collection-aware gRPC API on port `8053`.
- `archivingcache` supplies a shared DNS cache and collection-specific DNS
  archival.

The order of directives in a Corefile does not determine plugin execution
order. The compiled order in `main.go` is authoritative; the Corefile only
enables and configures plugins.

## Request paths and cache ownership

```text
Squid process
  └─ local Squid DNS cache
       └─ UDP/TCP port 53
            └─ archivingcache
                 ├─ shared Olric DMap
                 └─ CoreDNS forward → recursive DNS servers

Recorder proxy
  └─ gRPC Resolve on port 8053, with collection and execution IDs
       └─ A query through the same archivingcache and Olric DMap
```

The layers have different scope and responsibilities:

| Layer | Scope | Stores | Extends lifetime on a hit? |
| --- | --- | --- | --- |
| Squid DNS cache | One Squid process | Positive and failed hostname lookups | No |
| `archivingcache` | Shared by resolver pods through Olric | DNS responses keyed by the complete normalized question | No |
| Archive marker | Shared by resolver pods through Olric | Successful archival of one DNS generation for one collection | No |
| Squid HTTP object cache | One parent Squid process | HTTP responses | Not applicable to DNS |
| Upstream recursive resolver | External to Veidemann | Resolver-dependent DNS cache | Controlled by that resolver |

An HTTP object-cache hit may avoid origin connection and DNS resolution
entirely. HTTP cache directives such as `Cache-Control: no-store` control the
HTTP object cache; they do not control either DNS cache.

Both DNS entry points populate and read the same Olric cache. Port-53 requests,
including requests from Squid, carry no collection context and never archive.
The gRPC API always creates an A query and supplies collection and execution
metadata so that a positive result can be archived.

The deployed `arpa` server block is an exception. Reverse-DNS zones are
forwarded through `/etc/resolv.conf` without passing through `archivingcache`,
so they are neither stored in Olric nor archived.

This is intentional. The current crawl path does not issue PTR queries: the
gRPC API only creates A queries, and the Squid configuration does not enable a
reverse-DNS ACL or hostname-logging feature. Keeping `arpa` separate also
ensures that private and Kubernetes reverse zones use the cluster/system
resolver rather than the public recursive resolver set used for crawl
hostnames. Reverse lookups should be added to Olric only if a measured workload
needs them; they should still retain the `/etc/resolv.conf` forwarding path.

The CoreDNS `forward` plugin does not add another DNS response cache. The base
and development configurations forward to `DNS_SERVER` (the base Deployment
defaults to `8.8.8.8`), while production forwards across Google, Cloudflare,
Quad9, and OpenDNS recursive resolvers. Those external recursive resolvers have
their own caches, so an Olric miss means a recursive lookup, not necessarily a
query to an authoritative name server. The remaining TTL returned by the
recursive resolver starts the Veidemann cache lifetime.

## Resolver and Olric cache

The `archivingcache` plugin uses Olric as a shared cache. DNS TTLs determine
entry expiry; cache reads do not extend the lifetime. DNS entries are immutable
after they are stored. Startup depends on an Olric endpoint being reachable.

Relevant `archivingcache` settings:

    archivingcache {
        olricAddress localhost:3320
        olricDmap dns-resolver-archivingcache
        contentWriterHost localhost
        contentWriterPort 5010
        logHost localhost
        logPort 5011
    }

- `olricAddress` accepts either a comma-separated list or repeated directives for multiple peers.
- `olricDmap` selects the distributed map used for cached entries.

### Cache keys and eligibility

DNS responses use this key shape:

```text
dns|v3|<lowercase-fqdn>|<query-type>|<query-class>|<dnssec-do>|<checking-disabled>
```

The collection ID and transport are intentionally absent, allowing port-53
and gRPC requests to share DNS content. A request is eligible only when it is a
standard DNS query with exactly one question. Requests containing EDNS options
are passed to the next plugin without using Olric; an empty EDNS record, as
created by the gRPC API, remains eligible.

The response entry uses the `DNS3` format and contains the packed DNS message,
the upstream recursive resolver address, `storedAt`, and `expiresAt`. It does
not contain collection state.

### Lifetime rules

The cache applies these rules:

| Response | Cache lifetime | Behavior |
| --- | --- | --- |
| Positive answer or delegation | Shortest record TTL, capped at seven days | Return cached copies with every record TTL reduced by the entry's age. |
| NXDOMAIN or NODATA with SOA | `min(SOA TTL, SOA.MINIMUM)`, capped at seven days | Cache and age the negative response. |
| NXDOMAIN or NODATA without SOA | Not cached | Resolve upstream again on the next request. |
| SERVFAIL or upstream failure | Five seconds | Suppress immediate repeated failures; never serve an expired positive answer instead. |
| Truncated response | Not cached | Use only for the current transaction; a client may retry over TCP. |
| Otherwise cacheable response containing a zero non-OPT TTL | Not cached | Use only for the current transaction. |

For positive and negative entries, the shortest non-OPT record TTL across the
answer, authority, and additional sections controls the whole entry. All
record TTLs are capped at seven days before storage. Negative SOA TTLs are
normalized according to [RFC 2308](https://www.rfc-editor.org/rfc/rfc2308.html)
before the shortest TTL is selected.

`storedAt` and `expiresAt` provide logical expiry. Olric receives the same
lifetime as a physical per-entry TTL, but delayed physical removal can never
make an entry usable after `expiresAt`. On a hit, the resolver copies the
stored message and subtracts its age from every non-OPT record TTL. It does not
rewrite the entry or move either expiry.

There is no background refresh, sliding expiration, stale serving, or stale
fallback. The first request after logical expiry resolves upstream and creates
a new immutable generation. Process-local singleflight combines concurrent
misses within one resolver process. It is not a distributed lock, so resolver
replicas can occasionally perform duplicate upstream lookups; the shared
Olric result still becomes available to all replicas.

Olric read failures are treated as cache misses. DNS-entry and marker write
failures are logged without failing the DNS response. Resolution continues
through the upstream resolver, and the response is still returned when
possible. Startup nevertheless fails if Olric or a configured archival service
cannot be connected.

The production Olric configuration limits this DMap to `256 MiB` per pod and
uses LRU capacity eviction. Capacity pressure may therefore discard a DNS
entry or archive marker before its DNS-derived TTL. This causes an earlier
lookup or archival retry; it never permits serving beyond `expiresAt`. No
default DMap TTL replaces the per-entry DNS TTL.

### Per-collection archival

For a positive response with at least one answer, the plugin checks this
separate marker:

```text
key:   dns-archive|v1|<collection-id>|<dns-cache-key>
value: <DNS entry storedAt timestamp>
TTL:   <DNS entry lifetime remaining after archival>
```

If the marker contains the current generation, the response has already been
archived for that collection. Otherwise, the plugin writes the DNS WARC record
and configured crawl log, then writes the marker. A port-53 lookup may populate
the DNS cache first; a later gRPC hit still archives that cached generation for
its collection. Different collections share the DNS response but maintain
independent markers. Refreshing the DNS response changes `storedAt`, so each
collection can archive the new generation.

Archival is at least once. Concurrent first requests for the same collection
may create duplicate archive records, as may a marker write failure after a
successful archive. Archive and marker failures never fail the DNS response,
and a failed archive does not create a marker so a later request can retry.
Positive responses that cannot be cached may be archived for the current
request but do not receive a marker.

The cache format is intended for a fresh Olric DMap. There is no compatibility
reader or migration for earlier entry formats.

With the shipped Docker and Kubernetes configuration, the Olric values are
wired through `OLRIC_ADDRESS` and `OLRIC_DMAP` environment variables.

## Squid DNS caching

Both Squid role templates contain:

```squidconf
dns_nameservers ${DNS_IP}
negative_dns_ttl 1 second
positive_dns_ttl 7 days
```

The Kubernetes child/balancer Deployment and parent/cache StatefulSet both set
`DNS_SERVERS=dns-resolver`. The cache image's configuration handler resolves
each configured server name to IPv4 addresses and substitutes those addresses
into `dns_nameservers`. Each Squid process then sends DNS requests directly to
the `dns-resolver` Service and keeps its own local DNS cache; Squid replicas do
not share this local cache. The balancer's `cache deny all` and peer
`proxy-only` settings disable HTTP object storage, not its DNS cache. See the
[cache README](../cache/README.md) for the complete Squid parent/child and HTTP
object-cache topology.

For Squid 7, `positive_dns_ttl` is an upper limit rather than a forced
lifetime. Squid honors the remaining TTL returned by `dns-resolver`, capped at
seven days. `negative_dns_ttl` controls failed lookups and also establishes the
one-second lower bound for positive lookups. See the Squid documentation for
[`positive_dns_ttl`](https://www.squid-cache.org/Doc/config/positive_dns_ttl/)
and
[`negative_dns_ttl`](https://www.squid-cache.org/Doc/config/negative_dns_ttl/).

The combined behavior is:

| Resolver result | Squid behavior | Resolver/Olric behavior after Squid expires |
| --- | --- | --- |
| Positive with a 30-second remaining TTL | Cache locally for 30 seconds | The aligned Olric entry is normally expired, so the next request refreshes upstream. |
| Positive with a TTL above seven days | Cache locally for seven days | The resolver has already capped the response and Olric entry at seven days. |
| Positive with TTL zero | Cache locally for Squid's one-second minimum | The resolver did not store it, so the next Squid lookup resolves upstream again. |
| NXDOMAIN/NODATA with SOA | Cache the failed lookup locally for one second | Repeated Squid lookups can reuse the shared Olric negative entry until its SOA-derived expiry. |
| NXDOMAIN/NODATA without SOA | Cache the failed lookup locally for one second | The resolver did not store it and queries upstream again. |
| SERVFAIL or upstream failure | Cache the failed lookup locally for one second | Olric suppresses repeated upstream attempts for five seconds. |

Consequently, a Squid hit does not contact `dns-resolver` or Olric. A Squid
miss may still be an Olric hit, and only an Olric miss or logical expiry
requires a recursive upstream lookup. The resolver returns aged TTLs, keeping
independent Squid processes aligned with the shared Olric expiry rather than
starting a fresh full TTL whenever they first see an existing entry.

A Squid process restart loses only that process's local DNS cache; subsequent
requests can warm it from Olric without necessarily querying an upstream
resolver. Restarting a resolver pod loses its process-local singleflight state
but not the shared Olric entries. Losing or clearing the Olric DMap makes DNS
responses and collection markers cold, so responses are resolved again and
collections may archive them again.

## Observability

The plugin exports `coredns_cache_hits_total` and
`coredns_cache_misses_total`. These count DNS response-cache lookups, not
archive-marker checks or Squid-local hits. There is deliberately no exact
Olric cache-size metric: calculating one requires a distributed full-map scan
and must not occur in the DNS request path.

## Example

Run server:

    go run .

If `archivingcache` is enabled in the Corefile, make sure Olric is available at the configured address before starting the resolver.

Query server:

    $ go run ./cmd/resolve vg.no
    time: 143.04964ms
    host:"vg.no" port:80 textual_ip:"195.88.55.16" raw_ip:"\xc3X7\x10"
