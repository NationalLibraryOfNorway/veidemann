#!/bin/sh

set -eu

PROGRAM=${0##*/}

usage() {
  cat <<EOF
Usage: $PROGRAM [--apply] DEPLOYMENT_BASE

Preview changes needed to mirror Veidemann's Kubernetes base into an existing
deployment base directory. Review the diff and backport any needed changes
before applying the sync.

Options:
  --apply     apply the sync after showing the preview; deletes destination-only files
  -h, --help  show this help
EOF
}

die() {
  printf '%s: %s\n' "$PROGRAM" "$*" >&2
  exit 2
}

apply=false
destination=
while [ "$#" -gt 0 ]; do
  case $1 in
    --apply) apply=true ;;
    -h|--help) usage; exit 0 ;;
    --) shift; break ;;
    -*) die "unknown option: $1" ;;
    *)
      [ -z "$destination" ] || die "expected exactly one deployment base directory"
      destination=$1
      ;;
  esac
  shift
done
for argument do
  [ -z "$destination" ] || die "expected exactly one deployment base directory"
  destination=$argument
done
[ -n "$destination" ] || die "a deployment base directory is required"

command -v diff >/dev/null 2>&1 || die "diff is required"
command -v rsync >/dev/null 2>&1 || die "rsync is required"

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P) || die "cannot locate script directory"
source_dir=$script_dir/../deploy/k8s/base
[ -d "$source_dir" ] || die "source base directory does not exist: $source_dir"
[ -d "$destination" ] || die "deployment base directory does not exist: $destination"
source_dir=$(CDPATH='' cd -- "$source_dir" && pwd -P) || die "cannot access source base directory"
destination=$(CDPATH='' cd -- "$destination" && pwd -P) || die "cannot access deployment base directory"

case "${source_dir%/}/" in
  "${destination%/}/"*) die "source and deployment base directories must not overlap" ;;
esac
case "${destination%/}/" in
  "${source_dir%/}/"*) die "source and deployment base directories must not overlap" ;;
esac

printf 'Diff (deployment base -> Veidemann base):\n'
diff_status=0
diff -ru -- "$destination" "$source_dir" || diff_status=$?
[ "$diff_status" -le 1 ] || die "diff failed (exit $diff_status)"

printf '\nRsync preview (Veidemann base -> deployment base):\n'
rsync -aicn --delete -- "$source_dir/" "$destination/" || die "rsync preview failed"

if [ "$apply" = true ]; then
  printf '\nApplying sync:\n'
  rsync -aic --delete -- "$source_dir/" "$destination/" || die "rsync failed"
fi
