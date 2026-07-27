````markdown
# Monorepo Module Tagging

This script reports and optionally creates Git tags for independently versioned modules in a monorepo.

For each module, it determines:

- The latest commit that touched the module directory
- The most recent tag matching `<module>-v<major>.<minor>.<patch>`
- The proposed next version tag
- The subject and date of the latest module commit

Tags are created on the latest commit that changed the module, which may be an earlier commit than `HEAD`.

When multiple tags are pushed, the script pushes them one-by-one. This ensures that GitHub receives a separate push event for each tag, allowing tag-triggered GitHub Actions workflows to run independently.

## Tag convention

Module tags use the following format:

```text
<module>-v<major>.<minor>.<patch>
```

Examples:

```text
my-service-v0.2.0
billing-service-v1.4.3
shared-library-v2.0.0
```

## Usage

```text
module-tags.sh [options] [module ...]
```

### Options

```text
--root DIR
    Directory containing the modules.
    Default: repository root.

--changed-since REF
    Only include modules changed after REF.
    Example: origin/main

--tag
    Create annotated tags.
    Without this option, the script is read-only.

--bump TYPE
    Version bump when creating tags.
    Supported values: major, minor, patch
    Default: patch

--version VERSION
    Use an explicit semantic version, such as 0.3.0.
    Requires exactly one explicitly named module.

--push REMOTE
    Push created tags to the specified remote.
    Implies --tag.
    Tags are pushed one-by-one.

-h, --help
    Show usage information.
```

## Usage modes

### 1. Inventory and report

```bash
./module-tags.sh
```

Discovers all immediate module directories and reports:

- The latest commit that touched each module
- The previous module tag
- The proposed next patch tag
- The commit date and subject

No tags are created or pushed.

Example output:

```text
MODULE          COMMIT        DATE        PREVIOUS TAG          PROPOSED TAG
my-service      6d018ab8f91e  2026-07-24  my-service-v0.2.0     my-service-v0.2.1
billing         fd320fe0f65a  2026-07-22  billing-v1.4.3        billing-v1.4.4
```

This mode is useful for reviewing the repository before deciding which modules should be released.

### 2. Inspect selected modules

```bash
./module-tags.sh my-service billing
```

Reports only the explicitly named modules.

This mode is useful when a pull request or release contains changes in a known subset of the monorepo.

### 3. Inspect modules changed since a branch or commit

```bash
./module-tags.sh --changed-since origin/main
```

Reports only modules whose paths have changed relative to the merge base with `origin/main`.

A commit SHA or tag can also be used:

```bash
./module-tags.sh --changed-since 91f4eab
./module-tags.sh --changed-since platform-release-2026-07
```

The `--changed-since` option determines which modules are included. The displayed commit is still the latest commit that touched each selected module.

Fetch the comparison branch before running the command when necessary:

```bash
git fetch origin main
./module-tags.sh --changed-since origin/main
```

### 4. Create local patch tags

```bash
./module-tags.sh --tag my-service billing
```

Creates the next patch tag for each selected module.

For example:

```text
my-service-v0.2.1
billing-v1.4.4
```

The tags are created locally but are not pushed.

The resulting tags can be inspected before publication:

```bash
git show my-service-v0.2.1
git show billing-v1.4.4
```

### 5. Create a minor release tag

```bash
./module-tags.sh --tag --bump minor my-service
```

Given the previous tag:

```text
my-service-v0.2.4
```

The script creates:

```text
my-service-v0.3.0
```

### 6. Create a major release tag

```bash
./module-tags.sh --tag --bump major my-service
```

Given the previous tag:

```text
my-service-v0.8.2
```

The script creates:

```text
my-service-v1.0.0
```

### 7. Create an explicit version

```bash
./module-tags.sh --tag --version 2.0.0 my-service
```

Creates:

```text
my-service-v2.0.0
```

Explicit versions require exactly one module. This prevents accidentally assigning the same version to multiple independently versioned modules.

### 8. Create and push one module tag

```bash
./module-tags.sh --push origin my-service
```

The `--push` option implies `--tag`, so the following command is equivalent:

```bash
./module-tags.sh --tag --push origin my-service
```

The script creates the next patch tag locally and pushes it to `origin`.

### 9. Create and push multiple tags one-by-one

```bash
./module-tags.sh --push origin my-service billing notifications
```

The script creates all required local tags and then pushes each tag in a separate Git push operation.

The operations are equivalent to:

```bash
git push origin \
    refs/tags/my-service-v0.2.1:refs/tags/my-service-v0.2.1

git push origin \
    refs/tags/billing-v1.4.4:refs/tags/billing-v1.4.4

git push origin \
    refs/tags/notifications-v0.7.2:refs/tags/notifications-v0.7.2
```

This is required when GitHub Actions must receive a separate tag push event for each module.

The script waits for each Git push command to complete before pushing the next tag. It does not wait for the corresponding GitHub Actions workflow run to finish.

Repository or workflow concurrency settings may still cancel or supersede workflow runs even when the tags are pushed separately.

### 10. Tag all modules changed since the main branch

```bash
git fetch origin main
./module-tags.sh --changed-since origin/main --push origin
```

This mode:

1. Discovers modules changed since `origin/main`
2. Finds the latest commit that touched each module
3. Determines the next patch version
4. Creates an annotated tag for each module
5. Pushes each tag separately

This is the most automated release mode and is suitable for releasing several changed modules after a reviewed merge.

### 11. Use a separate module root

For repositories where modules are stored under a common directory:

```text
services/
├── my-service/
├── billing/
└── notifications/
```

Use:

```bash
./module-tags.sh --root services
```

Selected modules are specified relative to the module root:

```bash
./module-tags.sh \
    --root services \
    --push origin \
    my-service billing
```

## Tag commit behavior

A module tag points to the latest commit that changed that module directory, not necessarily to the current `HEAD`.

For example:

```text
A -- B -- C -- D  HEAD
     ^
     my-service-v0.2.1
```

If commit `B` is the latest commit that touched `my-service`, the module tag points to `B`.

Checking out the tag checks out the entire repository as it existed at commit `B`. It does not combine the module directory from commit `B` with unrelated files from a later commit.

This behavior is important when a module depends on shared files outside its own directory. Changes to those shared files will not count as module changes unless their paths are included in the module's tagging logic.

## Push failure behavior

When multiple tags are pushed, they are processed sequentially.

If a push fails:

- Tags pushed earlier remain on the remote
- The failed tag remains local
- Tags after the failed tag are not pushed
- The command exits with a non-zero status

After resolving the problem, local tags can be inspected with:

```bash
git tag --list
```

A specific remaining tag can be pushed manually:

```bash
git push origin \
    refs/tags/my-service-v0.2.1:refs/tags/my-service-v0.2.1
```

## Recommended release workflow

Review the proposed tags without modifying the repository:

```bash
git fetch origin main
./module-tags.sh --changed-since origin/main
```

Create the tags locally:

```bash
./module-tags.sh \
    --changed-since origin/main \
    --tag
```

Inspect the created tags:

```bash
git show my-service-v0.2.1
git show billing-v1.4.4
```

Push the selected tags one-by-one:

```bash
./module-tags.sh \
    --changed-since origin/main \
    --push origin
```
````
