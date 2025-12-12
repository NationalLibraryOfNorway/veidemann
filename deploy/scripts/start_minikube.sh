#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K="kubectl --context=minikube"

"${SCRIPT_DIR}/prerequisites.sh" kubectl minikube skaffold helm

if ! minikube status >/dev/null 2>&1; then
  minikube start --addons=ingress,ingress-dns
fi

$K wait --for=condition=Ready node --all --timeout=5m
