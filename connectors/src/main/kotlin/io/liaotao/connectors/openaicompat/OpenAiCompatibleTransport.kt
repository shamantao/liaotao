/*
 * OpenAiCompatibleTransport.kt - transport abstraction for connector HTTP calls.
 * Responsibilities: decouple connector logic from networking implementation
 * and provide test-friendly request/response primitives.
 */

package io.liaotao.connectors.openaicompat

import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.time.Duration
import kotlin.streams.asSequence

data class TransportResponse(
    val statusCode: Int,
    val body: String,
)

interface OpenAiCompatibleTransport {
    fun getJson(url: String, headers: Map<String, String>): TransportResponse

    fun postJson(url: String, headers: Map<String, String>, body: String): TransportResponse

    fun streamPostJson(url: String, headers: Map<String, String>, body: String): Sequence<String>
}

class JdkOpenAiCompatibleTransport(
    private val timeout: Duration = Duration.ofSeconds(30),
) : OpenAiCompatibleTransport {
    private val client: HttpClient = HttpClient.newBuilder()
        .connectTimeout(timeout)
        .build()

    override fun getJson(url: String, headers: Map<String, String>): TransportResponse {
        val requestBuilder = HttpRequest.newBuilder()
            .GET()
            .timeout(timeout)
            .uri(URI.create(url))
            .header("Accept", "application/json")

        headers.forEach { (name, value) -> requestBuilder.header(name, value) }

        val response = client.send(requestBuilder.build(), HttpResponse.BodyHandlers.ofString())
        return TransportResponse(statusCode = response.statusCode(), body = response.body())
    }

    override fun postJson(url: String, headers: Map<String, String>, body: String): TransportResponse {
        val requestBuilder = HttpRequest.newBuilder()
            .POST(HttpRequest.BodyPublishers.ofString(body))
            .timeout(timeout)
            .uri(URI.create(url))
            .header("Content-Type", "application/json")
            .header("Accept", "application/json")

        headers.forEach { (name, value) -> requestBuilder.header(name, value) }

        val response = client.send(requestBuilder.build(), HttpResponse.BodyHandlers.ofString())
        return TransportResponse(statusCode = response.statusCode(), body = response.body())
    }

    override fun streamPostJson(url: String, headers: Map<String, String>, body: String): Sequence<String> {
        val requestBuilder = HttpRequest.newBuilder()
            .POST(HttpRequest.BodyPublishers.ofString(body))
            .timeout(timeout)
            .uri(URI.create(url))
            .header("Content-Type", "application/json")
            .header("Accept", "text/event-stream")

        headers.forEach { (name, value) -> requestBuilder.header(name, value) }

        val response = client.send(requestBuilder.build(), HttpResponse.BodyHandlers.ofLines())
        if (response.statusCode() !in 200..299) {
            throw IllegalStateException("Streaming request failed with status ${response.statusCode()}")
        }

        return response.body().asSequence()
    }
}