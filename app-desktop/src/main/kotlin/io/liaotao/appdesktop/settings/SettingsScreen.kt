/*
 * SettingsScreen.kt - user-facing settings and provider CRUD.
 * Responsibilities: manage provider catalog (create/read/update/delete),
 * validate provider connectivity, and edit MCP endpoint settings.
 */

package io.liaotao.appdesktop.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import io.liaotao.connectors.aitao.AitaoConnector
import io.liaotao.connectors.core.ConnectorExecutionConfig
import io.liaotao.connectors.core.ConnectorRegistry
import io.liaotao.connectors.core.ConnectorType
import io.liaotao.connectors.litellm.LiteLlmConnector
import io.liaotao.connectors.ollama.OllamaConnector
import io.liaotao.shared.settings.ConnectionHealth
import io.liaotao.shared.settings.ConnectorSetting
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.time.Duration
import java.time.Instant
import java.util.UUID

@Composable
fun SettingsScreen() {
    val service = remember { ProviderSettingsService() }
    val providers = remember {
        mutableStateListOf<ProviderSettingUiState>()
    }

    var newName by remember { mutableStateOf("") }
    var newType by remember { mutableStateOf(ConnectorType.OLLAMA) }
    var newBaseUrl by remember { mutableStateOf(defaultBaseUrl(newType)) }
    var newModel by remember { mutableStateOf(defaultModel(newType)) }
    var newSecretRef by remember { mutableStateOf("") }

    var mcpName by remember { mutableStateOf("Main MCP") }
    var mcpUrl by remember { mutableStateOf("http://localhost:3333") }
    var mcpStatus by remember { mutableStateOf("Not checked") }

    fun refreshProviders() {
        providers.clear()
        providers.addAll(service.listAll().map { it.toUiState() })
    }

    LaunchedEffect(Unit) {
        service.ensureDefaults()
        refreshProviders()
    }

    Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
        Text("Provider Settings", style = MaterialTheme.typography.titleMedium)

        ProviderCreateCard(
            name = newName,
            type = newType,
            baseUrl = newBaseUrl,
            defaultModel = newModel,
            secretRef = newSecretRef,
            onNameChange = { newName = it },
            onTypeChange = {
                newType = it
                newBaseUrl = defaultBaseUrl(it)
                newModel = defaultModel(it)
            },
            onBaseUrlChange = { newBaseUrl = it },
            onDefaultModelChange = { newModel = it },
            onSecretRefChange = { newSecretRef = it },
            onCreate = {
                val trimmedName = newName.trim().ifBlank { "Provider ${providers.size + 1}" }
                val now = Instant.now()
                service.create(
                    ConnectorSetting(
                        id = UUID.randomUUID().toString(),
                        connectorType = newType.name,
                        displayName = trimmedName,
                        baseUrl = newBaseUrl.trim(),
                        defaultModel = newModel.trim(),
                        isEnabled = true,
                        secretRef = newSecretRef.trim().ifBlank { null },
                        createdAt = now,
                        updatedAt = now,
                        connectionHealth = ConnectionHealth.UNKNOWN,
                        connectionMessage = "Not checked",
                    ),
                )
                refreshProviders()
                newName = ""
                newType = ConnectorType.OLLAMA
                newBaseUrl = defaultBaseUrl(newType)
                newModel = defaultModel(newType)
                newSecretRef = ""
            },
        )

        providers.forEachIndexed { index, provider ->
            ProviderCrudCard(
                state = provider,
                onNameChange = { value -> providers[index] = provider.copy(name = value) },
                onTypeChange = { value ->
                    providers[index] = provider.copy(
                        type = value,
                        baseUrl = provider.baseUrl,
                        defaultModel = provider.defaultModel,
                    )
                },
                onBaseUrlChange = { value -> providers[index] = provider.copy(baseUrl = value) },
                onDefaultModelChange = { value -> providers[index] = provider.copy(defaultModel = value) },
                onSecretRefChange = { value -> providers[index] = provider.copy(secretRef = value) },
                onEnabledChange = { value -> providers[index] = provider.copy(isEnabled = value) },
                onValidate = {
                    val current = providers[index]
                    val connector = ConnectorRegistry.create(provider.type)
                    val result = connector.validateConfiguration(
                        ConnectorExecutionConfig(
                            baseUrl = current.baseUrl,
                            apiKey = null,
                            headers = if (current.secretRef.isNotBlank()) {
                                mapOf("X-Secret-Ref" to current.secretRef)
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
                    val updated = current.copy(connectionStatus = updatedStatus)
                    providers[index] = updated
                    service.update(updated.toDomain())
                },
                onSave = {
                    val current = providers[index]
                    service.update(current.toDomain())
                    refreshProviders()
                },
                onDelete = {
                    service.delete(provider.id)
                    refreshProviders()
                },
            )
        }

        Text("MCP Server Settings", style = MaterialTheme.typography.titleMedium)
        Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f))) {
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
private fun ProviderCreateCard(
    name: String,
    type: ConnectorType,
    baseUrl: String,
    defaultModel: String,
    secretRef: String,
    onNameChange: (String) -> Unit,
    onTypeChange: (ConnectorType) -> Unit,
    onBaseUrlChange: (String) -> Unit,
    onDefaultModelChange: (String) -> Unit,
    onSecretRefChange: (String) -> Unit,
    onCreate: () -> Unit,
) {
    Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f))) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text("Create provider", style = MaterialTheme.typography.titleSmall)
            OutlinedTextField(value = name, onValueChange = onNameChange, label = { Text("Display name") }, modifier = Modifier.fillMaxWidth())
            ProviderTypePicker(type = type, onTypeChange = onTypeChange)
            OutlinedTextField(value = baseUrl, onValueChange = onBaseUrlChange, label = { Text("Base URL") }, modifier = Modifier.fillMaxWidth())
            OutlinedTextField(value = defaultModel, onValueChange = onDefaultModelChange, label = { Text("Default model") }, modifier = Modifier.fillMaxWidth())
            OutlinedTextField(value = secretRef, onValueChange = onSecretRefChange, label = { Text("Secret Ref") }, modifier = Modifier.fillMaxWidth())
            Button(onClick = onCreate) { Text("Add provider") }
        }
    }
}

