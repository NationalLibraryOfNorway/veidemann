# Veidemann log service

The log service stores crawl and page logs as Parquet files. It writes files to
local storage first and can optionally hand finalized files to an S3-compatible
object store.

## Kubernetes requirements

### Use persistent, single-writer storage

Mount a persistent volume at the directory configured by `PARQUET_DIR`. The
directory contains both Parquet files and per-collection `.index.json` files.
The files and their indexes must be kept together.

Use one log-service replica per Parquet directory. Concurrent pods must not
write to or scan the same directory. The Kubernetes manifests provide a
`base/log-service-statefulset` alternative that gives each replica its own
claim and stable ordinal. `ReadWriteOnce` alone does not guarantee
single-process access because multiple pods on the same node may still be able
to mount the volume.

Do not delete, replace, unmount, or manually modify the volume while the
log-service pod is running or terminating. A utility pod may mount it for
inspection, but must not modify Parquet files or indexes concurrently.

### Allow graceful termination

The container entrypoint runs log-service directly, so Kubernetes `SIGTERM`
reaches the service. On termination, the service performs these operations in
order:

1. Gracefully stop the gRPC server and allow active RPCs to finish.
2. Stop the metrics/readiness HTTP server, with a five-second HTTP shutdown
   timeout.
3. Close Parquet storage, which finalizes open files and submits them for S3
   handoff when S3 is enabled.
4. Stop the S3 scanner, finish all queued and active uploads, update indexes,
   and only then exit.

Set `terminationGracePeriodSeconds` long enough for all four steps. The
Kubernetes default of 30 seconds is often too short when S3 upload is enabled.
A reasonable initial value is 300 seconds, but it must be sized from observed
active-RPC duration, maximum upload backlog, file size, network throughput, and
S3 failure behavior.

There is currently no command-line option that limits an individual S3 upload.
If S3 stalls indefinitely, shutdown can also wait indefinitely. When the grace
period expires, Kubernetes sends `SIGKILL`. Finalized files that were not
successfully uploaded remain recoverable only if `PARQUET_DIR` is persistent
and is mounted by the replacement pod.

A `preStop` sleep is not required for internal file cleanup and does not replace
an adequate termination grace period. If ingress propagation requires a
`preStop` delay, include that delay in the grace-period calculation.

### Recommended deployment shape

This excerpt shows the settings that matter to storage ownership and shutdown.
Adapt names, resources, probes, and the grace period to the cluster:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: log-service
spec:
  replicas: 1
  serviceName: log-service
  persistentVolumeClaimRetentionPolicy:
    whenDeleted: Retain
    whenScaled: Retain
  volumeClaimTemplates:
    - metadata:
        name: parquet
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
  template:
    spec:
      terminationGracePeriodSeconds: 300
      containers:
        - name: log-service
          image: ghcr.io/nationallibraryofnorway/veidemann/log-service:<version>
          env:
            - name: PARQUET_DIR
              value: /parquet
            - name: MAX_LINES_PER_FILE
              value: "100000"
            - name: S3_ENDPOINT
              value: s3.example.org
            - name: S3_BUCKET
              value: veidemann-parquet
            - name: S3_KEY_PREFIX
              value: logs
            - name: S3_UPLOAD_DELAY
              value: "0s"
            - name: S3_SCAN_INTERVAL
              value: "1m"
            - name: S3_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: log-service-s3
                  key: access-key
            - name: S3_SECRET_KEY
              valueFrom:
                secretKeyRef:
                  name: log-service-s3
                  key: secret-key
          volumeMounts:
            - name: parquet
              mountPath: /parquet
