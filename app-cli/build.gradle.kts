plugins {
    kotlin("jvm")
    kotlin("plugin.serialization")
    application
    id("com.gradleup.shadow") version "9.0.0-beta2"
}

dependencies {
    implementation(project(":shared"))
    implementation(project(":domain"))
    implementation(project(":connectors"))
    implementation(project(":persistence"))

    implementation("com.github.ajalt.clikt:clikt-jvm:5.0.1")
    implementation("com.github.ajalt.mordant:mordant:3.0.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.8.1")

    testImplementation(kotlin("test"))
}

application {
    mainClass = "io.liaotao.appcli.MainKt"
}


tasks.named<com.github.jengelman.gradle.plugins.shadow.tasks.ShadowJar>("shadowJar") {
    archiveBaseName.set("liaotao")
    archiveClassifier.set("")
    archiveVersion.set(project.version.toString())
    mergeServiceFiles()
}
