# veidemann-dns-resolver

>The ordering of the plugins in the Corefile does not determine the order of the >plugin chain. The order in which the plugins are executed is determined by the >ordering in `main.go`.

The Corefile only enables and configures plugins in the plugin chain.

## Archiving Cache

The `archivingcache` plugin now uses Olric as its cache backend instead of an embedded in-process cache. The plugin preserves the existing DNS response and archival behavior, but startup now depends on an Olric endpoint being reachable.

Relevant `archivingcache` settings:

    archivingcache {
        eviction 5m
        olricAddress localhost:3320
        olricDmap dns-resolver-archivingcache
        contentWriterHost localhost
        contentWriterPort 5010
        logHost localhost
        logPort 5011
    }

- `eviction` is the TTL applied to cached DNS entries in Olric.
- `olricAddress` accepts either a comma-separated list or repeated directives for multiple peers.
- `olricDmap` selects the distributed map used for cached entries.

With the shipped Docker and Kubernetes configuration, these values are wired through `OLRIC_ADDRESS` and `OLRIC_DMAP` environment variables.

## Example

Run server:

    go run .

If `archivingcache` is enabled in the Corefile, make sure Olric is available at the configured address before starting the resolver.

Query server:

    $ go run ./cmd/resolve vg.no
    time: 143.04964ms
    host:"vg.no" port:80 textual_ip:"195.88.55.16" raw_ip:"\xc3X7\x10"
