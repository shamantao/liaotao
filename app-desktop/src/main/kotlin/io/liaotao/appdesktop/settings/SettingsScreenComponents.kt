/*
 * SettingsScreenComponents.kt - reusable settings composables and mappers.
 * Responsibilities: render language/provider cards and map provider UI state
 * to persistence models.
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
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import io.liaotao.appdesktop.i18n.AppLanguage
import io.liaotao.appdesktop.i18n.AppStrings
import io.liaotao.appdesktop.i18n.LocalAppStrings
import io.liaotao.connectors.aitao.AitaoConnector
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

@Composable
internal fun LanguagePickerCard(
    selectedLanguage: AppLanguage,
    onLanguageChange: (AppLanguage) -> Unit,
) {
    val strings = LocalAppStrings.current
    var expanded by remember { mutableStateOf(false) }

    Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f))) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(strings.language, style = MaterialTheme.typography.titleSmall)
            Box {
                Button(onClick = { expanded = true }) {
                    Text(languageLabel(selectedLanguage, strings))
                }
                DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                    AppLanguage.entries.forEach { language ->
                        DropdownMenuItem(
                            text = { Text(languageLabel(language, strings)) },
                            onClick = {
                                onLanguageChange(language)
                                expanded = false
                            },
                        )
                    }
                }
            }
        }
    }
}

@Composable
internal fun ProviderCreateCard(
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
    val strings = LocalAppStrings.current
    Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f))) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(strings.createProvider, style = MaterialTheme.typography.titleSmall)
            OutlinedTextField(value = name, onValueChange = onNameChange, label = { Text(strings.displayName) }, modifier = Modifier.fillMaxWidth())
            ProviderTypePicker(type = type, onTypeChange = onTypeChange)
            OutlinedTextField(value = baseUrl, onValueChange = onBaseUrlChange, label = { Text(strings.baseUrl) }, modifier = Modifier.fillMaxWidth())
            OutlinedTextField(value = defaultModel, onValueChange = onDefaultModelChange, label = { Text(strings.defaultModel) }, modifier = Modifier.fillMaxWidth())
            OutlinedTextField(value = secretRef, onValueChange = onSecretRefChange, label = { Text(strings.secretRef) }, modifier = Modifier.fillMaxWidth())
            Button(onClick = onCreate) { Text(strings.addProvider) }
        }
    }
}

@Composable
internal fun ProviderCrudCard(
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
    val strings = LocalAppStrings.current
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
                    Text(strings.enabled, style = MaterialTheme.typography.bodySmall)
                    Switch(checked = state.isEnabled, onCheckedChange = onEnabledChange)
                }
            }
            OutlinedTextField(value = state.name, onValueChange = onNameChange, label = { Text(strings.displayName) }, modifier = Modifier.fillMaxWidth())
            ProviderTypePicker(type = state.type, onTypeChange = onTypeChange)
            OutlinedTextField(value = state.baseUrl, onValueChange = onBaseUrlChange, label = { Text(strings.baseUrl) }, modifier = Modifier.fillMaxWidth())
            OutlinedTextField(value = state.defaultModel, onValueChange = onDefaultModelChange, label = { Text(strings.defaultModel) }, modifier = Modifier.fillMaxWidth())
            OutlinedTextField(
                value = state.secretRef,
                onValueChange = onSecretRefChange,
                label = { Text(strings.secretRefHint) },
                modifier = Modifier.fillMaxWidth(),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = onSave) { Text(strings.save) }
                Button(onClick = onValidate) { Text(strings.validate) }
                TextButton(onClick = onDelete) { Text(strings.delete) }
            }
            Text(strings.statusLabel(state.connectionStatus), style = MaterialTheme.typography.bodySmall)
        }
    }
}

@Composable
private fun ProviderTypePicker(
    type: ConnectorType,
    onTypeChange: (ConnectorType) -> Unit,
) {
    val strings = LocalAppStrings.current
    var expanded by remember { mutableStateOf(false) }
    Box {
        Button(onClick = { expanded = true }) {
            Text("${strings.type}: ${type.name}")
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

internal fun validateMcpEndpoint(url: String): String {
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

internal fun defaultBaseUrl(type: ConnectorType): String {
    return when (type) {
        ConnectorType.OLLAMA -> OllamaConnector.DEFAULT_BASE_URL
        ConnectorType.LITELLM -> LiteLlmConnector.DEFAULT_BASE_URL
        ConnectorType.AITAO -> AitaoConnector.DEFAULT_BASE_URL
        ConnectorType.OPENAI_COMPAT -> LiteLlmConnector.DEFAULT_BASE_URL
    }
}

internal fun defaultModel(type: ConnectorType): String {
    return when (type) {
        ConnectorType.OLLAMA -> OllamaConnector.DEFAULT_MODEL
        ConnectorType.LITELLM -> "gpt-4o-mini"
        ConnectorType.AITAO -> "aitao-default"
        ConnectorType.OPENAI_COMPAT -> "gpt-4o-mini"
    }
}

private fun languageLabel(language: AppLanguage, strings: AppStrings): String {
    return when (language) {
        AppLanguage.ENGLISH -> strings.english
        AppLanguage.FRENCH -> strings.french
    }
}

internal data class ProviderSettingUiState(
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

internal fun ProviderSettingUiState.toDomain(): ConnectorSetting {
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

internal fun ConnectorSetting.toUiState(): ProviderSettingUiState {
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
