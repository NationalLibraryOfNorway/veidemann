#!/usr/bin/env bash

set -euo pipefail

if ! command -v resolvectl >/dev/null 2>&1; then
  echo "ERROR: resolvectl is required to configure Minikube DNS." >&2
  exit 1
fi

if ! systemctl is-active --quiet systemd-resolved; then
  echo "ERROR: systemd-resolved is not active." >&2
  exit 1
fi

# Minikube's Docker driver creates a `minikube` network whose Linux bridge is
# named `br-<first 12 characters of the Docker network ID>`.
network_id="$(docker network inspect --format '{{.Id}}' minikube)"
iface="br-${network_id:0:12}"
minikube_ip="$(minikube ip)"

if [[ ! -e "/sys/class/net/${iface}" ]]; then
  echo "ERROR: Docker bridge ${iface} for the Minikube network was not found." >&2
  exit 1
fi

# Route only the .test DNS zone to Minikube's ingress-dns nameserver. The `~`
# makes this a route-only domain, so other queries use the host's normal DNS.
echo "Configuring systemd-resolved to route .test queries to ${minikube_ip} on ${iface}"
sudo resolvectl dns "${iface}" "${minikube_ip}"
sudo resolvectl domain "${iface}" '~test'
