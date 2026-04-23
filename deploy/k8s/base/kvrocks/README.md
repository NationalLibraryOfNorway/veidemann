# Kvrocks Kustomize Base

This directory contains a plain Kubernetes and Kustomize base for Apache Kvrocks running as a replicated StatefulSet with Redis Sentinel sidecars.

The base is intended for a simple master/replica deployment with automatic failover. It is not a Kvrocks cluster-mode deployment.

## Topology

- `StatefulSet/kvrocks` with 3 replicas by default
- One `apache/kvrocks` container per pod
- One `redis` Sentinel sidecar per pod
- Headless service for stable pod DNS and peer discovery
- Sentinel service for master discovery
- General Kvrocks service for broad connectivity

## Included Resources

- `serviceaccount.yaml`
- `configmap.yaml`
- `scripts-configmap.yaml`
- `service.yaml`
- `statefulset.yaml`

## Bootstrap And Failover

On a cold start, `kvrocks-0` becomes the initial master.

When a pod starts later, it first asks existing Sentinels for the current master. If a master has already been elected, the pod joins as a replica of that node instead of falling back to `kvrocks-0`.

During termination of the active master, the `preStop` hook asks the local Sentinel to trigger failover and waits briefly for a new master to be elected.

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

### `kvrocks`

Exposes Kvrocks on port `6666` for all pods selected by the StatefulSet labels.

This is useful for diagnostics and some read-oriented traffic, but it is not a stable write endpoint in a Sentinel-managed topology. Kubernetes may route a connection to a replica.

If a stable write endpoint is required for non-Sentinel-aware clients, add a separate master-discovery component that updates a dedicated service after failover.

## Authentication

Authentication is optional.

If a Secret named `kvrocks-auth` exists in the same namespace and contains a `redis-password` key, the base enables password auth automatically.

When the secret is present, the startup scripts:

- add `requirepass` and `masterauth` to the generated Kvrocks configuration
- add `sentinel auth-pass kvrocks ...` to the generated Sentinel configuration
- use authenticated `redis-cli` calls for Kvrocks health checks and Kvrocks-side helper commands

Sentinel itself is not password protected in this base, so Sentinel control commands remain unauthenticated.

### Example Auth Overlay

An example overlay is provided in [deploy/k8s/overlays/auth/kvrocks](../../overlays/auth/kvrocks).

Render it with:

```sh
kustomize build deploy/k8s/overlays/auth/kvrocks
```

Before applying it, replace the example password in [deploy/k8s/overlays/auth/kvrocks/kustomization.yaml](../../overlays/auth/kvrocks/kustomization.yaml) with a real value or switch the secret generator to your normal secret source.

## Usage

1. Reference this base from an overlay.
2. Set the namespace, replica count, storage, images, and resources in that overlay.
3. Use the auth overlay pattern if password protection is required.
4. Use Sentinel-aware clients against `kvrocks-sentinel:26379` when writing to the current master.

## Overlay Recommendations

- Adjust `spec.replicas` if you need a different number of pods.
- Adjust the PVC size in `volumeClaimTemplates`.
- Add TLS through overlay patches if required.
- Pin container images to the versions approved for your environment.
- Tune probe thresholds if your cluster starts slowly.

## Scope And Limitations

- This base is for replicated Kvrocks with Sentinel, not cluster mode.
- The cluster-init Helm Job is intentionally not included because it is specific to Redis cluster bootstrap semantics.
- The base assumes no TLS.
- Sentinel quorum is hard-coded to `2`, which matches the default 3-pod topology.