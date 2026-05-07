// shared/build.gradle.kts - configures the shared module.
// Responsibilities: expose cross-cutting utilities such as configuration,
// logging, platform helpers, and shared data types.

plugins {
    kotlin("jvm")
    kotlin("plugin.serialization")
}

dependencies {
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-core:1.8.1")
}