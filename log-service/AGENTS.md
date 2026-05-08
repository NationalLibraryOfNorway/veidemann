# Log-Service Agent Notes

## Scope

These notes apply to work inside `log-service/`.

## Current Parquet Archival Behavior

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
- Important: deleting only `.index.json` while leaving old parquet files behind makes those old local parquet files invisible to log-service reads, because current reads are driven by the local index.

## Deferred Future Feature

- Historical reads from S3 endpoints are a separate future feature.
- They are not implemented in this iteration.
- Do not treat upload-to-S3 as implying read-from-S3 support.

## Relevant Constraints For Future S3 Reads

- Current read paths only read local parquet files referenced by `.index.json`.
- After successful S3 archival, local parquet files and local index entries are intentionally removed.
- A future S3 read feature will need explicit object discovery, query/filter behavior for `warcId` and `executionId`, pagination semantics, and a rule for merging local and remote history.
- Any future implementation should preserve the current write-side archival behavior unless the task explicitly changes retention semantics.