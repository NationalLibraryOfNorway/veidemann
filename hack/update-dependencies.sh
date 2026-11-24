#!/usr/bin/env bash

set -euo pipefail

usage() {
    cat <<EOF
Usage: ${0##*/} [--go-only | --java-only] [--patch] [--sync] [--dry-run]

Upgrade dependencies in every tracked Go module, including test and tool
dependencies, then run go mod tidy. Update the Gradle version catalog to
stable Java library and plugin releases. Leave all changes unstaged.

Options:
  --go-only   Update only the Go modules.
  --java-only Update only the Gradle version catalog.
  --patch    Use patch-only upgrades during go get.
  --sync     Align dependency versions through a temporary Go workspace.
  --dry-run  Print commands without running them.
  -h, --help Show this help.
EOF
}

dry_run=false
sync_workspace=false
upgrade=-u
language=all

while (($# > 0)); do
    case "$1" in
        --go-only | --java-only)
            if [[ $language != all && $language != "$1" ]]; then
                printf 'error: --go-only and --java-only are mutually exclusive\n' >&2
                exit 2
            fi
            language="$1"
            ;;
        --patch) upgrade=-u=patch ;;
        --sync) sync_workspace=true ;;
        --dry-run) dry_run=true ;;
        -h | --help)
            usage
            exit 0
            ;;
        *)
            printf 'error: unknown option: %s\n' "$1" >&2
            usage >&2
            exit 2
            ;;
    esac
    shift
done

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
repo_root="$(git -C "$script_dir" rev-parse --show-toplevel)"
cd -- "$repo_root"

run() {
    if "$dry_run"; then
        printf '  +'
        printf ' %q' "$@"
        printf '\n'
    else
        "$@"
    fi
}

sync_modules() (
    local workspace
    printf '\nSyncing dependency versions across Go modules\n'
    if "$dry_run"; then
        workspace='<temporary workspace>'
    else
        workspace="$(mktemp -d)" || exit 1
        trap 'rm -rf -- "$workspace"' EXIT
    fi
    run env GOWORK=off go -C "$workspace" work init "$@" &&
        run env GOWORK="$workspace/go.work" go -C "$workspace" work sync
)

update_modules() {
    local modfile module
    local -a packages failures=() modules=()

    while IFS= read -r -d '' modfile; do
        module="$(dirname -- "$modfile")"
        if [[ ! -f "$modfile" ]]; then
            printf 'error: tracked module file is missing: %s\n' "$modfile" >&2
            failures+=("$module")
            continue
        fi
        packages=(./...)
        if [[ $module == tools ]]; then
            packages=(tool)
        elif grep -Eq '^[[:space:]]*tool([[:space:]]|\()' "$modfile"; then
            packages+=(tool)
        fi
        modules+=("$repo_root/$module")

        printf 'Updating %s\n' "$module"
        if ! run env GOWORK=off go -C "$module" get -t "$upgrade" "${packages[@]}" ||
            ! run env GOWORK=off go -C "$module" mod tidy; then
            failures+=("$module")
        fi
    done

    if ((${#failures[@]} > 0)); then
        printf '\nDependency update failed in:\n' >&2
        printf '  %s\n' "${failures[@]}" >&2
        return 1
    fi
    if "$sync_workspace" && ((${#modules[@]} > 0)); then
        sync_modules "${modules[@]}"
    fi
}

status=0
if [[ $language != --java-only ]]; then
    if ! git ls-files --deduplicate -z -- go.mod ':(glob)**/go.mod' | update_modules; then
        status=1
    fi
fi
if [[ $language != --go-only ]]; then
    printf '\nUpdating the Gradle version catalog\n'
    if ! run ./gradlew versionCatalogUpdate; then
        printf 'error: Gradle version catalog update failed\n' >&2
        status=1
    fi
fi
exit "$status"
