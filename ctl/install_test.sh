#!/bin/sh

set -eu

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
readonly SCRIPT_DIR
readonly INSTALLER="${SCRIPT_DIR}/install.sh"

TEST_DIR=$(mktemp -d "${TMPDIR:-/tmp}/veidemannctl-installer-test.XXXXXX")
trap 'rm -rf "$TEST_DIR"' EXIT
trap 'exit 1' HUP INT TERM

fail() {
  printf 'install_test.sh: %s\n' "$*" >&2
  exit 1
}

assert_file() {
  [ -f "$1" ] || fail "expected file: $1"
}

assert_executable() {
  [ -x "$1" ] || fail "expected executable: $1"
}

assert_contains() {
  if ! grep -F "$2" "$1" >/dev/null; then
    sed -n '1,120p' "$1" >&2
    fail "expected '$2' in $1"
  fi
}

MOCK_BIN="${TEST_DIR}/bin"
FIXTURE_DIR="${TEST_DIR}/fixtures"
mkdir -p "$MOCK_BIN" "$FIXTURE_DIR"

cat >"${FIXTURE_DIR}/fake-binary" <<'EOF'
#!/bin/sh
case "${1:-}" in
  --version) printf '%s\n' 'veidemannctl version Client version: 2.3.4, Git commit: test' ;;
  completion) printf '# %s completion\n' "$2" ;;
  *) exit 1 ;;
esac
EOF
chmod 0755 "${FIXTURE_DIR}/fake-binary"

if command -v sha256sum >/dev/null 2>&1; then
  checksum=$(sha256sum "${FIXTURE_DIR}/fake-binary" | awk '{ print $1 }')
else
  checksum=$(shasum -a 256 "${FIXTURE_DIR}/fake-binary" | awk '{ print $1 }')
fi
{
  printf '%s\n' '{"assets": ['
  printf '  {\n    "name": "%s",\n    "digest": "sha256:%s"\n  },\n' veidemannctl_linux_amd64 "$checksum"
  printf '  {\n    "name": "%s",\n    "digest": "sha256:%s"\n  },\n' veidemannctl_linux_arm64 "$checksum"
  printf '  {\n    "name": "%s",\n    "digest": "sha256:%s"\n  },\n' veidemannctl_darwin_amd64 "$checksum"
  printf '  {\n    "name": "%s",\n    "digest": "sha256:%s"\n  }\n' veidemannctl_darwin_arm64 "$checksum"
  printf '%s\n' ']}'
} >"${FIXTURE_DIR}/release.json"
sed "s/sha256:${checksum}/sha256:$(printf '%064d' 0)/g" \
  "${FIXTURE_DIR}/release.json" >"${FIXTURE_DIR}/release.bad.json"

cat >"${FIXTURE_DIR}/releases.json" <<'EOF'
[
  {
    "tag_name": "api-v9.0.0"
  },
  {
    "tag_name": "ctl-v2.3.4"
  }
]
EOF

cat >"${MOCK_BIN}/curl" <<'EOF'
#!/bin/sh
set -eu

output=
url=
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o | -Lo)
      output=$2
      shift 2
      ;;
    -H | --proto)
      shift 2
      ;;
    --tlsv1.2 | -fsSL)
      shift
      ;;
    *)
      url=$1
      shift
      ;;
  esac
done

[ -n "$output" ]
[ -n "$url" ]
printf '%s\n' "$url" >>"$CURL_LOG"

case "$url" in
  *'/releases?'*) cp "${FIXTURE_DIR}/releases.json" "$output" ;;
  *'/releases/tags/'*)
    if [ "${MOCK_BAD_CHECKSUM:-0}" = 1 ]; then
      cp "${FIXTURE_DIR}/release.bad.json" "$output"
    else
      cp "${FIXTURE_DIR}/release.json" "$output"
    fi
    ;;
  *'/veidemannctl_'*) cp "${FIXTURE_DIR}/fake-binary" "$output" ;;
  *) exit 1 ;;
esac
EOF
chmod 0755 "${MOCK_BIN}/curl"

cat >"${MOCK_BIN}/uname" <<'EOF'
#!/bin/sh
case "${1:-}" in
  -s) printf '%s\n' "${MOCK_UNAME_S:-Linux}" ;;
  -m) printf '%s\n' "${MOCK_UNAME_M:-x86_64}" ;;
  *) exit 1 ;;
esac
EOF
chmod 0755 "${MOCK_BIN}/uname"

run_installer() {
  case_dir=$1
  shift
  mkdir -p "${case_dir}/home"
  : >"${case_dir}/curl.log"

  env \
    PATH="${MOCK_BIN}:${PATH}" \
    HOME="${case_dir}/home" \
    FIXTURE_DIR="$FIXTURE_DIR" \
    CURL_LOG="${case_dir}/curl.log" \
    "$@" \
    sh "$INSTALLER" >"${case_dir}/output.log" 2>&1
}

