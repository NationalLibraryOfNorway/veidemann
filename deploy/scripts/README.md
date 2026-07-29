# Local deployment scripts

Start Minikube with the ingress and ingress-dns add-ons:

```shell
./start_minikube.sh
```

DNS configuration is intentionally not part of `start_minikube.sh`, because it
changes the host's resolver settings with `sudo`. Run the helper that matches
the host's resolver after starting Minikube.

For `systemd-resolved` with Minikube's Docker driver:

```shell
./start_minikube.sh && ./configure_resolved.sh
```

This attaches the `.test` route to Minikube's Docker bridge. Run the helper again
after the bridge is recreated or the host restarts.

For NetworkManager with its `dnsmasq` plugin installed:

```shell
./start_minikube.sh && ./configure_networkmanager.sh
```

This writes NetworkManager drop-ins under `/etc/NetworkManager` and restarts
NetworkManager. The drop-ins persist across restarts and are updated with the
current Minikube IP each time the helper runs. Use only one DNS helper for a
given host.
