#!/usr/bin/env bash

set -Eeuo pipefail

readonly MODULE_PREFIX='github.com/NationalLibraryOfNorway/veidemann'
readonly API_MODULE="$MODULE_PREFIX/api"
readonly DEFAULT_REMOTE='origin'
readonly DEFAULT_WAIT_TIMEOUT_SECONDS=600

bump_kind=''
explicit_version=''
explicit_commit=''
remote="$DEFAULT_REMOTE"
wait_timeout_seconds=$DEFAULT_WAIT_TIMEOUT_SECONDS
dry_run=false
release_version=''
api_tag_commit=''

declare -a module_names=()
declare -A module_dirs=()
declare -A module_dependencies=()
declare -A module_consumers=()
declare -A reachable=()
declare -A visit_state=()
declare -A published_versions=()
declare -a summary_tags=()
declare -a summary_commits=()
declare -a summary_leaves=()
declare -A summary_leaf_seen=()

usage() {
    cat <<EOF
Usage: $(basename "$0") (--bump patch|minor|major | --version vX.Y.Z) [options]

Regenerate and validate the Go API, publish its Go module tag, update all
repository consumers in dependency order, and publish patch releases for
intermediate Go modules.

Options:
  --bump TYPE              Bump the API by patch, minor, or major.
  --version VERSION        Release or resume an exact API version.
  --commit REF             Commit to tag. Default: latest change under api/.
  --remote REMOTE          Git remote to push. Default: $DEFAULT_REMOTE
  --wait-timeout SECONDS   Proxy wait per module. Default: $DEFAULT_WAIT_TIMEOUT_SECONDS
  --dry-run                Show the release graph and commands without changes.
  -h, --help               Show this help.

Examples:
  $(basename "$0") --bump minor
  $(basename "$0") --bump minor --commit 0123456789abcdef
  $(basename "$0") --version v1.4.0
  $(basename "$0") --bump patch --remote origin --wait-timeout 900
  $(basename "$0") --bump minor --dry-run
EOF
}

die() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

on_exit() {
    local status=$?

    if ((status != 0)) && [[ -n "$release_version" ]] && ! "$dry_run"; then
        printf '\nRelease stopped after version selection. After resolving the error, resume with:\n' >&2
        printf '  %q --version %q' \
            "$repo_root/hack/update-api.sh" \
            "$release_version" >&2
        if [[ -n "$explicit_commit" ]]; then
            printf ' --commit %q' "$explicit_commit" >&2
        fi
        printf ' --remote %q --wait-timeout %q\n' \
            "$remote" \
            "$wait_timeout_seconds" >&2
    fi
}
trap on_exit EXIT

require_command() {
    command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

valid_version() {
    [[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

normalize_version() {
    local version="$1"
    version="${version#v}"
    valid_version "v$version" || die "invalid semantic version: $1"
    printf 'v%s\n' "$version"
}

bump_version() {
    local version="${1#v}"
    local kind="$2"
    local major minor patch

    IFS='.' read -r major minor patch <<<"$version"
    case "$kind" in
        major)
            ((major += 1))
            minor=0
            patch=0
            ;;
        minor)
            ((minor += 1))
            patch=0
            ;;
        patch)
            ((patch += 1))
            ;;
        *)
            die "invalid bump type: $kind"
            ;;
    esac
    printf 'v%d.%d.%d\n' "$major" "$minor" "$patch"
}

version_greater_than() {
    local left="${1#v}"
    local right="${2#v}"
    local left_major left_minor left_patch
    local right_major right_minor right_patch

    IFS='.' read -r left_major left_minor left_patch <<<"$left"
    IFS='.' read -r right_major right_minor right_patch <<<"$right"
    ((left_major > right_major)) ||
        ((left_major == right_major && left_minor > right_minor)) ||
        ((left_major == right_major && left_minor == right_minor && left_patch > right_patch))
}

