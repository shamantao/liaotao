/*
 * ThemeManager.kt - shared theme contract.
 * Responsibilities: expose semantic color tokens consumed by feature screens
 * instead of hardcoded values.
 */

package io.liaotao.shared.theme

interface ThemeManager {
    val sidebarBackgroundToken: String
    val sidebarCardToken: String
    val sidebarSelectedCardToken: String
    val workspaceCardToken: String
    val workspaceStatusToken: String
}
