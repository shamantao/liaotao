/*
 * ProviderSettingsServiceTest.kt - tests for provider settings orchestration.
 * Responsibilities: verify enabled provider filtering used by chat selector
 * and default seeding behavior.
 */

package io.liaotao.appdesktop.settings

import io.liaotao.shared.settings.ConnectionHealth
import io.liaotao.shared.settings.ConnectorSetting
import io.liaotao.shared.settings.ConnectorSettingsRepository
import java.time.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class ProviderSettingsServiceTest {
    @Test
    fun `listEnabled excludes disabled providers for chat selector`() {
        val now = Instant.parse("2026-05-08T10:00:00Z")
        val repository = InMemoryConnectorSettingsRepository(
            mutableListOf(
                setting("p1", "OLLAMA", true, now),
                setting("p2", "LITELLM", false, now),
                setting("p3", "AITAO", true, now),
            ),
        )
        val service = ProviderSettingsService(repository = repository, nowProvider = { now })

        val enabled = service.listEnabled()

        assertEquals(listOf("p1", "p3"), enabled.map { it.id })
    }

    @Test
    fun `ensureDefaults seeds providers only once`() {
        val now = Instant.parse("2026-05-08T10:00:00Z")
        val repository = InMemoryConnectorSettingsRepository(mutableListOf())
        val service = ProviderSettingsService(repository = repository, nowProvider = { now })

        service.ensureDefaults()
        service.ensureDefaults()

        assertEquals(3, repository.listAll().size)
        assertTrue(repository.listAll().all { it.defaultModel.isNotBlank() })
    }

    private fun setting(id: String, type: String, enabled: Boolean, now: Instant): ConnectorSetting {
        return ConnectorSetting(
            id = id,
            connectorType = type,
            displayName = id,
            baseUrl = "http://localhost",
            defaultModel = "model",
            isEnabled = enabled,
            secretRef = null,
            createdAt = now,
            updatedAt = now,
            connectionHealth = ConnectionHealth.UNKNOWN,
            connectionMessage = "Not checked",
        )
    }
}

private class InMemoryConnectorSettingsRepository(
    private val items: MutableList<ConnectorSetting>,
) : ConnectorSettingsRepository {
    override fun create(setting: ConnectorSetting): ConnectorSetting {
        items.add(setting)
        return setting
    }

    override fun update(setting: ConnectorSetting): ConnectorSetting {
        val index = items.indexOfFirst { it.id == setting.id }
        check(index >= 0) { "Missing setting: ${setting.id}" }
        items[index] = setting
        return setting
    }

    override fun getById(settingId: String): ConnectorSetting? {
        return items.firstOrNull { it.id == settingId }
    }

    override fun listAll(): List<ConnectorSetting> {
        return items.toList()
    }

    override fun delete(settingId: String): Boolean {
        return items.removeIf { it.id == settingId }
    }
}
