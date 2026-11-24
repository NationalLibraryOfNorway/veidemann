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

Parents mount `tls.crt` and `tls.key` individually using Secret `subPath`
mounts. Kubernetes does not propagate Secret updates into these mounts, so
renewal leaves running parents on their existing CA. Replace the parent pods
with a manual StatefulSet rollout to adopt the renewed CA and rebuild the
generated-certificate database. The helper validates TLS material once at
startup; its periodic loop only tracks DNS and parent discovery. It neither
monitors CA changes nor requests restarts. Child/balancer pods neither mount
the CA nor initialize generated-certificate state.

The dev Certificate requests a five-year lifetime (`duration: 43800h`) and
renewal 90 days before expiry (`renewBefore: 2160h`), with explicit
`privateKey.rotationPolicy: Always`. Apply the same settings to production's
`cache-ca` Certificate. Cert-manager still renews the Secret automatically;
an operator must roll the parents before their mounted CA expires. A fresh
Secret does not imply that the running parents use its certificate.

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
- Parent startup and readiness probes require the helper's ready file and a
  connection to Squid's loopback port 3128, bypassing any Linkerd listener.
  They do not test origin connectivity or TLS generation. Children have no
  explicit Squid probes, and neither role has a liveness probe.

Running more than one parent has a different benefit: it increases aggregate
memory, disk capacity, and origin-fetch throughput. CARP keeps an object on one
parent under a stable peer set, which avoids copying every cached object to
every parent. Losing a parent's volume loses its warm share of the cache until
requests repopulate it. Replacing its pod preserves that share when the same
claim is reused, although a changed peer address can still remap requests.

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
- A `100 MiB` AUFS object cache on a `250 MiB` persistent PVC mounted at
  `/var/spool`, leaving room for the certificate database and metadata.
- A Squid exporter on port `9301` in both tiers.

The shared base defaults to two parents; dev explicitly overrides this to one.
Production needs at least two healthy parents for a rolling CA change. Dev's
single parent pauses harvesting when replaced. The base uses `OrderedReady`
and `RollingUpdate`, with 45 seconds of continuous readiness before proceeding
to the next parent so balancer discovery and its throttled reconfigure can
settle. Startup allows five minutes to open Squid's port, readiness checks every
five seconds, and the 90-second termination grace period exceeds Squid's
30-second shutdown wait. Individual connections may still be interrupted.

The base StatefulSet defines a `cache` volume claim template with a `1 GiB`
default and no explicit storage class. Dev overrides its capacity to `250 MiB`
and uses the cluster's default storage class. Explicit cache sizing avoids
using the entire backing filesystem reported by Minikube's hostPath storage.
The base itself does not enable an object disk cache without a sizing variable.

The two tiers scale independently:

```sh
kubectl scale deployment/cache --replicas=2
kubectl scale statefulset/cache --replicas=3
```

The first command adds children; the second adds parent cache capacity. The dev
balancer init container waits for the first parent through the headless cache
Service in the pod's namespace before starting Squid.

## Parent storage lifecycle and migration

The base and dev overlay use the StatefulSet's `volumeClaimTemplates`, rather
than pod-owned generic ephemeral volumes. With template name `cache`, the claims
are named `cache-cache-0`, `cache-cache-1`, and so on. The default retention
policy is `Retain` for both StatefulSet deletion and scale-down. Pod replacement
reuses the existing claim; scaling down or removing the StatefulSet leaves
claims behind until an operator explicitly deletes them. PVC retention does
not override the PV's reclaim policy when a PVC is deliberately deleted.

Existing StatefulSets using the old generic ephemeral volumes cannot gain a
claim template through an ordinary update: `volumeClaimTemplates` is immutable.
Test the transition in dev:

1. Build an image without `VOLUME /var/spool/squid` and select it for the rollout.
   Otherwise an image-defined nested mount can hide the PVC mounted at
   `/var/spool`.
   Ensure the config-copy init container and Squid container use the same image.
2. Plan a StatefulSet recreation in the intended cluster and namespace. One
   approach is to delete only the StatefulSet with orphan propagation, preserving
   its pods, then apply the new manifests so the replacement controller adopts
   and rolls those pods. Expect new claims and a cold cache during migration;
   orphaning the StatefulSet does not preserve pod-owned claims when those pods
   are subsequently replaced.
3. Verify the new `cache-cache-N` claims are bound at the expected capacity and
   storage class. `/var/spool`, `/var/spool/squid`, and `/var/spool/squid/cache`
   must resolve to the same filesystem inside a parent. Old claims such as
   `cache-0-cache` are not automatically migrated or reused. If old data must be
   preserved, arrange a separate offline copy before deleting its owner pod.
