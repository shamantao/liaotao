/*
 * LiteLlmConnector.kt - LiteLLM provider connector.
 * Responsibilities: reuse OpenAI-compatible behavior with LiteLLM defaults
 * and connector typing for registry and settings integration.
 */

package io.liaotao.connectors.litellm

import io.liaotao.connectors.core.ConnectorType
import io.liaotao.connectors.openaicompat.JdkOpenAiCompatibleTransport
import io.liaotao.connectors.openaicompat.OpenAiCompatibleConnector
import io.liaotao.connectors.openaicompat.OpenAiCompatibleTransport
import kotlinx.serialization.json.Json

class LiteLlmConnector(
    transport: OpenAiCompatibleTransport = JdkOpenAiCompatibleTransport(),
    json: Json = Json { ignoreUnknownKeys = true },
) : OpenAiCompatibleConnector(transport = transport, json = json) {
    override val type: ConnectorType = ConnectorType.LITELLM

    companion object {
        const val DEFAULT_BASE_URL: String = "http://localhost:4000/v1"
    }
}