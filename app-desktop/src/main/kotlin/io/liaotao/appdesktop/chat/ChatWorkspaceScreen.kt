/*
 * ChatWorkspaceScreen.kt - chat workspace orchestration and state.
 * Responsibilities: hold UI state, delegate rendering to panel components,
 * and coordinate user actions with chat services.
 */

package io.liaotao.appdesktop.chat

import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.unit.dp
import io.liaotao.appdesktop.i18n.AppI18n
import io.liaotao.appdesktop.i18n.AppLanguage
import io.liaotao.appdesktop.i18n.LocalAppStrings
import io.liaotao.appdesktop.settings.SettingsScreen
import io.liaotao.appdesktop.settings.DesktopUserPreferences
import io.liaotao.appdesktop.settings.DesktopFeatureFlags
import io.liaotao.appdesktop.settings.ProviderSettingsService
import io.liaotao.connectors.core.ConnectorType
import io.liaotao.domain.history.ConversationHistoryQuery
import io.liaotao.domain.history.ConversationHistoryService
import io.liaotao.domain.routing.ExecutionAttempt
import io.liaotao.shared.settings.ConnectorSetting
import java.time.Instant

@Composable
fun ChatWorkspaceScreen() {
    val controller = remember { ChatWorkspaceController() }
    val providerService = remember { ProviderSettingsService() }
    var language by rememberSaveable { mutableStateOf(DesktopUserPreferences.loadLanguage()) }
    val strings = remember(language) { AppI18n.strings(language) }
    var prompt by rememberSaveable { mutableStateOf("") }
    val enabledProviders = remember { mutableStateListOf<ConnectorSetting>() }
    var selectedProviderId by rememberSaveable { mutableStateOf<String?>(null) }
    var showSettings by rememberSaveable { mutableStateOf(false) }
    var sidebarExpanded by rememberSaveable { mutableStateOf(true) }
    var modelMenuExpanded by remember { mutableStateOf(false) }
    var attachedFileLabel by rememberSaveable { mutableStateOf<String?>(null) }
    var attachedFilePath by rememberSaveable { mutableStateOf<String?>(null) }
    var workspaceStatus by rememberSaveable { mutableStateOf<String?>(null) }
    var folderCounter by rememberSaveable { mutableStateOf(2) }
    var selectedFolderId by rememberSaveable { mutableStateOf("default") }
    var selectedConversationId by rememberSaveable { mutableStateOf<String?>(null) }

    fun refreshProviders() {
        if (!DesktopFeatureFlags.usePersistedProviderSelector) {
            enabledProviders.clear()
            enabledProviders.addAll(legacyProviderFallbacks())
            if (enabledProviders.none { it.id == selectedProviderId }) {
                selectedProviderId = enabledProviders.firstOrNull()?.id
            }
            return
        }

        providerService.ensureDefaults()
        enabledProviders.clear()
        enabledProviders.addAll(providerService.listEnabled())
        if (enabledProviders.none { it.id == selectedProviderId }) {
            selectedProviderId = enabledProviders.firstOrNull()?.id
        }
    }

    LaunchedEffect(showSettings) {
        if (!showSettings) {
            refreshProviders()
        }
    }

    val folders = remember {
        mutableStateListOf(
            UiFolder(id = "default", label = strings.folderLabel(1), icon = "📁"),
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

    CompositionLocalProvider(LocalAppStrings provides strings) {
        Row(
            modifier = Modifier
                .fillMaxSize()
                .background(
                    brush = Brush.horizontalGradient(
                        colors = listOf(
                            MaterialTheme.colorScheme.background,
                            MaterialTheme.colorScheme.surface.copy(alpha = 0.92f),
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
                    folders.add(UiFolder(id = id, label = strings.folderLabel(folderCounter), icon = "📁"))
                    selectedFolderId = id
                    folderCounter += 1
                },
                onExportFolders = {
                    val path = ChatWorkspaceDesktopActions.chooseExportTarget(
                        defaultFileName = "liaotao-folders-export.json",
                        dialogTitle = strings.exportDialog,
                    )
                    if (path != null) {
                        val result = controller.exportAllFolders(path)
                        workspaceStatus = result.fold(
                            onSuccess = { strings.foldersExported(it.toString()) },
                            onFailure = { strings.exportFailed(it.message) },
                        )
                    }
                },
                onSelectFolder = { selectedFolderId = it },
                onAddConversation = {
                    controller.startConversation()
                    selectedConversationId = null
                    showSettings = false
                },
                onExportConversations = {
                    val path = ChatWorkspaceDesktopActions.chooseExportTarget(
                        defaultFileName = "liaotao-conversations-export.json",
                        dialogTitle = strings.exportDialog,
                    )
                    if (path != null) {
                        val result = controller.exportCurrentFolder(path, selectedFolderId)
                        workspaceStatus = result.fold(
                            onSuccess = { strings.conversationsExported(it.toString()) },
                            onFailure = { strings.exportFailed(it.message) },
                        )
                    }
                },
                onSelectConversation = { selectedConversationId = it },
            )

            if (showSettings) {
                SettingsWorkspace(
                    modifier = Modifier
                        .weight(1f)
                        .fillMaxHeight()
                        .padding(16.dp),
                    language = language,
                    onLanguageChange = {
                        language = it
                        DesktopUserPreferences.saveLanguage(it)
                    },
                    onBackToChat = { showSettings = false },
                )
            } else {
                ConversationWorkspace(
                    modifier = Modifier
                        .weight(1f)
                        .fillMaxHeight()
                        .padding(16.dp),
                    messages = controller.messages,
                    prompt = prompt,
                    providers = enabledProviders,
                    selectedProviderId = selectedProviderId,
                    modelMenuExpanded = modelMenuExpanded,
                    attachedFileLabel = attachedFileLabel,
                    statusMessage = workspaceStatus,
                    executionAttempts = controller.executionAttempts,
                    onPromptChange = { prompt = it },
                    onToggleModelMenu = { modelMenuExpanded = !modelMenuExpanded },
                    onSelectProvider = { providerId ->
                        selectedProviderId = providerId
                        modelMenuExpanded = false
                    },
                    onAttach = {
                        val selected = ChatWorkspaceDesktopActions.chooseAttachment(strings.attachFileDialog)
                        if (selected != null) {
                            attachedFilePath = selected.toString()
                            attachedFileLabel = selected.fileName.toString()
                            workspaceStatus = strings.attached(selected.fileName.toString())
                        }
                    },
                    onSend = {
                        val text = prompt.trim()
                        val selectedProvider = enabledProviders.firstOrNull { it.id == selectedProviderId }
                        if (text.isNotEmpty()) {
                            val withAttachment = if (!attachedFilePath.isNullOrBlank()) {
                                runCatching {
                                    val preview = ChatWorkspaceDesktopActions.readAttachmentPreview(java.nio.file.Paths.get(attachedFilePath))
                                    "$text\n\n--- Attached file context (${attachedFileLabel ?: "file"}) ---\n$preview"
                                }.getOrElse {
                                    workspaceStatus = strings.attachmentReadFailed(it.message)
                                    text
                                }
                            } else {
                                text
                            }
                            if (selectedProvider != null) {
                                controller.send(
                                    prompt = withAttachment,
                                    provider = selectedProvider.toChatProviderConfig(),
                                    availableProviders = enabledProviders.map { it.toChatProviderConfig() },
                                    projectId = selectedFolderId,
                                )
                            }
                            prompt = ""
                            attachedFileLabel = null
                            attachedFilePath = null
                        }
                    },
                    onRetry = {
                        val selectedProvider = enabledProviders.firstOrNull { it.id == selectedProviderId }
                        if (selectedProvider != null) {
                            controller.retryLast(
                                provider = selectedProvider.toChatProviderConfig(),
                                availableProviders = enabledProviders.map { it.toChatProviderConfig() },
                            )
                        }
                    },
                    onEditUserMessage = { prompt = it },
                )
            }
        }
    }
}

@Composable
private fun SettingsWorkspace(
    modifier: Modifier,
    language: AppLanguage,
    onLanguageChange: (AppLanguage) -> Unit,
    onBackToChat: () -> Unit,
) {
    val strings = LocalAppStrings.current
    Card(
        modifier = modifier,
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
    ) {
        Column(modifier = Modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(strings.settings, style = MaterialTheme.typography.headlineSmall)
                TextButton(
                    onClick = onBackToChat,
                    colors = androidx.compose.material3.ButtonDefaults.textButtonColors(
                        contentColor = MaterialTheme.colorScheme.onSurface,
                    ),
                ) { Text(strings.backToChat) }
            }
            SettingsScreen(
                selectedLanguage = language,
                onLanguageChange = onLanguageChange,
            )
        }
    }
}

internal class ChatWorkspaceController {
    private val delegate = ChatWorkspaceControllerDelegate()

    val messages = delegate.messages
    val executionAttempts = delegate.executionAttempts
    val conversations = delegate.conversations

    fun startConversation() = delegate.startConversation()

    fun send(
        prompt: String,
        provider: ChatProviderConfig,
        availableProviders: List<ChatProviderConfig>,
        projectId: String,
    ) = delegate.send(prompt, provider, availableProviders, projectId)

    fun retryLast(provider: ChatProviderConfig, availableProviders: List<ChatProviderConfig>) {
        delegate.retryLast(provider, availableProviders)
    }

    fun exportAllFolders(outputPath: java.nio.file.Path): Result<java.nio.file.Path> {
        return delegate.exportAllFolders(outputPath)
    }

    fun exportCurrentFolder(outputPath: java.nio.file.Path, projectId: String): Result<java.nio.file.Path> {
        return delegate.exportCurrentFolder(outputPath, projectId)
    }
}

private fun ConnectorSetting.toChatProviderConfig(): ChatProviderConfig {
    val type = runCatching { ConnectorType.valueOf(connectorType) }.getOrDefault(ConnectorType.OLLAMA)
    return ChatProviderConfig(
        id = id,
        connectorType = type,
        displayName = displayName,
        baseUrl = baseUrl,
        defaultModel = defaultModel.ifBlank { "gpt-4o-mini" },
    )
}

private fun legacyProviderFallbacks(): List<ConnectorSetting> {
    val now = Instant.now()
    return listOf(
        ConnectorSetting(
            id = "legacy-ollama",
            connectorType = "OLLAMA",
            displayName = "Ollama (legacy)",
            baseUrl = "http://localhost:11434",
            defaultModel = "llama3.1",
            isEnabled = true,
            secretRef = null,
            createdAt = now,
            updatedAt = now,
            connectionHealth = io.liaotao.shared.settings.ConnectionHealth.UNKNOWN,
            connectionMessage = "Legacy selector enabled",
        ),
        ConnectorSetting(
            id = "legacy-litellm",
            connectorType = "LITELLM",
            displayName = "LiteLLM (legacy)",
            baseUrl = "http://localhost:4000",
            defaultModel = "gpt-4o-mini",
            isEnabled = true,
            secretRef = null,
            createdAt = now,
            updatedAt = now,
            connectionHealth = io.liaotao.shared.settings.ConnectionHealth.UNKNOWN,
            connectionMessage = "Legacy selector enabled",
        ),
    )
}
