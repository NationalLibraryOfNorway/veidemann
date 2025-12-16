# First Article Inspection (FAI)

FAI loops trough files in a given directory that match a given pattern
and for every matching file:

- calculates md5 checksum
- uploads file to S3
- updates a file size histogram metric and a upload duration metric
- logs the name, size and etag
- removes the file

```text
Usage of fai:
      --dir string                    path to source directory
      --metrics-address string        address to expose metrics on (default ":8081")
      --metrics-path string           path to expose metrics on (default "/metrics")
      --pattern string                glob pattern used to match filenames in source directory (default "*.warc.gz")
      --s3-access-key-id string       access key ID
      --s3-address string             s3 endpoint (address:port) (default "localhost:9000")
      --s3-bucket-name string         name of bucket to upload files to
      --s3-secret-access-key string   secret access key
      --s3-token string               token to use for s3 authentication (optional)
      --sleep duration                sleep duration between directory listings, set to 0 to only do a single pass (default 5s)
```
