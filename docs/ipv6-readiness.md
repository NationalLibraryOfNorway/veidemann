# IPv6 readiness

## Status

Veidemann's supported crawl path is currently IPv4-only. Continue using DNS A
records for crawl scheduling and recorder metadata. The repository contains
several IPv6-capable data types and parsing paths, but IPv6 origin crawling has
not been implemented or verified end to end.

Do not infer IPv6 readiness from the AAAA response branch in
[`dns-resolver/plugin/resolve/resolve.go`](../dns-resolver/plugin/resolve/resolve.go).
The same method constructs an A query unconditionally, so that branch is not
normally reached by the gRPC resolver path.

## Current behavior by subsystem

| Subsystem | Current state | Consequence |
| --- | --- | --- |
| DnsResolver gRPC API | `Resolve` always issues `dns.TypeA` and returns one `textual_ip`/`raw_ip` pair. `raw_ip` can technically hold either 4 or 16 bytes. | There is no A/AAAA selection, fallback, or ordered address list. |
| DNS archival cache | Cache keys include the DNS question type and can store native AAAA responses. Collection-aware gRPC resolution currently generates only A queries. | Existing crawl DNS records normally describe A lookups. A future dual-query design must decide how A and AAAA records are archived and identified. |
| Frontier | Java's `InetAddress.getByAddress` can consume a 16-byte reply, and queued URIs store the selected IP as a string. Frontier relies on the A-only gRPC result for DNS preconditions and politeness grouping. | An AAAA-only queued target receives A NODATA and is not scheduled successfully. IPv6 politeness ranges and grouping have no dedicated coverage. |
| Recorderproxy | URL/authority handling supports bracketed IPv6 literals and removes IPv6 zone identifiers. Crawl metadata stores an IP string. The executable requires Squid; direct origin dialing is unsupported. | Recorderproxy's recorded IP comes from the A-only resolver call, while Squid independently chooses the origin address. For browser subresources, an empty A result can therefore coexist with a Squid connection chosen through separate DNS behavior. |
| Squid cache | Squid performs the actual origin connection. Its configured DNS server endpoints are resolved to IPv4 only by `cache/helpers/iputil`. | The DNS control path assumes IPv4. Squid's actual origin address is not returned to recorderproxy, so an IPv6 origin connection could not currently be attributed reliably in the crawl log. |
| ContentWriter and logs | IP addresses are represented as strings and written to the WARC `WARC-IP-Address` header. | Storage is representation-capable, but does not prove that the selected address was contacted. |
| Kubernetes | Services do not declare `ipFamilies` or `ipFamilyPolicy`; overlays contain no tested dual-stack contract. | IPv6 pod egress, service discovery, DNS reachability, and NetworkPolicy behavior depend on unverified cluster defaults. |

Relevant implementation points:

- [`dns-resolver/plugin/resolve/resolve.go`](../dns-resolver/plugin/resolve/resolve.go)
  constructs the A query and maps a single answer into `ResolveReply`.
- [`frontier/.../DnsServiceClient.java`](../frontier/src/main/java/no/nb/nna/veidemann/frontier/worker/DnsServiceClient.java)
  converts `raw_ip` into an `InetAddress`.
- [`frontier/.../CrawlHostGroupCalculator.java`](../frontier/src/main/java/no/nb/nna/veidemann/frontier/worker/CrawlHostGroupCalculator.java)
  parses addresses for politeness grouping, but lacks IPv6-specific tests.
- [`recorderproxy/recorderproxy/dns_lookup_filter.go`](../recorderproxy/recorderproxy/dns_lookup_filter.go)
  copies the resolver's single textual address into recorder state.
- [`cache/helpers/iputil/iputil.go`](../cache/helpers/iputil/iputil.go) selects
  only IPv4 addresses for Squid's `dns_nameservers` configuration.
- [`deploy/k8s/base/dns-resolver/service.yaml`](../deploy/k8s/base/dns-resolver/service.yaml)
  leaves Kubernetes IP-family selection implicit.

The IP address in a `dns:<host>` crawl log is a separate concept: it records
the upstream DNS server associated with the archived response, not an A or AAAA
answer. See
[`dns-resolver/plugin/archivingcache/logwriter.go`](../dns-resolver/plugin/archivingcache/logwriter.go).

## Requirements before enabling IPv6 crawling

IPv6 should be treated as a feature project rather than enabled by changing the
gRPC query from A to AAAA. Complete all of the following:

1. Define address selection semantics: A-first with AAAA fallback, IPv6-first,
   or a timed multi-address strategy. Define retry behavior when one family is
   unreachable.
2. Extend the resolver contract to return the required family information or
   ordered address candidates. Preserve compatibility for current clients.
3. Align Frontier, recorderproxy, and Squid so the address used for scope and
   politeness checks, the address contacted, and the address stored in the crawl
   log cannot silently disagree. If Squid remains the origin dialer, expose or
   otherwise verify its selected origin address.
4. Decide whether A and AAAA responses produce separate DNS WARC/crawl-log
   records and how their query types are represented.
5. Audit Frontier crawl-host-group range matching for IPv6, mixed address
   families, and ranges crossing the high bit; add backend tests matching the
   dashboard's IPv6 input support.
6. Remove IPv4-only infrastructure assumptions where necessary, including
   Squid DNS-server discovery, and declare the intended Kubernetes dual-stack
   service and egress behavior in deployment overlays.
7. Add end-to-end tests for AAAA-only and dual-stack hostnames, IPv6 literals,
   HTTP and HTTPS through Squid, CONNECT and empty-SNI TLS, cache hits and
   misses, family fallback, failures, WARC metadata, and crawl-log attribution.
8. Verify the chosen production cluster/CNI has IPv6 pod egress and that
   NetworkPolicy and observability cover IPv6 traffic before rollout.

## Re-evaluation triggers

Reassess this document when any of these change:

- the DnsResolver gRPC query type or reply shape;
- Frontier DNS preconditions or crawl-host-group calculation;
- recorderproxy begins dialing the resolved origin directly;
- Squid exposes its selected origin address or changes DNS/address-family
  policy;
- Kubernetes environments adopt an explicit dual-stack networking contract;
- IPv6 origin integration tests are added.
