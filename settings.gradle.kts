rootProject.name = "veidemann"

include(
    "java-api",
    "rethinkdbadapter",
    "commons",
    "frontier",
    "controller",
)

pluginManagement {
    repositories {
        gradlePluginPortal()
    }

    plugins {
        id("com.google.cloud.tools.jib") version "3.5.1"
    }
}
