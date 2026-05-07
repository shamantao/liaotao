/*
 * SettingsRepositories.kt - settings persistence contracts.
 * Responsibilities: define CRUD access for connector and MCP settings while
 * keeping storage technology outside of business and UI layers.
 */

package io.liaotao.shared.settings

interface ConnectorSettingsRepository {
    fun create(setting: ConnectorSetting): ConnectorSetting
    fun update(setting: ConnectorSetting): ConnectorSetting
    fun getById(settingId: String): ConnectorSetting?
    fun listAll(): List<ConnectorSetting>
    fun delete(settingId: String): Boolean
}

interface McpServerSettingsRepository {
    fun create(setting: McpServerSetting): McpServerSetting
    fun update(setting: McpServerSetting): McpServerSetting
    fun getById(settingId: String): McpServerSetting?
    fun listAll(): List<McpServerSetting>
    fun delete(settingId: String): Boolean
}