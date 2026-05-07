/*
 * AppShell.kt - desktop shell for Liaotao P1.1.
 * Responsibilities: provide navigation structure, adaptive layout behavior,
 * and a global runtime status surface shared across all primary screens.
 */

package io.liaotao.appdesktop

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import io.liaotao.appdesktop.chat.ChatWorkspaceScreen
import io.liaotao.appdesktop.settings.SettingsScreen

private enum class AppSection(val label: String) {
    Chat("Chat"),
    Projects("Projects"),
    Settings("Settings"),
}

@Composable
fun LiaotaoAppShell() {
    var selectedSection by remember { mutableStateOf(AppSection.Chat) }
    val status = remember {
        RuntimeStatus(
            title = "System status",
            detail = "Ready for desktop initialization",
            isHealthy = true,
        )
    }

    Surface(modifier = Modifier.fillMaxSize()) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(
                    brush = Brush.verticalGradient(
                        colors = listOf(
                            MaterialTheme.colorScheme.background,
                            MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.65f),
                        ),
                    ),
                ),
        ) {
            BoxWithConstraints(modifier = Modifier.fillMaxSize()) {
                val compactLayout = maxWidth < 980.dp
                if (compactLayout) {
                    CompactShell(
                        selectedSection = selectedSection,
                        status = status,
                        onSelectSection = { selectedSection = it },
                    )
                } else {
                    WideShell(
                        selectedSection = selectedSection,
                        status = status,
                        onSelectSection = { selectedSection = it },
                    )
                }
            }
        }
    }
}

@Composable
private fun WideShell(
    selectedSection: AppSection,
    status: RuntimeStatus,
    onSelectSection: (AppSection) -> Unit,
) {
    Row(modifier = Modifier.fillMaxSize()) {
        NavigationRail(
            selectedSection = selectedSection,
            onSelectSection = onSelectSection,
            modifier = Modifier
                .fillMaxHeight()
                .width(220.dp)
                .padding(16.dp),
        )

        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(top = 16.dp, end = 16.dp, bottom = 16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            RuntimeStatusBanner(status = status)
            MainContent(section = selectedSection, paddingValues = PaddingValues(20.dp))
        }
    }
}

@Composable
private fun CompactShell(
    selectedSection: AppSection,
    status: RuntimeStatus,
    onSelectSection: (AppSection) -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(12.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        RuntimeStatusBanner(status = status)
        CompactTopNav(selectedSection = selectedSection, onSelectSection = onSelectSection)
        MainContent(section = selectedSection, paddingValues = PaddingValues(16.dp))
    }
}

@Composable
private fun NavigationRail(
    selectedSection: AppSection,
    onSelectSection: (AppSection) -> Unit,
    modifier: Modifier = Modifier,
) {
    Card(
        modifier = modifier,
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface.copy(alpha = 0.95f)),
    ) {
        Column(
            modifier = Modifier.fillMaxSize().padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text(text = "Liaotao", style = MaterialTheme.typography.titleLarge)
            Text(
                text = "Desktop workspace",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            AppSection.entries.forEach { section ->
                if (section == selectedSection) {
                    Button(
                        modifier = Modifier.fillMaxWidth(),
                        onClick = { onSelectSection(section) },
                    ) {
                        Text(section.label)
                    }
                } else {
                    TextButton(
                        modifier = Modifier.fillMaxWidth(),
                        onClick = { onSelectSection(section) },
                    ) {
                        Text(section.label)
                    }
                }
            }
        }
    }
}

@Composable
private fun CompactTopNav(
    selectedSection: AppSection,
    onSelectSection: (AppSection) -> Unit,
) {
    Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface)) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(6.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            AppSection.entries.forEach { section ->
                if (section == selectedSection) {
                    Button(onClick = { onSelectSection(section) }) {
                        Text(section.label)
                    }
                } else {
                    TextButton(onClick = { onSelectSection(section) }) {
                        Text(section.label)
                    }
                }
            }
        }
    }
}

@Composable
private fun RuntimeStatusBanner(status: RuntimeStatus) {
    val tone = if (status.isHealthy) {
        MaterialTheme.colorScheme.tertiary.copy(alpha = 0.2f)
    } else {
        Color(0x66CF6679)
    }

    Card(
        colors = CardDefaults.cardColors(containerColor = tone),
        elevation = CardDefaults.cardElevation(defaultElevation = 0.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(status.title, style = MaterialTheme.typography.labelLarge)
                Text(status.detail, style = MaterialTheme.typography.bodySmall)
            }
            Text(
                text = if (status.isHealthy) "Healthy" else "Degraded",
                style = MaterialTheme.typography.labelMedium,
                color = if (status.isHealthy) MaterialTheme.colorScheme.tertiary else Color(0xFFCF6679),
            )
        }
    }
}

@Composable
private fun MainContent(section: AppSection, paddingValues: PaddingValues) {
    Card(
        modifier = Modifier.fillMaxSize(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
    ) {
        Box(modifier = Modifier.fillMaxSize().padding(paddingValues), contentAlignment = Alignment.CenterStart) {
            when (section) {
                AppSection.Chat -> ChatWorkspaceScreen()

                AppSection.Projects -> ScreenPlaceholder(
                    title = "Projects",
                    body = "Project and conversation organization UI will be attached here.",
                )

                AppSection.Settings -> SettingsScreen()
            }
        }
    }
}

@Composable
private fun ScreenPlaceholder(title: String, body: String) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(text = title, style = MaterialTheme.typography.headlineSmall)
        Text(text = body, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

private data class RuntimeStatus(
    val title: String,
    val detail: String,
    val isHealthy: Boolean,
)