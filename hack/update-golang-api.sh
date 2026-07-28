#!/usr/bin/env bash

set -Eeuo pipefail

readonly API_MODULE='github.com/NationalLibraryOfNorway/veidemann/api'

usage() {
    cat <<EOF
Usage: $(basename "$0") [--dry-run] VERSION

Update all Go modules in the repository that depend on:
  $API_MODULE

Examples:
  $(basename "$0") v0.2.0
  $(basename "$0") --dry-run v0.2.0
EOF
}

die() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

dry_run=false

case "${1:-}" in
    --dry-run)
        dry_run=true
        shift
        ;;
    -h|--help)
        usage
        exit 0
        ;;
esac

[[ $# -eq 1 ]] || {
    usage >&2
    exit 2
}

readonly api_version="$1"

[[ "$api_version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]] ||
    die "invalid version: $api_version"

readonly script_dir="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &&
        pwd -P
)"
readonly repo_root="$(
    git -C "$script_dir" rev-parse --show-toplevel
)"

run() {
    if "$dry_run"; then
        printf '  +'
        printf ' %q' "$@"
        printf '\n'
    else
        "$@"
    fi
}

module_depends_on_api() {
    local dir="$1"

    GOWORK=off go -C "$dir" mod edit -json |
        grep -Fq "\"Path\": \"$API_MODULE\""
}

printf 'API version: %s@%s\n' "$API_MODULE" "$api_version"

updated=0

while IFS= read -r -d '' modfile; do
    dir="${modfile%/go.mod}"

    # Do not update the API module itself or the isolated tools module.
    case "$dir" in
        "$repo_root/api"|"$repo_root/tools")
            continue
            ;;
    esac

    if ! module_depends_on_api "$dir"; then
        continue
    fi

    relative_dir="${dir#"$repo_root"/}"
    printf '\nUpdating %s\n' "$relative_dir"

    run env GOWORK=off \
        go -C "$dir" get "$API_MODULE@$api_version"

    run env GOWORK=off \
        go -C "$dir" mod tidy

    ((updated += 1))
done < <(
    find "$repo_root" \
        \( -type d \( \
            -name .git -o \
            -name vendor -o \
            -name node_modules \
        \) -prune \) -o \
        \( -type f -name go.mod -print0 \)
)

printf '\n%s %d module(s).\n' \
    "$("$dry_run" && printf 'Would update' || printf 'Updated')" \
    "$updated"
