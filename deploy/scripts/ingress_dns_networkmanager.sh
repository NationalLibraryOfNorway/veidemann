#!/usr/bin/env bash

# See https://minikube.sigs.k8s.io/docs/handbook/addons/ingress-dns

set -euo pipefail

# Get the current Minikube IP address
LOCAL_IP=$(minikube ip)

if [ -z "${LOCAL_IP}" ]; then
    echo "ERROR: Minikube IP not found. Is Minikube running?"
    exit 1
fi
echo "Minikube IP found: ${LOCAL_IP}"

TARGET_DOMAIN="test"
COREDNS="coredns"
COREDNS_NS="kube-system"

# Enable dnsmasq in NetworkManager and configure it to resolve .test to Minikube IP
grep ^dns=dnsmasq /etc/NetworkManager/NetworkManager.conf >/dev/null 2>&1 || {
	echo "Enabling dnsmasq in NetworkManager"
	sudo sed -i 's/^#\s*dns=dnsmasq/dns=dnsmasq/' /etc/NetworkManager/NetworkManager.conf
}
sudo mkdir -p /etc/NetworkManager/dnsmasq.d/
echo "Configuring NetworkManager to resolve .${TARGET_DOMAIN} to: ${LOCAL_IP}"

sudo tee /etc/NetworkManager/dnsmasq.d/minikube.conf >/dev/null <<EOF
server=/${TARGET_DOMAIN}/${LOCAL_IP}
EOF

sudo systemctl restart NetworkManager

echo "Configuring CoreDNS to forward *.${TARGET_DOMAIN} to: ${LOCAL_IP}"

# Construct the new CoreDNS block using the 'forward' plugin
FORWARD_BLOCK=$(cat <<EOF
${TARGET_DOMAIN}:53 {
    errors
    cache 30
    # Forward all queries for this domain to the Minikube host IP
    forward . ${LOCAL_IP}
}
EOF
)

# Get the current Corefile from the ConfigMap
CURRENT_COREFILE="$(
  kubectl get configmap "${COREDNS}" -n "${COREDNS_NS}" \
    -o jsonpath='{.data.Corefile}'
)"

# Check if the block is already present
if echo "${CURRENT_COREFILE}" | grep -q "^${TARGET_DOMAIN}:53"; then
    echo "CoreDNS configuration for '*.${TARGET_DOMAIN}' already exists. Skipping Corefile patch."
else
    # Append the new block to the existing Corefile content.
    # The 'forward' block must be outside the existing '.:53' block.
    NEW_COREFILE=$(printf '%s\n\n%s\n' "${CURRENT_COREFILE}" "${FORWARD_BLOCK}")

    # Recreate the ConfigMap manifest from literal and apply it.
    # This avoids JSON escaping hell and preserves the newlines correctly.
    kubectl create configmap "${COREDNS}" \
        -n "${COREDNS_NS}" \
        --from-literal=Corefile="${NEW_COREFILE}" \
        --dry-run=client -o yaml \
      | kubectl apply -f -

fi

echo "CoreDNS will auto-reload due to the reload plugin."
echo
echo "All services inside your Minikube cluster should now resolve any hostname ending in '.${TARGET_DOMAIN}' to the external service reachable at ${LOCAL_IP}."