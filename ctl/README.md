# Veidemannctl

## Usage

To get a list of available commands and configuration flags:

```console
veidemanctl -h
```

## Build

```console
go build
```

## Test

```console
go test ./...
```

## Generate documentation

```console
go generate
```

## Known limitations

### Default server error message

When no `--server <address>` is provided or previously set using `veidemannctl
config set-address <address>` you might experience the following error message:

```
$ veidemannctl get seed
Error: failed to build resolver: passthrough: received empty target in Build()
```

Setting `--server` or `veidemannctl config set-address <address>` to something
other than an empty string will resolve this specific error.
