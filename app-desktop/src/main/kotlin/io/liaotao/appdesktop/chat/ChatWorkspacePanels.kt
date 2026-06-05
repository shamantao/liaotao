/*
 * ChatWorkspacePanels.kt - presentational panels for chat workspace.
 * Responsibilities: render sidebar, message timeline, and composer widgets
 * used by the workspace controller screen.
 */

package io.liaotao.appdesktop.chat

import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import io.liaotao.appdesktop.i18n.LocalAppStrings
import io.liaotao.appdesktop.theme.DesktopThemeManager
import io.liaotao.app_desktop.generated.resources.Res
import io.liaotao.app_desktop.generated.resources.liaotao_logo
import io.liaotao.domain.conversations.Conversation
import io.liaotao.domain.routing.ExecutionAttempt
import io.liaotao.shared.settings.ConnectorSetting
import org.jetbrains.compose.resources.painterResource
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
internal fun WorkspaceSidebar(
    expanded: Boolean,
    width: Dp,
    folders: List<UiFolder>,
    selectedFolderId: String,
    conversations: List<Conversation>,
    selectedConversationId: String?,
    onToggleExpanded: () -> Unit,
    onOpenSettings: () -> Unit,
    onAddFolder: () -> Unit,
    onExportFolders: () -> Unit,
    onSelectFolder: (String) -> Unit,
    onAddConversation: () -> Unit,
    onExportConversations: () -> Unit,
    onSelectConversation: (String) -> Unit,
) {
    val strings = LocalAppStrings.current
    val sidebarBackground = DesktopThemeManager.sidebarBackground()
    val sidebarCard = DesktopThemeManager.sidebarCard()
    val logoPainter = painterResource(Res.drawable.liaotao_logo)

    Card(
        modifier = Modifier
            .fillMaxHeight()
            .width(width)
            .padding(12.dp),
        colors = CardDefaults.cardColors(containerColor = sidebarBackground),
    ) {
        Column(
            modifier = Modifier.fillMaxSize().padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                Image(
                    painter = logoPainter,
                    contentDescription = "Liaotao logo",
                    modifier = Modifier.size(34.dp),
                )
                if (expanded) {
                    Text(
                        strings.appName,
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.SemiBold,
                    )
                }
            }

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                SideIconButton(label = "☰", onClick = onToggleExpanded)
                SideIconButton(label = "⚙", onClick = onOpenSettings)
            }

            SidebarSectionHeader(
                expanded = expanded,
                icon = "🗂",
                title = strings.folders,
                onAdd = onAddFolder,
                onExportAll = onExportFolders,
            )

            LazyColumn(
                modifier = Modifier.weight(0.45f),
                verticalArrangement = Arrangement.spacedBy(6.dp),
                contentPadding = PaddingValues(bottom = 8.dp),
            ) {
                items(folders, key = { it.id }) { folder ->
                    SidebarEntry(
                        expanded = expanded,
                        icon = folder.icon,
                        label = folder.label,
                        trailing = null,
                        selected = folder.id == selectedFolderId,
                        onClick = { onSelectFolder(folder.id) },
                        cardColor = sidebarCard,
                    )
                }
            }

            SidebarSectionHeader(
                expanded = expanded,
                icon = "💬",
                title = strings.conversations,
                onAdd = onAddConversation,
                onExportAll = onExportConversations,
            )

            LazyColumn(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(6.dp),
                contentPadding = PaddingValues(bottom = 6.dp),
            ) {
                items(conversations, key = { it.id }) { conversation ->
                    SidebarEntry(
                        expanded = expanded,
                        icon = "💬",
                        label = conversation.title,
                        trailing = formatDay(conversation.lastActivityAt),
                        selected = conversation.id == selectedConversationId,
                        onClick = { onSelectConversation(conversation.id) },
                        cardColor = sidebarCard,
                    )
                }
            }
        }
    }
}

@Composable
private fun SideIconButton(label: String, onClick: () -> Unit) {
    Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.65f))) {
        IconButton(onClick = onClick) {
            Text(label)
        }
    }
}

@Composable
private fun SidebarSectionHeader(
    expanded: Boolean,
    icon: String,
    title: String,
    onAdd: () -> Unit,
    onExportAll: () -> Unit,
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Text(icon)
        if (expanded) {
            Text(title, style = MaterialTheme.typography.titleSmall, modifier = Modifier.weight(1f))
        }
        SideIconButton(label = "+", onClick = onAdd)
        SideIconButton(label = "⇪", onClick = onExportAll)
    }
}

