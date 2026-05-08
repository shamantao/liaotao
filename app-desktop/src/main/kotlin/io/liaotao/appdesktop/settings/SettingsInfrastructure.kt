/*
 * SettingsInfrastructure.kt - desktop settings infrastructure wiring.
 * Responsibilities: provide path management and SQLite-backed repositories
 * used by settings and chat screens without hardcoded path scattering.
 */

package io.liaotao.appdesktop.settings

import io.liaotao.persistence.sqlite.SqliteConnectorSettingsRepository
import io.liaotao.persistence.sqlite.SqliteDatabase
import io.liaotao.shared.settings.ConnectionHealth
import io.liaotao.shared.settings.ConnectorSetting
import io.liaotao.shared.settings.ConnectorSettingsRepository
import java.time.Instant
import java.util.UUID

internal object DesktopSettingsInfrastructure {
    private val database: SqliteDatabase by lazy {
        SqliteDatabase.fromPath(DesktopPathManager.appDatabaseFile("liaotao.db")).also { it.migrate() }
    }

    val connectorRepository: ConnectorSettingsRepository by lazy {
        SqliteConnectorSettingsRepository(database)
    }
}

internal class ProviderSettingsService(
    private val repository: ConnectorSettingsRepository = DesktopSettingsInfrastructure.connectorRepository,
    private val nowProvider: () -> Instant = { Instant.now() },
) {
    fun ensureDefaults() {
        if (repository.listAll().isNotEmpty()) {
            return
        }
        val now = nowProvider()
        val defaults = listOf(
            ConnectorSetting(
                id = UUID.randomUUID().toString(),
                connectorType = "OLLAMA",
                displayName = "Ollama",
                baseUrl = "http://localhost:11434",
                defaultModel = "llama3.1",
                isEnabled = true,
                secretRef = null,
                createdAt = now,
                updatedAt = now,
                connectionHealth = ConnectionHealth.UNKNOWN,
                connectionMessage = "Not checked",
            ),
            ConnectorSetting(
                id = UUID.randomUUID().toString(),
                connectorType = "LITELLM",
                displayName = "LiteLLM",
                baseUrl = "http://localhost:4000",
                defaultModel = "gpt-4o-mini",
                isEnabled = true,
                secretRef = null,
                createdAt = now,
                updatedAt = now,
                connectionHealth = ConnectionHealth.UNKNOWN,
                connectionMessage = "Not checked",
            ),
            ConnectorSetting(
                id = UUID.randomUUID().toString(),
                connectorType = "AITAO",
                displayName = "Aitao",
                baseUrl = "http://localhost:5001",
                defaultModel = "aitao-default",
                isEnabled = true,
                secretRef = null,
                createdAt = now,
                updatedAt = now,
                connectionHealth = ConnectionHealth.UNKNOWN,
                connectionMessage = "Not checked",
            ),
        )
        defaults.forEach { repository.create(it) }
    }

    fun listAll(): List<ConnectorSetting> = repository.listAll()

    fun listEnabled(): List<ConnectorSetting> = repository.listAll().filter { it.isEnabled }

    fun create(setting: ConnectorSetting): ConnectorSetting = repository.create(setting)

    fun update(setting: ConnectorSetting): ConnectorSetting = repository.update(setting)

    fun delete(settingId: String): Boolean = repository.delete(settingId)
}
