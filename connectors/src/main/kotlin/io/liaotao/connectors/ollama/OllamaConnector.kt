/*
 * OllamaConnector.kt - Ollama connector with local-first defaults.
 * Responsibilities: provide an Ollama-ready connector configuration and
 * model discovery via Ollama tags API when available.
 */

package io.liaotao.connectors.ollama

import io.liaotao.connectors.core.ConnectorExecutionConfig
import io.liaotao.connectors.core.ConnectorModel
import io.liaotao.connectors.core.ConnectorModelsResult
import io.liaotao.connectors.core.ConnectorType
import io.liaotao.connectors.openaicompat.JdkOpenAiCompatibleTransport
import io.liaotao.connectors.openaicompat.OpenAiCompatibleConnector
import io.liaotao.connectors.openaicompat.OpenAiCompatibleTransport
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

class OllamaConnector(
    private val transport: OpenAiCompatibleTransport = JdkOpenAiCompatibleTransport(),
    private val json: Json = Json { ignoreUnknownKeys = true },
) : OpenAiCompatibleConnector(transport = transport, json = json) {
    override val type: ConnectorType = ConnectorType.OLLAMA

    override fun discoverModels(config: ConnectorExecutionConfig): ConnectorModelsResult {
        val effectiveBaseUrl = if (config.baseUrl.isBlank()) DEFAULT_BASE_URL else config.baseUrl

        return try {
            val response = transport.getJson(
                url = effectiveBaseUrl.trimEnd('/') + "/api/tags",
                headers = emptyMap(),
            )
            if (response.statusCode !in 200..299) {
                return super.discoverModels(config.copy(baseUrl = effectiveBaseUrl))
            }

            val payload = json.parseToJsonElement(response.body).jsonObject
            val models = payload["models"]?.jsonArray
                ?.mapNotNull { model ->
                    val name = model.jsonObject["name"]?.jsonPrimitive?.contentOrNull ?: return@mapNotNull null
                    ConnectorModel(id = name, displayName = name, supportsStreaming = true)
                }
                ?: emptyList()

            ConnectorModelsResult.Success(models)
        } catch (_: Exception) {
            super.discoverModels(config.copy(baseUrl = effectiveBaseUrl))
        }
    }

    companion object {
        const val DEFAULT_BASE_URL: String = "http://localhost:11434"
        const val DEFAULT_MODEL: String = "qwen3:latest"
    }
}