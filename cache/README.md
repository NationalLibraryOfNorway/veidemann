# Veidemann cache

This module builds the Squid proxy image used by Veidemann. The same image has
two roles in Kubernetes:

| Squid role | Kubernetes workload | Responsibility |
| --- | --- | --- |
| Child/balancer | `Deployment/cache`, selected by `Service/cache-balancer` | Accept recorderproxy connections and use CARP to select a parent. It does not cache objects locally. |
| Parent/cache | `StatefulSet/cache`, selected by `Service/cache` | Perform TLS bump, normalize Store-IDs, and cache fetched objects in its own memory and disk store. |

“Parent” and “child” are the terminology used by
[Squid cache hierarchies](https://wiki.squid-cache.org/Features/CacheHierarchy).
The `cache_peer ... parent` declarations make the balancer the child side of
those peer relationships. Recorderproxy is the only client of that child
proxy; it is not itself a cache child.

## Request flow

```text
recorderproxy
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

The parent sets `maximum_object_size 16 MB`, so larger responses are still
delivered to the crawler but are not retained. It also writes out Squid's three
default `refresh_pattern` values and adds only `ignore-reload`. Chromium's local
cache remains disabled so recorderproxy observes every request, while Squid may
serve a fresh cached response despite the controlled client sending
`Cache-Control: no-cache` or `Pragma: no-cache`.

Writing all three patterns preserves Squid's normal FTP, CGI/query-string, and
catch-all freshness calculations; defining any explicit pattern disables the
implicit default list. `ignore-reload` does not make stale responses fresh or
override origin `private`, `no-store`, expiry, or validation requirements.
Store-ID keys remain scoped by job execution, so this exception does not enable
reuse across different `veidemann_jeid` values.

## Dynamic parent TLS certificates

TLS bump requires Squid to terminate each client TLS connection so it can apply
Store-ID normalization and caching to the decrypted HTTP exchange. Parent pods
mount the cert-manager-managed signing CA Secret named `cache` at
`/ca-certificates`. Squid uses that CA to generate a certificate matching each
requested origin through `security_file_certgen`; generated certificates use a
32 MiB memory cache and a 64 MiB database under `/var/spool/squid/ssl_db`.

Before becoming ready, `confighandler` verifies that the CA certificate and
private key are readable, match, are currently valid, and permit certificate
signing. The entrypoint initializes the generated-certificate database and
stores the CA fingerprint alongside it. If the fingerprint differs at the next
startup, the database is rebuilt before Squid starts.

At runtime, `confighandler` fingerprints the mounted CA files every five
seconds. A valid CA rotation causes the helper and container to exit so
Kubernetes restarts the parent; startup then rebuilds the generated-certificate
database. Transiently incomplete or invalid Secret updates are retried without
disturbing the running Squid process. Child/balancer pods neither mount the CA
nor initialize generated-certificate state.

## Why run more than one child?

Multiple child/balancer replicas provide two main benefits:

- **Availability:** `Service/cache-balancer` can send new recorderproxy connections
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
- A cert-manager-generated signing CA mounted at `/ca-certificates` in parent
  pods only.
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
| `--tls-cert-file` | Parent | TLS signing CA certificate path. Defaults to `/ca-certificates/tls.crt`. |
| `--tls-key-file` | Parent | TLS signing CA private-key path. Defaults to `/ca-certificates/tls.key`. |
| `/ca-certificates/tls.crt` and `tls.key` | Parent | CA certificate and matching private key used to generate per-origin TLS certificates. |
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
  parent endpoints in child mode, validates parent CA material, triggers safe
  Squid reconfiguration for ordinary changes, and requests a container restart
  when the signing CA rotates.
- `storeid` combines the crawl job identifier and request URL into a stable
  Store-ID so crawl requests for the same job and URL share a cache key.

Run the helper tests from the Go module:

```sh
cd cache/helpers
go test ./...
```

## Certificate rollout

Apply the `cache-ca` Certificate and wait for its `cache` Secret to be Ready
before rolling out the parent StatefulSet. In production, allow the ordered
rollout to replace and verify one parent at a time. Validate HTTPS MISS/HIT
behavior and confirm the generated peer certificate matches the requested
origin. The obsolete `cache-server` Certificate and `cache-server-tls` Secret
are not used by this configuration and may be removed after rollback.
