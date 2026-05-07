// build.gradle.kts - defines shared build conventions for the Liaotao workspace.
// Responsibilities: register core plugins, centralize group/version, and apply
// Kotlin JVM conventions consistently across all modules.

plugins {
    kotlin("jvm") version "2.1.21" apply false
    kotlin("plugin.serialization") version "2.1.21" apply false
    id("org.jetbrains.kotlin.plugin.compose") version "2.1.21" apply false
    id("org.jetbrains.compose") version "1.8.0" apply false
}

group = "io.liaotao"
version = "1.0.0"

subprojects {
    group = rootProject.group
    version = rootProject.version

    pluginManager.withPlugin("org.jetbrains.kotlin.jvm") {
        extensions.configure(org.jetbrains.kotlin.gradle.dsl.KotlinJvmProjectExtension::class.java) {
            jvmToolchain(21)
        }
    }
}