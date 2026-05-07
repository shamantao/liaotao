/*
 * ConnectorContracts.kt - normalized connector contracts for Liaotao.
 * Responsibilities: define provider-agnostic DTOs and execution interfaces
 * used by connector implementations and settings validation flows.
 */

package io.liaotao.connectors.core

enum class ConnectorType {
    OPENAI_COMPAT,
    OLLAMA,
    LITELLM,
    AITAO,
}

data class ConnectorExecutionConfig(
    val baseUrl: String,
    val apiKey: String? = null,
    val headers: Map<String, String> = emptyMap(),
)

data class ConnectorMessage(
    val role: String,
    val content: String,
)

data class ConnectorChatRequest(
    val model: String,
    val messages: List<ConnectorMessage>,
    val temperature: Double? = null,
    val maxTokens: Int? = null,
)

data class ConnectorChatResponse(
    val content: String,
    val model: String,
    val providerConversationId: String? = null,
)

data class ConnectorChatChunk(
    val content: String,
    val isFinal: Boolean,
)

data class ConnectorModel(
    val id: String,
    val displayName: String,
    val supportsStreaming: Boolean,
)

data class ConnectorError(
    val code: String,
    val message: String,
    val retryable: Boolean,
    val httpStatus: Int? = null,
)

data class ConnectorValidationResult(
    val isValid: Boolean,
    val message: String,
    val latencyMs: Long,
    val error: ConnectorError? = null,
)

sealed interface ConnectorModelsResult {
    data class Success(val models: List<ConnectorModel>) : ConnectorModelsResult
    data class Failure(val error: ConnectorError) : ConnectorModelsResult
}

sealed interface ConnectorChatResult {
    data class Success(val response: ConnectorChatResponse) : ConnectorChatResult
    data class Failure(val error: ConnectorError) : ConnectorChatResult
}

sealed interface ConnectorStreamResult {
    data class Success(val chunks: Sequence<ConnectorChatChunk>) : ConnectorStreamResult
    data class Failure(val error: ConnectorError) : ConnectorStreamResult
}

interface AiConnector {
    val type: ConnectorType

    fun validateConfiguration(config: ConnectorExecutionConfig): ConnectorValidationResult

    fun discoverModels(config: ConnectorExecutionConfig): ConnectorModelsResult

    fun chat(config: ConnectorExecutionConfig, request: ConnectorChatRequest): ConnectorChatResult

    fun streamChat(config: ConnectorExecutionConfig, request: ConnectorChatRequest): ConnectorStreamResult
}