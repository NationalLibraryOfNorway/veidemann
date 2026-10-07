# Harvester writable storage

The browser uses a generic ephemeral PVC for `/tmp` and
`/home/blessuser`, mounted as separate subdirectories of `browser-storage`.
The base requests 4 GiB and uses the cluster's default StorageClass. Overlays
should select a suitable class and increase the claim size for their workload.

Choose a StorageClass backed by capacity separate from the filesystem used for
container images, writable layers, and logs. A PVC backed by a directory on
that same filesystem does not isolate the browser from node disk pressure.
Thin-provisioned volumes also require free space in their backing pool; the claim
size is logical capacity, not a reservation of physical space in the thin pool.
Allow for one claim per replica plus additional claims during rolling updates.

`init-browser-storage` creates the subdirectories as UID 999, the `blessuser`
UID in the pinned Browserless image. The CSI driver must support the pod's
`fsGroup: 65532` so the initializer can write to the volume. Check the image UID
when upgrading Browserless. The browser's root filesystem is read-only.

The remaining local `ephemeral-storage` request and limit budget container
logs. Browser files on the PVC are accounted separately. `/dev/shm` remains a
1 GiB memory-backed `emptyDir`; its usage counts against memory. The recorder
certificate volume also remains memory-backed.

## Cleanup and recovery

Browserless v2.55.2 attempts to delete automatically generated profiles when
sessions close. It retries failed deletions, but does not sweep all of `/tmp`
or guarantee recovery from a full filesystem. Its retry queue is lost if the
Browserless process is killed. See the pinned
[cleanup implementation](https://github.com/browserless/browserless/blob/v2.55.2/src/browsers/index.ts).

Both browser directories survive a container restart. The PVC belongs to the
pod and is garbage-collected when that pod is deleted; backing-volume disposal
depends on the StorageClass reclaim policy. Failed pods that remain in the API
can retain their claims. Do not reuse this scratch claim for durable crawl data.

Monitor used bytes and inodes and alert before the volume is full. If usage
continues to grow after sessions finish, identify the accumulating directories.
Drain the harvester and replace its pod when a fresh scratch volume is needed.
Do not delete arbitrary files belonging to active browser sessions. Increasing
the claim size postpones exhaustion but does not fix accumulating orphan files.

## Validation

Render the base and the dev overlay before rollout:

```sh
kustomize build deploy/k8s/base/harvester
kustomize build deploy/k8s/overlays/dev/harvester
```

For a staged rollout, verify that:

- Each harvester gets its own bound PVC with the intended class and capacity.
- `init-browser-storage` completes and the browser can write both mounted paths.
- The injected Linkerd proxy has its configured local-storage request/limit and
  a read-only root filesystem. Injection requires a Linkerd version supporting
  the `proxy-ephemeral-storage-request` and `proxy-ephemeral-storage-limit`
  annotations; this cannot be checked by a Kustomize render.
- A representative crawl succeeds, including screenshots and graceful shutdown.
- Scratch usage falls after sessions complete, and alerts cover bytes and inodes.
- Deleting a drained test pod removes its scratch PVC and reclaims its volume
  according to the storage policy.
