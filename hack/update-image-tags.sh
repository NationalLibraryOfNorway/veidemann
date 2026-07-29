#!/bin/sh

set -eu

PROGRAM=${0##*/}
IMAGE_PREFIX=ghcr.io/nationallibraryofnorway/veidemann

usage() {
  cat <<EOF
Usage: $PROGRAM [--write | --check] [COMPONENT ...]

Find the newest stable image versions represented by local Git tags.

With no COMPONENT arguments, all known Veidemann images are inspected.
By default, the command only reports available updates. Use --write to update
the workload YAML files, or --check to exit with status 1 when updates exist.

Only tags named <component>-vX.Y.Z are considered. This command never fetches
tags; run "git fetch --tags" yourself before invoking it when fresh tags are
required.

Options:
  --write  update image fields in deploy/k8s/base
  --check  report updates and exit with status 1 when any are available
  -h, --help
           show this help
EOF
}

components() {
  # CLI name | Git tag prefix | GHCR image name | comma-separated manifests
  cat <<'EOF'
cache|cache|cache|deploy/k8s/base/cache/deployment.yaml,deploy/k8s/base/cache/statefulset.yaml
contentwriter|contentwriter|contentwriter|deploy/k8s/base/contentwriter/deployment.yaml
controller|controller|controller|deploy/k8s/base/controller/deployment.yaml
dashboard|dashboard|dashboard|deploy/k8s/base/dashboard/deployment.yaml
dns-resolver|dns-resolver|dns-resolver|deploy/k8s/base/dns-resolver/deployment.yaml
fai|fai|fai|deploy/k8s/base/fai/deployment.yaml
frontier-queue-workers|frontier-queue-workers|frontier-queue-workers|deploy/k8s/base/frontier-queue-workers/deployment.yaml
frontier|frontier|frontier|deploy/k8s/base/frontier/deployment.yaml
browser-controller|browser-controller|browser-controller|deploy/k8s/base/harvester/deployment.yaml
recorderproxy|recorderproxy|recorderproxy|deploy/k8s/base/harvester/deployment.yaml
log-service|log-service|log-service|deploy/k8s/base/log-service/deployment.yaml
metrics|metrics|metrics|deploy/k8s/base/metrics/deployment.yaml
olricd|olricd|olricd|deploy/k8s/base/olric/statefulset.yaml
ooshandler|ooshandler|ooshandler|deploy/k8s/base/ooshandler/deployment.yaml
rethinkdb|rethinkdb|rethinkdb|deploy/k8s/base/rethinkdb/deployment.yaml,deploy/k8s/base/rethinkdb/statefulset.yaml
rethinkdbadapter|rethinkdbadapter|rethinkdbadapter|deploy/k8s/base/rethinkdbadapter/job.yaml
rethinkdb-ast-service|rethinkdb-ast-service|rethinkdb-ast-service|deploy/k8s/base/controller/deployment.yaml
robots-evaluator|robots-evaluator|robots-evaluator|deploy/k8s/base/robots/deployment.yaml
scopeservice|scopeservice|scopeservice|deploy/k8s/base/scopeservice/deployment.yaml
EOF
}

die() {
  printf '%s: %s\n' "$PROGRAM" "$*" >&2
  exit 2
}

command -v git >/dev/null 2>&1 || die "git is required"
command -v yq >/dev/null 2>&1 || die "yq is required"

mode=report
requested=
for argument in "$@"; do
  case $argument in
    --write)
      [ "$mode" = report ] || die "--write and --check cannot be combined"
      mode='write'
      ;;
    --check)
      [ "$mode" = report ] || die "--write and --check cannot be combined"
      mode=check
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --*) die "unknown option: $argument" ;;
    *) requested="${requested}${requested:+ }$argument" ;;
  esac
done

repo_root=$(git rev-parse --show-toplevel 2>/dev/null) || die "not inside a Git repository"
cd "$repo_root"