4. On dev, record the new PVC's UID and write a temporary marker in the spool.
   Replace `cache-0`, then verify that the PVC UID and marker survive. Remove
   the marker afterward. Also verify successful requests and log forwarding
   after Squid reconfiguration, stable restart counts, and no filesystem errors.

Both cache roles have read-only root filesystems. `/run` (`16 MiB`),
`/etc/squid/conf.d` (`4 MiB`), and `/tmp` (`32 MiB`) are bounded `emptyDir`
mounts. An init container copies the image's configuration fragments into the
writable config volume before runtime configuration is generated. It also puts
the netdb journal under `/var/spool/squid`. The parent uses its PVC for that
spool; the balancer uses a separate `128 MiB` `emptyDir` and stores no objects.
There must be no extra parent mount at `/var/spool/squid` hiding the PVC.

Ephemeral-storage requests and limits budget the local runtime volumes and
container logs, separately from the parent PVC. Squid defaults to a `256 MiB`
request and `512 MiB` limit; the exporter defaults to `32 MiB` and `128 MiB`.
Validate that only the intended volume mounts accept writes and that writing
elsewhere, for example `/etc`, fails with a read-only filesystem error.

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

The image owns `/etc/squid/squid.conf` and the role templates. The base manifests
mount a writable `/etc/squid/conf.d` seeded from the image by the config-copy
init container. Overlays must preserve this initialization and runtime writes.
Add optional settings as individual, numbered files such as
`/etc/squid/conf.d/00-custom.conf`, mounted with a ConfigMap `subPath`.

`CACHE_DIR_SIZE_PERCENT` and `CACHE_DIR_SIZE_MB` are mutually exclusive. When
neither is set, the entrypoint does not configure a disk cache. Generated
sizing also cannot be combined with a manually supplied `cache_dir` fragment.
The percentage is recalculated at container startup, so restart the parent
container after expanding its filesystem. Resize existing PVCs explicitly;
editing a claim template does not resize existing claims and cannot be applied
as an ordinary StatefulSet update.

Squid runs as the unprivileged `proxy` user. Access logs are forwarded to
container stdout, diagnostic/cache logs to stderr, and store logs are disabled.
The forwarding FIFOs live only in `/run/squid`; no persistent log files or
custom logfile daemon are used.

## Helpers

- `confighandler` renders the role configuration, validates changes, discovers
  parent endpoints in child mode, validates parent CA material at startup, and
  triggers Squid reconfiguration for DNS and parent changes. CA adoption is
  owned by the manual pod rollout.
- `storeid` combines the crawl job identifier and request URL into a stable
  Store-ID so crawl requests for the same job and URL share a cache key.

Run the helper tests from the Go module:

```sh
cd cache/helpers
go test ./...
```

## Manual certificate rollout

Use Bash with `kubectl`, `cmctl`, `jq`, `yq` v4, and OpenSSL on the operator's
machine. Check the intended cluster and namespace before running these
commands. The following examples use dev's namespace; production must use its
own namespace and deployment overlay.

```bash
set -euo pipefail
CACHE_NAMESPACE=default
CACHE_OVERLAY=deploy/k8s/overlays/dev/cache
CACHE_CA_SOURCE=cache-0
CACHE_CA_BACKUP="cache-ca-backup-$(date -u +%Y%m%d-%H%M%S)"
kubectl config current-context
kubectl -n "$CACHE_NAMESPACE" get statefulset/cache
kubectl -n "$CACHE_NAMESPACE" get pods \
  -l app.kubernetes.io/name=cache,app.kubernetes.io/component=cache
```

### Initial migration

From the repository root, build the updated cache image locally with:

```bash
docker build -t ghcr.io/nationallibraryofnorway/veidemann/cache:0.12.0 cache
```

The manifests select `0.12.0` for both roles and their config-copy init
containers. Make that image available to the target cluster before applying
the manifests. Publish it through the existing `cache-v0.12.0` release workflow
when shipping the release; editing manifests does not publish the image.

Apply the updated workloads first, leaving the deployed Certificate and
Issuer specifications untouched. Do not apply the full overlay yet: changing
the Certificate's duration can trigger reissuance while old pods still have
live Secret mounts.

```bash
kustomize build "$CACHE_OVERLAY" \
  | yq 'select(.kind != "Certificate" and .kind != "Issuer")' \
  | kubectl -n "$CACHE_NAMESPACE" apply -f -
kubectl -n "$CACHE_NAMESPACE" rollout status statefulset/cache --timeout=15m
kubectl -n "$CACHE_NAMESPACE" rollout status deployment/cache --timeout=15m
```