pinned_dir="${TEST_DIR}/pinned"
mkdir -p "$pinned_dir"
run_installer "$pinned_dir" \
  INSTALL_DIR="${pinned_dir}/install" \
  VEIDEMANNCTL_VERSION=v1.2.3 \
  SHELL=/bin/bash
assert_executable "${pinned_dir}/install/veidemannctl"
assert_file "${pinned_dir}/home/.local/share/bash-completion/completions/veidemannctl"
assert_contains "${pinned_dir}/curl.log" '/ctl-v1.2.3/veidemannctl_linux_amd64'
assert_contains "${pinned_dir}/curl.log" '/releases/tags/ctl-v1.2.3'

latest_dir="${TEST_DIR}/latest"
mkdir -p "$latest_dir"
run_installer "$latest_dir" \
  INSTALL_DIR="${latest_dir}/install" \
  VEIDEMANNCTL_COMPLETION=none \
  MOCK_UNAME_S=Darwin \
  MOCK_UNAME_M=arm64
assert_executable "${latest_dir}/install/veidemannctl"
assert_contains "${latest_dir}/curl.log" 'api.github.com/repos/NationalLibraryOfNorway/veidemann/releases?per_page=100&page=1'
assert_contains "${latest_dir}/curl.log" '/ctl-v2.3.4/veidemannctl_darwin_arm64'

bad_checksum_dir="${TEST_DIR}/bad-checksum"
mkdir -p "${bad_checksum_dir}/home"
: >"${bad_checksum_dir}/curl.log"
if env \
  PATH="${MOCK_BIN}:${PATH}" \
  HOME="${bad_checksum_dir}/home" \
  FIXTURE_DIR="$FIXTURE_DIR" \
  CURL_LOG="${bad_checksum_dir}/curl.log" \
  INSTALL_DIR="${bad_checksum_dir}/install" \
  VEIDEMANNCTL_VERSION=1.2.3 \
  VEIDEMANNCTL_COMPLETION=none \
  MOCK_BAD_CHECKSUM=1 \
  sh "$INSTALLER" >"${bad_checksum_dir}/output.log" 2>&1; then
  fail 'installer accepted a mismatched checksum'
fi
[ ! -e "${bad_checksum_dir}/install/veidemannctl" ] || fail 'binary installed after checksum failure'
assert_contains "${bad_checksum_dir}/output.log" 'checksum verification failed'

unsupported_dir="${TEST_DIR}/unsupported"
mkdir -p "${unsupported_dir}/home"
: >"${unsupported_dir}/curl.log"
if env \
  PATH="${MOCK_BIN}:${PATH}" \
  HOME="${unsupported_dir}/home" \
  FIXTURE_DIR="$FIXTURE_DIR" \
  CURL_LOG="${unsupported_dir}/curl.log" \
  INSTALL_DIR="${unsupported_dir}/install" \
  VEIDEMANNCTL_VERSION=1.2.3 \
  VEIDEMANNCTL_COMPLETION=none \
  MOCK_UNAME_M=mips64 \
  sh "$INSTALLER" >"${unsupported_dir}/output.log" 2>&1; then
  fail 'installer accepted an unsupported architecture'
fi
assert_contains "${unsupported_dir}/output.log" 'unsupported architecture: mips64'

cat >"${MOCK_BIN}/veidemannctl" <<'EOF'
#!/bin/sh
printf '%s\n' 'veidemannctl version Client version: 1.0.0, Git commit: test'
EOF
chmod 0755 "${MOCK_BIN}/veidemannctl"

prerequisite_dir="${TEST_DIR}/prerequisite"
mkdir -p "${prerequisite_dir}/home"
: >"${prerequisite_dir}/curl.log"
printf 'y\n' | env \
  PATH="${MOCK_BIN}:${PATH}" \
  HOME="${prerequisite_dir}/home" \
  FIXTURE_DIR="$FIXTURE_DIR" \
  CURL_LOG="${prerequisite_dir}/curl.log" \
  bash "${SCRIPT_DIR}/../deploy/scripts/prerequisites.sh" veidemannctl \
  >"${prerequisite_dir}/output.log" 2>&1
assert_contains "${prerequisite_dir}/output.log" 'Installing veidemannctl 0.11.0'
assert_contains "${prerequisite_dir}/curl.log" \
  '/NationalLibraryOfNorway/veidemann/releases/download/ctl-v0.11.0/veidemannctl_linux_amd64'
assert_executable "${prerequisite_dir}/home/.local/bin/veidemannctl"
assert_file "${prerequisite_dir}/home/.local/share/bash-completion/completions/veidemannctl"

printf '%s\n' 'installer tests passed'
