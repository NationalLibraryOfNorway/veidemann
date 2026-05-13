# Kvrocks Backup Component

This component adds scheduled backup export to the Kvrocks StatefulSet without using `kubectl`, `pods/exec`, or Kubernetes API discovery.

## Design

- each Kvrocks pod mounts one shared RWX PVC at `/var/lib/kvrocks-backups`
- `kvrocks-start.sh` enables backup settings only when the component adds the backup env vars
- each pod writes to a unique per-pod directory, for example `/var/lib/kvrocks-backups/kvrocks-0`
- Kvrocks generates local backups with `bgsave-cron`
- a separate uploader `CronJob` mounts the same RWX PVC read-only and uploads full per-pod archives to S3-compatible storage

The component assumes `bgsave-cron` may run in every Kvrocks server process, including replicas. It therefore isolates backup output by pod name on the shared RWX volume.

## Add To An Overlay

Reference the component from a production-style overlay:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../../base/kvrocks

components:
  - ../../../components/kvrocks-backup
```

Patch the component defaults in the overlay for your storage class, bucket, schedules, and credentials.

## RWX Volume Requirement

The PVC in this component requests `ReadWriteMany` storage:

- claim name: `kvrocks-backups`
- default size: `100Gi`
- no `storageClassName` is set in the component

Your overlay must provide an RWX-capable default storage class or patch the PVC with an explicit `storageClassName`.

If the cluster cannot provide RWX storage, do not use this component as-is.

## Required S3 Configuration

The component provides a placeholder `ConfigMap` named `kvrocks-backup-config` with these required keys:

- `bucket-name`
- `bucket-prefix`
- `endpoint-url`
- `region`
- `backup-root`
- `statefulset-name`

The default placeholder values are intentionally invalid so an unpatched overlay fails fast.

The uploader job also references a Secret named `kvrocks-backup-s3` with these optional keys:

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_SESSION_TOKEN`

If you use workload identity instead of static credentials, patch the CronJob env list accordingly and remove the secret dependency from your overlay.

## Schedule Relationship

- Kvrocks per-pod backup schedule default: `0 4 * * *`
- uploader schedule default: `0 5 * * *`
- uploader `CronJob` timezone: `Etc/UTC`

This component assumes the Kvrocks container is using UTC for `bgsave-cron`. If your image or runtime uses another local timezone, set `TZ=Etc/UTC` in the Kvrocks container or patch the cron expression accordingly.

The uploader runs one hour after the default `bgsave-cron` to leave time for scheduled checkpoint creation.

## Generated Kvrocks Settings

When the component is enabled, the Kvrocks container appends these settings to its generated config:

```properties
backup-dir /var/lib/kvrocks-backups/<pod-name>
bgsave-cron 0 4 * * *
max-backup-to-keep 1
max-backup-keep-hours 24
```

The exact values come from container env vars, so overlays can patch them without editing the base `kvrocks.conf`.

## Object Layout

The uploader archives each per-pod directory separately and stores it at:

```text
s3://<bucket>/<prefix>/<namespace>/<statefulset>/<pod-name>/kvrocks-<pod-name>-<timestamp>.tar.gz
```

Example:

```text
s3://veidemann-prod/kvrocks/prod/kvrocks/kvrocks-1/kvrocks-kvrocks-1-20260427T050000Z.tar.gz
```

## Restore Outline

Treat restore as an explicit operational workflow, not an implicit feature of this component.

Recommended outline:

1. Download one uploaded archive for a single pod into an isolated workspace.
2. Extract the archive and inspect that the per-pod directory contains a valid Kvrocks backup with `CURRENT` and the expected checkpoint files.
3. Provision an isolated namespace and storage for a one-off recovery test.
4. Copy the extracted backup directory into the location your recovery workflow expects for Kvrocks backup input.
5. Start an isolated Kvrocks recovery instance or recovery job that restores from that backup data and verify it opens successfully.
6. Prove the restored instance contains expected keys before declaring the backup workflow production-ready.

The exact restore mechanics are environment-specific and must be documented alongside the overlay that ships this component.

## Validation Checklist

1. Deploy Kvrocks without this component and verify the base still works.
2. Deploy with this component in a non-production test overlay.
3. Verify each Kvrocks pod mounts the RWX backup volume.
4. Verify each generated Kvrocks config contains `backup-dir /var/lib/kvrocks-backups/<pod-name>` and the expected `bgsave-cron`.
5. Wait for scheduled `BGSAVE` or trigger one manually and confirm each configured pod writes to its own directory.
6. Verify the directories do not collide across pods.
7. Run the uploader `CronJob` manually.
8. Verify uploaded objects exist in the expected bucket and prefix.
9. Restore at least one uploaded backup into an isolated namespace and prove Kvrocks starts with expected data.
10. Repeat backup generation and upload validation after Sentinel failover so a different master lifecycle is exercised.