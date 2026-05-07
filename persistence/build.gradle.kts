// persistence/build.gradle.kts - configures the persistence module.
// Responsibilities: host database access, file serialization, and secret storage
// abstractions needed by the desktop application.

plugins {
    kotlin("jvm")
    kotlin("plugin.serialization")
}

dependencies {
    implementation(project(":shared"))
    implementation(project(":domain"))
    implementation("org.xerial:sqlite-jdbc:3.46.1.3")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.8.1")

    testImplementation(kotlin("test"))
}