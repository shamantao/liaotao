/*
 * AppShell.kt - root shell for Liaotao desktop workspace.
 * Responsibilities: provide the application root container and delegate
 * the complete workspace layout to the chat workspace screen.
 */

package io.liaotao.appdesktop

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import io.liaotao.appdesktop.chat.ChatWorkspaceScreen

@Composable
fun LiaotaoAppShell() {
    Surface(modifier = Modifier.fillMaxSize()) {
        ChatWorkspaceScreen()
    }
}