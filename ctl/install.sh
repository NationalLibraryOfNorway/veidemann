#!/bin/sh

set -eu

readonly REPOSITORY="NationalLibraryOfNorway/veidemann"
readonly RELEASES_API="https://api.github.com/repos/${REPOSITORY}/releases"
readonly RELEASES_BASE="https://github.com/${REPOSITORY}/releases/download"

TEMP_DIR=
INSTALL_TEMP=
COMPLETION_TEMP=

log() {
  printf '%s\n' "$*"
}

fail() {
  printf 'veidemannctl installer: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [ -n "$COMPLETION_TEMP" ]; then
    rm -f "$COMPLETION_TEMP"
  fi
  if [ -n "$INSTALL_TEMP" ]; then
    rm -f "$INSTALL_TEMP"
  fi
  if [ -n "$TEMP_DIR" ]; then
    rm -rf "$TEMP_DIR"
  fi
}

trap cleanup EXIT
trap 'exit 1' HUP INT TERM

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

fetch() {
  url=$1
  output=$2

  if [ -n "${GITHUB_TOKEN:-}" ]; then
    curl --proto '=https' --tlsv1.2 -fsSL \
      -H "Authorization: Bearer ${GITHUB_TOKEN}" \
      -H 'Accept: application/vnd.github+json' \
      -o "$output" "$url"
  else
    curl --proto '=https' --tlsv1.2 -fsSL \
      -H 'Accept: application/vnd.github+json' \
      -o "$output" "$url"
  fi
}

normalize_version() {
  requested=$1

  case "$requested" in
    ctl-v*) version=${requested#ctl-v} ;;
    v*) version=${requested#v} ;;
    *) version=$requested ;;
  esac

  case "$version" in
    '' | *[!0-9A-Za-z.+-]*) fail "invalid version: $requested" ;;
  esac
  case "$version" in
    [0-9]*) ;;
    *) fail "invalid version: $requested" ;;
  esac

  VERSION=$version
  RELEASE_TAG="ctl-v${version}"
}

find_latest_version() {
  page=1
  while [ "$page" -le 10 ]; do
    response="${TEMP_DIR}/releases-${page}.json"
    fetch "${RELEASES_API}?per_page=100&page=${page}" "$response"

    tag=$(sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\(ctl-v[^"[:space:]]*\)".*/\1/p' "$response" | sed -n '1p')
    if [ -n "$tag" ]; then
      normalize_version "$tag"
      return
    fi

    if ! grep -q '"tag_name"' "$response"; then
      break
    fi
    page=$((page + 1))
  done

  fail "could not find a published ctl-v* release; set VEIDEMANNCTL_VERSION explicitly"
}

detect_platform() {
  case $(uname -s) in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) fail "unsupported operating system: $(uname -s)" ;;
  esac

  case $(uname -m) in
    x86_64 | amd64) arch=amd64 ;;
    arm64 | aarch64) arch=arm64 ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac

  ASSET="veidemannctl_${os}_${arch}"
}

