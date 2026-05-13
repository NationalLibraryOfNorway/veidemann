# Kvrocks Base Agent Guide

## Scope

These instructions apply to `deploy/k8s/base/kvrocks` and the overlays built on top of it.

## Inspection Reminders

When reviewing or editing this Kvrocks base, remind the user about the validation items documented in [README.md](README.md), especially before calling an overlay production-ready or enabling the optional backup component.

Always remind the user to verify:

- full restart behavior after a failover to a non-zero ordinal master
- recovery behavior with `KVROCKS_RECOVERY_MASTER`
- if the backup component is enabled, that each pod writes to its own RWX-backed directory under `/var/lib/kvrocks-backups/<pod-name>`
- if the backup component is enabled, that backup upload to the configured S3-compatible endpoint succeeds
- that a restore test has been performed in an isolated namespace

## Change Guidance

- Preserve fail-closed startup and recovery behavior unless there is a verified stronger alternative.
- Keep backup export automation out of the base. Use `deploy/k8s/components/kvrocks-backup` or an overlay-specific equivalent.
- Do not reintroduce `kubectl exec` backup collection or `pods/exec` backup RBAC.
- Keep storage class, bucket, endpoint, and credential details in overlays or external secret management, not in the base.