while (($# > 0)); do
    case "$1" in
        --bump)
            (($# >= 2)) || die "--bump requires patch, minor, or major"
            [[ -z "$bump_kind" ]] || die "--bump was specified more than once"
            bump_kind="$2"
            shift 2
            ;;
        --version)
            (($# >= 2)) || die "--version requires vX.Y.Z"
            [[ -z "$explicit_version" ]] || die "--version was specified more than once"
            explicit_version="$(normalize_version "$2")"
            shift 2
            ;;
        --commit)
            (($# >= 2)) || die "--commit requires a commit or ref"
            [[ -z "$explicit_commit" ]] || die "--commit was specified more than once"
            [[ -n "$2" ]] || die "--commit requires a non-empty commit or ref"
            explicit_commit="$2"
            shift 2
            ;;
        --remote)
            (($# >= 2)) || die "--remote requires a remote name"
            remote="$2"
            shift 2
            ;;
        --wait-timeout)
            (($# >= 2)) || die "--wait-timeout requires seconds"
            wait_timeout_seconds="$2"
            shift 2
            ;;
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
            (($# == 0)) || die "unexpected arguments: $*"
            ;;
        *)
            die "unknown argument: $1"
            ;;
    esac
done

[[ -z "$bump_kind" || -z "$explicit_version" ]] ||
    die "--bump and --version are mutually exclusive"
[[ -n "$bump_kind" || -n "$explicit_version" ]] ||
    die "one of --bump or --version is required"
[[ -z "$bump_kind" || "$bump_kind" =~ ^(patch|minor|major)$ ]] ||
    die "--bump must be patch, minor, or major"
if ! [[ "$wait_timeout_seconds" =~ ^[0-9]+$ ]] || ((wait_timeout_seconds <= 0)); then
    die "invalid wait timeout: $wait_timeout_seconds"
fi
[[ "$remote" =~ ^[A-Za-z0-9._-]+$ ]] || die "invalid remote name: $remote"

script_dir="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &&
        pwd -P
)"
readonly script_dir

repo_root="$(git -C "$script_dir" rev-parse --show-toplevel)"
readonly repo_root
readonly update_helper="$repo_root/hack/update-golang-api.sh"

run() {
    if "$dry_run"; then
        printf '  +'
        printf ' %q' "$@"
        printf '\n'
    else
        "$@"
    fi
}

stable_tags() {
    local module_dir="$1"
    local tag

    while IFS= read -r tag; do
        [[ "$tag" =~ ^${module_dir//\//\\/}/v[0-9]+\.[0-9]+\.[0-9]+$ ]] &&
            printf '%s\n' "$tag"
    done < <(git tag --list "$module_dir/v*" --sort=-version:refname)
    return 0
}

latest_stable_tag() {
    local -a tags=()

    mapfile -t tags < <(stable_tags "$1")
    ((${#tags[@]} == 0)) || printf '%s\n' "${tags[0]}"
}

module_version_from_tag() {
    local module_dir="$1"
    local tag="$2"
    printf 'v%s\n' "${tag#"$module_dir/v"}"
}

discover_modules() {
    local modfile dir module dependency json

    while IFS= read -r -d '' modfile; do
        dir="${modfile%/go.mod}"
        json="$(env GOWORK=off go -C "$dir" mod edit -json)"
        module="$(jq -r '.Module.Path' <<<"$json")"
        [[ "$module" == "$MODULE_PREFIX/"* ]] || continue
        [[ -z "${module_dirs[$module]+x}" ]] || die "duplicate module path: $module"

        module_names+=("$module")
        module_dirs["$module"]="${dir#"$repo_root/"}"
        module_dependencies["$module"]=''

        while IFS= read -r dependency; do
            [[ "$dependency" == "$MODULE_PREFIX/"* ]] || continue
            module_dependencies["$module"]+=" $dependency"
            module_consumers["$dependency"]+=" $module"
        done < <(jq -r '.Require[]?.Path' <<<"$json")
    done < <(
        find "$repo_root" \
            \( -type d \( -name .git -o -name vendor -o -name node_modules \) -prune \) -o \
            \( -type f -name go.mod -print0 \) |
            sort -z
    )

    [[ -n "${module_dirs[$API_MODULE]+x}" ]] || die "API module not found"

    for module in "${module_names[@]}"; do
        for dependency in ${module_dependencies[$module]-}; do
            [[ -n "${module_dirs[$dependency]+x}" ]] ||
                die "$module depends on repository module not present in this checkout: $dependency"
        done
    done
}

visit_module() {
    local module="$1"
    local dependency

    case "${visit_state[$module]-}" in
        visiting)
            die "repository Go module dependency cycle includes $module"
            ;;
        visited)
            return
            ;;
    esac

    visit_state["$module"]='visiting'
    for dependency in ${module_dependencies[$module]-}; do
        visit_module "$dependency"
    done
    visit_state["$module"]='visited'
}

mark_reachable() {
    local dependency="$1"
    local consumer

    [[ -z "${reachable[$dependency]+x}" ]] || return 0
    reachable["$dependency"]=1
    for consumer in ${module_consumers[$dependency]-}; do
        mark_reachable "$consumer"
    done
}

validate_graph() {
    local module

    for module in "${module_names[@]}"; do
        visit_module "$module"
    done
    mark_reachable "$API_MODULE"
}

preflight() {
    local upstream expected_upstream counts behind

    require_command git
    require_command go
    require_command jq
    [[ -x "$update_helper" ]] || die "update helper is not executable: $update_helper"
    git -C "$repo_root" remote get-url "$remote" >/dev/null 2>&1 ||
        die "unknown Git remote: $remote"
    [[ -z "$(git -C "$repo_root" status --porcelain)" ]] ||
        die "the worktree must be clean"
    [[ -n "$(git -C "$repo_root" config user.name)" ]] || die "Git user.name is not configured"
    [[ -n "$(git -C "$repo_root" config user.email)" ]] || die "Git user.email is not configured"

    branch="$(git -C "$repo_root" symbolic-ref --quiet --short HEAD)" ||
        die "HEAD is detached"
    readonly branch
    upstream="$(git -C "$repo_root" rev-parse --abbrev-ref --symbolic-full-name '@{upstream}')" ||
        die "branch $branch has no upstream"
    expected_upstream="$remote/$branch"
    [[ "$upstream" == "$expected_upstream" ]] ||
        die "branch upstream is $upstream; expected $expected_upstream"

    if "$dry_run"; then
        run git -C "$repo_root" fetch "$remote" "$branch" --tags
    else
        git -C "$repo_root" fetch "$remote" "$branch" --tags
    fi

    counts="$(git -C "$repo_root" rev-list --left-right --count "$upstream...HEAD")"
    read -r behind _ <<<"$counts"
    ((behind == 0)) || die "branch is behind or diverged from $upstream"

    run git -C "$repo_root" push --dry-run --atomic "$remote" "HEAD:refs/heads/$branch"
}

select_api_version() {
    local previous_tag previous_version proposed_tag existing_tag target_description
    local requested_commit

    previous_tag="$(latest_stable_tag api)"
    [[ -n "$previous_tag" ]] || die "no stable api/vX.Y.Z tag found"
    git merge-base --is-ancestor "${previous_tag}^{commit}" HEAD ||
        die "latest API tag is not reachable from HEAD: $previous_tag"

    if [[ -n "$explicit_version" ]]; then
        release_version="$explicit_version"
    else
        previous_version="$(module_version_from_tag api "$previous_tag")"
        release_version="$(bump_version "$previous_version" "$bump_kind")"
    fi

    proposed_tag="api/$release_version"
    if git rev-parse --verify --quiet "refs/tags/$proposed_tag" >/dev/null; then
        [[ -n "$explicit_version" ]] || die "tag already exists: $proposed_tag"
        git merge-base --is-ancestor "${proposed_tag}^{commit}" HEAD ||
            die "$proposed_tag is not reachable from HEAD"
        git diff --quiet "${proposed_tag}^{commit}" HEAD -- api ||
            die "current API contents differ from existing $proposed_tag"
        api_tag_commit="$(git rev-parse "${proposed_tag}^{commit}")"
        if [[ -n "$explicit_commit" ]]; then
            requested_commit="$(git rev-parse --verify --quiet --end-of-options \
                "${explicit_commit}^{commit}")" ||
                die "cannot resolve --commit ref: $explicit_commit"
            [[ "$requested_commit" == "$api_tag_commit" ]] ||
                die "--commit $explicit_commit does not match existing tag $proposed_tag"
        fi
        target_description="existing tag $proposed_tag"
        existing_tag=true
    else
        existing_tag=false
        previous_version="$(module_version_from_tag api "$previous_tag")"
        version_greater_than "$release_version" "$previous_version" ||
            die "$release_version must be newer than $previous_version"

        if [[ -n "$explicit_commit" ]]; then
            api_tag_commit="$(git rev-parse --verify --quiet --end-of-options \
                "${explicit_commit}^{commit}")" ||
                die "cannot resolve --commit ref: $explicit_commit"
            target_description="--commit $explicit_commit"
        else
            api_tag_commit="$(git log -1 --format='%H' HEAD -- api)"
            [[ -n "$api_tag_commit" ]] || die "no commit found for api/"
            target_description='latest change under api/'
        fi

        git merge-base --is-ancestor "$api_tag_commit" HEAD ||
            die "API tag commit is not reachable from HEAD: $api_tag_commit"
        git merge-base --is-ancestor "${previous_tag}^{commit}" "$api_tag_commit" ||
            die "API tag commit predates the latest API tag: $previous_tag"
        git diff --quiet "$api_tag_commit" HEAD -- api ||
            die "API contents at tag commit differ from HEAD: $api_tag_commit"
        git diff --quiet "${previous_tag}^{commit}" "$api_tag_commit" -- api &&
            die "no API changes since $previous_tag"
    fi

    printf 'API version: %s (%s)\n' "$release_version" \
        "$([[ "$existing_tag" == true ]] && printf 'resume' || printf 'new release')"
    printf 'API tag commit: %s (%s)\n' "${api_tag_commit:0:12}" "$target_description"
}

verify_generated_api() {
    local before

    printf '\nRegenerating and testing the Go API\n'
    before="$(git -C "$repo_root" rev-parse HEAD)"
    run env GOWORK=off go -C "$repo_root/api" generate ./...
    run env GOWORK=off go -C "$repo_root/api" test ./...

    "$dry_run" && return
    [[ "$before" == "$(git -C "$repo_root" rev-parse HEAD)" ]] ||
        die "HEAD changed while validating the API"
    if [[ -n "$(git -C "$repo_root" status --porcelain)" ]]; then
        die "API regeneration changed tracked files; review and commit them before releasing"
    fi
}

remote_tag_state() {
    local tag="$1"
    local local_oid remote_oid

    local_oid="$(git -C "$repo_root" rev-parse "refs/tags/$tag")"
    remote_oid="$(git -C "$repo_root" ls-remote --refs "$remote" "refs/tags/$tag" | awk 'NR == 1 { print $1 }')"
    if [[ -z "$remote_oid" ]]; then
        printf 'missing\n'
    elif [[ "$remote_oid" == "$local_oid" ]]; then
        printf 'published\n'
    else
        die "remote tag conflicts with local tag: $tag"
    fi
}

push_stage() {
    local tag state
    local -a refspecs=("HEAD:refs/heads/$branch")

    for tag in "$@"; do
        state="$(remote_tag_state "$tag")"
        [[ "$state" == published ]] || refspecs+=("refs/tags/$tag:refs/tags/$tag")
    done

    if "$dry_run"; then
        run git -C "$repo_root" push --atomic "$remote" "${refspecs[@]}"
        return
    fi

    if ((${#refspecs[@]} == 1)) &&
        [[ "$(git -C "$repo_root" rev-parse "$remote/$branch")" == \
            "$(git -C "$repo_root" rev-parse HEAD)" ]]; then
        return
    fi

    git -C "$repo_root" push --atomic "$remote" "${refspecs[@]}"
}

publish_api() {
    local tag="api/$release_version"

    if ! git -C "$repo_root" rev-parse --verify --quiet "refs/tags/$tag" >/dev/null; then
        run git -C "$repo_root" tag --annotate "$tag" \
            --message "Release API version ${release_version#v}" "$api_tag_commit"
    fi
    summary_tags+=("$tag")
    push_stage "$tag"
    published_versions["$API_MODULE"]="$release_version"
}

current_required_version() {
    local module="$1"
    local dependency="$2"
    local dir="${module_dirs[$module]}"

    awk -v dependency="$dependency" '
        $1 == "require" && $2 == dependency { print $3; exit }
        $1 == dependency { print $2; exit }
    ' "$repo_root/$dir/go.mod"
}

tag_matches_current_module() {
    local module="$1"
    local tag="$2"
    local dependency expected actual dir

    dir="${module_dirs[$module]}"
    git merge-base --is-ancestor "${tag}^{commit}" HEAD || return 1
    git diff --quiet "${tag}^{commit}" HEAD -- "$dir" || return 1

    for dependency in ${module_dependencies[$module]-}; do
        [[ -n "${reachable[$dependency]+x}" ]] || continue
        expected="${published_versions[$dependency]-}"
        [[ -n "$expected" ]] || return 1
        actual="$(current_required_version "$module" "$dependency")"
        [[ "$actual" == "$expected" ]] || return 1
    done
}

matching_module_tag() {
    local module="$1"
    local dir="${module_dirs[$module]}"
    local tag
    local -a tags=()

    mapfile -t tags < <(stable_tags "$dir")
    for tag in "${tags[@]}"; do
        if tag_matches_current_module "$module" "$tag"; then
            printf '%s\n' "$tag"
            return
        fi
    done
    return 0
}

module_ready_to_publish() {
    local module="$1"
    local dependency expected actual

    for dependency in ${module_dependencies[$module]-}; do
        [[ -n "${reachable[$dependency]+x}" ]] || continue
        expected="${published_versions[$dependency]-}"
        [[ -n "$expected" ]] || return 1
        actual="$(current_required_version "$module" "$dependency")"
        [[ "$actual" == "$expected" ]] || return 1
    done
}

select_intermediate_tag() {
    local module="$1"
    local dir="${module_dirs[$module]}"
    local matching latest previous version tag

    matching="$(matching_module_tag "$module")"
    if [[ -n "$matching" ]]; then
        printf '%s\n' "$matching"
        return
    fi

    latest="$(latest_stable_tag "$dir")"
    if [[ -n "$latest" ]]; then
        previous="$(module_version_from_tag "$dir" "$latest")"
        version="$(bump_version "$previous" patch)"
    else
        version='v0.1.0'
    fi
    tag="$dir/$version"
    git -C "$repo_root" rev-parse --verify --quiet "refs/tags/$tag" >/dev/null &&
        die "cannot create existing tag: $tag"
    run git -C "$repo_root" tag --annotate "$tag" \
        --message "Release $dir version ${version#v}" HEAD
    printf '%s\n' "$tag"
}

allowed_changes_only() {
    local module path
    declare -A allowed=()

    for module in "$@"; do
        allowed["${module_dirs[$module]}/go.mod"]=1
        allowed["${module_dirs[$module]}/go.sum"]=1
    done

    while IFS= read -r -d '' path; do
        [[ -n "${allowed[$path]+x}" ]] || die "unexpected file changed by dependency update: $path"
    done < <(
        {
            git -C "$repo_root" diff --name-only -z
            git -C "$repo_root" diff --cached --name-only -z
            git -C "$repo_root" ls-files --others --exclude-standard -z
        }
    )
}

changed_modules() {
    local module

    for module in "$@"; do
        if ! git -C "$repo_root" diff --quiet -- \
            "${module_dirs[$module]}/go.mod" "${module_dirs[$module]}/go.sum"; then
            printf '%s\n' "$module"
        fi
    done
}

dependency_summary() {
    local module version output=''

    for module in "$@"; do
        version="${published_versions[$module]}"
        [[ -z "$output" ]] || output+=', '
        output+="${module#"$MODULE_PREFIX/"} $version"
    done
    printf '%s\n' "$output"
}

update_wave() {
    local -a dependencies=("$@")
    local dependency consumer module tag version message
    local -a expected=()
    local -a changed=()
    local -a intermediate=()
    local -a wave_tags=()
    local -a next_dependencies=()
    local -a helper_args=(--wait-timeout "$wait_timeout_seconds")
    declare -A expected_set=()
    declare -A changed_set=()

    for dependency in "${dependencies[@]}"; do
        if [[ "$dependency" == "$API_MODULE" ]]; then
            helper_args+=("${published_versions[$dependency]}")
        else
            helper_args+=(--module "${dependency#"$MODULE_PREFIX/"}@${published_versions[$dependency]}")
        fi
        for consumer in ${module_consumers[$dependency]-}; do
            [[ -n "${reachable[$consumer]+x}" ]] || continue
            expected_set["$consumer"]=1
        done
    done

    mapfile -t expected < <(printf '%s\n' "${!expected_set[@]}" | sort)
    ((${#expected[@]} > 0)) || return 0

    printf '\nUpdating consumers of %s\n' "$(dependency_summary "${dependencies[@]}")"
    run "$update_helper" "${helper_args[@]}"
    "$dry_run" && return

    allowed_changes_only "${expected[@]}"
    mapfile -t changed < <(changed_modules "${expected[@]}")
    for module in "${changed[@]}"; do
        changed_set["$module"]=1
    done

    for module in "${changed[@]}"; do
        printf 'Testing %s\n' "${module_dirs[$module]}"
        env GOWORK=off go -C "$repo_root/${module_dirs[$module]}" test ./...
    done
    allowed_changes_only "${expected[@]}"

    if ((${#changed[@]} > 0)); then
        for module in "${changed[@]}"; do
            git -C "$repo_root" add -- \
                "${module_dirs[$module]}/go.mod" "${module_dirs[$module]}/go.sum"
        done
        message="Update Go modules to use $(dependency_summary "${dependencies[@]}")"
        git -C "$repo_root" commit --message "$message"
        summary_commits+=("$(git -C "$repo_root" rev-parse --short HEAD) $message")
    fi

    for module in "${expected[@]}"; do
        [[ -n "${module_consumers[$module]-}" ]] || {
            if [[ -n "${changed_set[$module]+x}" && -z "${summary_leaf_seen[$module]+x}" ]]; then
                summary_leaves+=("${module_dirs[$module]}")
                summary_leaf_seen["$module"]=1
            fi
            continue
        }
        [[ -z "${published_versions[$module]+x}" ]] || continue
        module_ready_to_publish "$module" || continue
        intermediate+=("$module")
    done

    for module in "${intermediate[@]}"; do
        tag="$(select_intermediate_tag "$module")"
        version="$(module_version_from_tag "${module_dirs[$module]}" "$tag")"
        published_versions["$module"]="$version"
        wave_tags+=("$tag")
        next_dependencies+=("$module")
        summary_tags+=("$tag")
    done

    push_stage "${wave_tags[@]}"

    if ((${#next_dependencies[@]} > 0)); then
        update_wave "${next_dependencies[@]}"
    fi
}

print_dry_run_plan() {
    local -a dependencies=("$API_MODULE")
    local -a next=()
    local -a expected=()
    local dependency consumer module module_dependency dir latest version ready
    local -a helper_args=()
    declare -A planned=(["$API_MODULE"]="$release_version")
    declare -A expected_set=()
    declare -A next_set=()

    printf '\nDependency propagation plan\n'
    while ((${#dependencies[@]} > 0)); do
        printf '  publish:'
        for dependency in "${dependencies[@]}"; do
            printf ' %s@%s' "$dependency" "${planned[$dependency]}"
        done
        printf '\n'

        helper_args=(--wait-timeout "$wait_timeout_seconds" --dry-run)
        for dependency in "${dependencies[@]}"; do
            if [[ "$dependency" == "$API_MODULE" ]]; then
                helper_args+=("${planned[$dependency]}")
            else
                helper_args+=(--module "${dependency#"$MODULE_PREFIX/"}@${planned[$dependency]}")
            fi
        done
        run "$update_helper" "${helper_args[@]}"

        expected_set=()
        next_set=()
        for dependency in "${dependencies[@]}"; do
            for consumer in ${module_consumers[$dependency]-}; do
                [[ -n "${reachable[$consumer]+x}" ]] || continue
                expected_set["$consumer"]=1
            done
        done
        mapfile -t expected < <(printf '%s\n' "${!expected_set[@]}" | sort)
        printf '  update:'
        printf ' %s' "${expected[@]}"
        printf '\n'

        for module in "${expected[@]}"; do
            [[ -n "${module_consumers[$module]-}" ]] || continue
            [[ -z "${planned[$module]+x}" ]] || continue
            ready=true
            for module_dependency in ${module_dependencies[$module]-}; do
                [[ -n "${reachable[$module_dependency]+x}" ]] || continue
                if [[ -z "${planned[$module_dependency]+x}" ]]; then
                    ready=false
                    break
                fi
            done
            "$ready" && next_set["$module"]=1
        done

        next=()
        if ((${#next_set[@]} > 0)); then
            mapfile -t next < <(printf '%s\n' "${!next_set[@]}" | sort)
        fi
        for module in "${next[@]}"; do
            dir="${module_dirs[$module]}"
            latest="$(latest_stable_tag "$dir")"
            if [[ -n "$latest" ]]; then
                version="$(bump_version "$(module_version_from_tag "$dir" "$latest")" patch)"
            else
                version='v0.1.0'
            fi
            planned["$module"]="$version"
        done
        dependencies=("${next[@]}")
    done
}

print_summary() {
    local item

    printf '\nGo API release complete\n'
    printf '  API version: %s\n' "$release_version"
    if ((${#summary_tags[@]} > 0)); then
        printf '  Module tags:\n'
        printf '    %s\n' "${summary_tags[@]}"
    fi
    if ((${#summary_commits[@]} > 0)); then
        printf '  Dependency commits:\n'
        for item in "${summary_commits[@]}"; do
            printf '    %s\n' "$item"
        done
    fi
    if ((${#summary_leaves[@]} > 0)); then
        printf '  Updated leaf modules:\n'
        printf '    %s\n' "${summary_leaves[@]}"
    fi
}

main() {
    cd "$repo_root"
    preflight
    discover_modules
    validate_graph
    select_api_version
    verify_generated_api

    if "$dry_run"; then
        if ! git -C "$repo_root" rev-parse --verify --quiet \
            "refs/tags/api/$release_version" >/dev/null; then
            run git -C "$repo_root" tag --annotate "api/$release_version" \
                --message "Release API version ${release_version#v}" "$api_tag_commit"
        fi
        run git -C "$repo_root" push --atomic "$remote" \
            "HEAD:refs/heads/$branch" "refs/tags/api/$release_version:refs/tags/api/$release_version"
        print_dry_run_plan
        return
    fi

    publish_api
    update_wave "$API_MODULE"
    print_summary
}

main
