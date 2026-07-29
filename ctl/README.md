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

## OIDC scopes

If an OIDC context does not configure scopes, `veidemannctl` uses the legacy
Dex-compatible defaults:

```yaml
auth-provider:
  name: oidc
  config:
    scopes:
      - openid
      - profile
      - email
      - groups
      - audience:server:client_id:veidemann-api
```

Providers which do not support Dex's cross-client audience scope, such as a
direct Keycloak deployment, can set an explicit list in the context's OIDC
auth-provider configuration:

```yaml
auth-provider:
  name: oidc
  config:
    scopes:
      - openid
      - profile
      - email
      - groups
```

Do not add `offline_access` to this list. Use `veidemannctl login --offline` to
request and require a renewable session.

## Known limitations

### Default server error message

When no `--server <address>` is provided or previously set using `veidemannctl
config set-address <address>` you might experience the following error message:

```console
$ veidemannctl get seed
Error: failed to build resolver: passthrough: received empty target in Build()
```

Setting `--server` or `veidemannctl config set-address <address>` to something
other than an empty string will resolve this specific error.
