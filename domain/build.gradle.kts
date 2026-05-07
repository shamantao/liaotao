// domain/build.gradle.kts - configures the domain module.
// Responsibilities: expose business rules and application use cases without UI
// or provider-specific runtime dependencies.

plugins {
    kotlin("jvm")
}

dependencies {
    implementation(project(":shared"))

    testImplementation(kotlin("test"))
}