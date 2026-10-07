#!/usr/bin/env bash

set -euo pipefail

K="kubectl --context=minikube"

if ! minikube status >/dev/null 2>&1; then
  minikube start --driver=docker --container-runtime=docker --addons=ingress,ingress-dns
fi

$K wait --for=condition=Ready node --all --timeout=5m
