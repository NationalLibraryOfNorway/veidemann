# Log-Service Agent Notes

## Scope

These notes apply to work inside `log-service/`.

## Current Storage Behavior

- Every accepted log is appended to Parquet first and then synchronously written
  to a bounded SQLite recent-log store.
- All production gRPC reads query SQLite only. Parquet and S3 are archival-only;
  they are never queried or backfilled into the recent store.
- Crawl and page retention limits are independent, and page resources/outlinks
  remain embedded in the retained page-log protobuf.
- The recent SQLite database and Parquet archive require separate persistent,
  single-writer volumes.

## Parquet Archival Behavior

- Parquet files are written locally under the configured `parquet-dir` and indexed with per-collection `.index.json` files.
- If S3 is not configured, finalized parquet files remain on local disk.
- If S3 is configured and `s3-upload-delay` is `0`, finalized parquet files are uploaded after close.
- If S3 is configured and `s3-upload-delay` is greater than `0`, finalized parquet files stay local until the retention threshold is reached. A background scan re-checks indexed finalized files so eligible uploads survive process restarts.
- Successful S3 upload deletes the local parquet file and removes its index entry.
- Current S3 object keys are built as `<keyPrefix>/<table>/<collection>/<file>.parquet`.
- Current S3 object metadata includes `veidemann-table`, `veidemann-collection`, and `veidemann-row-count`.

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