is_requested() {
  [ -z "$requested" ] && return 0
  case " $requested " in
    *" $1 "*) return 0 ;;
    *) return 1 ;;
  esac
}

for name in $requested; do
  components | awk -F '|' -v wanted="$name" '$1 == wanted { found = 1 } END { exit !found }' ||
    die "unknown component: $name"
done

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/update-image-tags.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM
plan_file=$work_dir/plan
: >"$plan_file"
updates=0

while IFS='|' read -r name tag_prefix image_name files; do
  is_requested "$name" || continue

  latest=$(git tag --list "${tag_prefix}-v*" |
    awk -v prefix="${tag_prefix}-v" 'index($0, prefix) == 1 { version = substr($0, length(prefix) + 1); if (version ~ /^[0-9]+\.[0-9]+\.[0-9]+$/) print version }' |
    sort -V | tail -n 1)
  [ -n "$latest" ] || die "no stable release tag found for $name (${tag_prefix}-vX.Y.Z)"

  image_repo=$IMAGE_PREFIX/$image_name
  old_ifs=$IFS
  IFS=,
  current_versions=
  for file in $files; do
    [ -f "$file" ] || die "mapped manifest does not exist: $file"
    matches=$(IMAGE_REPO="$image_repo" yq eval '.. | select(tag == "!!map" and has("image")) | .image | select(test("^" + strenv(IMAGE_REPO) + ":[^:]+$"))' "$file")
    count=$(printf '%s\n' "$matches" | awk 'NF { count++ } END { print count + 0 }')
    [ "$count" -eq 1 ] || die "expected exactly one $image_repo image in $file, found $count"
    current=${matches#"$image_repo:"}
    printf '%s\n' "$current" | awk '$0 ~ /^[0-9]+\.[0-9]+\.[0-9]+$/ { valid = 1 } END { exit !valid }' ||
      die "unexpected tag for $image_repo in $file: $current"
    newest=$(printf '%s\n%s\n' "$current" "$latest" | sort -V | tail -n 1)
    [ "$newest" = "$latest" ] ||
      die "manifest tag $current for $name is newer than local Git tag $latest; fetch tags before updating"
    case ",$current_versions," in
      *",$current,"*) ;;
      *) current_versions="${current_versions}${current_versions:+,}$current" ;;
    esac
  done
  IFS=$old_ifs

  status=current
  if [ "$current_versions" != "$latest" ]; then
    status='update available'
    updates=$((updates + 1))
  fi
  printf '%-24s current=%-15s latest=%-10s %s\n' "$name" "$current_versions" "$latest" "$status"
  printf '%s|%s|%s|%s|%s\n' "$name" "$image_repo" "$latest" "$files" "$status" >>"$plan_file"
done <<EOF
$(components)
EOF

if [ "$mode" = write ] && [ "$updates" -gt 0 ]; then
  staged_files=$work_dir/files
  mkdir -p "$staged_files"

  while IFS='|' read -r name image_repo latest files status; do
    [ "$status" = 'update available' ] || continue
    old_ifs=$IFS
    IFS=,
    for file in $files; do
      staged=$staged_files/$file
      mkdir -p "${staged%/*}"
      [ -f "$staged" ] || cp "$file" "$staged"
      IMAGE_REPO="$image_repo" NEW_TAG="$latest" yq eval -i \
        '(.. | select(tag == "!!map" and has("image")) | .image | select(test("^" + strenv(IMAGE_REPO) + ":[^:]+$"))) = strenv(IMAGE_REPO) + ":" + strenv(NEW_TAG)' \
        "$staged"
    done
    IFS=$old_ifs
  done <"$plan_file"

  find "$staged_files" -type f | while IFS= read -r staged; do
    file=${staged#"$staged_files/"}
    cp "$staged" "$file"
    printf 'updated %s\n' "$file"
  done
fi

if [ "$mode" = check ] && [ "$updates" -gt 0 ]; then
  exit 1
fi
