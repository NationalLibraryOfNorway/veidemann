# Veidemann Agent Guide

## Project Overview

Veidemann is a multi-language monorepo. The repository contains:

- Java services and shared libraries built with Gradle under `commons`, `controller`, `frontier`, `java-api`, and `rethinkdbadapter`
- Multiple Go modules for operational services and tools, for example `api`, `browser-controller`, `contentwriter`, `ctl`, `dns-resolver`, and `frontier-queue-workers`
- An Angular dashboard in `dashboard`
- Kubernetes deployment manifests under `deploy/k8s`

Treat the repository as a collection of independent build units. Do not assume one top-level build command covers every Go module or frontend workflow.

## Build And Test Commands

Use the narrowest command that proves your change.

### Gradle

- Root metadata and shared plugin resolution: `./gradlew help`
- Full Java build: `./gradlew build`
- Targeted module build: `./gradlew :commons:build` or `./gradlew :controller:test`

### Go

Run Go commands from the module you changed.

- Example: `cd ctl && go test ./...`
- Example: `cd browser-controller && go test ./...`
- Example: `cd contentwriter && go test ./...`

If a change spans multiple Go modules, test each affected module separately.

### Dashboard

- `cd dashboard && npm run build`
- `cd dashboard && npm run lint`
- `cd dashboard && npm run test:chromium`

### Kubernetes Manifests

- Render a base: `kustomize build deploy/k8s/base/kvrocks`
- Render an overlay: `kustomize build deploy/k8s/overlays/dev/kvrocks`

Always render the affected Kustomize base or overlay after editing manifests.

## Code Style Guidelines

- Match the style already used in the touched module.
- Keep changes minimal and localized. Do not reformat unrelated files.
- Prefer configuration through overlays instead of baking environment-specific values into shared Kubernetes bases.
- For shell scripts in manifests, keep them POSIX `sh` compatible unless the image clearly guarantees `bash`.
- For Kubernetes YAML, favor explicit labels, pinned images, and conservative rollout behavior for stateful services.

## Testing Instructions

- Validate the smallest affected surface first, then the rendered or integrated surface.
- For stateful Kubernetes changes, prefer render validation plus an operational checklist in docs when live-cluster testing is not available.
- For Kvrocks changes specifically, validate bootstrap, failover, full restart, and backup behavior through the overlay you expect to ship.
- If you cannot run a meaningful test locally, say so explicitly in the final handoff.

## Security Considerations

- Never hardcode real credentials in the repository.
- Kubernetes bases may include placeholder secret references, but overlays or deployment systems must provide the real values.
- Preserve fail-closed behavior for stateful recovery paths unless there is a verified stronger alternative.
- Be careful with generic Services for leader-replica topologies; names should not imply safe write routing when they select all pods.
- Tighten `NetworkPolicy` in overlays for production. Same-namespace access is only a baseline posture.

## Deployment Notes

- Stateful workloads under `deploy/k8s` should be treated as operational code, not just configuration.
- When editing `deploy/k8s/base/kvrocks`, keep Sentinel recovery safety, backup behavior, and service naming aligned with the README in that directory.
- Prefer overlay patches for storage class, bucket names, endpoint URLs, replica counts, and any cluster-specific values.

## Pull Request And Change Hygiene

- Mention which module or manifest set you validated.
- Call out operational risks for stateful or security-sensitive changes.
- If a change adds a new required secret, config map value, or manual recovery step, update the relevant README in the same change.