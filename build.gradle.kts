import nl.littlerobots.vcu.plugin.resolver.VersionSelectors

plugins {
    alias(libs.plugins.versions)
    alias(libs.plugins.version.catalog.update)
}

subprojects {
    plugins.withType<JavaPlugin> {
        tasks.withType<JavaCompile>().configureEach {
            options.encoding = "UTF-8"
        }
    }
}

versionCatalogUpdate {
    sortByKey.set(true)

    versionSelector(VersionSelectors.STABLE)

    keep {
        keepUnusedVersions.set(true)
    }
}
