# Log-Service Agent Notes

## Scope

These notes apply to work inside `log-service/`.

## Current Storage Behavior

- The binary supports `writer`, `recent`, and backwards-compatible `combined`
  modes behind the same gRPC API.
- Writer mode appends to Parquet first, then non-blockingly queues a best-effort
  copy for a configured recent service. Forward failures never fail an archived
  write and are never replayed from Parquet.
- Recent mode writes synchronously to a bounded SQLite store and returns SQLite
  errors to direct callers. Combined mode keeps the original synchronous
  Parquet-first/SQLite-best-effort behavior.
- All production gRPC reads query SQLite only. Parquet and S3 are archival-only;
  they are never queried or backfilled into the recent store.
- Crawl and page retention limits are independent, and page resources/outlinks
  remain embedded in the retained page-log protobuf.
- Each writer replica requires its own persistent, single-writer Parquet volume.
  The singleton recent Deployment exclusively owns its SQLite volume.

## Split-Service Routing

- Producers use `log-service-writer`; controller/dashboard and other readers use
  `log-service`.
- Writer list RPCs return gRPC `Unimplemented`; do not return an empty success for
  misrouted reads.
- The production overlay runs three writer StatefulSet replicas and one recent
  Deployment. Development aliases `log-service-writer` to the SQLite-only recent
  pod and does not mount Parquet.
- The recent-forward queue is volatile and bounded. Preserve timeout, drop,
  queue-depth, and shutdown-drain observability when changing it.

## Parquet Archival Behavior

- Parquet files are written locally under the configured `parquet-dir` and indexed with per-collection `.index.json` files.
- If S3 is not configured, finalized parquet files remain on local disk.
- If S3 is configured and `s3-upload-delay` is `0`, finalized parquet files are uploaded after close.
- If S3 is configured and `s3-upload-delay` is greater than `0`, finalized parquet files stay local until the retention threshold is reached. A background scan re-checks indexed finalized files so eligible uploads survive process restarts.
- Successful S3 upload deletes the local parquet file and removes its index entry.
- Current S3 object keys are built as `<keyPrefix>/<table>/<collection>/<file>.parquet`.
- S3 object user metadata contains only `md5`, the lowercase hexadecimal MD5 checksum of the uploaded Parquet file.

## Manual Local Cleanup

- With no S3 configured, operators may manually copy finalized parquet files out of the volume and remove the local parquet files.
- If the `.index.json` file is also removed, future closes recreate the index file automatically.
- Parquet cleanup does not change gRPC read results, which are driven only by the
  SQLite retention window. Do not modify open files or indexes while the service
  is running.

## Archival Reads

- Historical Parquet/S3 access is deliberately separate from the log-service
  gRPC API.
- Do not add a Parquet fallback, merge local and remote history, or backfill
  SQLite unless a task explicitly changes the architecture.
- Keep Parquet indexes for finalized-file discovery and S3 retry/handoff only.
- Parquet decoding belongs in archival tests and offline tooling, not production
  read paths.
