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
import io.liaotao.connectors.ollama.OllamaConnector
import io.liaotao.connectors.litellm.LiteLlmConnector
import io.liaotao.connectors.aitao.AitaoConnector
import io.liaotao.domain.conversations.Conversation
import io.liaotao.domain.routing.InMemoryExecutionHistoryRepository
import io.liaotao.domain.routing.ProviderExecutionResult
import io.liaotao.domain.routing.RoutingPolicyService
import java.time.Instant
import java.util.UUID

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

    fun execute(
        prompt: String,
        provider: ConnectorType,
        onAssistantChunk: (String) -> Unit = {},
    ): ChatExecutionResult {
        var finalReply = ""
        var finalSource = provider.name
        var finalModel = "unknown"

        val fallback = ConnectorType.entries
            .filter { it != provider }
            .map { it.name }

        val execution = routing.executeWithFallback(
            primaryProvider = provider.name,
            fallbackProviders = fallback,
            maxRetries = 1,
            backoffMs = 100,
        ) { providerId ->
            val type = runCatching { ConnectorType.valueOf(providerId) }.getOrNull()
                ?: return@executeWithFallback ProviderExecutionResult(false, "Unknown provider")
            val connector = ConnectorRegistry.create(type)
            val config = ConnectorExecutionConfig(
                baseUrl = defaultBaseUrl(type),
            )
            val request = ConnectorChatRequest(
                model = defaultModel(type),
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
                    finalSource = providerId
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
                            finalSource = providerId
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
        conversationHistory.add(
            Conversation(
                id = UUID.randomUUID().toString(),
                projectId = "default",
                title = prompt.take(42),
                source = finalSource,
                model = finalModel,
                createdAt = now,
                updatedAt = now,
                lastActivityAt = now,
                archivedAt = null,
            ),
        )

        return ChatExecutionResult(
            reply = finalReply.ifBlank { "No provider could answer." },
            source = finalSource,
            model = finalModel,
            attempts = execution.attempts,
        )
    }

    fun conversationHistory(): List<Conversation> = conversationHistory.toList()

    private fun defaultBaseUrl(type: ConnectorType): String {
        return when (type) {
            ConnectorType.OLLAMA -> OllamaConnector.DEFAULT_BASE_URL
            ConnectorType.LITELLM -> LiteLlmConnector.DEFAULT_BASE_URL
            ConnectorType.AITAO -> AitaoConnector.DEFAULT_BASE_URL
            ConnectorType.OPENAI_COMPAT -> LiteLlmConnector.DEFAULT_BASE_URL
        }
    }

    private fun defaultModel(type: ConnectorType): String {
        return when (type) {
            ConnectorType.OLLAMA -> OllamaConnector.DEFAULT_MODEL
            ConnectorType.LITELLM -> "gpt-4o-mini"
            ConnectorType.AITAO -> "aitao-default"
            ConnectorType.OPENAI_COMPAT -> "gpt-4o-mini"
        }
    }
}