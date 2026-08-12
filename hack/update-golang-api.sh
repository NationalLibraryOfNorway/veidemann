#!/usr/bin/env bash

set -Eeuo pipefail

readonly MODULE_PREFIX='github.com/NationalLibraryOfNorway/veidemann'
readonly API_MODULE="$MODULE_PREFIX/api"
readonly DEFAULT_WAIT_TIMEOUT_SECONDS=600
readonly RETRY_INTERVAL_SECONDS=10

usage() {
    cat <<EOF
Usage: $(basename "$0") [options] [API_VERSION]

Update repository Go modules in release order. API_VERSION updates every direct
consumer of:
  $API_MODULE

After API-consuming library modules have been committed and published with Go
module tags, use --module to update their direct consumers.

Options:
  --module MODULE@VERSION  Update direct consumers of a published repository
                           module. May be specified more than once. MODULE may
                           be a short name such as log-service or a full path.
  --wait-timeout SECONDS   Maximum time to wait for each version to become
                           available through the configured Go module proxy.
                           Default: $DEFAULT_WAIT_TIMEOUT_SECONDS
  --dry-run                Print commands without waiting or changing files.
  -h, --help               Show this help.

At least API_VERSION or one --module option is required.

Examples:
  # Stage 1: after pushing api/v1.4.0
  $(basename "$0") v1.4.0

  # Stage 2: after committing stage 1 and pushing these slash-tags
  $(basename "$0") \\
      --module log-service@v0.8.2 \\
      --module recorderproxy@v0.9.4

  $(basename "$0") --dry-run v1.4.0
EOF
}

die() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

