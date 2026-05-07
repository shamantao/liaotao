/*
 * LiaotaoTheme.kt - central theme definitions for the desktop shell.
 * Responsibilities: define the dark-first color system and expose a single
 * Material theme wrapper used by all app surfaces.
 */

package io.liaotao.appdesktop.theme

import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

private val LiaotaoBlue = Color(0xFF1E3D59)
private val LiaotaoGreen = Color(0xFF3E8E7E)
private val LiaotaoBlack = Color(0xFF1B1B1B)
private val LiaotaoBeige = Color(0xFFE9E3D5)
private val LiaotaoSurface = Color(0xFF202226)
private val LiaotaoSurfaceAlt = Color(0xFF252A30)

private val LiaotaoDarkColors: ColorScheme = darkColorScheme(
    primary = LiaotaoBlue,
    onPrimary = LiaotaoBeige,
    secondary = LiaotaoGreen,
    onSecondary = LiaotaoBeige,
    background = LiaotaoBlack,
    onBackground = LiaotaoBeige,
    surface = LiaotaoSurface,
    onSurface = LiaotaoBeige,
    surfaceVariant = LiaotaoSurfaceAlt,
    onSurfaceVariant = LiaotaoBeige.copy(alpha = 0.82f),
    tertiary = LiaotaoGreen.copy(alpha = 0.9f),
    error = Color(0xFFCF6679),
)

@Composable
fun LiaotaoTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = LiaotaoDarkColors,
        content = content,
    )
}