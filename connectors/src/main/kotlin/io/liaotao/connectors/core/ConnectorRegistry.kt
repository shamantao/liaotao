/*
 * ConnectorRegistry.kt - connector factory registry.
 * Responsibilities: centralize connector instantiation by type to support
 * settings-driven routing and validation in desktop UI and services.
 */

package io.liaotao.connectors.core

import io.liaotao.connectors.aitao.AitaoConnector
import io.liaotao.connectors.litellm.LiteLlmConnector
import io.liaotao.connectors.ollama.OllamaConnector
import io.liaotao.connectors.openaicompat.OpenAiCompatibleConnector

object ConnectorRegistry {
    fun create(type: ConnectorType): AiConnector {
        return when (type) {
            ConnectorType.OPENAI_COMPAT -> OpenAiCompatibleConnector()
            ConnectorType.OLLAMA -> OllamaConnector()
            ConnectorType.LITELLM -> LiteLlmConnector()
            ConnectorType.AITAO -> AitaoConnector()
        }
    }

    fun create(type: String): AiConnector? {
        val normalized = type.trim().uppercase()
        return runCatching { ConnectorType.valueOf(normalized) }
            .getOrNull()
            ?.let { create(it) }
    }
}