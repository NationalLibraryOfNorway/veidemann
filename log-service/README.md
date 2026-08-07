# Veidemann log service

The log service writes every accepted crawl and page log to two destinations:

- Parquet files provide the permanent archive and may be handed to an
  S3-compatible object store after finalization.
- A bounded SQLite database provides the operational read window used by every
  gRPC list request.

SQLite is updated synchronously after the Parquet append, so operators can query
new logs immediately without waiting for a Parquet file to rotate. Existing
Parquet files are never read or backfilled into SQLite.

## Kubernetes requirements

### Use persistent, single-writer storage

Mount separate persistent volumes at the directory configured by `PARQUET_DIR`
and the parent directory of `RECENT_LOG_DB_PATH`. The Parquet directory contains
archive files and per-collection `.index.json` files. The recent-log volume
contains the SQLite main database, write-ahead log, and shared-memory sidecar.
The supplied manifests request a dedicated 50 GiB recent-log volume so archive
backlog cannot consume the read-store capacity.

Use one log-service replica per pair of volumes. Concurrent pods must not write
to or scan the same Parquet directory or open the same SQLite database. The
Kubernetes manifests provide a `base/log-service/statefulset` alternative that
gives each replica its own claims and stable ordinal. `ReadWriteOnce` alone does
not guarantee single-process access because multiple pods on the same node may
still be able to mount the volume.

Do not delete, replace, unmount, or manually modify either volume while the
log-service pod is running or terminating. A utility pod may mount it for
inspection, but must not modify Parquet files, indexes, or the SQLite database
concurrently.

### Allow graceful termination

The container entrypoint runs log-service directly, so Kubernetes `SIGTERM`
reaches the service. On termination, the service performs these operations in
order:

1. Gracefully stop the gRPC server and allow active RPCs to finish.
2. Stop the metrics/readiness HTTP server, with a five-second HTTP shutdown
   timeout.
3. Checkpoint and close the SQLite recent-log database.
4. Close Parquet storage, which finalizes open files and submits them for S3
   handoff when S3 is enabled.
5. Stop the S3 scanner, finish all queued and active uploads, update indexes,
   and only then exit.

Set `terminationGracePeriodSeconds` long enough for all five steps. The
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
    - metadata:
        name: recent-logs
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 50Gi
  template:
    spec:
      terminationGracePeriodSeconds: 300
      containers:
        - name: log-service
          image: ghcr.io/nationallibraryofnorway/veidemann/log-service:<version>
          env:
            - name: PARQUET_DIR
              value: /parquet
            - name: RECENT_LOG_DB_PATH
              value: /recent-logs/logs.db
            - name: RECENT_CRAWL_LOG_MAX_ENTRIES
              value: "1000000"
            - name: RECENT_PAGE_LOG_MAX_ENTRIES
              value: "250000"
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
            - name: recent-logs
              mountPath: /recent-logs
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
| `--recent-log-db-path` | `RECENT_LOG_DB_PATH` | `./data/recent-logs.db` | SQLite database used exclusively for gRPC reads. Mount the dedicated recent-log PVC at its parent directory. |
| `--recent-crawl-log-max-entries` | `RECENT_CRAWL_LOG_MAX_ENTRIES` | `1000000` | Independently retained crawl-log row limit. Must be at least one. |
| `--recent-page-log-max-entries` | `RECENT_PAGE_LOG_MAX_ENTRIES` | `250000` | Independently retained page-log row limit. Must be at least one. |
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

## Recent read store

All gRPC reads query SQLite only. The database starts empty on the first rollout;
logs that exist only in local Parquet or S3 remain archival-only and are
intentionally invisible to the read API. Refreshing or repeating a query is
enough to see a newly accepted log; Parquet rotation is unrelated to read
visibility.

The write policy is deliberately Parquet-first. A Parquet append failure fails
the RPC and skips SQLite. After a successful archive append, the SQLite write is
attempted synchronously. A SQLite failure is logged and counted but does not fail
the RPC, retry from Parquet, or enable a Parquet fallback, so it can leave a
permanent gap in the recent window.

Crawl and page retention are independent. Each non-empty WARC ID has at most one
row per log type; writing it again atomically replaces the prior payload and
makes it the newest row. Execution-ID queries return newest-first and apply the
existing offset and page-size fields. Page resources and outlinks are embedded
in the page-log protobuf and count as one retained page row. Lowering either
limit prunes that table before the service becomes ready.

Requests without WARC-ID or execution-ID filters return logs newest-first across
the selected log type. An empty request returns only the latest inserted row;
setting a positive page size returns that many rows, and offset skips newer rows.
This ordering uses SQLite ingestion sequence rather than embedded log timestamps.

The database stores protobuf payloads uncompressed. Core SQLite does not provide
native transparent compression, and the proprietary ZIPVFS extension is not
compatible with the pure-Go `modernc.org/sqlite` driver used to keep
`CGO_ENABLED=0` builds working. Memory use is bounded by four pooled connections
with approximately 2 MiB of SQLite page cache each, active query/write payloads,
and driver overhead; it does not grow with the entire retained row set.

Observed planning ranges are approximately 0.75–1.37 GiB for one million crawl
payloads averaging 512 B–1 KiB. At 250,000 rows, page payloads averaging 4–16 KiB
require roughly 1.1–4 GiB and can be larger for resource-heavy pages. Retention
is a row-count bound rather than a byte bound, so monitor the dedicated 50 GiB
PVC and size each window with:

```text
retained entries = peak logs/second × desired QA seconds × headroom
```

The service exports these recent-store metrics:

- `veidemann_recent_logs_entries{type}`
- `veidemann_recent_logs_evicted_total{type}`
- `veidemann_recent_logs_write_failures_total{type}`
- `veidemann_recent_logs_payload_bytes{type}`
- `veidemann_recent_logs_database_file_bytes{file="main|wal|shm"}`

If an upload fails, the finalized Parquet file and its index entry remain in
`PARQUET_DIR`. The error is logged as `Parquet S3 handoff failed`, the service
continues running, and the scanner retries the file after `S3_SCAN_INTERVAL`.
The persistent Parquet directory therefore serves as failed-upload storage; a
separate fallback directory is not required.

`S3_UPLOAD_DELAY` has direct storage implications. During the delay, finalized
files remain on the Parquet PVC. Capacity must cover the write rate multiplied
by the delay, plus open files and operational headroom. After a successful
upload, the local file and its index entry are removed. Neither local Parquet nor
S3 is queried by the gRPC read API.

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
- Both PVCs survive pod deletion and are mounted at exactly `PARQUET_DIR` and
  the parent directory of `RECENT_LOG_DB_PATH` by the replacement pod.
- Only one pod can own and write the volume pair at a time.
- Network policy permits S3 traffic throughout termination.
- Pod eviction, node shutdown, and rolling replacement have been tested while
  an upload is active.
- Alerts detect S3 errors, repeated restart recovery, PVC capacity pressure, and
  pods stuck in `Terminating`.

After an ungraceful termination, do not remove either PVC. The replacement
service recovers SQLite through WAL and scans indexed finalized Parquet files to
retry eligible uploads. Files that were successfully uploaded are deleted
locally only after the upload call returns.

## Local tests

Run unit tests from this module:

```shell
go test ./...
CGO_ENABLED=0 go build ./...
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
