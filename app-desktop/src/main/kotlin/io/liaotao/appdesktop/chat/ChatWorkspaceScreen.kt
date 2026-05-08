/*
 * ChatWorkspaceScreen.kt - primary desktop workspace UI.
 * Responsibilities: render the collapsible sidebar, conversation timeline,
 * bottom composer, and settings toggle while preserving chat orchestration.
 */

package io.liaotao.appdesktop.chat

import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.TextButton
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.Composable
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import io.liaotao.appdesktop.settings.SettingsScreen
import io.liaotao.connectors.core.ConnectorType
import io.liaotao.domain.history.ConversationHistoryQuery
import io.liaotao.domain.history.ConversationHistoryService
import io.liaotao.domain.conversations.Conversation
import io.liaotao.domain.routing.ExecutionAttempt
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.UUID

@Composable
fun ChatWorkspaceScreen() {
    val controller = remember { ChatWorkspaceController() }
    var prompt by rememberSaveable { mutableStateOf("") }
    var selectedProvider by rememberSaveable { mutableStateOf(ConnectorType.OLLAMA) }
    var showSettings by rememberSaveable { mutableStateOf(false) }
    var sidebarExpanded by rememberSaveable { mutableStateOf(true) }
    var modelMenuExpanded by remember { mutableStateOf(false) }
    var attachedFileLabel by rememberSaveable { mutableStateOf<String?>(null) }
    var folderCounter by rememberSaveable { mutableStateOf(2) }
    var selectedFolderId by rememberSaveable { mutableStateOf("default") }
    var selectedConversationId by rememberSaveable { mutableStateOf<String?>(null) }

    val folders = remember {
        mutableStateListOf(
            UiFolder(id = "default", label = "Folder #1", icon = "📁"),
        )
    }

    val visibleConversations = remember(controller.conversations.toList(), selectedFolderId) {
        ConversationHistoryService().query(
            conversations = controller.conversations.toList(),
            request = ConversationHistoryQuery(
                keyword = "",
                projectId = selectedFolderId,
                source = null,
            ),
        ).sortedByDescending { it.lastActivityAt }
    }

    val sidebarWidth by animateDpAsState(if (sidebarExpanded) 336.dp else 92.dp)

    Row(
        modifier = Modifier
            .fillMaxSize()
            .background(
                brush = Brush.horizontalGradient(
                    colors = listOf(
                        Color(0xFFEEF5F4),
                        Color(0xFFFDF9F2),
                    ),
                ),
            ),
    ) {
        WorkspaceSidebar(
            expanded = sidebarExpanded,
            width = sidebarWidth,
            folders = folders,
            selectedFolderId = selectedFolderId,
            conversations = visibleConversations,
            selectedConversationId = selectedConversationId,
            onToggleExpanded = { sidebarExpanded = !sidebarExpanded },
            onOpenSettings = { showSettings = true },
            onAddFolder = {
                val id = "folder-$folderCounter"
                folders.add(UiFolder(id = id, label = "Folder #$folderCounter", icon = "📁"))
                selectedFolderId = id
                folderCounter += 1
            },
            onExportFolders = {},
            onSelectFolder = { selectedFolderId = it },
            onAddConversation = {
                controller.startConversation()
                selectedConversationId = null
                showSettings = false
            },
            onExportConversations = {},
            onSelectConversation = { selectedConversationId = it },
        )

        if (showSettings) {
            Card(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxHeight()
                    .padding(16.dp),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
            ) {
                Column(modifier = Modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text("Settings", style = MaterialTheme.typography.headlineSmall)
                        TextButton(onClick = { showSettings = false }) { Text("Back to chat") }
                    }
                    SettingsScreen()
                }
            }
        } else {
            ConversationWorkspace(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxHeight()
                    .padding(16.dp),
                messages = controller.messages,
                prompt = prompt,
                selectedProvider = selectedProvider,
                modelMenuExpanded = modelMenuExpanded,
                attachedFileLabel = attachedFileLabel,
                executionAttempts = controller.executionAttempts,
                onPromptChange = { prompt = it },
                onToggleModelMenu = { modelMenuExpanded = !modelMenuExpanded },
                onSelectProvider = {
                    selectedProvider = it
                    modelMenuExpanded = false
                },
                onAttach = {
                    attachedFileLabel = if (attachedFileLabel == null) "attached-context.txt" else null
                },
                onSend = {
                    val text = prompt.trim()
                    if (text.isNotEmpty()) {
                        controller.send(text, selectedProvider)
                        prompt = ""
                        attachedFileLabel = null
                    }
                },
                onRetry = { controller.retryLast(selectedProvider) },
                onEditUserMessage = { prompt = it },
            )
        }
    }
}

