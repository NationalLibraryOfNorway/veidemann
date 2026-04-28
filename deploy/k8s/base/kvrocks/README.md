# Kvrocks Kustomize Base

This directory contains a plain Kubernetes and Kustomize base for Apache Kvrocks running as a replicated StatefulSet with Redis Sentinel sidecars.

The base is intended for a simple master/replica deployment with automatic failover. It is not a Kvrocks cluster-mode deployment.

This base requires password-protected Kvrocks and Sentinel, and uses durable Sentinel state, conservative rollout behavior, explicit resource requests and limits, and a default PVC size. It still needs overlay-specific choices for TLS, metrics, and final network-policy narrowing.

## Topology

- `StatefulSet/kvrocks` with 3 replicas by default
- One `apache/kvrocks` container per pod
- One `redis` Sentinel sidecar per pod
- Headless service for stable pod DNS and peer discovery
- Sentinel service for master discovery
- `kvrocks-any` service for diagnostics and read-oriented access to arbitrary pods; not a write endpoint

## Included Resources

- `serviceaccount.yaml`
- `configmap.yaml`
- `scripts-configmap.yaml`
- `service.yaml`
- `pdb.yaml`
- `networkpolicy.yaml`
- `statefulset.yaml`

## Bootstrap And Failover

Only an empty `kvrocks-0` PVC is allowed to bootstrap automatically.

When that happens, `kvrocks-start.sh` records the decision on the PVC as:

- `/var/lib/kvrocks/cluster-state/initial-bootstrap-authorized`

The marker only authorizes the Sentinel sidecar on `kvrocks-0` to create initial Sentinel state after local Kvrocks reports `role=master`. It is not a recovery signal and must not be treated as proof that `kvrocks-0` is the current master after the cluster has been running. Once Sentinel state exists, Sentinel state is the source of truth for master identity.

During that first handoff, the Sentinel sidecar waits for the local StatefulSet pod FQDN to resolve and then monitors the bootstrap master by that stable DNS name. This avoids the initial DNS race without persisting a pod IP as durable master identity.

On startup, pods first ask existing Sentinels for the current master. Sentinel state is stored on the pod PVC, so normal restarts preserve the last known master and config epoch.

If a pod has existing data but no Sentinel can provide the current master, the pod now fails closed instead of falling back to `kvrocks-0`. This prevents stale-master resurrection after a full outage.

If durable Sentinel state is unavailable and operator recovery is required, set `KVROCKS_RECOVERY_MASTER` to the chosen pod DNS name before restarting the StatefulSet, for example `kvrocks-1.kvrocks-headless.<namespace>.svc`.
Choose `KVROCKS_RECOVERY_MASTER` only after identifying the member with the most recent valid data; setting it incorrectly can replicate stale data over newer data.

During termination of the active master, the `preStop` hook asks the local Sentinel to trigger failover and waits briefly for a new master to be elected.

StatefulSet updates use `OnDelete` so normal image or config rollouts do not automatically restart the current master.

The StatefulSet uses `podManagementPolicy: Parallel` intentionally. With fail-closed startup, ordered startup can deadlock after a failover if `kvrocks-0` starts first, cannot discover the durable master, and blocks later pods from starting. Parallel startup avoids that recovery deadlock at the cost of some initial restart noise during first bootstrap.

The base uses best-effort pod anti-affinity so multi-node clusters spread replicas when possible, while still allowing single-node development clusters. Production overlays should replace this with required anti-affinity and stricter topology spread constraints.

## Services

### `kvrocks-headless`

Provides stable per-pod DNS names for the StatefulSet, for example:

- `kvrocks-0.kvrocks-headless.<namespace>.svc`
- `kvrocks-1.kvrocks-headless.<namespace>.svc`

This service is used internally for bootstrap, replica wiring, and Sentinel peer discovery.

### `kvrocks-sentinel`

Exposes Sentinel on port `26379` for clients that support Sentinel-based master discovery.

Example:

```sh
redis-cli -h kvrocks-sentinel -p 26379 sentinel get-master-addr-by-name kvrocks
```

### `kvrocks-any`

Exposes Kvrocks on port `6666` for all pods selected by the StatefulSet labels.

This is useful for diagnostics and some read-oriented traffic, but it is not a stable write endpoint in a Sentinel-managed topology. Kubernetes may route a connection to a replica.

Authentication still applies; this service only changes routing, not access control.

If a stable write endpoint is required for non-Sentinel-aware clients, add a separate master-discovery component that updates a dedicated service after failover.

## Authentication

Authentication is required.

Create a Secret named `kvrocks-auth` in the same namespace with:

- `redis-password`
- `sentinel-password`

The startup scripts:

- add `requirepass` and `masterauth` to the generated Kvrocks configuration
- add `requirepass` to Sentinel so both Sentinel clients and peer Sentinels must authenticate
- add `sentinel sentinel-pass ...` so each Sentinel authenticates to its peers explicitly
- add `sentinel auth-pass kvrocks ...` to the generated Sentinel configuration
- use authenticated `redis-cli` calls for Kvrocks health checks and Kvrocks-side helper commands
- use authenticated `redis-cli` calls for Sentinel health checks and helper commands

