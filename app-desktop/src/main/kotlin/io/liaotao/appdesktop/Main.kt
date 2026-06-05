/*
 * Main.kt - desktop entry point for Liaotao.
 * Responsibilities: boot the Compose Desktop application and expose the first
 * application shell that will later host navigation, chat, and settings.
 */

package io.liaotao.appdesktop

import io.liaotao.appdesktop.theme.LiaotaoTheme
import io.liaotao.app_desktop.generated.resources.Res
import io.liaotao.app_desktop.generated.resources.liaotao_logo
import org.jetbrains.compose.resources.painterResource
import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application

fun main() = application {
    val appIcon = painterResource(Res.drawable.liaotao_logo)
    Window(onCloseRequest = ::exitApplication, title = "Liaotao", icon = appIcon) {
        LiaotaoTheme {
            LiaotaoAppShell()
        }
    }
}