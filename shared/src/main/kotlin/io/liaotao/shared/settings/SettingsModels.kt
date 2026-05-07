/*
 * SettingsModels.kt - provider and MCP settings models.
 * Responsibilities: hold non-secret settings data and connection status fields
 * so secrets are stored separately through dedicated secret stores.
 */

package io.liaotao.shared.settings

import java.time.Instant

enum class ConnectionHealth {
    UNKNOWN,
    HEALTHY,
    DEGRADED,
    OFFLINE,
}

data class ConnectorSetting(
    val id: String,
    val connectorType: String,
    val displayName: String,
    val baseUrl: String,
    val isEnabled: Boolean,
    val secretRef: String?,
    val createdAt: Instant,
    val updatedAt: Instant,
    val connectionHealth: ConnectionHealth = ConnectionHealth.UNKNOWN,
    val connectionMessage: String = "Not checked",
)

data class McpServerSetting(
    val id: String,
    val name: String,
    val url: String,
    val isEnabled: Boolean,
    val authRef: String?,
    val createdAt: Instant,
    val updatedAt: Instant,
    val connectionHealth: ConnectionHealth = ConnectionHealth.UNKNOWN,
    val connectionMessage: String = "Not checked",
)