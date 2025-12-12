#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${SCRIPT_DIR}"/start_minikube.sh
"${SCRIPT_DIR}"/ingress_dns_networkmanager.sh
