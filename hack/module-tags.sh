#!/usr/bin/env bash
#
# module-tags.sh
#
# Reports, and optionally creates, per-module Git tags in a monorepo.
#
# Tag convention:
#   <module-name>-v<major>.<minor>.<patch>
#
# Examples:
#   ./module-tags.sh
#   ./module-tags.sh my-service another-service
#   ./module-tags.sh --changed-since origin/main
#   ./module-tags.sh --tag --bump patch my-service
#   ./module-tags.sh --tag --version 0.3.0 my-service
#
set -Eeuo pipefail

readonly SCRIPT_NAME="${0##*/}"

CREATE_TAGS=false
BUMP_KIND="patch"
EXPLICIT_VERSION=""
CHANGED_SINCE=""
MODULE_ROOT="."
TAG_REMOTE=""
declare -a REQUESTED_MODULES=()

readonly -a EXEMPT_MODULES=(
    "api"
    "commons"
    "deploy"
    ".devcontainer"
    ".gradle"
    "docs"
    "gradle"
    "hack"
    "java-api"
    "proto"
)

usage() {
    cat <<EOF
Usage:
  ${SCRIPT_NAME} [options] [module ...]

Options:
  --root DIR                Directory containing the modules.
                            Default: current repository root.

  --changed-since REF       Only include modules changed after REF.
                            Example: origin/main

  --tag                     Create annotated tags.
                            Without this option, the script is read-only.

  --bump TYPE               Version bump when creating tags:
                            major, minor, or patch.
                            Default: patch.

  --version VERSION         Use an explicit version, such as 0.3.0.
                            Requires exactly one module.

  --push REMOTE             Push created tags to REMOTE.
                            Implies --tag.

  -h, --help                Show this help.

Tag format:
  <module>-v<major>.<minor>.<patch>

Examples:
  ${SCRIPT_NAME}
  ${SCRIPT_NAME} my-service billing-service
  ${SCRIPT_NAME} --changed-since origin/main
  ${SCRIPT_NAME} --tag --bump minor my-service
  ${SCRIPT_NAME} --tag --version 1.0.0 my-service
  ${SCRIPT_NAME} --tag --push origin my-service
EOF
}

die() {
    printf 'Error: %s\n' "$*" >&2
    exit 1
}

require_command() {
    command -v "$1" >/dev/null 2>&1 ||
        die "Required command not found: $1"
}

repository_root() {
    git rev-parse --show-toplevel 2>/dev/null
}

normalize_version() {
    local version="$1"
    version="${version#v}"

    [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] ||
        die "Invalid semantic version: $1"

    printf '%s\n' "$version"
}

bump_version() {
    local version="$1"
    local bump="$2"
    local major minor patch

    IFS='.' read -r major minor patch <<<"$version"

    case "$bump" in
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
            die "Unsupported bump type: $bump"
            ;;
    esac

    printf '%d.%d.%d\n' "$major" "$minor" "$patch"
}

is_exempt_module() {
    local module="$1"
    local exempt_module

    for exempt_module in "${EXEMPT_MODULES[@]}"; do
        if [[ "$module" == "$exempt_module" ]]; then
            return 0
        fi
    done

    return 1
}

module_tag_pattern() {
    local module="$1"

    # Escape glob-significant characters so module names are treated literally.
    module="${module//\\/\\\\}"
    module="${module//\*/\\*}"
    module="${module//\?/\\?}"
    module="${module//\[/\\[}"

    printf '%s-v[0-9]*.[0-9]*.[0-9]*\n' "$module"
}

latest_module_tag() {
    local module="$1"
    local pattern
    pattern="$(module_tag_pattern "$module")"

    git tag \
        --list "$pattern" \
        --sort=-version:refname |
        head -n 1
}

latest_module_commit() {
    local module_path="$1"

    git log \
        -1 \
        --format='%H' \
        -- "$module_path"
}

latest_module_commit_date() {
    local commit="$1"

    git show \
        -s \
        --format='%cs' \
        "$commit"
}

latest_module_commit_subject() {
    local commit="$1"

    git show \
        -s \
        --format='%s' \
        "$commit"
}

module_changed_since() {
    local ref="$1"
    local module_path="$2"

    ! git diff --quiet "$ref"...HEAD -- "$module_path"
}

discover_modules() {
    local root="$1"
    local directory

    while IFS= read -r -d '' directory; do
        basename "$directory"
    done < <(
        find "$root" \
            -mindepth 1 \
            -maxdepth 1 \
            -type d \
            ! -name '.git' \
            ! -name '.github' \
            ! -name 'node_modules' \
            ! -name 'target' \
            ! -name 'build' \
            ! -name 'dist' \
            -print0 |
            sort -z
    )
}

