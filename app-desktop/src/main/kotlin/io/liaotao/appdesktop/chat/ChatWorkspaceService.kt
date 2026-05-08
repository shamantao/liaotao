/*
 * ChatWorkspaceService.kt - chat orchestration for desktop UI.
 * Responsibilities: run provider calls with fallback/retry policy, aggregate
 * attempt diagnostics, and maintain searchable conversation metadata.
 */

package io.liaotao.appdesktop.chat

import io.liaotao.connectors.core.ConnectorChatRequest
import io.liaotao.connectors.core.ConnectorChatResult
import io.liaotao.connectors.core.ConnectorExecutionConfig
import io.liaotao.connectors.core.ConnectorMessage
import io.liaotao.connectors.core.ConnectorRegistry
import io.liaotao.connectors.core.ConnectorStreamResult
import io.liaotao.connectors.core.ConnectorType
import io.liaotao.domain.conversations.Conversation
import io.liaotao.domain.routing.InMemoryExecutionHistoryRepository
import io.liaotao.domain.routing.ProviderExecutionResult
import io.liaotao.domain.routing.RoutingPolicyService
import io.liaotao.persistence.files.ConversationTranscript
import io.liaotao.persistence.files.TranscriptMessage
import java.time.Instant
import java.util.UUID

internal data class ChatProviderConfig(
    val id: String,
    val connectorType: ConnectorType,
    val displayName: String,
    val baseUrl: String,
    val defaultModel: String,
)

internal data class ChatExecutionResult(
    val reply: String,
    val source: String,
    val model: String,
    val attempts: List<io.liaotao.domain.routing.ExecutionAttempt>,
)

internal class ChatWorkspaceService {
    private val historyRepository = InMemoryExecutionHistoryRepository()
    private val routing = RoutingPolicyService(historyRepository = historyRepository)
    private val conversationHistory = mutableListOf<Conversation>()
    private val transcriptHistory = mutableListOf<ConversationTranscript>()

    fun execute(
        prompt: String,
        provider: ChatProviderConfig,
        availableProviders: List<ChatProviderConfig>,
        projectId: String,
        onAssistantChunk: (String) -> Unit = {},
    ): ChatExecutionResult {
        var finalReply = ""
        var finalSource = provider.displayName
        var finalModel = "unknown"

        val providersById = availableProviders.associateBy { it.id }
        val fallback = availableProviders
            .filter { it.id != provider.id }
            .map { it.id }

        val execution = routing.executeWithFallback(
            primaryProvider = provider.id,
            fallbackProviders = fallback,
            maxRetries = 1,
            backoffMs = 100,
        ) { providerId ->
            val targetProvider = providersById[providerId]
                ?: return@executeWithFallback ProviderExecutionResult(false, "Unknown provider")
            val connector = ConnectorRegistry.create(targetProvider.connectorType)
            val config = ConnectorExecutionConfig(
                baseUrl = targetProvider.baseUrl,
            )
            val request = ConnectorChatRequest(
                model = targetProvider.defaultModel,
                messages = listOf(ConnectorMessage(role = "user", content = prompt)),
            )
            when (val stream = connector.streamChat(config, request)) {
                is ConnectorStreamResult.Success -> {
                    val builder = StringBuilder()
                    stream.chunks.forEach { chunk ->
                        if (chunk.content.isNotEmpty()) {
                            builder.append(chunk.content)
                            onAssistantChunk(chunk.content)
                        }
                    }
                    finalReply = builder.toString()
                    finalSource = targetProvider.displayName
                    finalModel = request.model
                    if (finalReply.isNotBlank()) {
                        ProviderExecutionResult(isSuccess = true)
                    } else {
                        when (val fallbackChat = connector.chat(config, request)) {
                            is ConnectorChatResult.Success -> {
                                finalReply = fallbackChat.response.content
                                finalModel = fallbackChat.response.model
                                onAssistantChunk(finalReply)
                                ProviderExecutionResult(isSuccess = true)
                            }

                            is ConnectorChatResult.Failure -> {
                                ProviderExecutionResult(isSuccess = false, errorMessage = fallbackChat.error.message)
                            }
                        }
                    }
                }

                is ConnectorStreamResult.Failure -> {
                    when (val result = connector.chat(config, request)) {
                        is ConnectorChatResult.Success -> {
                            finalReply = result.response.content
                            finalSource = targetProvider.displayName
                            finalModel = result.response.model
                            onAssistantChunk(finalReply)
                            ProviderExecutionResult(isSuccess = true)
                        }

                        is ConnectorChatResult.Failure -> {
                            ProviderExecutionResult(isSuccess = false, errorMessage = result.error.message)
                        }
                    }
                }
            }
        }

        val now = Instant.now()
        val conversationId = UUID.randomUUID().toString()
        val replyForTranscript = finalReply.ifBlank { "No provider could answer." }
        conversationHistory.add(
            Conversation(
            id = conversationId,
            projectId = projectId,
                title = prompt.take(42),
                source = finalSource,
                model = finalModel,
                createdAt = now,
                updatedAt = now,
                lastActivityAt = now,
                archivedAt = null,
            ),
        )

        transcriptHistory.add(
            ConversationTranscript(
                conversation = conversationHistory.last(),
                messages = listOf(
                    TranscriptMessage(role = "user", content = prompt, createdAt = now),
                    TranscriptMessage(role = "assistant", content = replyForTranscript, createdAt = now),
                ),
            ),
        )

        return ChatExecutionResult(
            reply = replyForTranscript,
            source = finalSource,
            model = finalModel,
            attempts = execution.attempts,
        )
    }

    fun conversationHistory(): List<Conversation> = conversationHistory.toList()

    fun transcriptHistory(): List<ConversationTranscript> = transcriptHistory.toList()
}