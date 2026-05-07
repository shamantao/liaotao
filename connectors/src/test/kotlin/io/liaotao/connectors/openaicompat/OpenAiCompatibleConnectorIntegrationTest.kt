/*
 * OpenAiCompatibleConnectorIntegrationTest.kt - local HTTP integration tests.
 * Responsibilities: validate connector behavior over real HTTP transport
 * against a local fake OpenAI-compatible server.
 */

package io.liaotao.connectors.openaicompat

import com.sun.net.httpserver.HttpExchange
import com.sun.net.httpserver.HttpServer
import io.liaotao.connectors.core.ConnectorChatRequest
import io.liaotao.connectors.core.ConnectorChatResult
import io.liaotao.connectors.core.ConnectorExecutionConfig
import io.liaotao.connectors.core.ConnectorMessage
import io.liaotao.connectors.core.ConnectorModelsResult
import io.liaotao.connectors.core.ConnectorStreamResult
import java.net.InetSocketAddress
import kotlin.test.AfterTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlin.test.assertTrue

class OpenAiCompatibleConnectorIntegrationTest {
    private var server: HttpServer? = null

    @AfterTest
    fun stopServer() {
        server?.stop(0)
    }

    @Test
    fun `connector performs validate discover chat and stream over http`() {
        server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0).apply {
            createContext("/v1/models") { exchange ->
                respondJson(exchange, 200, """{"data":[{"id":"demo-model"}]}""")
            }
            createContext("/v1/chat/completions") { exchange ->
                val accept = exchange.requestHeaders.getFirst("Accept") ?: ""
                if (accept.contains("text/event-stream")) {
                    val payload = """
                        data: {"choices":[{"delta":{"content":"Hel"},"finish_reason":null}]}

                        data: {"choices":[{"delta":{"content":"lo"},"finish_reason":"stop"}]}

                        data: [DONE]

                    """.trimIndent()
                    respondPlain(exchange, 200, payload)
                } else {
                    respondJson(
                        exchange,
                        200,
                        """{"id":"r1","model":"demo-model","choices":[{"message":{"content":"Hello"}}]}""",
                    )
                }
            }
            start()
        }

        val port = server!!.address.port
        val connector = OpenAiCompatibleConnector(transport = JdkOpenAiCompatibleTransport())
        val config = ConnectorExecutionConfig(baseUrl = "http://127.0.0.1:$port/v1")

        val validation = connector.validateConfiguration(config)
        assertTrue(validation.isValid)

        val models = connector.discoverModels(config)
        val modelsSuccess = assertIs<ConnectorModelsResult.Success>(models)
        assertEquals(listOf("demo-model"), modelsSuccess.models.map { it.id })

        val chat = connector.chat(
            config,
            ConnectorChatRequest(
                model = "demo-model",
                messages = listOf(ConnectorMessage(role = "user", content = "hello")),
            ),
        )
        val chatSuccess = assertIs<ConnectorChatResult.Success>(chat)
        assertEquals("Hello", chatSuccess.response.content)

        val stream = connector.streamChat(
            config,
            ConnectorChatRequest(
                model = "demo-model",
                messages = listOf(ConnectorMessage(role = "user", content = "hello")),
            ),
        )
        val streamSuccess = assertIs<ConnectorStreamResult.Success>(stream)
        val chunks = streamSuccess.chunks.toList()
        assertEquals("Hello", chunks.take(2).joinToString(separator = "") { it.content })
        assertTrue(chunks.last().isFinal)
    }

    private fun respondJson(exchange: HttpExchange, status: Int, json: String) {
        exchange.responseHeaders.add("Content-Type", "application/json")
        respondPlain(exchange, status, json)
    }

    private fun respondPlain(exchange: HttpExchange, status: Int, body: String) {
        val bytes = body.toByteArray(Charsets.UTF_8)
        exchange.sendResponseHeaders(status, bytes.size.toLong())
        exchange.responseBody.use { it.write(bytes) }
    }
}