validate_module() {
    local module="$1"
    local module_path="${MODULE_ROOT%/}/$module"

    [[ -d "$module_path" ]] ||
        die "Module directory does not exist: $module_path"

    [[ "$module" != */* ]] ||
        die "Module names must be direct children of '$MODULE_ROOT': $module"
}

push_tags_individually() {
    local remote="$1"
    shift

    local tag

    git remote get-url "$remote" >/dev/null 2>&1 ||
        die "Unknown Git remote: $remote"

    for tag in "$@"; do
        printf 'Pushing tag %s to %s\n' "$tag" "$remote"

        if ! git push "$remote" "refs/tags/$tag:refs/tags/$tag"; then
            printf 'Failed to push tag: %s\n' "$tag" >&2
            printf 'Tags after this one were not pushed.\n' >&2
            return 1
        fi
    done
}

tag_version() {
    local module="$1"
    local tag="$2"

    if [[ -z "$tag" ]]; then
        printf '%s\n' "-"
        return
    fi

    printf '%s\n' "${tag#"$module-v"}"
}

parse_arguments() {
    while (($# > 0)); do
        case "$1" in
            --root)
                (($# >= 2)) || die "--root requires a directory"
                MODULE_ROOT="$2"
                shift 2
                ;;
            --changed-since)
                (($# >= 2)) || die "--changed-since requires a Git ref"
                CHANGED_SINCE="$2"
                shift 2
                ;;
            --tag)
                CREATE_TAGS=true
                shift
                ;;
            --bump)
                (($# >= 2)) || die "--bump requires major, minor, or patch"
                BUMP_KIND="$2"
                shift 2
                ;;
            --version)
                (($# >= 2)) || die "--version requires a semantic version"
                EXPLICIT_VERSION="$(normalize_version "$2")"
                shift 2
                ;;
            --push)
                (($# >= 2)) || die "--push requires a remote name"
                TAG_REMOTE="$2"
                CREATE_TAGS=true
                shift 2
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            --)
                shift
                REQUESTED_MODULES+=("$@")
                break
                ;;
            -*)
                die "Unknown option: $1"
                ;;
            *)
                REQUESTED_MODULES+=("$1")
                shift
                ;;
        esac
    done
}

main() {
    require_command git
    require_command find
    require_command sort

    parse_arguments "$@"

    [[ "$BUMP_KIND" =~ ^(major|minor|patch)$ ]] ||
        die "--bump must be major, minor, or patch"

    local repo_root
    repo_root="$(repository_root)" ||
        die "Current directory is not inside a Git repository"

    cd "$repo_root"

    [[ -d "$MODULE_ROOT" ]] ||
        die "Module root does not exist: $MODULE_ROOT"

    if [[ -n "$CHANGED_SINCE" ]]; then
        git rev-parse --verify "${CHANGED_SINCE}^{commit}" >/dev/null 2>&1 ||
            die "Unknown Git ref: $CHANGED_SINCE"
    fi

    if [[ -n "$EXPLICIT_VERSION" && ${#REQUESTED_MODULES[@]} -ne 1 ]]; then
        die "--version requires exactly one explicitly named module"
    fi

    local -a modules=()

    if ((${#REQUESTED_MODULES[@]} > 0)); then
        modules=("${REQUESTED_MODULES[@]}")
    else
        mapfile -t modules < <(discover_modules "$MODULE_ROOT")
    fi

    ((${#modules[@]} > 0)) ||
        die "No module directories found"

    local module
    local module_path
    local commit
    local short_commit
    local commit_date
    local subject
    local previous_tag
    local previous_tag_display
    local previous_version
    local next_version
    local proposed_tag
    local proposed_tag_display
    local status
    local -a created_tags=()

    printf '%-28s %-12s %-10s %-16s %-16s %s\n' \
        "MODULE" \
        "COMMIT" \
        "DATE" \
        "PREVIOUS VERSION" \
        "PROPOSED VERSION" \
        "SUBJECT"

    printf '%-28s %-12s %-10s %-16s %-16s %s\n' \
        "----------------------------" \
        "------------" \
        "----------" \
        "------------------------" \
        "------------------------" \
        "----------------------------------------"

    for module in "${modules[@]}"; do
        if is_exempt_module "$module"; then
            continue
        fi

        validate_module "$module"
        module_path="${MODULE_ROOT%/}/$module"

        if [[ -n "$CHANGED_SINCE" ]] &&
            ! module_changed_since "$CHANGED_SINCE" "$module_path"; then
            continue
        fi

        commit="$(latest_module_commit "$module_path")"

        # The directory may exist locally without ever having been committed.
        if [[ -z "$commit" ]]; then
            printf '%-28s %-12s %-10s %-24s %-24s %s\n' \
                "$module" \
                "-" \
                "-" \
                "-" \
                "-" \
                "No committed changes"
            continue
        fi

        short_commit="${commit:0:12}"
        commit_date="$(latest_module_commit_date "$commit")"
        subject="$(latest_module_commit_subject "$commit")"
        previous_tag="$(latest_module_tag "$module")"

        if [[ -n "$EXPLICIT_VERSION" ]]; then
            next_version="$EXPLICIT_VERSION"
        elif [[ -n "$previous_tag" ]]; then
            previous_version="${previous_tag#"$module-v"}"
            next_version="$(bump_version "$previous_version" "$BUMP_KIND")"
        else
            next_version="0.1.0"
        fi

        proposed_tag="${module}-v${next_version}"
        
        previous_tag_display="$(tag_version "$module" "$previous_tag")"
        proposed_tag_display="$next_version"

        if git rev-parse --verify --quiet "refs/tags/$proposed_tag" >/dev/null; then
            proposed_tag_display="${next_version} (exists)"
        fi

        printf '%-28s %-12s %-10s %-16s %-16s %s\n' \
            "$module" \
            "$short_commit" \
            "$commit_date" \
            "$previous_tag_display" \
            "$proposed_tag_display" \
            "$subject"

        if [[ "$CREATE_TAGS" == true ]]; then
            if git rev-parse --verify --quiet "refs/tags/$proposed_tag" >/dev/null; then
                printf 'Skipping existing tag %s\n' "$proposed_tag" >&2
                continue
            fi

            git tag \
                --annotate "$proposed_tag" \
                --message "Release $module version $next_version" \
                "$commit"

            created_tags+=("$proposed_tag")
        fi
    done

    if [[ -n "$TAG_REMOTE" && ${#created_tags[@]} -gt 0 ]]; then
        push_tags_individually "$TAG_REMOTE" "${created_tags[@]}"
    fi
}

main "$@"