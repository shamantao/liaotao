/*
 * ChatWorkspaceController.kt - controller delegate for chat workspace.
 * Responsibilities: hold message state, invoke chat execution service,
 * support retry, and expose export actions.
 */

package io.liaotao.appdesktop.chat

import androidx.compose.runtime.mutableStateListOf
import io.liaotao.connectors.core.ConnectorType
import io.liaotao.domain.routing.ExecutionAttempt
import java.nio.file.Path
import java.time.Instant
import java.util.UUID

internal class ChatWorkspaceControllerDelegate {
    private val chatService = ChatWorkspaceService()

    val messages = mutableStateListOf<ChatUiMessage>()
    val executionAttempts = mutableStateListOf<ExecutionAttempt>()
    val conversations = mutableStateListOf<io.liaotao.domain.conversations.Conversation>()

    private var lastPrompt: String? = null
    private var lastProvider: ChatProviderConfig? = null

    fun startConversation() {
        messages.clear()
        lastPrompt = null
    }

    fun send(
        prompt: String,
        provider: ChatProviderConfig,
        availableProviders: List<ChatProviderConfig>,
        projectId: String,
    ) {
        val sentAt = Instant.now()
        messages.add(
            ChatUiMessage(
                id = UUID.randomUUID().toString(),
                role = "user",
                content = prompt,
                source = provider.displayName,
                model = "user-input",
                createdAt = sentAt,
                completedAt = sentAt,
                tokensUsed = estimateTokens(prompt),
            ),
        )

        val draftId = UUID.randomUUID().toString()
        messages.add(
            ChatUiMessage(
                id = draftId,
                role = "assistant",
                content = "",
                source = provider.displayName,
                model = "streaming",
                createdAt = Instant.now(),
                completedAt = null,
                tokensUsed = null,
            ),
        )

        val result = chatService.execute(prompt, provider, availableProviders, projectId) { chunk ->
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
            messages[draftIndex] = draft.copy(
                content = finalContent,
                source = result.source,
                model = result.model,
                completedAt = Instant.now(),
                tokensUsed = estimateTokens(finalContent),
            )
        }

        conversations.clear()
        conversations.addAll(chatService.conversationHistory())

        lastPrompt = prompt
        lastProvider = provider
    }

    fun retryLast(provider: ChatProviderConfig, availableProviders: List<ChatProviderConfig>) {
        val prompt = lastPrompt ?: return
        val effectiveProvider = if (provider.connectorType != ConnectorType.OPENAI_COMPAT) {
            provider
        } else {
            lastProvider ?: provider
        }
        val projectId = conversations.firstOrNull()?.projectId ?: "default"
        send(prompt, effectiveProvider, availableProviders, projectId)
    }

    fun exportAllFolders(outputPath: Path): Result<Path> {
        return ChatWorkspaceDesktopActions.exportAllConversations(chatService.transcriptHistory(), outputPath)
    }

    fun exportCurrentFolder(outputPath: Path, projectId: String): Result<Path> {
        return ChatWorkspaceDesktopActions.exportProjectConversations(
            transcripts = chatService.transcriptHistory(),
            projectId = projectId,
            outputPath = outputPath,
        )
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

private fun estimateTokens(content: String): Int {
    val trimmed = content.trim()
    if (trimmed.isEmpty()) {
        return 0
    }
    return trimmed.split(Regex("\\s+")).size
}
