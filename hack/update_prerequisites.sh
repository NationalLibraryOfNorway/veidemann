#!/bin/sh

set -eu

PROGRAM=${0##*/}
SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH='' cd -- "${SCRIPT_DIR}/.." && pwd)
PREREQUISITES_FILE=${PREREQUISITES_FILE:-${REPO_ROOT}/deploy/scripts/prerequisites.sh}

usage() {
  cat <<EOF
Usage: $PROGRAM

Update the tool versions in deploy/scripts/prerequisites.sh from their official
release sources. Linkerd is set to the newest release carrying its RECOMMENDED
status label, which may be older than the newest Linkerd edge release.
EOF
}

die() {
  printf '%s: %s\n' "$PROGRAM" "$*" >&2
  exit 1
}

fetch() {
  curl --proto '=https' --tlsv1.2 -fsSL "$1"
}

fetch_github() {
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    curl --proto '=https' --tlsv1.2 -fsSL \
      -H "Authorization: Bearer ${GITHUB_TOKEN}" \
      -H 'Accept: application/vnd.github+json' \
      "$1"
  else
    curl --proto '=https' --tlsv1.2 -fsSL \
      -H 'Accept: application/vnd.github+json' \
      "$1"
  fi
}

latest_github_tag() {
  fetch_github "https://api.github.com/repos/$1/releases/latest" |
    jq -er '.tag_name | select(type == "string" and length > 0)'
}

validate_version() {
  name=$1
  version=$2
  pattern=$3
  printf '%s\n' "$version" | grep -Eq "$pattern" ||
    die "unexpected ${name} version from release source: ${version}"
}

case ${1:-} in
  -h | --help)
    usage
    exit 0
    ;;
  '') ;;
  *)
    usage >&2
    exit 2
    ;;
esac

command -v curl >/dev/null 2>&1 || die 'curl is required'
command -v jq >/dev/null 2>&1 || die 'jq is required'
command -v awk >/dev/null 2>&1 || die 'awk is required'
[ -f "$PREREQUISITES_FILE" ] || die "file not found: $PREREQUISITES_FILE"

linkerd_version=$(fetch_github 'https://api.github.com/repos/linkerd/linkerd2/releases?per_page=100' |
  jq -er '[.[] | select(.draft == false) | select((.body // "") | contains("release_status-RECOMMENDED-lightgreen"))][0].tag_name // empty')
kubectl_version=$(fetch 'https://dl.k8s.io/release/stable.txt')
minikube_version=$(latest_github_tag 'kubernetes/minikube')
helm_version=$(latest_github_tag 'helm/helm')
skaffold_version=$(latest_github_tag 'GoogleContainerTools/skaffold')
step_version=$(latest_github_tag 'smallstep/cli')
step_version=${step_version#v}
veidemannctl_version=$(fetch_github 'https://api.github.com/repos/NationalLibraryOfNorway/veidemann/releases?per_page=100' |
  jq -er '[.[] | select(.draft == false and .prerelease == false) | .tag_name | select(startswith("ctl-v"))][0] // empty')
veidemannctl_version=${veidemannctl_version#ctl-v}

validate_version Linkerd "$linkerd_version" '^edge-[0-9]+\.[0-9]+\.[0-9]+$'
validate_version kubectl "$kubectl_version" '^v[0-9]+\.[0-9]+\.[0-9]+$'
validate_version Minikube "$minikube_version" '^v[0-9]+\.[0-9]+\.[0-9]+$'
validate_version Helm "$helm_version" '^v[0-9]+\.[0-9]+\.[0-9]+$'
validate_version Skaffold "$skaffold_version" '^v[0-9]+\.[0-9]+\.[0-9]+$'
validate_version Step "$step_version" '^[0-9]+\.[0-9]+\.[0-9]+$'
validate_version veidemannctl "$veidemannctl_version" '^[0-9]+\.[0-9]+\.[0-9]+$'

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/update-prerequisites.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT
trap 'exit 1' HUP INT TERM
staged_file="${work_dir}/prerequisites.sh"

awk \
  -v linkerd="$linkerd_version" \
  -v kubectl="$kubectl_version" \
  -v minikube="$minikube_version" \
  -v veidemannctl="$veidemannctl_version" \
  -v helm="$helm_version" \
  -v skaffold="$skaffold_version" \
  -v step="$step_version" '
    /^LINKERD_VERSION=/ { print "LINKERD_VERSION=" linkerd; found_linkerd++; next }
    /^KUBECTL_VERSION=/ { print "KUBECTL_VERSION=" kubectl; found_kubectl++; next }
    /^MINIKUBE_VERSION=/ { print "MINIKUBE_VERSION=" minikube; found_minikube++; next }
    /^VEIDEMANNCTL_VERSION=/ { print "VEIDEMANNCTL_VERSION=" veidemannctl; found_veidemannctl++; next }
    /^HELM_VERSION=/ { print "HELM_VERSION=" helm; found_helm++; next }
    /^SKAFFOLD_VERSION=/ { print "SKAFFOLD_VERSION=" skaffold; found_skaffold++; next }
    /^STEP_VERSION=/ { print "STEP_VERSION=" step; found_step++; next }
    { print }
    END {
      if (found_linkerd != 1 || found_kubectl != 1 || found_minikube != 1 ||
          found_veidemannctl != 1 || found_helm != 1 || found_skaffold != 1 ||
          found_step != 1) {
        print "expected exactly one assignment for every prerequisite version" > "/dev/stderr"
        exit 1
      }
    }
  ' "$PREREQUISITES_FILE" >"$staged_file"

if cmp -s "$staged_file" "$PREREQUISITES_FILE"; then
  printf '%s\n' "Prerequisite versions are already current."
  exit 0
fi

chmod 0755 "$staged_file"
mv "$staged_file" "$PREREQUISITES_FILE"

printf '%s\n' "Updated ${PREREQUISITES_FILE}:"
printf '  Linkerd       %s\n' "$linkerd_version"
printf '  kubectl       %s\n' "$kubectl_version"
printf '  Minikube      %s\n' "$minikube_version"
printf '  veidemannctl  %s\n' "$veidemannctl_version"
printf '  Helm          %s\n' "$helm_version"
printf '  Skaffold      %s\n' "$skaffold_version"
printf '  Step CLI      %s\n' "$step_version"
