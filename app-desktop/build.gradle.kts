// app-desktop/build.gradle.kts - configures the desktop application module.
// Responsibilities: declare Compose Desktop runtime dependencies, wire module
// dependencies, and define native packaging targets for desktop releases.

plugins {
    kotlin("jvm")
    id("org.jetbrains.kotlin.plugin.compose")
    id("org.jetbrains.compose")
}

dependencies {
    implementation(compose.desktop.currentOs)
    implementation(compose.material3)
    implementation(project(":domain"))
    implementation(project(":connectors"))
    implementation(project(":persistence"))
    implementation(project(":shared"))
}

compose.desktop {
    application {
        mainClass = "io.liaotao.appdesktop.MainKt"

        nativeDistributions {
            targetFormats(
                org.jetbrains.compose.desktop.application.dsl.TargetFormat.Dmg,
                org.jetbrains.compose.desktop.application.dsl.TargetFormat.Msi,
                org.jetbrains.compose.desktop.application.dsl.TargetFormat.Deb,
            )
            packageName = "Liaotao"
            packageVersion = project.version.toString()
            description = "Desktop AI workspace for multi-source conversations"
            vendor = "Liaotao"
        }
    }
}