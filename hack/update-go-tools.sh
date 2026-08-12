#!/usr/bin/env bash

set -Eeuo pipefail

usage() {
    cat <<EOF
Usage: $(basename "$0") [options] [TOOL[@VERSION] ...]

Update tools declared in tools/go.mod. With no TOOL arguments, every declared
tool is updated to its latest version. A TOOL without a VERSION also uses latest.

Options:
  --dry-run  Print commands without changing files.
  -h, --help Show this help.

Examples:
  $(basename "$0")
  $(basename "$0") github.com/bufbuild/buf/cmd/buf@v1.73.0
  $(basename "$0") --dry-run
EOF
}

die() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

dry_run=false
declare -a requested_specs=()

while (($# > 0)); do
    case "$1" in
        --dry-run)
            dry_run=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        --)
            shift
            requested_specs+=("$@")
            break
            ;;
        -*)
            die "unknown option: $1"
            ;;
        *)
            requested_specs+=("$1")
            shift
            ;;
    esac
done

script_dir="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &&
        pwd -P
)"
readonly script_dir

repo_root="$(
    git -C "$script_dir" rev-parse --show-toplevel
)"
readonly repo_root
readonly tools_dir="$repo_root/tools"

run() {
    if "$dry_run"; then
        printf '  +'
        printf ' %q' "$@"
        printf '\n'
    else
        "$@"
    fi
}

configured_plugin_version() {
    local plugin="$1"

    awk -v plugin="$plugin" '
        $1 == "-" && $2 == "remote:" {
            spec = $3
            prefix = plugin ":"
            if (spec == plugin) {
                print "unversioned"
                exit
            }
            if (substr(spec, 1, length(prefix)) == prefix) {
                print substr(spec, length(prefix) + 1)
                exit
            }
        }
    ' "$repo_root/buf.gen.go.yaml"
}

print_remote_plugin_versions() {
    local index plugin module pinned latest
    local -a plugins=(
        buf.build/protocolbuffers/go
        buf.build/grpc/go
    )
    local -a modules=(
        google.golang.org/protobuf
        google.golang.org/grpc/cmd/protoc-gen-go-grpc
    )

    printf '\nRemote protobuf generators (advisory; pins are unchanged):\n'
    for index in "${!plugins[@]}"; do
        plugin="${plugins[$index]}"
        module="${modules[$index]}"
        pinned="$(configured_plugin_version "$plugin")"
        [[ -n "$pinned" ]] || pinned="not configured"

        if latest="$({
            env GOWORK=off go -C "$tools_dir" list -m -f '{{.Version}}' "$module@latest"
        } 2>/dev/null)" && [[ -n "$latest" ]]; then
            printf '  %s: pinned %s, latest upstream %s\n' "$plugin" "$pinned" "$latest"
        else
            printf '  %s: pinned %s, latest upstream unavailable\n' "$plugin" "$pinned"
            printf 'warning: could not query the latest version of %s\n' "$module" >&2
        fi
    done
}

declared_output="$({
    env GOWORK=off go -C "$tools_dir" list tool
} 2>&1)" || die "could not read tools/go.mod: $declared_output"

declare -a declared_tools=()
while IFS= read -r tool; do
    [[ -n "$tool" ]] && declared_tools+=("$tool")
done <<<"$declared_output"

((${#declared_tools[@]} > 0)) || die "tools/go.mod does not declare any tools"

is_declared_tool() {
    local requested_tool="$1"
    local declared_tool

    for declared_tool in "${declared_tools[@]}"; do
        [[ "$requested_tool" != "$declared_tool" ]] || return 0
    done
    return 1
}

declare -a update_specs=()
declare -a selected_tools=()

add_tool() {
    local spec="$1"
    local tool version selected_tool

    if [[ "$spec" == *@* ]]; then
        tool="${spec%@*}"
        version="${spec##*@}"
        [[ -n "$tool" && -n "$version" ]] || die "invalid tool specification: $spec"
    else
        tool="$spec"
        version=latest
    fi

    is_declared_tool "$tool" || die "tool is not declared in tools/go.mod: $tool"

    for selected_tool in "${selected_tools[@]}"; do
        [[ "$selected_tool" != "$tool" ]] || die "tool specified more than once: $tool"
    done

    selected_tools+=("$tool")
    update_specs+=("$tool@$version")
}

if ((${#requested_specs[@]} == 0)); then
    for tool in "${declared_tools[@]}"; do
        add_tool "$tool"
    done
else
    for spec in "${requested_specs[@]}"; do
        add_tool "$spec"
    done
fi

printf 'Updating Go tools:\n'
printf '  %s\n' "${update_specs[@]}"

run env GOWORK=off \
    go -C "$tools_dir" get -tool "${update_specs[@]}"

run env GOWORK=off \
    go -C "$tools_dir" mod tidy

print_remote_plugin_versions

printf '\nReview tools/go.mod and tools/go.sum.\n'
printf 'Remote plugin pins in buf.gen.go.yaml are not updated by this command.\n'
printf 'Then regenerate the Go API:\n'
printf '  cd %q && go generate ./...\n' "$repo_root/api"
