/*
 * Main.kt - desktop entry point for Liaotao.
 * Responsibilities: boot the Compose Desktop application and expose the first
 * application shell that will later host navigation, chat, and settings.
 */

package io.liaotao.appdesktop

import io.liaotao.appdesktop.theme.LiaotaoTheme
import androidx.compose.ui.window.Window
import androidx.compose.ui.window.application

fun main() = application {
    Window(onCloseRequest = ::exitApplication, title = "Liaotao") {
        LiaotaoTheme {
            LiaotaoAppShell()
        }
    }
}