```

Never put real S3 credentials directly in a manifest. Supply them through a
Kubernetes `Secret` or the deployment system's secret integration.

## Configuration

Every command-line flag can also be set as an environment variable by replacing
hyphens with underscores. For example, `--parquet-dir=/parquet` is equivalent
to `PARQUET_DIR=/parquet`.

| Flag | Environment variable | Default | Operational meaning |
| --- | --- | --- | --- |
| `--host` | `HOST` | all interfaces | Interface for the gRPC API. |
| `--port` | `PORT` | `8090` | gRPC API port. Must match the Service and container port. |
| `--metrics-address` | `METRICS_ADDRESS` | `:9153` | Address for `/metrics` and `/readyz`. |
| `--parquet-dir` | `PARQUET_DIR` | `./data/parquet` | Local Parquet and index directory. Mount the PVC here. |
| `--max-lines-per-file` | `MAX_LINES_PER_FILE` | `100000` | Finalizes/rotates a file after this many rows. Smaller values create more upload jobs and objects. |
| `--s3-endpoint` | `S3_ENDPOINT` | empty | Enables S3 handoff when non-empty. May be `host:port`, `https://host`, or `http://host`; URL paths are not supported. |
| `--s3-bucket` | `S3_BUCKET` | empty | Required when `S3_ENDPOINT` is set. |
| `--s3-access-key` | `S3_ACCESS_KEY` | empty | Required when `S3_ENDPOINT` is set; provide from a Secret. |
| `--s3-secret-key` | `S3_SECRET_KEY` | empty | Required when `S3_ENDPOINT` is set; provide from a Secret. |
| `--s3-key-prefix` | `S3_KEY_PREFIX` | empty | Optional prefix. Objects are stored as `<prefix>/<table>/<collection>/<file>.parquet`. |
| `--s3-insecure` | `S3_INSECURE` | `false` | Uses HTTP only when the endpoint has no scheme. An explicit `http://` or `https://` scheme takes precedence. |
| `--s3-upload-delay` | `S3_UPLOAD_DELAY` | `0s` | With `0`, upload finalized files immediately. A positive Go duration such as `72h` retains them locally until eligible. |
| `--s3-scan-interval` | `S3_SCAN_INTERVAL` | `1m` | How often indexed files are checked for delayed or restart-recovery upload. |
| `--log-level` | `LOG_LEVEL` | `info` | Log verbosity. Supported values are `panic`, `fatal`, `error`, `warn`, `info`, `debug`, and `trace`. |
| `--log-formatter` | `LOG_FORMATTER` | `logfmt` | `logfmt` or `json`. JSON is usually easier to process in Kubernetes. |
| `--log-method` | `LOG_METHOD` | `false` | Include the source file and line in logs. |

If `S3_ENDPOINT` is empty, all finalized files remain on the local volume. If it
is set, bucket and credentials are mandatory and invalid configuration prevents
the service from starting.

If an upload fails, the finalized Parquet file and its index entry remain in
`PARQUET_DIR`. The error is logged as `Parquet S3 handoff failed`, the service
continues running, and the scanner retries the file after `S3_SCAN_INTERVAL`.
The persistent Parquet directory therefore serves as failed-upload storage; a
separate fallback directory is not required.

`S3_UPLOAD_DELAY` has direct storage implications. During the delay, finalized
files remain on the PVC and continue to be available to local reads. Capacity
must cover the write rate multiplied by the retention period, plus open files
and operational headroom. After a successful upload, the local file and its
index entry are removed. Historical reads directly from S3 are not currently
implemented.

`S3_SCAN_INTERVAL` affects how soon an eligible or previously interrupted file
is retried after startup; it does not bound upload time and does not shorten the
shutdown wait.

Changing `S3_UPLOAD_DELAY` changes eligibility against each file's recorded
finalization time. Reducing the delay and restarting the StatefulSet causes the
startup scan to enqueue files that are eligible under the new value. Setting it
to `0s` makes every indexed finalized file eligible immediately.

There are currently no dedicated S3 upload failure or backlog metrics. Monitor
`Parquet S3 handoff failed` log entries, PVC usage, and storage-pool capacity.

## Shutdown and recovery checks

Before deploying S3 handoff, verify the following in the target overlay:

- The pod receives `SIGTERM` and is not launched through a wrapper that swallows
  signals.
- `terminationGracePeriodSeconds` covers active RPCs and the largest expected S3
  backlog.
- The PVC survives pod deletion and is mounted at exactly `PARQUET_DIR` by the
  replacement pod.
- Only one pod can write or scan that directory at a time.
- Network policy permits S3 traffic throughout termination.
- Pod eviction, node shutdown, and rolling replacement have been tested while
  an upload is active.
- Alerts detect S3 errors, repeated restart recovery, PVC capacity pressure, and
  pods stuck in `Terminating`.

After an ungraceful termination, do not remove the PVC. The replacement service
scans indexed finalized files and retries eligible uploads. Files that were
successfully uploaded are deleted locally only after the upload call returns.

## Local tests

Run unit tests from this module:

```shell
go test ./...
```

The integration tests are tagged `integration` and use test containers:

```shell
go clean -testcache && go test -tags=integration ./...
```

With Podman:

```shell
go clean -testcache && TESTCONTAINERS_RYUK_DISABLED=true \
  DOCKER_HOST=unix:///var/run/user/${UID}/podman/podman.sock \
  go test -tags=integration ./...
```