get_release_checksum() {
  metadata="${TEMP_DIR}/release.json"
  fetch "${RELEASES_API}/tags/${RELEASE_TAG}" "$metadata"

  expected=$(awk -v asset="$ASSET" '
    index($0, "\"name\": \"" asset "\"") { in_asset = 1 }
    in_asset && /"digest":[[:space:]]*"sha256:/ {
      line = $0
      sub(/^.*"digest":[[:space:]]*"sha256:/, "", line)
      sub(/".*$/, "", line)
      print line
      exit
    }
  ' "$metadata")
  case "$expected" in
    '' | *[!0-9A-Fa-f]*) fail "release ${RELEASE_TAG} does not publish a valid SHA-256 digest for ${ASSET}" ;;
  esac
  if [ "${#expected}" -ne 64 ]; then
    fail "release ${RELEASE_TAG} does not publish a valid SHA-256 digest for ${ASSET}"
  fi

  EXPECTED_CHECKSUM=$expected
}

verify_checksum() {
  binary=$1

  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$binary" | awk '{ print $1 }')
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$binary" | awk '{ print $1 }')
  else
    fail "sha256sum or shasum is required to verify the download"
  fi

  if [ "$actual" != "$EXPECTED_CHECKSUM" ]; then
    fail "checksum verification failed for ${ASSET}"
  fi
}

install_completion() {
  shell_name=${VEIDEMANNCTL_COMPLETION:-auto}
  if [ "$shell_name" = auto ]; then
    shell_name=${SHELL:-}
    shell_name=${shell_name##*/}
    case "$shell_name" in
      bash | zsh | fish) ;;
      *)
        log "Shell completion skipped: set VEIDEMANNCTL_COMPLETION to bash, zsh, or fish."
        return
        ;;
    esac
  fi

  case "$shell_name" in
    none) return ;;
    bash)
      [ -n "${HOME:-}" ] || fail "HOME is required to install bash completion"
      completion_path="${XDG_DATA_HOME:-${HOME}/.local/share}/bash-completion/completions/veidemannctl"
      ;;
    zsh)
      [ -n "${HOME:-}" ] || fail "HOME is required to install zsh completion"
      completion_path="${XDG_DATA_HOME:-${HOME}/.local/share}/zsh/site-functions/_veidemannctl"
      ;;
    fish)
      [ -n "${HOME:-}" ] || fail "HOME is required to install fish completion"
      completion_path="${XDG_CONFIG_HOME:-${HOME}/.config}/fish/completions/veidemannctl.fish"
      ;;
    *) fail "unsupported completion shell: $shell_name" ;;
  esac

  completion_dir=${completion_path%/*}
  mkdir -p "$completion_dir"
  [ -w "$completion_dir" ] || fail "completion directory is not writable: $completion_dir"

  COMPLETION_TEMP="${completion_path}.tmp.$$"
  "$TARGET" completion "$shell_name" >"$COMPLETION_TEMP"
  chmod 0644 "$COMPLETION_TEMP"
  mv -f "$COMPLETION_TEMP" "$completion_path"
  COMPLETION_TEMP=
  log "Installed ${shell_name} completion to ${completion_path}"

  if [ "$shell_name" = zsh ]; then
    log "Ensure ${completion_dir} is present in your zsh fpath."
  fi
}

require_command curl
require_command awk
require_command sed
require_command grep
require_command uname
require_command mktemp

if [ -z "${INSTALL_DIR:-}" ]; then
  [ -n "${HOME:-}" ] || fail "HOME is required when INSTALL_DIR is not set"
  INSTALL_DIR="${HOME}/.local/bin"
fi

TEMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/veidemannctl-install.XXXXXX")
detect_platform

if [ -n "${VEIDEMANNCTL_VERSION:-}" ]; then
  normalize_version "$VEIDEMANNCTL_VERSION"
else
  find_latest_version
fi

release_url="${RELEASES_BASE}/${RELEASE_TAG}"
binary="${TEMP_DIR}/${ASSET}"

log "Downloading veidemannctl ${VERSION} for ${os}/${arch}..."
get_release_checksum
fetch "${release_url}/${ASSET}" "$binary"
verify_checksum "$binary"
chmod 0755 "$binary"
"$binary" --version >/dev/null

mkdir -p "$INSTALL_DIR"
[ -w "$INSTALL_DIR" ] || fail "install directory is not writable: $INSTALL_DIR"

TARGET="${INSTALL_DIR}/veidemannctl"
INSTALL_TEMP="${INSTALL_DIR}/.veidemannctl.tmp.$$"
cp "$binary" "$INSTALL_TEMP"
chmod 0755 "$INSTALL_TEMP"
mv -f "$INSTALL_TEMP" "$TARGET"
INSTALL_TEMP=

log "Installed veidemannctl ${VERSION} to ${TARGET}"
install_completion

case ":${PATH}:" in
  *:"${INSTALL_DIR}":*) ;;
  *) log "Add ${INSTALL_DIR} to PATH to run veidemannctl." ;;
esac
