# Veidemann log service

The same binary supports three storage roles while keeping one gRPC API:

- `writer` appends logs to the permanent Parquet archive and optionally forwards
  a best-effort copy to a separate recent service.
- `recent` writes logs to a bounded SQLite operational window and serves every
  list request. It does not initialize Parquet or S3.
- `combined` preserves the original Parquet-first, synchronous SQLite
  best-effort behavior for compatibility.

Production writers send to `log-service-writer`; readers send to `log-service`.
The writer acknowledges a request as soon as its Parquet append succeeds and the
recent copy has either been queued or dropped. Existing Parquet files are never
read or backfilled into SQLite.

## Kubernetes requirements

### Use persistent, single-writer storage

Each writer pod owns one Parquet volume containing archive files and
per-collection `.index.json` files. Writers may scale horizontally only when
every ordinal has a separate claim. The recent Deployment remains at one replica
and exclusively owns the SQLite main database, write-ahead log, and shared-memory
sidecar. `ReadWriteOnce` alone does not guarantee single-process access when pods
can be scheduled on the same node.

Do not modify a mounted Parquet or SQLite volume while its owning process is
running or terminating. The supplied recent base requests a dedicated 50 GiB
claim so archive backlog cannot consume the read-store capacity.

### Allow graceful termination

The container entrypoint runs log-service directly, so Kubernetes `SIGTERM`
reaches the service. On termination, the service performs these operations in
order:

1. Gracefully stop the gRPC server and allow active RPCs to finish.
2. Stop the metrics/readiness HTTP server, with a five-second HTTP shutdown
   timeout.
3. In writer mode, drain queued recent forwards for at most
   `RECENT_FORWARD_SHUTDOWN_TIMEOUT`; cancel and count the remainder as dropped.
4. In recent or combined mode, checkpoint and close SQLite.
5. In writer or combined mode, close Parquet storage, then finish S3 handoff.

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

### Deployment shapes

- `deploy/k8s/base/log-service/statefulset` supplies the scalable writer and one
  Parquet claim per ordinal.
- `deploy/k8s/base/log-service/recent` supplies the singleton recent Deployment,
  Service, and SQLite claim.
- `deploy/k8s/overlays/prod/log-service` composes those bases with three writer
  replicas and one recent replica. It expects a separately provisioned
  `log-service-minio` Secret containing the keys referenced by its writer patch.
- `deploy/k8s/base/log-service/deployment` retains combined mode and exposes both
  Service names to the same pod.
- The development overlay changes that Deployment to recent-only mode. Its
  `log-service-writer` Service is only an alias, so development uses SQLite and
  no Parquet volume.

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
| `--mode` | `MODE` | `combined` | Runtime role: `combined`, `writer`, or `recent`. |
| `--metrics-address` | `METRICS_ADDRESS` | `:9153` | Address for `/metrics` and `/readyz`. |
| `--parquet-dir` | `PARQUET_DIR` | `./data/parquet` | Local Parquet and index directory used by writer and combined modes. |
| `--max-lines-per-file` | `MAX_LINES_PER_FILE` | `1000000` | Finalizes/rotates a file after this many rows. Smaller values create more upload jobs and objects. |
| `--recent-log-db-path` | `RECENT_LOG_DB_PATH` | `./data/recent-logs.db` | SQLite database used by recent and combined modes. |
| `--recent-crawl-log-max-entries` | `RECENT_CRAWL_LOG_MAX_ENTRIES` | `1000000` | Independently retained crawl-log row limit. Must be at least one. |
| `--recent-page-log-max-entries` | `RECENT_PAGE_LOG_MAX_ENTRIES` | `250000` | Independently retained page-log row limit. Must be at least one. |
| `--recent-log-service-address` | `RECENT_LOG_SERVICE_ADDRESS` | empty | Optional `host:port` forwarded to asynchronously by writer mode. Empty means archive-only. |
| `--recent-forward-queue-size` | `RECENT_FORWARD_QUEUE_SIZE` | `1024` | Bounded in-memory forwarding queue. A full queue drops the recent copy. |
| `--recent-forward-workers` | `RECENT_FORWARD_WORKERS` | `2` | Concurrent recent-forward workers. |
| `--recent-forward-timeout` | `RECENT_FORWARD_TIMEOUT` | `5s` | Timeout for one forwarding attempt. |
| `--recent-forward-shutdown-timeout` | `RECENT_FORWARD_SHUTDOWN_TIMEOUT` | `30s` | Maximum graceful queue-drain time. |
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

Parquet and S3 settings are ignored in recent mode. SQLite settings are ignored
in writer mode. If `S3_ENDPOINT` is empty, finalized files remain on the local
volume; otherwise bucket and credentials are mandatory.

## Recent read store

All gRPC reads query SQLite only. The database starts empty on the first rollout;
logs that exist only in local Parquet or S3 remain archival-only and are
intentionally invisible to the read API. Refreshing or repeating a query is
enough to see a newly accepted log; Parquet rotation is unrelated to read
visibility.

Writer mode is deliberately Parquet-first. A Parquet append failure fails the RPC
and skips forwarding. After a successful append, enqueueing is non-blocking and
the client is acknowledged without waiting for SQLite. Each queued log receives
one timed forwarding attempt. A full queue, forward failure, process crash, or
shutdown deadline can therefore leave a permanent gap in the recent window.
There is no retry or Parquet backfill. Direct writes to recent mode are
synchronous and return SQLite errors to the caller. Combined mode retains the
old synchronous SQLite best-effort behavior.

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

Writer mode also exports:

- `veidemann_recent_forward_queue_depth`
- `veidemann_recent_forward_total{type,outcome}`

The bounded outcomes are `success`, `failure`, `queue_full_drop`, and
`shutdown_drop`. Writer readiness intentionally does not depend on the recent
service because the secondary copy is best-effort.

Each uploaded Parquet object has a single user-metadata field, `md5`, containing
the lowercase hexadecimal MD5 checksum of the uploaded file.

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
- Every writer ordinal has its own retained Parquet claim, and only that ordinal
  mounts it at `PARQUET_DIR`.
- Exactly one recent pod mounts the SQLite claim at the parent directory of
  `RECENT_LOG_DB_PATH`.
- Network policy permits S3 traffic throughout termination.
- Pod eviction, node shutdown, and rolling replacement have been tested while
  an upload is active.
- Alerts detect S3 errors, repeated restart recovery, PVC capacity pressure, and
  pods stuck in `Terminating`.

After an ungraceful termination, do not remove any PVC. The recent replacement
recovers SQLite through WAL, while each writer scans its own indexed finalized
Parquet files to retry eligible uploads. Files that were successfully uploaded
are deleted locally only after the upload call returns.

### Migrating a combined StatefulSet

Use a controlled write outage for the initial split:

1. Stop producers and gracefully terminate the combined pod so open Parquet
   files are finalized and SQLite is checkpointed.
2. Confirm the old Parquet claim has been uploaded or copy its finalized files
   before detaching it.
3. Retain the old SQLite claim and configure the recent Deployment to mount that
   claim, or migrate it offline to `log-service-recent-logs`.
4. Deploy the singleton recent service and the new writer StatefulSet, whose
   ordinals receive new Parquet claims.
5. Verify both Services, then restart producers against `log-service-writer`.

Do not automatically delete the detached legacy Parquet claim. Deploying the new
writer against the old combined service is unsafe because the forwarded write
would be archived a second time.

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