@Composable
private fun ProviderCrudCard(
    state: ProviderSettingUiState,
    onNameChange: (String) -> Unit,
    onTypeChange: (ConnectorType) -> Unit,
    onBaseUrlChange: (String) -> Unit,
    onDefaultModelChange: (String) -> Unit,
    onSecretRefChange: (String) -> Unit,
    onEnabledChange: (Boolean) -> Unit,
    onValidate: () -> Unit,
    onSave: () -> Unit,
    onDelete: () -> Unit,
) {
    Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.25f))) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text(state.name, style = MaterialTheme.typography.titleSmall)
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text("Enabled", style = MaterialTheme.typography.bodySmall)
                    Switch(checked = state.isEnabled, onCheckedChange = onEnabledChange)
                }
            }
            OutlinedTextField(value = state.name, onValueChange = onNameChange, label = { Text("Display name") }, modifier = Modifier.fillMaxWidth())
            ProviderTypePicker(type = state.type, onTypeChange = onTypeChange)
            OutlinedTextField(value = state.baseUrl, onValueChange = onBaseUrlChange, label = { Text("Base URL") }, modifier = Modifier.fillMaxWidth())
            OutlinedTextField(value = state.defaultModel, onValueChange = onDefaultModelChange, label = { Text("Default model") }, modifier = Modifier.fillMaxWidth())
            OutlinedTextField(
                value = state.secretRef,
                onValueChange = onSecretRefChange,
                label = { Text("Secret Ref (stored in OS keychain)") },
                modifier = Modifier.fillMaxWidth(),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = onSave) { Text("Save") }
                Button(onClick = onValidate) { Text("Validate") }
                TextButton(onClick = onDelete) { Text("Delete") }
            }
            Text("Status: ${state.connectionStatus}", style = MaterialTheme.typography.bodySmall)
        }
    }
}

@Composable
private fun ProviderTypePicker(
    type: ConnectorType,
    onTypeChange: (ConnectorType) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    Box {
        Button(onClick = { expanded = true }) {
            Text("Type: ${type.name}")
        }
        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            ConnectorType.entries.forEach { connectorType ->
                DropdownMenuItem(
                    text = { Text(connectorType.name) },
                    onClick = {
                        onTypeChange(connectorType)
                        expanded = false
                    },
                )
            }
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

private fun defaultBaseUrl(type: ConnectorType): String {
    return when (type) {
        ConnectorType.OLLAMA -> OllamaConnector.DEFAULT_BASE_URL
        ConnectorType.LITELLM -> LiteLlmConnector.DEFAULT_BASE_URL
        ConnectorType.AITAO -> AitaoConnector.DEFAULT_BASE_URL
        ConnectorType.OPENAI_COMPAT -> LiteLlmConnector.DEFAULT_BASE_URL
    }
}

private fun defaultModel(type: ConnectorType): String {
    return when (type) {
        ConnectorType.OLLAMA -> OllamaConnector.DEFAULT_MODEL
        ConnectorType.LITELLM -> "gpt-4o-mini"
        ConnectorType.AITAO -> "aitao-default"
        ConnectorType.OPENAI_COMPAT -> "gpt-4o-mini"
    }
}

private data class ProviderSettingUiState(
    val id: String,
    val name: String,
    val type: ConnectorType,
    val baseUrl: String,
    val defaultModel: String,
    val createdAt: Instant,
    val secretRef: String = "",
    val isEnabled: Boolean = true,
    val connectionStatus: String = "Not checked",
)

private fun ProviderSettingUiState.toDomain(): ConnectorSetting {
    return ConnectorSetting(
        id = id,
        connectorType = type.name,
        displayName = name,
        baseUrl = baseUrl,
        defaultModel = defaultModel,
        isEnabled = isEnabled,
        secretRef = secretRef.ifBlank { null },
        createdAt = createdAt,
        updatedAt = Instant.now(),
        connectionHealth = when {
            connectionStatus.startsWith("Healthy") -> ConnectionHealth.HEALTHY
            connectionStatus.startsWith("Offline") -> ConnectionHealth.OFFLINE
            connectionStatus.startsWith("Degraded") -> ConnectionHealth.DEGRADED
            else -> ConnectionHealth.UNKNOWN
        },
        connectionMessage = connectionStatus,
    )
}

private fun ConnectorSetting.toUiState(): ProviderSettingUiState {
    val parsedType = runCatching { ConnectorType.valueOf(connectorType) }.getOrDefault(ConnectorType.OLLAMA)
    return ProviderSettingUiState(
        id = id,
        name = displayName,
        type = parsedType,
        baseUrl = baseUrl,
        defaultModel = defaultModel,
        createdAt = createdAt,
        secretRef = secretRef ?: "",
        isEnabled = isEnabled,
        connectionStatus = connectionMessage,
    )
}