@Composable
private fun WorkspaceSidebar(
    expanded: Boolean,
    width: androidx.compose.ui.unit.Dp,
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
    Card(
        modifier = Modifier
            .fillMaxHeight()
            .width(width)
            .padding(12.dp),
        colors = CardDefaults.cardColors(containerColor = Color(0xFF18252A)),
    ) {
        Column(
            modifier = Modifier.fillMaxSize().padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("◉", color = Color(0xFF8EE3A1), style = MaterialTheme.typography.titleMedium)
                if (expanded) {
                    Text("Liaotao", color = Color.White, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold)
                }
            }

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                SideIconButton(label = "☰", isActive = false, onClick = onToggleExpanded)
                SideIconButton(label = "⚙", isActive = false, onClick = onOpenSettings)
            }

            SidebarSectionHeader(
                expanded = expanded,
                icon = "🗂",
                title = "Folders",
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
                    )
                }
            }

            SidebarSectionHeader(
                expanded = expanded,
                icon = "💬",
                title = "Conversations",
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
                    )
                }
            }
        }
    }
}

@Composable
private fun SideIconButton(label: String, isActive: Boolean, onClick: () -> Unit) {
    val container = if (isActive) Color(0xFF30697A) else Color(0xFF21343B)
    Card(colors = CardDefaults.cardColors(containerColor = container)) {
        IconButton(onClick = onClick) {
            Text(label, color = Color.White)
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
        Text(icon, color = Color(0xFFB6DCE8))
        if (expanded) {
            Text(title, color = Color(0xFFEAF4F8), style = MaterialTheme.typography.titleSmall, modifier = Modifier.weight(1f))
        }
        SideIconButton(label = "+", isActive = false, onClick = onAdd)
        SideIconButton(label = "⇪", isActive = false, onClick = onExportAll)
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
) {
    val tone = if (selected) Color(0xFF2C4954) else Color(0xFF1F3037)
    Card(colors = CardDefaults.cardColors(containerColor = tone), onClick = onClick) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 10.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(icon, color = Color.White)
            if (expanded) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(label, color = Color.White, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    if (!trailing.isNullOrBlank()) {
                        Text(trailing, color = Color(0xFFA4BDC7), style = MaterialTheme.typography.labelSmall)
                    }
                }
            }
        }
    }
}

