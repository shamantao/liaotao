/*
 * SettingsScreen.kt - user-facing settings and provider CRUD.
 * Responsibilities: manage provider catalog (create/read/update/delete),
 * validate provider connectivity, and edit MCP endpoint settings.
 */

package io.liaotao.appdesktop.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import io.liaotao.appdesktop.i18n.AppLanguage
import io.liaotao.appdesktop.i18n.LocalAppStrings
import io.liaotao.connectors.core.ConnectorExecutionConfig
import io.liaotao.connectors.core.ConnectorRegistry
import io.liaotao.connectors.core.ConnectorType
import io.liaotao.shared.settings.ConnectionHealth
import io.liaotao.shared.settings.ConnectorSetting
import java.time.Instant
import java.util.UUID
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

@Composable
fun SettingsScreen(
    selectedLanguage: AppLanguage,
    onLanguageChange: (AppLanguage) -> Unit,
) {
    val strings = LocalAppStrings.current
    val scope = rememberCoroutineScope()
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
    var mcpStatus by remember { mutableStateOf(strings.notChecked) }

    fun refreshProviders() {
        providers.clear()
        providers.addAll(service.listAll().map { it.toUiState() })
    }

    LaunchedEffect(Unit) {
        service.ensureDefaults()
        refreshProviders()
    }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        item {
            Text(strings.settings, style = MaterialTheme.typography.titleMedium)
        }

        item {
            LanguagePickerCard(
                selectedLanguage = selectedLanguage,
                onLanguageChange = onLanguageChange,
            )
        }

        item {
            Text(strings.providerSettings, style = MaterialTheme.typography.titleMedium)
        }

        item {
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
                    val trimmedName = newName.trim().ifBlank { strings.providerAutoName(providers.size + 1) }
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
                            connectionMessage = strings.notChecked,
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
        }

        itemsIndexed(providers, key = { _, provider -> provider.id }) { index, provider ->
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
                    scope.launch {
                        val current = providers[index]
                        val connector = ConnectorRegistry.create(provider.type)
                        val result = withContext(Dispatchers.IO) {
                            connector.validateConfiguration(
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
                        }
                        val updatedStatus = if (result.isValid) {
                            strings.healthLatency(result.latencyMs)
                        } else {
                            strings.degraded(result.message)
                        }
                        val updated = current.copy(connectionStatus = updatedStatus)
                        providers[index] = updated
                        service.update(updated.toDomain())
                    }
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

        item {
            Text(strings.mcpServerSettings, style = MaterialTheme.typography.titleMedium)
        }

        item {
            Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f))) {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(12.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    OutlinedTextField(
                        value = mcpName,
                        onValueChange = { mcpName = it },
                        label = { Text(strings.serverName) },
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = mcpUrl,
                        onValueChange = { mcpUrl = it },
                        label = { Text(strings.serverUrl) },
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Button(onClick = {
                            scope.launch {
                                mcpStatus = withContext(Dispatchers.IO) { validateMcpEndpoint(mcpUrl) }
                            }
                        }) {
                            Text(strings.validate)
                        }
                    }
                    Text(strings.statusLabel(mcpStatus), style = MaterialTheme.typography.bodySmall)
                }
            }
        }

        item {
            androidx.compose.foundation.layout.Spacer(modifier = Modifier.height(16.dp))
        }
    }
}
