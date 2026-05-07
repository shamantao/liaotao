/*
 * OpenAiCompatibleConnectorTest.kt - tests for normalized connector behavior.
 * Responsibilities: validate error normalization, model discovery parsing,
 * and streaming chunk extraction for OpenAI-compatible connectors.
 */

package io.liaotao.connectors.openaicompat

import io.liaotao.connectors.core.ConnectorChatRequest
import io.liaotao.connectors.core.ConnectorChatResult
import io.liaotao.connectors.core.ConnectorExecutionConfig
import io.liaotao.connectors.core.ConnectorMessage
import io.liaotao.connectors.core.ConnectorModelsResult
import io.liaotao.connectors.core.ConnectorStreamResult
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlin.test.assertTrue

class OpenAiCompatibleConnectorTest {
    @Test
    fun `discover models parses data array`() {
        val connector = OpenAiCompatibleConnector(
            transport = FakeTransport(
                getResponse = TransportResponse(
                    200,
                    """{"data":[{"id":"gpt-4o"},{"id":"qwen3"}]}""",
                ),
            ),
        )

        val result = connector.discoverModels(ConnectorExecutionConfig(baseUrl = "http://provider/v1"))
        val success = assertIs<ConnectorModelsResult.Success>(result)
        assertEquals(listOf("gpt-4o", "qwen3"), success.models.map { it.id })
    }

    @Test
    fun `chat normalizes http errors`() {
        val connector = OpenAiCompatibleConnector(
            transport = FakeTransport(
                postResponse = TransportResponse(
                    401,
                    """{"error":{"message":"invalid api key"}}""",
                ),
            ),
        )

        val result = connector.chat(
            config = ConnectorExecutionConfig(baseUrl = "http://provider/v1", apiKey = "bad"),
            request = ConnectorChatRequest(
                model = "gpt-4o",
                messages = listOf(ConnectorMessage(role = "user", content = "hello")),
            ),
        )

        val failure = assertIs<ConnectorChatResult.Failure>(result)
        assertEquals("http_401", failure.error.code)
        assertEquals("invalid api key", failure.error.message)
    }

    @Test
    fun `stream parses sse data and done marker`() {
        val connector = OpenAiCompatibleConnector(
            transport = FakeTransport(
                streamLines = sequenceOf(
                    "data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"},\"finish_reason\":null}]}",
                    "data: {\"choices\":[{\"delta\":{\"content\":\"lo\"},\"finish_reason\":\"stop\"}]}",
                    "data: [DONE]",
                ),
            ),
        )

        val result = connector.streamChat(
            config = ConnectorExecutionConfig(baseUrl = "http://provider/v1"),
            request = ConnectorChatRequest(
                model = "gpt-4o",
                messages = listOf(ConnectorMessage(role = "user", content = "hello")),
            ),
        )

        val success = assertIs<ConnectorStreamResult.Success>(result)
        val chunks = success.chunks.toList()
        assertEquals("Hello", chunks.take(2).joinToString(separator = "") { it.content })
        assertTrue(chunks.last().isFinal)
    }
}

private class FakeTransport(
    private val getResponse: TransportResponse = TransportResponse(200, "{}"),
    private val postResponse: TransportResponse = TransportResponse(200, """{"id":"x","model":"m","choices":[{"message":{"content":"ok"}}]}"""),
    private val streamLines: Sequence<String> = emptySequence(),
) : OpenAiCompatibleTransport {
    override fun getJson(url: String, headers: Map<String, String>): TransportResponse = getResponse

    override fun postJson(url: String, headers: Map<String, String>, body: String): TransportResponse = postResponse

    override fun streamPostJson(url: String, headers: Map<String, String>, body: String): Sequence<String> = streamLines
}