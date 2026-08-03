# Veidemann cache

This module builds the Squid proxy image used by Veidemann. The same image has
two roles in Kubernetes:

| Squid role | Kubernetes workload | Responsibility |
| --- | --- | --- |
| Child/balancer | `Deployment/cache`, selected by `Service/cache-balancer` | Accept harvester proxy connections and use CARP to select a parent. It does not cache objects locally. |
| Parent/cache | `StatefulSet/cache`, selected by `Service/cache` | Perform TLS bump, normalize Store-IDs, and cache fetched objects in its own memory and disk store. |

“Parent” and “child” are the terminology used by
[Squid cache hierarchies](https://wiki.squid-cache.org/Features/CacheHierarchy).
The `cache_peer ... parent` declarations make the balancer the child side of
those peer relationships. A harvester is a client of that child proxy; it is
not itself a cache child.

## Request flow

```text
Harvester
  │  CACHE_HOST=cache-balancer, CACHE_PORT=3128
  ▼
Service/cache-balancer
  ▼
Squid child/balancer Deployment
  │  CARP-selected cache_peer ... parent
  ▼
Service/cache EndpointSlices
  ▼
Squid parent/cache StatefulSet pods
  │  TLS bump, Store-ID, memory/disk cache
  ▼
Origin server
```

The child starts the image with `-b`. In this mode, `confighandler`:

1. Reads ready addresses from EndpointSlices belonging to `SERVICE_NAME` in
   `NAMESPACE`.
2. Sorts and deduplicates those addresses.
3. Generates one peer per address using this shape:

   ```squidconf
   cache_peer 10.0.0.10 parent 3128 0 carp no-query no-digest \
       proxy-only no-netdb-exchange connect-timeout=5 connect-fail-limit=2
   ```

4. Validates the generated configuration before committing it.
5. Rechecks discovery every five seconds and reconfigures Squid after a change,
   with a minimum 30-second interval between reconfigurations.

The child configuration uses `cache deny all`; `proxy-only` also prevents
objects received from a parent from being stored by the child. Object storage
therefore belongs exclusively to the parent tier.

Both Squid roles resolve names through the configured `DNS_SERVERS`. Their DNS
cache accepts positive answers for at most seven days and uses a one-second
negative TTL, which is also Squid's minimum positive-cache lifetime. The DNS
resolver supplies the remaining authoritative TTL, so shorter positive TTLs
remain shorter except for Squid's unavoidable one-second minimum.

The parent renders `squid.conf.template`, which configures TLS bump, the
`storeid` helper, origin-compliant caching, and origin access. When disk-cache
sizing is enabled, the entrypoint measures the filesystem mounted at
`/var/spool/squid/cache` and writes the resulting `cache_dir` directive to
`/etc/squid/conf.d/95-cache-dir.conf` before Squid configuration is rendered.

The parent sets `maximum_object_size 16 MB` as its only explicit cache-policy
tuning. Larger responses are still delivered to the crawler, but are not
retained. With no custom `refresh_pattern`, Squid uses its conservative
defaults and honors origin freshness, validation, `private`, and `no-store`
instructions. This prioritizes TLS certificate generation and keeps crawling
standards compliant while retaining ordinary reusable web resources to reduce
repeat origin traffic.

## Why run more than one child?

Multiple child/balancer replicas provide two main benefits:

- **Availability:** `Service/cache-balancer` can send new harvester connections
  to another child when a child pod is not ready.
- **Connection capacity:** accepting proxy connections, parsing requests, and
  forwarding traffic are spread across more Squid processes and nodes.

Adding children does not fragment the parent caches in normal operation. Each
child discovers the same parent addresses in the same sorted order, and CARP
deterministically maps a URL to a parent. A request can therefore arrive through
different children and still reach the parent that is likely to hold that
object.

There are important limits:

- Children are stateless and have `cache deny all`; adding them does not add
  object-cache capacity.
- Parent objects are partitioned by CARP, not replicated. Changing parent
  membership remaps part of the URL space and temporarily increases misses.
- Multiple children do not protect a single parent from failure or provide
  more parent storage. Scale both tiers when both front-end availability and
  cache capacity are required.
- The current manifests do not define explicit Squid readiness or liveness
  probes. Kubernetes detects a process/container exit, but not every possible
  hung or degraded Squid state.

Running more than one parent has a different benefit: it increases aggregate
memory, disk capacity, and origin-fetch throughput. CARP keeps an object on one
parent under a stable peer set, which avoids copying every cached object to
every parent. The tradeoff is that losing or replacing a parent loses its warm
share of the cache until requests repopulate it.

## Dev Kubernetes configuration

Render the configuration with:

```sh
kustomize build deploy/k8s/overlays/dev/cache
```

The rendered dev overlay currently contains:

- One child/balancer replica behind `Service/cache-balancer`.
- One parent/cache replica behind `Service/cache`.
- EndpointSlice discovery permissions for the child’s ServiceAccount.
- A cert-manager-generated CA mounted at `/ca-certificates` for parent TLS
  bumping.
- An AUFS cache sized to 80% of a `250 MiB` generic ephemeral PVC mounted below
  `/var/spool`.
- A Squid exporter on port `9301` in both tiers.

The dev cache volume is a generic ephemeral volume. It survives a container
restart inside the same pod, but Kubernetes deletes it with the pod. It is not
a durable cache across pod replacement.

The two tiers scale independently:

```sh
kubectl scale deployment/cache --replicas=2
kubectl scale statefulset/cache --replicas=3
```

The first command adds children; the second adds parent cache capacity. The dev
balancer init container waits for the first parent through the headless cache
Service in the pod's namespace before starting Squid.

## Production Kubernetes configuration

Render the configuration with:

```sh
kustomize build deploy/k8s/overlays/prod/cache
```

The production overlay runs two child/balancer replicas and three parent/cache
replicas. Each parent receives a `50 GiB` generic ephemeral volume from
`topolvm-provisioner-thin`. Squid uses 80% of the mounted filesystem (roughly
`40 GiB`) and the image defaults of `16` first-level and `256` second-level
AUFS directories.

These thin-provisioned volumes are disposable caches. Replacing a parent pod
deletes its volume and starts that parent cold. Changing parent membership also
changes the CARP mapping, temporarily increasing origin traffic while the new
peer set warms. Monitor both parent cache utilization and the TopoLVM thin pool
to avoid overcommitting physical storage.

## Runtime configuration

| Input | Used by | Purpose |
| --- | --- | --- |
| `-b` | `entrypoint.sh` / `confighandler` | Select child/balancer mode. Without it, the image runs as a parent cache. |
| `DNS_SERVERS` | Both roles | Space-separated DNS server names or addresses resolved to IPv4 addresses for Squid. |
| `SERVICE_NAME` | Child | Service whose EndpointSlices contain parent pods. |
| `NAMESPACE` | Child | Namespace used for EndpointSlice discovery. |
| `CACHE_DIR_SIZE_PERCENT` | Parent | Integer percentage from 1 through 90 used to size the AUFS cache from the filesystem mounted at `/var/spool/squid/cache`. |
| `CACHE_DIR_SIZE_MB` | Parent | Explicit positive cache size in MiB. This is an alternative to percentage-based sizing. |
| `CACHE_DIR_L1` / `CACHE_DIR_L2` | Parent | AUFS directory fan-out. Defaults to `16` and `256`. |
| `/ca-certificates/tls.crt` and `tls.key` | Parent | Signing CA used for TLS bumping. |
| `/etc/squid/conf.d/*.conf` | Both roles | Deployment-specific Squid configuration fragments. |

The image owns `/etc/squid/squid.conf` and the role templates. Deployment
overlays must not replace them or mount over the whole `/etc/squid/conf.d`
directory: that would hide image fragments and prevent runtime-generated role
and disk-cache configuration. Add optional settings as individual, numbered
files such as `/etc/squid/conf.d/00-production.conf`, mounted with a ConfigMap
`subPath`.

`CACHE_DIR_SIZE_PERCENT` and `CACHE_DIR_SIZE_MB` are mutually exclusive. When
neither is set, the entrypoint does not configure a disk cache. Generated
sizing also cannot be combined with a manually supplied `cache_dir` fragment.
The percentage is recalculated at pod startup, so restart the parent after
expanding its volume.

Squid runs as the unprivileged `proxy` user. Access logs are forwarded to
container stdout, diagnostic/cache logs to stderr, and store logs are disabled.
The forwarding FIFOs live only in `/run/squid`; no persistent log files or
custom logfile daemon are used.

## Helpers

- `confighandler` renders the role configuration, validates changes, discovers
  parent endpoints in child mode, and triggers safe Squid reconfiguration.
- `storeid` combines the crawl job identifier and request URL into a stable
  Store-ID so crawl requests for the same job and URL share a cache key.

Run the helper tests from the Go module:

```sh
cd cache/helpers
go test ./...
```
