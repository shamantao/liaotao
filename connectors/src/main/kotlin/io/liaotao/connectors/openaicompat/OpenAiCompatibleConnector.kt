/*
 * OpenAiCompatibleConnector.kt - reusable OpenAI-compatible connector base.
 * Responsibilities: implement validation, model discovery, chat completion,
 * streaming chunk extraction, and provider error normalization.
 */

package io.liaotao.connectors.openaicompat

import io.liaotao.connectors.core.AiConnector
import io.liaotao.connectors.core.ConnectorChatChunk
import io.liaotao.connectors.core.ConnectorChatRequest
import io.liaotao.connectors.core.ConnectorChatResponse
import io.liaotao.connectors.core.ConnectorChatResult
import io.liaotao.connectors.core.ConnectorError
import io.liaotao.connectors.core.ConnectorExecutionConfig
import io.liaotao.connectors.core.ConnectorModel
import io.liaotao.connectors.core.ConnectorModelsResult
import io.liaotao.connectors.core.ConnectorStreamResult
import io.liaotao.connectors.core.ConnectorType
import io.liaotao.connectors.core.ConnectorValidationResult
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

open class OpenAiCompatibleConnector(
    private val transport: OpenAiCompatibleTransport = JdkOpenAiCompatibleTransport(),
    private val json: Json = Json { ignoreUnknownKeys = true },
) : AiConnector {
    override val type: ConnectorType = ConnectorType.OPENAI_COMPAT

    override fun validateConfiguration(config: ConnectorExecutionConfig): ConnectorValidationResult {
        val start = System.currentTimeMillis()
        return try {
            val response = transport.getJson(
                url = normalizeBaseUrl(config.baseUrl) + "/models",
                headers = authHeaders(config),
            )
            if (response.statusCode in 200..299) {
                ConnectorValidationResult(
                    isValid = true,
                    message = "Connection validated",
                    latencyMs = System.currentTimeMillis() - start,
                )
            } else {
                val error = normalizeError(response.statusCode, response.body)
                ConnectorValidationResult(
                    isValid = false,
                    message = error.message,
                    latencyMs = System.currentTimeMillis() - start,
                    error = error,
                )
            }
        } catch (exception: Exception) {
            val error = ConnectorError(
                code = "network_error",
                message = exception.message ?: "Unable to reach provider",
                retryable = true,
            )
            ConnectorValidationResult(
                isValid = false,
                message = error.message,
                latencyMs = System.currentTimeMillis() - start,
                error = error,
            )
        }
    }

    override fun discoverModels(config: ConnectorExecutionConfig): ConnectorModelsResult {
        return try {
            val response = transport.getJson(
                url = normalizeBaseUrl(config.baseUrl) + "/models",
                headers = authHeaders(config),
            )

            if (response.statusCode !in 200..299) {
                return ConnectorModelsResult.Failure(normalizeError(response.statusCode, response.body))
            }

            val payload = json.parseToJsonElement(response.body).jsonObject
            val models = payload["data"]?.jsonArray
                ?.mapNotNull { item ->
                    val id = item.jsonObject["id"]?.jsonPrimitive?.contentOrNull ?: return@mapNotNull null
                    ConnectorModel(id = id, displayName = id, supportsStreaming = true)
                }
                ?: emptyList()

            ConnectorModelsResult.Success(models)
        } catch (exception: Exception) {
            ConnectorModelsResult.Failure(
                ConnectorError(
                    code = "model_discovery_failed",
                    message = exception.message ?: "Model discovery failed",
                    retryable = true,
                ),
            )
        }
    }

    override fun chat(config: ConnectorExecutionConfig, request: ConnectorChatRequest): ConnectorChatResult {
        return try {
            val response = transport.postJson(
                url = normalizeBaseUrl(config.baseUrl) + "/chat/completions",
                headers = authHeaders(config),
                body = buildChatBody(request, stream = false),
            )

            if (response.statusCode !in 200..299) {
                return ConnectorChatResult.Failure(normalizeError(response.statusCode, response.body))
            }

            val payload = json.parseToJsonElement(response.body).jsonObject
            val model = payload["model"]?.jsonPrimitive?.contentOrNull ?: request.model
            val conversationId = payload["id"]?.jsonPrimitive?.contentOrNull
            val content = payload["choices"]?.jsonArray
                ?.firstOrNull()
                ?.jsonObject
                ?.get("message")
                ?.jsonObject
                ?.get("content")
                ?.jsonPrimitive
                ?.contentOrNull
                ?: ""

            ConnectorChatResult.Success(
                ConnectorChatResponse(
                    content = content,
                    model = model,
                    providerConversationId = conversationId,
                ),
            )
        } catch (exception: Exception) {
            ConnectorChatResult.Failure(
                ConnectorError(
                    code = "chat_failed",
                    message = exception.message ?: "Chat request failed",
                    retryable = true,
                ),
            )
        }
    }

    override fun streamChat(config: ConnectorExecutionConfig, request: ConnectorChatRequest): ConnectorStreamResult {
        return try {
            val lines = transport.streamPostJson(
                url = normalizeBaseUrl(config.baseUrl) + "/chat/completions",
                headers = authHeaders(config),
                body = buildChatBody(request, stream = true),
            )

            val chunks = lines
                .mapNotNull { line ->
                    val trimmed = line.trim()
                    if (!trimmed.startsWith("data:")) {
                        return@mapNotNull null
                    }
                    val data = trimmed.removePrefix("data:").trim()
                    if (data == "[DONE]") {
                        return@mapNotNull ConnectorChatChunk(content = "", isFinal = true)
                    }

                    val payload = json.parseToJsonElement(data).jsonObject
                    val delta = payload["choices"]?.jsonArray
                        ?.firstOrNull()
                        ?.jsonObject
                        ?.get("delta")
                        ?.jsonObject
                        ?.get("content")
                        ?.jsonPrimitive
                        ?.contentOrNull
                        .orEmpty()

                    val finishReason = payload["choices"]?.jsonArray
                        ?.firstOrNull()
                        ?.jsonObject
                        ?.get("finish_reason")
                        ?.jsonPrimitive
                        ?.contentOrNull

                    ConnectorChatChunk(content = delta, isFinal = finishReason != null)
                }

            ConnectorStreamResult.Success(chunks)
        } catch (exception: Exception) {
            ConnectorStreamResult.Failure(
                ConnectorError(
                    code = "stream_failed",
                    message = exception.message ?: "Streaming failed",
                    retryable = true,
                ),
            )
        }
    }

    protected fun normalizeBaseUrl(baseUrl: String): String = baseUrl.trimEnd('/')

    protected fun authHeaders(config: ConnectorExecutionConfig): Map<String, String> {
        val headers = mutableMapOf<String, String>()
        headers.putAll(config.headers)
        if (!config.apiKey.isNullOrBlank()) {
            headers["Authorization"] = "Bearer ${config.apiKey}"
        }
        return headers
    }

    protected fun normalizeError(statusCode: Int, body: String): ConnectorError {
        val message = runCatching {
            val payload = json.parseToJsonElement(body).jsonObject
            payload["error"]?.jsonObject?.get("message")?.jsonPrimitive?.contentOrNull
                ?: payload["message"]?.jsonPrimitive?.contentOrNull
        }.getOrNull() ?: "Provider request failed"

        return ConnectorError(
            code = "http_$statusCode",
            message = message,
            retryable = statusCode == 429 || statusCode >= 500,
            httpStatus = statusCode,
        )
    }

    private fun buildChatBody(request: ConnectorChatRequest, stream: Boolean): String {
        val payload = buildJsonObject {
            put("model", JsonPrimitive(request.model))
            put("stream", JsonPrimitive(stream))
            request.temperature?.let { put("temperature", JsonPrimitive(it)) }
            request.maxTokens?.let { put("max_tokens", JsonPrimitive(it)) }
            put(
                "messages",
                buildJsonArray {
                    request.messages.forEach { message ->
                        add(
                            buildJsonObject {
                                put("role", JsonPrimitive(message.role))
                                put("content", JsonPrimitive(message.content))
                            },
                        )
                    }
                },
            )
        }
        return json.encodeToString(kotlinx.serialization.json.JsonElement.serializer(), payload)
    }
}