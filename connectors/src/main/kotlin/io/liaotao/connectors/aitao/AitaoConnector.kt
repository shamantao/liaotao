/*
 * AitaoConnector.kt - Aitao connector entrypoint.
 * Responsibilities: provide Aitao-specific defaults while using the shared
 * OpenAI-compatible request, streaming, and error normalization behavior.
 */

package io.liaotao.connectors.aitao

import io.liaotao.connectors.core.ConnectorType
import io.liaotao.connectors.openaicompat.JdkOpenAiCompatibleTransport
import io.liaotao.connectors.openaicompat.OpenAiCompatibleConnector
import io.liaotao.connectors.openaicompat.OpenAiCompatibleTransport
import kotlinx.serialization.json.Json

class AitaoConnector(
    transport: OpenAiCompatibleTransport = JdkOpenAiCompatibleTransport(),
    json: Json = Json { ignoreUnknownKeys = true },
) : OpenAiCompatibleConnector(transport = transport, json = json) {
    override val type: ConnectorType = ConnectorType.AITAO

    companion object {
        const val DEFAULT_BASE_URL: String = "http://localhost:8080/v1"
    }
}