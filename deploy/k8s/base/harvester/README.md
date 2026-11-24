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

Browserless v2.64.3 attempts to delete automatically generated profiles and
session scratch directories after the browser closes. It retries failed deletions
and queues remaining directories for a background retry every five minutes, but
does not sweep all of `/tmp` or guarantee recovery from a full filesystem. Its
retry queue is lost if the Browserless process is killed. See the pinned
[cleanup implementation](https://github.com/browserless/browserless/blob/v2.64.3/src/browsers/index.ts).

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

## Recorder-owned attempts

Deploy browser-controller 0.11.0 and recorderproxy 0.10.0 together. Both the
recorder sidecar and the certificate generator use the recorderproxy image.
Include robots-evaluator 0.13.0 and Frontier 0.21.5 in the cutover. The robots
recording budget is 30 seconds per refresh and Frontier allows 60 seconds per
evaluator call.

The Go services use published modules. Browser-controller consumes
`recorderproxy/v0.10.0` and the recorder contract is in `api/v1.6.0`.
Build individual images from the repository root with service-local contexts:

```sh
docker build -t veidemann-robots-evaluator:local ./robots-evaluator
docker build -t veidemann-browser-controller:local ./browser-controller
docker build -t veidemann-recorderproxy:local ./recorderproxy
```

For local Minikube testing, run `skaffold run -m app` from the repository root.
The app module uses ko for browser-controller and robots-evaluator, and Docker
for recorderproxy. Skaffold substitutes the recorder image in both init-container
references. The development overlay requires cluster infrastructure and secrets.

Recorder control listens on pod-local port 7780. Robots evaluator configuration
belongs to recorderproxy (`robots:7053`). Browser-controller has no callback
listener or log-service connection. Recorderproxy owns crawl/page-log writes to
`log-service-writer:8080`. FinishHarvest streams outlinks and confirms persistence
in one reply.
Robots-evaluator uses DNS, ContentWriter, log-service and Squid directly.
The harvester Service exposes no public proxy port. `PROXY_COUNT=1` in both
components preserves the previous effective single harvest concurrency; slot 1
now uses port 9900. No downstream service contract changes.

This base uses Kubernetes native sidecars (`initContainers.restartPolicy: Always`)
for browser and recorder. The cluster must support native sidecars.
Keep the rendered init-container order as `init-browser-storage`,
`generate-recorder-mitm-cert`, `browser`, then `recorderproxy`. The certificate
generator creates the pod's MITM identity automatically; no manually supplied
certificate is required. Strategic merge patches that modify an init container
must include the preceding container names in order, or Kustomize can move the
patched container ahead of its prerequisites. Starting recorderproxy before the
generator causes a restart loop with a missing `/var/run/recorderproxy/tls.crt`.
Browser starts
before recorder; both remain running until browser-controller has stopped
scheduling, finished active attempts, persisted logs and reported to Frontier.
Recorder then drains before browser exits. The 600-second termination grace period
covers the default 300-second outer fetch limit, 30-second capture, 45-second
recorder finalization including log persistence, 30-second screenshot persistence, 5-second browser
cleanup, 60-second report and recorder/telemetry shutdown margin. Increase grace
when increasing any of those budgets. Do not use a short independent recorder
preStop delay as a substitute for this ordering.

Application-container restarts leave the native sidecars alive. Browser-controller
therefore calls recorder `RecoverSlot` on startup, before readiness and before
fetching Frontier work. Recovery closes the previous attempt and is bounded by
the recorder's `FINALIZATION_TIMEOUT` and the controller's
`COMPLETION_DRAIN_TIMEOUT`. It refuses genuinely undrained work; a subsequent
restart can recover after late terminal work completes. Requests carry an attempt
token, so old browser contexts cannot send requests into the new harvest.
Keep one controller per recorder slot set; do not point multiple controllers at
the same recorder. A permanently wedged handler still requires replacing the pod.

In staging, kill only browser-controller during a harvest, then verify that its
restart recovers with the same recorder container ID and completes a new harvest.
Repeat after injecting a recorder finalization timeout and releasing the blocked
terminal operation. While that operation remains blocked, verify readiness stays
false and the restarting controller does not take another Frontier item. These
container checks complement the gRPC reconnect and HTTP/HTTPS regression tests;
they require a live cluster.

Rollback is coordinated too: drain the new pods, restore the previous deployment
(including its container layout/configuration) and the previous browser-controller, recorderproxy, robots-evaluator and Frontier images. Restore the robots environment proxy settings
and the harvester public proxy Service port.
Never connect an old browser-controller to a new recorder slot. No journal or
cross-version protocol bridge is required; Frontier recovers interrupted attempts.

Before cutover, pause scheduling and drain active harvests; replace the robots-evaluator, Frontier,
recorderproxy and browser-controller images plus configuration together. Resume
after checking robots disallow decisions, request/response WARC pairs and crawl
logs, including a robots evaluation with browser-controller stopped. Validate
that an idle harvest listener closes incoming connections, only one harvest can
run, and recording summary failures do not alter robots decisions. Robots refreshes
wait for recording before replying; cache hits use the cached rules immediately.
Render both bases and the deployment overlay before applying.

## Harvest logging

Browser-controller and recorderproxy use INFO in this base; recorderproxy's
`LOG_FORMATTER` is explicitly `json`. The development overlay sets both services
to DEBUG while retaining JSON. INFO provides the harvest phase timeline and
final summaries; DEBUG adds individual browser requests and recorder terminal
outcomes. Routine admission, cache-hit, connection, and body-handling messages are TRACE.
DEBUG contains resource results; INFO contains no routine per-resource messages.

Follow `browser-controller` and `recorderproxy` separately with `kubectl logs`,
and correlate their JSON records using `harvestId` and `slot`. See
[reading harvest logs](../../../../browser-controller/README.md#reading-harvest-logs)
for filtering commands and the distinction between browser events, recorded
resources, and final harvest success. Validate the rendered base and development
overlay when changing log settings.
