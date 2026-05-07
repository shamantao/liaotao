// connectors/build.gradle.kts - configures the connector module.
// Responsibilities: host provider contracts, transport abstractions, and
// concrete integrations for OpenAI-compatible endpoints, Aitao, LiteLLM, and MCP.

plugins {
    kotlin("jvm")
    kotlin("plugin.serialization")
}

dependencies {
    implementation(project(":shared"))
    implementation(project(":domain"))
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.8.1")

    testImplementation(kotlin("test"))
}