@Composable
private fun SidebarEntry(
    expanded: Boolean,
    icon: String,
    label: String,
    trailing: String?,
    selected: Boolean,
    onClick: () -> Unit,
    cardColor: Color,
) {
    val selectedColor = DesktopThemeManager.sidebarSelectedCard()
    Card(
        colors = CardDefaults.cardColors(containerColor = if (selected) selectedColor else cardColor),
        onClick = onClick,
    ) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 10.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(icon)
            if (expanded) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(label, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    if (!trailing.isNullOrBlank()) {
                        Text(trailing, style = MaterialTheme.typography.labelSmall)
                    }
                }
            }
        }
    }
}

@Composable
internal fun ConversationWorkspace(
    modifier: Modifier = Modifier,
    messages: List<ChatUiMessage>,
    prompt: String,
    providers: List<ConnectorSetting>,
    selectedProviderId: String?,
    modelMenuExpanded: Boolean,
    attachedFileLabel: String?,
    statusMessage: String?,
    executionAttempts: List<ExecutionAttempt>,
    onPromptChange: (String) -> Unit,
    onToggleModelMenu: () -> Unit,
    onSelectProvider: (String) -> Unit,
    onAttach: () -> Unit,
    onSend: () -> Unit,
    onRetry: () -> Unit,
    onEditUserMessage: (String) -> Unit,
) {
    val strings = LocalAppStrings.current
    val selectedProvider = providers.firstOrNull { it.id == selectedProviderId }
    val providerLabel = selectedProvider?.displayName ?: strings.noProviderEnabled

    Card(
        modifier = modifier,
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
    ) {
        Column(modifier = Modifier.fillMaxSize().padding(14.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            if (!statusMessage.isNullOrBlank()) {
                Card(colors = CardDefaults.cardColors(containerColor = DesktopThemeManager.workspaceStatus())) {
                    Text(
                        text = statusMessage,
                        modifier = Modifier.fillMaxWidth().padding(horizontal = 10.dp, vertical = 8.dp),
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }

            Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f))) {
                Column(modifier = Modifier.fillMaxWidth().padding(10.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Text(strings.executionAttempts, style = MaterialTheme.typography.titleSmall)
                    executionAttempts.takeLast(4).forEach { attempt ->
                        val detail = "${attempt.providerId} · ${attempt.status} · retry ${attempt.retryIndex}" +
                            (attempt.errorMessage?.let { " · $it" } ?: "")
                        Text(detail, style = MaterialTheme.typography.labelSmall)
                    }
                }
            }

            LazyColumn(
                modifier = Modifier.fillMaxWidth().weight(1f),
                verticalArrangement = Arrangement.spacedBy(10.dp),
                contentPadding = PaddingValues(vertical = 8.dp),
            ) {
                items(messages, key = { it.id }) { message ->
                    ChatBubble(message = message, onEditUserMessage = onEditUserMessage)
                }
            }

            Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f))) {
                Column(modifier = Modifier.fillMaxWidth().padding(12.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = prompt,
                        onValueChange = onPromptChange,
                        label = { Text(strings.askYourQuestion) },
                        minLines = 3,
                        maxLines = 12,
                        modifier = Modifier.fillMaxWidth(),
                    )

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Box {
                                TextButton(
                                    onClick = onToggleModelMenu,
                                    colors = ButtonDefaults.textButtonColors(
                                        contentColor = MaterialTheme.colorScheme.onSurface,
                                    ),
                                ) {
                                    Text(strings.modelLabel(providerLabel))
                                }
                                DropdownMenu(expanded = modelMenuExpanded, onDismissRequest = onToggleModelMenu) {
                                    providers.forEach { provider ->
                                        DropdownMenuItem(
                                            text = { Text(provider.displayName) },
                                            onClick = { onSelectProvider(provider.id) },
                                        )
                                    }
                                }
                            }
                            TextButton(
                                onClick = onAttach,
                                colors = ButtonDefaults.textButtonColors(
                                    contentColor = MaterialTheme.colorScheme.onSurface,
                                ),
                            ) {
                                Text(attachedFileLabel ?: strings.attachFile)
                            }
                        }
                        BoxWithConstraints {
                            val compact = maxWidth < 220.dp
                            Row(horizontalArrangement = Arrangement.spacedBy(if (compact) 6.dp else 8.dp)) {
                                Button(
                                    onClick = onRetry,
                                    contentPadding = if (compact) PaddingValues(horizontal = 10.dp, vertical = 8.dp) else ButtonDefaults.ContentPadding,
                                ) {
                                    Text("↻")
                                }
                                Button(
                                    onClick = onSend,
                                    enabled = selectedProvider != null,
                                    contentPadding = if (compact) PaddingValues(horizontal = 10.dp, vertical = 8.dp) else ButtonDefaults.ContentPadding,
                                ) {
                                    Text("➤")
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private val dayFormatter: DateTimeFormatter = DateTimeFormatter.ofPattern("yyyy-MM-dd")
private val localZone: ZoneId = ZoneId.systemDefault()

private fun formatDay(instant: Instant): String = dayFormatter.format(instant.atZone(localZone))
