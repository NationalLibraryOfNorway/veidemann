#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${SCRIPT_DIR}"/prerequisites.sh veidemannctl

veidemannctl config create-context minikube || true
veidemannctl config use-context minikube
veidemannctl config set-address veidemann.test:443

# Create a temp file and ensure it's removed no matter what happens
ca_file="$(mktemp -p "${SCRIPT_DIR}" ca.crt.XXXXXX)"
cleanup() {
  rm -f "$ca_file"
}
trap cleanup EXIT INT TERM HUP

kubectl --context=minikube get secrets veidemann-tls -o jsonpath="{.data.tls\.crt}" \
  | base64 -d \
  > "$ca_file"

veidemannctl config import-ca "$ca_file"
