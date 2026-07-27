# Harvester browser policy

The dev harvester overlay mounts a generated ConfigMap directly into the upstream `ghcr.io/browserless/chromium:v2.49.0` browser container.

Files involved:

- `harvester-policy.json` contains the Chromium managed-policy JSON.
- `kustomization.yaml` generates the `chromium-policy` ConfigMap from that file.
- `debug-patch.yaml` mounts that ConfigMap at `/etc/chromium/policies/managed` on the `browser` container.

This removes the need for a separate `harvester-browser` image directory in the repo.

Render the overlay with:

```sh
kubectl kustomize deploy/k8s/overlays/dev/harvester
```