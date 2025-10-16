# Veidemannctl

[![License Apache](https://img.shields.io/github/license/nlnwa/veidemannctl.svg)](https://github.com/NationalLibraryOfNorway/veidemann/ctl/blob/main/LICENSE)
[![GitHub release](https://img.shields.io/github/release/nlnwa/veidemannctl.svg)](https://github.com/NationalLibraryOfNorway/veidemann/ctl/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/nlnwa/veidemannctl?style=flat-square)](https://goreportcard.com/report/github.com/nlnwa/veidemannctl)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/nlnwa/veidemannctl)](https://pkg.go.dev/github.com/nlnwa/veidemannctl)

## Install

Install the latest release version

```console
curl -sL https://raw.githubusercontent.com/NationalLibraryOfNorway/veidemann/ctl/main/install.sh | bash
```

## Usage

To get a list of available commands and configuration flags:

```console
veidemanctl -h
```

## Documentation

Usage documentation: <https://nlnwa.github.io/veidemannctl>

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
