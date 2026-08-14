# Updating Gradle dependencies

Run all commands in this document from the repository root and use the Gradle
wrapper (`./gradlew`). Java library and plugin versions are centralized in
[`libs.versions.toml`](libs.versions.toml).

The root build applies two complementary plugins:

- The [Gradle Versions plugin](https://github.com/ben-manes/gradle-versions-plugin)
  reports available dependency and Gradle updates. It does not change the
  version catalog.
- The [Version Catalog Update plugin](https://github.com/littlerobots/version-catalog-update-plugin)
  checks or updates `libs.versions.toml`.

The catalog updater is configured in the root
[`build.gradle.kts`](../build.gradle.kts) to select only stable releases, sort
catalog entries by key, and retain unused entries in the `[versions]` section.
The Versions plugin uses its own reporting policy, so its report can include
milestones, release candidates, alpha versions, or beta versions that the
catalog updater will not select.

## Inspect available updates

Generate a consolidated report for the root project and all Java subprojects:

```bash
./gradlew dependencyUpdates
```

The report is printed to the terminal and written to
`build/dependencyUpdates/report.txt`. This command is advisory and does not
modify `libs.versions.toml`. To force Gradle to refresh cached repository
metadata, run:

```bash
./gradlew dependencyUpdates --refresh-dependencies
```

To check only the stable updates that the catalog updater would apply, without
changing the catalog, run:

```bash
./gradlew versionCatalogUpdate --check
```

This task exits successfully when no applicable updates exist and fails after
listing updates when the catalog is outdated. This makes it suitable for a CI
check.

Catalog keys can be checked individually. Use keys from `[libraries]` and
`[plugins]` in `libs.versions.toml`, not Maven coordinates:

```bash
./gradlew versionCatalogUpdate --check --library checker-qual
./gradlew versionCatalogUpdate --check --plugin versions
```

Repeat `--library` or `--plugin` to select multiple entries.

## Apply updates

### Review updates before applying them

Interactive mode is the preferred workflow when several updates are available:

```bash
./gradlew versionCatalogUpdate --interactive
```

This leaves `libs.versions.toml` unchanged and creates
`gradle/libs.versions.updates.toml`. Review that staging file and comment out or
remove any entries that should not be applied. Then apply the remaining entries:

```bash
./gradlew versionCatalogApplyUpdates
git diff -- gradle/libs.versions.toml
```

### Apply every stable update directly

To update the catalog without the interactive staging step, run:

```bash
./gradlew versionCatalogUpdate
git diff -- gradle/libs.versions.toml
```

### Apply selected updates directly

Use a catalog alias to limit the update to one library or plugin:

```bash
./gradlew versionCatalogUpdate --library checker-qual
./gradlew versionCatalogUpdate --plugin versions
```

An alias may use a shared `version.ref`. Updating it changes the corresponding
entry in `[versions]` and therefore updates every library or plugin using that
version key. For example, the gRPC aliases share the `grpc` version. Always
review the resulting catalog diff rather than assuming only one dependency was
affected.

## Validate an update

Review release notes and migration guides, especially for major-version
updates. Then run the full Java build because a catalog version can be shared by
multiple subprojects:

```bash
./gradlew build
./gradlew versionCatalogUpdate --check
```

The final check should succeed with no applicable stable updates. The broader
`dependencyUpdates` report may still list pre-release versions because it uses a
different reporting policy.

Commit `gradle/libs.versions.toml` together with any source or configuration
changes required by upgraded dependencies. Do not commit generated reports
under `build/` or the interactive `gradle/libs.versions.updates.toml` staging
file.

## Update the Gradle wrapper

The plugins can report a newer Gradle release, but they do not update the
wrapper. Update it separately, review `gradlew`, `gradlew.bat`, and the files
under `gradle/wrapper`, then run the build again:

```bash
./gradlew wrapper --gradle-version X.Y.Z
./gradlew wrapper
./gradlew build
```