The scripts reject secrets with control characters before generating config files.

Kvrocks replicas announce their stable StatefulSet pod FQDNs to the master. This keeps durable Sentinel state tied to StatefulSet member identity instead of transient pod IPs across restarts.

### Example Overlay Secret

The development overlay in [deploy/k8s/overlays/dev/kvrocks](../../overlays/dev/kvrocks) shows the required secret shape.

Render it with:

```sh
kustomize build deploy/k8s/overlays/dev/kvrocks
```

Before applying it, replace the example passwords in [deploy/k8s/overlays/dev/kvrocks/kustomization.yaml](../../overlays/dev/kvrocks/kustomization.yaml) with real values or switch the secret generator to your normal secret source.

## Usage

1. Reference this base from an overlay.
2. Set namespace, images, resources, storage class, and PVC size in the overlay.
3. Narrow the NetworkPolicy selectors for the workloads that should access Kvrocks and Sentinel.
4. Provide both required passwords through `kvrocks-auth`.
5. Use Sentinel-aware clients against `kvrocks-sentinel:26379` when writing to the current master.
6. Add backup export automation only through an overlay or optional component, not by editing this base directly.

## Backups

The base does not include backup export automation.

Production overlays can add the optional [deploy/k8s/components/kvrocks-backup](../../components/kvrocks-backup) component instead of patching backup logic into the base.

That component uses Kvrocks' own backup mechanism:

- `kvrocks-start.sh` appends `backup-dir` and `bgsave-cron` only when backup env vars are enabled by an overlay or component
- each Kvrocks pod mounts the same RWX backup volume at `/var/lib/kvrocks-backups`
- each pod writes to its own subdirectory, for example `/var/lib/kvrocks-backups/kvrocks-1`
- a separate uploader `CronJob` mounts the same RWX volume read-only and uploads per-pod archives to S3-compatible storage
- the uploader does not use `kubectl`, `pods/exec`, or backup-specific RBAC

The backup component assumes `bgsave-cron` may run in every Kvrocks server process, including replicas. It therefore treats every pod as a possible backup writer and isolates backup output by pod name on the shared RWX volume.

Restore validation is mandatory before treating any backup design as production-ready. The component README includes the expected object layout, restore outline, and validation checklist.

## Readiness Semantics

The Kvrocks readiness probe is intentionally conservative:

- a replica is ready only when it reports `master_link_status=connected`
- a master is ready only when it has at least one connected replica

This means a lone surviving master in degraded mode will keep serving internally but will not be marked Ready through Kubernetes Endpoints. That is intentional for this base.

## Overlay Recommendations

- Adjust `spec.replicas` if you need a different number of pods.
- Adjust the PVC size and `storageClassName` in `volumeClaimTemplates`.
- Narrow the base `NetworkPolicy` from same-namespace access to only the exact workloads that should reach ports `6666` and `26379`.
- Add [deploy/k8s/components/kvrocks-backup](../../components/kvrocks-backup) only in overlays that also provide a suitable RWX storage class and S3 configuration.
- Replace the base placement policy with required anti-affinity and stricter topology spread constraints in production overlays.
- Add TLS through overlay patches if required.
- Pin container images to the versions approved for your environment.
- Tune probe thresholds if your cluster starts slowly.
- Exercise full restart and post-failover recovery tests before promoting an overlay to production.

## Scope And Limitations

- This base is for replicated Kvrocks with Sentinel, not cluster mode.
- The cluster-init Helm Job is intentionally not included because it is specific to Redis cluster bootstrap semantics.
- The base assumes no TLS.
- Sentinel quorum is hard-coded to `2`, which matches the default 3-pod topology.
- The base does not include backup export automation. Use an overlay or [deploy/k8s/components/kvrocks-backup](../../components/kvrocks-backup) if you need scheduled backup export.
- The base does not include a metrics exporter or dashboards.

## Validation Checklist

Before calling an overlay production-ready, validate at least these cases:

1. Deploy the base without the backup component and verify the Sentinel-managed StatefulSet still works.
2. Empty PVC bootstrap from scratch.
3. Simultaneous pod deletion and recreation.
4. Simultaneous restart after failover to a non-zero ordinal master.
5. Full cluster/node outage followed by restart, where all pods and Sentinels are unavailable at the same time.
6. Recovery using `KVROCKS_RECOVERY_MASTER`.
7. If using the backup component, verify each pod mounts the shared RWX backup volume and renders `backup-dir /var/lib/kvrocks-backups/<pod-name>` plus `bgsave-cron ...` into the generated Kvrocks config.
8. If using the backup component, verify per-pod backup directories do not collide and that `BGSAVE` updates `CURRENT` as expected on the target Kvrocks image.
9. If using the backup component, run the uploader job manually once and verify objects land in the expected bucket prefix.
10. If using the backup component, restore at least one uploaded backup into an isolated namespace and repeat after Sentinel failover.