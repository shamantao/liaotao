/*
 * DesktopThemeManager.kt - desktop implementation of semantic theme tokens.
 * Responsibilities: map shared theme tokens to Material color scheme values.
 */

package io.liaotao.appdesktop.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import io.liaotao.shared.theme.ThemeManager

internal object DesktopThemeTokens : ThemeManager {
    override val sidebarBackgroundToken: String = "sidebar.background"
    override val sidebarCardToken: String = "sidebar.card"
    override val sidebarSelectedCardToken: String = "sidebar.selected"
    override val workspaceCardToken: String = "workspace.card"
    override val workspaceStatusToken: String = "workspace.status"
}

internal object DesktopThemeManager {
    @Composable
    fun sidebarBackground(): Color = MaterialTheme.colorScheme.surface

    @Composable
    fun sidebarCard(): Color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.55f)

    @Composable
    fun sidebarSelectedCard(): Color = MaterialTheme.colorScheme.primary.copy(alpha = 0.28f)

    @Composable
    fun workspaceCard(): Color = MaterialTheme.colorScheme.surface

    @Composable
    fun workspaceStatus(): Color = MaterialTheme.colorScheme.tertiary.copy(alpha = 0.22f)
}
