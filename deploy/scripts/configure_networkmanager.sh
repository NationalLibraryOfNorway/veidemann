#!/usr/bin/env bash

set -euo pipefail

# See https://minikube.sigs.k8s.io/docs/handbook/addons/ingress-dns

if ! systemctl is-active --quiet NetworkManager; then
  echo "ERROR: NetworkManager is not active." >&2
  exit 1
fi

if ! command -v dnsmasq >/dev/null 2>&1; then
  echo "ERROR: dnsmasq is required by NetworkManager for Minikube DNS." >&2
  exit 1
fi

minikube_ip="$(minikube ip)"

if [[ -z "${minikube_ip}" ]]; then
  echo "ERROR: Minikube IP not found. Is Minikube running?" >&2
  exit 1
fi

# Enable NetworkManager's dnsmasq plugin through a drop-in instead of editing
# the distribution-owned NetworkManager.conf file.
sudo mkdir -p /etc/NetworkManager/conf.d /etc/NetworkManager/dnsmasq.d
sudo tee /etc/NetworkManager/conf.d/minikube-dnsmasq.conf >/dev/null <<'EOF'
[main]
dns=dnsmasq
EOF

# Send only the .test DNS zone to Minikube's ingress-dns nameserver.
echo "Configuring NetworkManager to route .test queries to ${minikube_ip}"
sudo tee /etc/NetworkManager/dnsmasq.d/minikube.conf >/dev/null <<EOF
server=/test/${minikube_ip}
EOF

sudo systemctl restart NetworkManager
