/*
 * SettingsPersistenceTest.kt - tests for settings persistence repositories.
 * Responsibilities: validate connector and MCP settings CRUD operations in
 * SQLite while preserving non-secret storage boundaries.
 */

package io.liaotao.persistence.sqlite

import io.liaotao.shared.settings.ConnectionHealth
import io.liaotao.shared.settings.ConnectorSetting
import io.liaotao.shared.settings.McpServerSetting
import java.nio.file.Files
import java.time.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

class SettingsPersistenceTest {
    @Test
    fun `connector settings repository persists and retrieves rows`() {
        val database = newDatabase()
        val repository = SqliteConnectorSettingsRepository(database)
        val now = Instant.parse("2026-05-07T12:00:00Z")

        repository.create(
            ConnectorSetting(
                id = "conn-1",
                connectorType = "OLLAMA",
                displayName = "Local Ollama",
                baseUrl = "http://localhost:11434",
                defaultModel = "llama3.1",
                isEnabled = true,
                secretRef = "secret-ollama",
                createdAt = now,
                updatedAt = now,
                connectionHealth = ConnectionHealth.UNKNOWN,
                connectionMessage = "Not checked",
            ),
        )

        val loaded = repository.getById("conn-1")
        assertNotNull(loaded)
        assertEquals("OLLAMA", loaded.connectorType)
        assertEquals("Local Ollama", loaded.displayName)
        assertEquals("http://localhost:11434", loaded.baseUrl)
        assertEquals("llama3.1", loaded.defaultModel)
        assertEquals("secret-ollama", loaded.secretRef)
    }

    @Test
    fun `connector settings repository supports update disable and delete lifecycle`() {
        val database = newDatabase()
        val repository = SqliteConnectorSettingsRepository(database)
        val now = Instant.parse("2026-05-07T12:00:00Z")

        val created = repository.create(
            ConnectorSetting(
                id = "conn-lifecycle",
                connectorType = "LITELLM",
                displayName = "LiteLLM A",
                baseUrl = "http://localhost:4000",
                defaultModel = "gpt-4o-mini",
                isEnabled = true,
                secretRef = null,
                createdAt = now,
                updatedAt = now,
                connectionHealth = ConnectionHealth.UNKNOWN,
                connectionMessage = "Not checked",
            ),
        )

        val updated = created.copy(
            displayName = "LiteLLM Disabled",
            isEnabled = false,
            updatedAt = now.plusSeconds(60),
            connectionHealth = ConnectionHealth.DEGRADED,
            connectionMessage = "Degraded: timeout",
        )
        repository.update(updated)

        val loaded = repository.getById(created.id)
        assertNotNull(loaded)
        assertEquals("LiteLLM Disabled", loaded.displayName)
        assertEquals(false, loaded.isEnabled)
        assertEquals(ConnectionHealth.DEGRADED, loaded.connectionHealth)

        assertTrue(repository.delete(created.id))
        assertEquals(null, repository.getById(created.id))
    }

    @Test
    fun `mcp settings repository persists and retrieves rows`() {
        val database = newDatabase()
        val repository = SqliteMcpServerSettingsRepository(database)
        val now = Instant.parse("2026-05-07T12:00:00Z")

        repository.create(
            McpServerSetting(
                id = "mcp-1",
                name = "Main MCP",
                url = "http://localhost:3333",
                isEnabled = true,
                authRef = "secret-mcp",
                createdAt = now,
                updatedAt = now,
                connectionHealth = ConnectionHealth.UNKNOWN,
                connectionMessage = "Not checked",
            ),
        )

        val loaded = repository.getById("mcp-1")
        assertNotNull(loaded)
        assertEquals("Main MCP", loaded.name)
        assertEquals("http://localhost:3333", loaded.url)
        assertTrue(loaded.isEnabled)
    }

    private fun newDatabase(): SqliteDatabase {
        val dbPath = Files.createTempFile("liaotao-settings-", ".db")
        val database = SqliteDatabase.fromPath(dbPath)
        database.migrate()
        return database
    }
}