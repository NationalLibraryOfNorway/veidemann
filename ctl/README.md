# Veidemannctl

## Install

The installer downloads the latest `ctl-v*` GitHub release for Linux or macOS
on AMD64 or ARM64, verifies it against the release asset's GitHub-published
SHA-256 digest, and installs it to `~/.local/bin`. Releases also include a
`SHA256SUMS` file for manual verification.

```console
curl --proto '=https' --tlsv1.2 -fsSL \
  https://raw.githubusercontent.com/NationalLibraryOfNorway/veidemann/main/ctl/install.sh | sh
```

For an inspect-then-run workflow, download the installer first:

```console
installer="$(mktemp)"
curl --proto '=https' --tlsv1.2 -fsSL \
  -o "$installer" \
  https://raw.githubusercontent.com/NationalLibraryOfNorway/veidemann/main/ctl/install.sh
less "$installer"
sh "$installer"
rm "$installer"
```

By default, completion is installed for the shell named by `$SHELL`. The
installer does not modify shell startup files. Set these environment variables
to customize the installation:

| Variable | Purpose |
| --- | --- |
| `VEIDEMANNCTL_VERSION` | Install a specific version, such as `0.11.0`, instead of the latest release. |
| `INSTALL_DIR` | Install the binary somewhere other than `~/.local/bin`. The directory must be writable. |
| `VEIDEMANNCTL_COMPLETION` | Use `bash`, `zsh`, `fish`, or `none` instead of automatic shell detection. |
| `GITHUB_TOKEN` | Authenticate GitHub API and release requests when anonymous rate limits are insufficient. |

For example, to install a specific version without completion:

```console
curl --proto '=https' --tlsv1.2 -fsSL \
  https://raw.githubusercontent.com/NationalLibraryOfNorway/veidemann/main/ctl/install.sh \
  | VEIDEMANNCTL_VERSION=0.11.0 VEIDEMANNCTL_COMPLETION=none sh
```

If `~/.local/bin` is not already in `PATH`, the installer prints a reminder.
Zsh users may also need to add the printed completion directory to `fpath`.

## Usage

To get a list of available commands and configuration flags:

```console
veidemannctl -h
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
