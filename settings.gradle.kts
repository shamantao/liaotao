// settings.gradle.kts - configures the Gradle multi-module build for Liaotao.
// Responsibilities: declare plugin repositories, dependency repositories,
// root project identity, and the initial module boundaries for the desktop app.

pluginManagement {
    repositories {
        gradlePluginPortal()
        mavenCentral()
        google()
        maven("https://maven.pkg.jetbrains.space/public/p/compose/dev")
    }
}

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        mavenCentral()
        google()
        maven("https://maven.pkg.jetbrains.space/public/p/compose/dev")
    }
}

rootProject.name = "liaotao"

include(":app-desktop")
include(":domain")
include(":connectors")
include(":persistence")
include(":shared")