@Composable
private fun ConversationWorkspace(
    modifier: Modifier = Modifier,
    messages: List<ChatUiMessage>,
    prompt: String,
    selectedProvider: ConnectorType,
    modelMenuExpanded: Boolean,
    attachedFileLabel: String?,
    executionAttempts: List<ExecutionAttempt>,
    onPromptChange: (String) -> Unit,
    onToggleModelMenu: () -> Unit,
    onSelectProvider: (ConnectorType) -> Unit,
    onAttach: () -> Unit,
    onSend: () -> Unit,
    onRetry: () -> Unit,
    onEditUserMessage: (String) -> Unit,
) {
    Card(
        modifier = modifier,
        colors = CardDefaults.cardColors(containerColor = Color(0xFFFDFDFB)),
    ) {
        Column(modifier = Modifier.fillMaxSize().padding(14.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Card(colors = CardDefaults.cardColors(containerColor = Color(0xFFEAF3F8))) {
                Column(modifier = Modifier.fillMaxWidth().padding(10.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    Text("Execution attempts", style = MaterialTheme.typography.titleSmall)
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

            Card(colors = CardDefaults.cardColors(containerColor = Color(0xFFF3F9F7))) {
                Column(modifier = Modifier.fillMaxWidth().padding(12.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = prompt,
                        onValueChange = onPromptChange,
                        label = { Text("Ask your question") },
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
                                TextButton(onClick = onToggleModelMenu) {
                                    Text("Model: ${selectedProvider.name}")
                                }
                                DropdownMenu(expanded = modelMenuExpanded, onDismissRequest = onToggleModelMenu) {
                                    ConnectorType.entries.forEach { type ->
                                        DropdownMenuItem(
                                            text = { Text(type.name) },
                                            onClick = { onSelectProvider(type) },
                                        )
                                    }
                                }
                            }
                            TextButton(onClick = onAttach) {
                                Text(attachedFileLabel ?: "Attach file")
                            }
                        }
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            Button(onClick = onRetry) { Text("Retry") }
                            Button(onClick = onSend) { Text("Send") }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ChatBubble(message: ChatUiMessage, onEditUserMessage: (String) -> Unit) {
    val isUser = message.role == "user"
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = if (isUser) Arrangement.End else Arrangement.Start,
    ) {
        Card(
            modifier = Modifier
                .fillMaxWidth(0.82f)
                .heightIn(min = 72.dp),
            colors = CardDefaults.cardColors(
                containerColor = if (isUser) Color(0xFFE2F6E9) else Color(0xFFEAF1FF),
            ),
        ) {
            Column(modifier = Modifier.fillMaxWidth().padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(message.content.ifBlank { "..." })

                if (isUser) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.End,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(formatDateTime(message.createdAt), style = MaterialTheme.typography.labelSmall)
                        Spacer(modifier = Modifier.size(8.dp))
                        TextButton(onClick = { onEditUserMessage(message.content) }) { Text("✎") }
                    }
                } else {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                        AssistantMetaTag("Model", message.model)
                        AssistantMetaTag("Tokens", (message.tokensUsed ?: 0).toString())
                        AssistantMetaTag("Done", formatDateTime(message.completedAt ?: message.createdAt))
                    }
                }
            }
        }
    }
}

@Composable
private fun AssistantMetaTag(label: String, value: String) {
    Card(colors = CardDefaults.cardColors(containerColor = Color(0xFFD8E9F7))) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text("$label:", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.SemiBold)
            Text(value, style = MaterialTheme.typography.labelSmall)
        }
    }
}

internal class ChatWorkspaceController {
    private val chatService = ChatWorkspaceService()

    val messages = mutableStateListOf<ChatUiMessage>()
    val executionAttempts = mutableStateListOf<ExecutionAttempt>()
    val conversations = mutableStateListOf<io.liaotao.domain.conversations.Conversation>()

    private var lastPrompt: String? = null
    private var lastProvider: ConnectorType = ConnectorType.OLLAMA

    fun startConversation() {
        messages.clear()
        lastPrompt = null
    }

    fun send(prompt: String, provider: ConnectorType) {
        val sentAt = Instant.now()
        val userMessage = ChatUiMessage(
            id = UUID.randomUUID().toString(),
            role = "user",
            content = prompt,
            source = provider.name,
            model = "user-input",
            createdAt = sentAt,
            completedAt = sentAt,
            tokensUsed = estimateTokens(prompt),
        )
        messages.add(userMessage)

        val draftId = UUID.randomUUID().toString()
        val draftCreatedAt = Instant.now()
        messages.add(
            ChatUiMessage(
                id = draftId,
                role = "assistant",
                content = "",
                source = provider.name,
                model = "streaming",
                createdAt = draftCreatedAt,
                completedAt = null,
                tokensUsed = null,
            ),
        )

        val result = chatService.execute(prompt, provider) { chunk ->
            val index = messages.indexOfFirst { it.id == draftId }
            if (index >= 0) {
                val current = messages[index]
                messages[index] = current.copy(content = current.content + chunk)
            }
        }
        executionAttempts.addAll(result.attempts)

        val draftIndex = messages.indexOfFirst { it.id == draftId }
        if (draftIndex >= 0) {
            val draft = messages[draftIndex]
            val finalContent = if (draft.content.isNotBlank()) draft.content else result.reply
            val completedAt = Instant.now()
            messages[draftIndex] = draft.copy(
                content = finalContent,
                source = result.source,
                model = result.model,
                completedAt = completedAt,
                tokensUsed = estimateTokens(finalContent),
            )
        }

        conversations.clear()
        conversations.addAll(chatService.conversationHistory())

        lastPrompt = prompt
        lastProvider = provider
    }

    fun retryLast(provider: ConnectorType) {
        val prompt = lastPrompt ?: return
        send(prompt, provider.takeIf { it != ConnectorType.OPENAI_COMPAT } ?: lastProvider)
    }
}

internal data class ChatUiMessage(
    val id: String,
    val role: String,
    val content: String,
    val source: String,
    val model: String,
    val createdAt: Instant,
    val completedAt: Instant?,
    val tokensUsed: Int?,
)

internal data class UiFolder(
    val id: String,
    val label: String,
    val icon: String,
)

private val dayFormatter: DateTimeFormatter = DateTimeFormatter.ofPattern("yyyy-MM-dd")
private val dateTimeFormatter: DateTimeFormatter = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")
private val localZone: ZoneId = ZoneId.systemDefault()

private fun formatDay(instant: Instant): String = dayFormatter.format(instant.atZone(localZone))

private fun formatDateTime(instant: Instant): String = dateTimeFormatter.format(instant.atZone(localZone))

private fun estimateTokens(content: String): Int {
    val trimmed = content.trim()
    if (trimmed.isEmpty()) {
        return 0
    }
    return trimmed.split(Regex("\\s+")).size
}