Confirm all parents use the individual `subPath` mounts and run successful
HTTP/HTTPS harvests. Production must have at least two ready parents and no
rollout in progress. For a dev rehearsal, temporarily scale the parent
StatefulSet to two replicas after applying the overlay. Applying the dev
overlay again resets it to one.

Preserve the currently mounted CA using the backup step below. Only after
the workloads are stable, apply the new certificate policy:

```bash
kustomize build "$CACHE_OVERLAY" \
  | yq 'select(.kind == "Certificate" or .kind == "Issuer")' \
  | kubectl -n "$CACHE_NAMESPACE" apply -f -
```

Wait for completed issuance and validate the Secret as described below, then
roll the parents. The same staged order applies to production overlays.

### Preserve the current CA

Before requesting renewal, record the parents' Ready conditions, container
restart counts, mounted certificate fingerprints, and expiry dates. Require
at least two ready parents in production and successful HTTPS harvesting. Pick
a healthy parent with the previous valid CA as `CACHE_CA_SOURCE`; do not run
another rollout while taking the backup.

Back up the mounted pair rather than the managed Secret: cert-manager might
already have renewed the Secret. These commands keep the private key out of
terminal output and create an immutable Secret without cert-manager ownership.

```bash
(
  umask 077
  CACHE_CA_TMP=$(mktemp -d)
  trap 'rm -rf "$CACHE_CA_TMP"' EXIT
  kubectl -n "$CACHE_NAMESPACE" exec "$CACHE_CA_SOURCE" -c cache -- \
    cat /ca-certificates/tls.crt > "$CACHE_CA_TMP/tls.crt"
  kubectl -n "$CACHE_NAMESPACE" exec "$CACHE_CA_SOURCE" -c cache -- \
    cat /ca-certificates/tls.key > "$CACHE_CA_TMP/tls.key"
  openssl x509 -in "$CACHE_CA_TMP/tls.crt" -checkend 0 -noout
  kubectl -n "$CACHE_NAMESPACE" create secret tls "$CACHE_CA_BACKUP" \
    --cert="$CACHE_CA_TMP/tls.crt" --key="$CACHE_CA_TMP/tls.key" \
    --dry-run=client -o json \
    | jq '.immutable = true' \
    | kubectl -n "$CACHE_NAMESPACE" create -f -
)
```

### Issue and verify

Inspect the managed Secret's public certificate and the Certificate's status:

```bash
cache_secret_certificate() {
  kubectl -n "$CACHE_NAMESPACE" get secret cache \
    -o jsonpath='{.data.tls\.crt}' | base64 --decode
}
cache_secret_certificate | openssl x509 -noout -dates -fingerprint -sha256
kubectl -n "$CACHE_NAMESPACE" get certificate cache-ca \
  -o jsonpath='{.status.revision}{"\n"}{.status.conditions}{"\n"}'
```

If automatic renewal or the initial duration change has already produced a
new valid certificate, skip the renewal request. Otherwise record the current
Secret fingerprint and Certificate revision, then request renewal:

```bash
cmctl renew -n "$CACHE_NAMESPACE" cache-ca
```

Repeat the inspection commands until the revision has advanced, the Secret's
certificate fingerprint has changed, and `Ready=True` describes the completed
issuance. `Ready=True` alone may still refer to the previous issuance. If
renewal does not complete, diagnose `kubectl describe certificate cache-ca`
and its CertificateRequests; keep the existing parents serving and do not
restart them onto unverified material.

Before rollout, validate the actual new pair and CA properties locally:

```bash
(
  umask 077
  CACHE_CA_TMP=$(mktemp -d)
  trap 'rm -rf "$CACHE_CA_TMP"' EXIT
  # One API read captures a consistent certificate/key pair.
  kubectl -n "$CACHE_NAMESPACE" get secret cache -o json > "$CACHE_CA_TMP/secret.json"
  jq -r '.data["tls.crt"]' "$CACHE_CA_TMP/secret.json" \
    | base64 --decode > "$CACHE_CA_TMP/tls.crt"
  jq -r '.data["tls.key"]' "$CACHE_CA_TMP/secret.json" \
    | base64 --decode > "$CACHE_CA_TMP/tls.key"
  openssl x509 -in "$CACHE_CA_TMP/tls.crt" -noout -dates -fingerprint -sha256
  openssl verify -CAfile "$CACHE_CA_TMP/tls.crt" "$CACHE_CA_TMP/tls.crt"
  openssl x509 -in "$CACHE_CA_TMP/tls.crt" -noout \
    -ext basicConstraints,keyUsage
  openssl x509 -in "$CACHE_CA_TMP/tls.crt" -pubkey -noout \
    > "$CACHE_CA_TMP/cert.pub"
  openssl pkey -in "$CACHE_CA_TMP/tls.key" -pubout \
    > "$CACHE_CA_TMP/key.pub"
  cmp "$CACHE_CA_TMP/cert.pub" "$CACHE_CA_TMP/key.pub"
)
```