valid_version() {
    [[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]
}

dry_run=false
wait_timeout_seconds=$DEFAULT_WAIT_TIMEOUT_SECONDS
api_version=''
declare -a requested_module_specs=()

while (($# > 0)); do
    case "$1" in
        --dry-run)
            dry_run=true
            shift
            ;;
        --module)
            (($# >= 2)) || die "--module requires MODULE@VERSION"
            requested_module_specs+=("$2")
            shift 2
            ;;
        --wait-timeout)
            (($# >= 2)) || die "--wait-timeout requires a number of seconds"
            wait_timeout_seconds="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        --)
            shift
            if (($# > 0)); then
                [[ -z "$api_version" ]] || die "API_VERSION was specified more than once"
                api_version="$1"
                shift
            fi
            (($# == 0)) || die "unexpected arguments: $*"
            ;;
        -*)
            die "unknown option: $1"
            ;;
        *)
            [[ -z "$api_version" ]] || die "API_VERSION was specified more than once"
            api_version="$1"
            shift
            ;;
    esac
done

if ! [[ "$wait_timeout_seconds" =~ ^[0-9]+$ ]] || ((wait_timeout_seconds <= 0)); then
    die "invalid wait timeout: $wait_timeout_seconds"
fi

if [[ -n "$api_version" ]] && ! valid_version "$api_version"; then
    die "invalid API version: $api_version"
fi

if [[ -z "$api_version" ]] && ((${#requested_module_specs[@]} == 0)); then
    usage >&2
    exit 2
fi

script_dir="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &&
        pwd -P
)"
readonly script_dir

repo_root="$(
    git -C "$script_dir" rev-parse --show-toplevel
)"
readonly repo_root

run() {
    if "$dry_run"; then
        printf '  +'
        printf ' %q' "$@"
        printf '\n'
    else
        "$@"
    fi
}

normalize_module() {
    local module="$1"

    if [[ "$module" != "$MODULE_PREFIX/"* ]]; then
        module="$MODULE_PREFIX/$module"
    fi
    printf '%s\n' "$module"
}

validate_repo_module() {
    local module="$1"
    local relative_dir declared_module

    [[ "$module" == "$MODULE_PREFIX/"* ]] ||
        die "module is outside this repository: $module"

    relative_dir="${module#"$MODULE_PREFIX/"}"
    [[ -f "$repo_root/$relative_dir/go.mod" ]] ||
        die "repository module not found: $module"

    declared_module="$(awk '$1 == "module" { print $2; exit }' "$repo_root/$relative_dir/go.mod")"
    [[ "$declared_module" == "$module" ]] ||
        die "module path mismatch in $relative_dir/go.mod: $declared_module"
}

module_depends_on() {
    local dir="$1"
    local dependency="$2"
    local module_json

    module_json="$(GOWORK=off go -C "$dir" mod edit -json)"
    grep -Fq "\"Path\": \"$dependency\"" <<<"$module_json"
}

wait_for_module_version() {
    local module="$1"
    local version="$2"
    local spec="$module@$version"
    local started elapsed remaining delay output

    printf 'Checking Go module proxy for %s\n' "$spec"

    if "$dry_run"; then
        run env GOWORK=off \
            go -C "$repo_root/tools" list -m "$spec"
        return
    fi

    started=$SECONDS
    while true; do
        if output="$(
            env GOWORK=off \
                go -C "$repo_root/tools" list -m "$spec" 2>&1
        )"; then
            printf '  available: %s\n' "$output"
            return
        fi

        elapsed=$((SECONDS - started))
        if ((elapsed >= wait_timeout_seconds)); then
            printf '%s\n' "$output" >&2
            die "$spec was not available after ${wait_timeout_seconds}s"
        fi

        remaining=$((wait_timeout_seconds - elapsed))
        delay=$RETRY_INTERVAL_SECONDS
        if ((delay > remaining)); then
            delay=$remaining
        fi

        printf '  not available yet; retrying in %ss (%ss remaining)\n' \
            "$delay" "$remaining" >&2
        sleep "$delay"
    done
}

update_consumers() {
    local dependency="$1"
    local version="$2"
    local dependency_dir modfile dir relative_dir
    local updated=0

    printf '\nDependency: %s@%s\n' "$dependency" "$version"
    wait_for_module_version "$dependency" "$version"

    dependency_dir="$repo_root/${dependency#"$MODULE_PREFIX/"}"

    while IFS= read -r -d '' modfile; do
        dir="${modfile%/go.mod}"

        # go mod edit -json includes the module's own Path as well as requires.
        [[ "$dir" != "$dependency_dir" ]] || continue

        if ! module_depends_on "$dir" "$dependency"; then
            continue
        fi

        relative_dir="${dir#"$repo_root/"}"
        printf '\nUpdating %s\n' "$relative_dir"

        run env GOWORK=off \
            go -C "$dir" get "$dependency@$version"

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
            \( -type f -name go.mod -print0 \) |
            sort -z
    )

    printf '\n%s %d direct consumer(s) of %s.\n' \
        "$("$dry_run" && printf 'Would update' || printf 'Updated')" \
        "$updated" \
        "$dependency"
}

declare -a dependency_modules=()
declare -a dependency_versions=()

add_dependency() {
    local module="$1"
    local version="$2"
    local existing

    validate_repo_module "$module"
    valid_version "$version" || die "invalid version for $module: $version"

    for existing in "${dependency_modules[@]}"; do
        [[ "$existing" != "$module" ]] ||
            die "module specified more than once: $module"
    done

    dependency_modules+=("$module")
    dependency_versions+=("$version")
}

if [[ -n "$api_version" ]]; then
    add_dependency "$API_MODULE" "$api_version"
fi

for spec in "${requested_module_specs[@]}"; do
    [[ "$spec" == *@* && "$spec" != @* && "$spec" != *@ ]] ||
        die "invalid module specification: $spec (expected MODULE@VERSION)"

    module="$(normalize_module "${spec%@*}")"
    version="${spec##*@}"
    add_dependency "$module" "$version"
done

for index in "${!dependency_modules[@]}"; do
    update_consumers \
        "${dependency_modules[$index]}" \
        "${dependency_versions[$index]}"
done
