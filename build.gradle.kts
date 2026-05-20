import nl.littlerobots.vcu.plugin.resolver.VersionSelectors

plugins {
    alias(libs.plugins.versions)
    alias(libs.plugins.version.catalog.update)
}

versionCatalogUpdate {
    sortByKey.set(true)

    versionSelector(VersionSelectors.STABLE)

    keep {
        keepUnusedVersions.set(true)
    }
}