Require currently valid dates, `CA:TRUE`, certificate-signing permission, and
matching public keys. The parent helper independently enforces these checks
at startup before signalling readiness.

### Roll and verify the parents

Record the target Secret fingerprint, then trigger one ordered rollout:

```bash
kubectl -n "$CACHE_NAMESPACE" rollout restart statefulset/cache
kubectl -n "$CACHE_NAMESPACE" rollout status statefulset/cache --timeout=15m
```

Monitor parent readiness, restart counts, and harvesting during the rollout.
Kubernetes replaces the highest ordinal first and waits for the replacement
to remain ready for 45 seconds before replacing the next parent. A replacement
that fails startup or readiness halts further progression. Do not delete all
parents or initiate repeated restarts to work around a stalled rollout.

Inspect the CA actually mounted in every parent:

```bash
for CACHE_POD in $(kubectl -n "$CACHE_NAMESPACE" get pods \
  -l app.kubernetes.io/name=cache,app.kubernetes.io/component=cache \
  -o jsonpath='{.items[*].metadata.name}'); do
  printf '%s\n' "$CACHE_POD"
  kubectl -n "$CACHE_NAMESPACE" exec "$CACHE_POD" -c cache -- \
    cat /ca-certificates/tls.crt \
    | openssl x509 -noout -dates -fingerprint -sha256
done
```

Require every fingerprint to match the target, stable readiness, successful
HTTPS harvesting, and expected MISS/HIT behavior. Confirm startup logs show
the generated-certificate database rebuilt when the CA changed. Pod replacement
preserves the object-cache PVC; it only invalidates the generated TLS database.

### Rollback

If a replacement fails, retain the healthy parents and inspect its logs and
events. If the new CA is the cause, verify the backup remains valid and point
the StatefulSet's certificate volume at it. This changes the pod template and
initiates an ordered rollback without modifying the cert-manager-managed Secret.

```bash
jq -n --arg secret "$CACHE_CA_BACKUP" \
  '{spec: {template: {spec: {volumes: [
    {name: "cache-certificate", secret: {secretName: $secret}}
  ]}}}}' \
  | kubectl -n "$CACHE_NAMESPACE" patch statefulset/cache \
      --type=strategic --patch-file=/dev/stdin
```

An OrderedReady StatefulSet can remain stuck waiting for its already-created
bad pod after reverting the template. Identify that failed replacement, then
delete only it so Kubernetes recreates it using the rollback template:

```bash
# Set this to the failed replacement after inspecting its status and logs.
FAILED_CACHE_POD=cache-1
kubectl -n "$CACHE_NAMESPACE" delete pod "$FAILED_CACHE_POD"
kubectl -n "$CACHE_NAMESPACE" rollout status statefulset/cache --timeout=15m
```

Verify mounted fingerprints and HTTPS harvesting again. While using the backup,
prevent GitOps or another overlay apply from restoring `secretName: cache`
prematurely. Once the managed Secret has been corrected and verified, restore
the normal volume reference through the deployment overlay and monitor that
ordered rollout. Retain the backup until recovery is verified, then remove it
according to the deployment's Secret retention policy.

### Rehearsal and ongoing checks

Render the base and dev overlay before deployment. Rehearse with two parents:
renewal alone must not change pod UIDs, restart counts, or mounted fingerprints;
manual rollout must adopt the new fingerprint one parent at a time while
harvesting continues. Exercise a failed replacement and the rollback procedure.
Check HTTP/HTTPS MISS/HIT behavior, TLS generation, and PVC reuse. Live-cluster
validation cannot be inferred from a successful Kustomize render.

At least annually, inspect every parent's mounted certificate using the commands
above and rehearse rotation well before expiry. After any automatic Secret
renewal, complete the manual rollout within the old CA's remaining lifetime.
The helper no longer polls certificate validity at runtime; startup validation
and a TCP readiness probe do not monitor the active CA's future expiry.
