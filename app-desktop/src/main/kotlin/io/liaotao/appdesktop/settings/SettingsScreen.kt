/*
 * SettingsScreen.kt - settings management UI for providers and MCP servers.
 * Responsibilities: edit non-secret connector/MCP settings and display
 * connection validation status for configured endpoints.
 */

package io.liaotao.appdesktop.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import io.liaotao.connectors.aitao.AitaoConnector
import io.liaotao.connectors.core.ConnectorExecutionConfig
import io.liaotao.connectors.core.ConnectorRegistry
import io.liaotao.connectors.core.ConnectorType
import io.liaotao.connectors.litellm.LiteLlmConnector
import io.liaotao.connectors.ollama.OllamaConnector
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.time.Duration

@Composable
fun SettingsScreen() {
    val providers = remember {
        mutableStateListOf(
            ProviderSettingUiState("Ollama", ConnectorType.OLLAMA, OllamaConnector.DEFAULT_BASE_URL),
            ProviderSettingUiState("LiteLLM", ConnectorType.LITELLM, LiteLlmConnector.DEFAULT_BASE_URL),
            ProviderSettingUiState("Aitao", ConnectorType.AITAO, AitaoConnector.DEFAULT_BASE_URL),
        )
    }

    var mcpName by remember { mutableStateOf("Main MCP") }
    var mcpUrl by remember { mutableStateOf("http://localhost:3333") }
    var mcpStatus by remember { mutableStateOf("Not checked") }

    Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
        Text("Provider Settings", style = MaterialTheme.typography.titleMedium)

        providers.forEachIndexed { index, provider ->
            ProviderCard(
                state = provider,
                onBaseUrlChange = { value -> providers[index] = provider.copy(baseUrl = value) },
                onSecretRefChange = { value -> providers[index] = provider.copy(secretRef = value) },
                onValidate = {
                    val connector = ConnectorRegistry.create(provider.type)
                    val result = connector.validateConfiguration(
                        ConnectorExecutionConfig(
                            baseUrl = provider.baseUrl,
                            apiKey = null,
                            headers = if (provider.secretRef.isNotBlank()) {
                                mapOf("X-Secret-Ref" to provider.secretRef)
                            } else {
                                emptyMap()
                            },
                        ),
                    )
                    val updatedStatus = if (result.isValid) {
                        "Healthy (${result.latencyMs} ms)"
                    } else {
                        "Degraded: ${result.message}"
                    }
                    providers[index] = provider.copy(connectionStatus = updatedStatus)
                },
            )
        }

        Text("MCP Server Settings", style = MaterialTheme.typography.titleMedium)
        Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.25f))) {
            Column(
                modifier = Modifier.fillMaxWidth().padding(12.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                OutlinedTextField(
                    value = mcpName,
                    onValueChange = { mcpName = it },
                    label = { Text("Server Name") },
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = mcpUrl,
                    onValueChange = { mcpUrl = it },
                    label = { Text("Server URL") },
                    modifier = Modifier.fillMaxWidth(),
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Button(onClick = {
                        mcpStatus = validateMcpEndpoint(mcpUrl)
                    }) {
                        Text("Validate")
                    }
                }
                Text("Status: $mcpStatus", style = MaterialTheme.typography.bodySmall)
            }
        }
    }
}

@Composable
private fun ProviderCard(
    state: ProviderSettingUiState,
    onBaseUrlChange: (String) -> Unit,
    onSecretRefChange: (String) -> Unit,
    onValidate: () -> Unit,
) {
    Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.25f))) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(state.name, style = MaterialTheme.typography.titleSmall)
            OutlinedTextField(
                value = state.baseUrl,
                onValueChange = onBaseUrlChange,
                label = { Text("Base URL") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = state.secretRef,
                onValueChange = onSecretRefChange,
                label = { Text("Secret Ref (stored in OS keychain)") },
                modifier = Modifier.fillMaxWidth(),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = onValidate) {
                    Text("Validate")
                }
            }
            Text("Status: ${state.connectionStatus}", style = MaterialTheme.typography.bodySmall)
        }
    }
}

private fun validateMcpEndpoint(url: String): String {
    val client = HttpClient.newBuilder()
        .connectTimeout(Duration.ofSeconds(3))
        .build()
    return try {
        val request = HttpRequest.newBuilder()
            .GET()
            .timeout(Duration.ofSeconds(3))
            .uri(URI.create(url))
            .build()
        val response = client.send(request, HttpResponse.BodyHandlers.discarding())
        if (response.statusCode() in 200..499) {
            "Healthy (${response.statusCode()})"
        } else {
            "Degraded (${response.statusCode()})"
        }
    } catch (exception: Exception) {
        "Offline: ${exception.message ?: "Unreachable"}"
    }
}

private data class ProviderSettingUiState(
    val name: String,
    val type: ConnectorType,
    val baseUrl: String,
    val secretRef: String = "",
    val connectionStatus: String = "Not checked",
)