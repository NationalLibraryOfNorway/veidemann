# veidemann-dns-resolver

>The ordering of the plugins in the Corefile does not determine the order of the >plugin chain. The order in which the plugins are executed is determined by the >ordering in `main.go`.

The Corefile only enables and configures plugins in the plugin chain.

## Example

Run server:

    go run .

Query server:

    $ go run ./cmd/resolve vg.no
    time: 143.04964ms
    host:"vg.no" port:80 textual_ip:"195.88.55.16" raw_ip:"\xc3X7\x10"
