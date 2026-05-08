/*
 * SqliteConnectorSettingsRepository.kt - SQLite connector settings repository.
 * Responsibilities: persist provider settings metadata while keeping secrets
 * out of plain storage by referencing external secret identifiers.
 */

package io.liaotao.persistence.sqlite

import io.liaotao.shared.settings.ConnectionHealth
import io.liaotao.shared.settings.ConnectorSetting
import io.liaotao.shared.settings.ConnectorSettingsRepository
import java.time.Instant

class SqliteConnectorSettingsRepository(
    private val database: SqliteDatabase,
) : ConnectorSettingsRepository {
    override fun create(setting: ConnectorSetting): ConnectorSetting {
        database.withConnection { connection ->
            connection.prepareStatement(
                """
                INSERT INTO connector_instances(
                    id, connector_type, display_name, base_url,
                    default_model, is_enabled, secret_ref,
                    created_at, updated_at, connection_health, connection_message
                )
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """.trimIndent(),
            ).use { prepared ->
                prepared.setString(1, setting.id)
                prepared.setString(2, setting.connectorType)
                prepared.setString(3, setting.displayName)
                prepared.setString(4, setting.baseUrl)
                prepared.setString(5, setting.defaultModel)
                prepared.setInt(6, if (setting.isEnabled) 1 else 0)
                prepared.setString(7, setting.secretRef)
                prepared.setString(8, setting.createdAt.toString())
                prepared.setString(9, setting.updatedAt.toString())
                prepared.setString(10, setting.connectionHealth.name)
                prepared.setString(11, setting.connectionMessage)
                prepared.executeUpdate()
            }
        }
        return setting
    }

    override fun update(setting: ConnectorSetting): ConnectorSetting {
        val updated = database.withConnection { connection ->
            connection.prepareStatement(
                """
                UPDATE connector_instances
                SET connector_type = ?, display_name = ?, base_url = ?,
                    default_model = ?, is_enabled = ?, secret_ref = ?,
                    updated_at = ?, connection_health = ?, connection_message = ?
                WHERE id = ?
                """.trimIndent(),
            ).use { prepared ->
                prepared.setString(1, setting.connectorType)
                prepared.setString(2, setting.displayName)
                prepared.setString(3, setting.baseUrl)
                prepared.setString(4, setting.defaultModel)
                prepared.setInt(5, if (setting.isEnabled) 1 else 0)
                prepared.setString(6, setting.secretRef)
                prepared.setString(7, setting.updatedAt.toString())
                prepared.setString(8, setting.connectionHealth.name)
                prepared.setString(9, setting.connectionMessage)
                prepared.setString(10, setting.id)
                prepared.executeUpdate()
            }
        }

        check(updated == 1) { "Connector setting not found: ${setting.id}" }
        return setting
    }

    override fun getById(settingId: String): ConnectorSetting? {
        return database.withConnection { connection ->
            connection.prepareStatement(
                """
                SELECT id, connector_type, display_name, base_url,
                      default_model, is_enabled, secret_ref,
                      created_at, updated_at, connection_health, connection_message
                FROM connector_instances
                WHERE id = ?
                """.trimIndent(),
            ).use { prepared ->
                prepared.setString(1, settingId)
                prepared.executeQuery().use { resultSet ->
                    if (!resultSet.next()) {
                        return@withConnection null
                    }
                    map(resultSet)
                }
            }
        }
    }

    override fun listAll(): List<ConnectorSetting> {
        return database.withConnection { connection ->
            connection.prepareStatement(
                """
                SELECT id, connector_type, display_name, base_url,
                      default_model, is_enabled, secret_ref,
                      created_at, updated_at, connection_health, connection_message
                FROM connector_instances
                ORDER BY updated_at DESC
                """.trimIndent(),
            ).use { prepared ->
                prepared.executeQuery().use { resultSet ->
                    val items = mutableListOf<ConnectorSetting>()
                    while (resultSet.next()) {
                        items.add(map(resultSet))
                    }
                    items
                }
            }
        }
    }

    override fun delete(settingId: String): Boolean {
        val deleted = database.withConnection { connection ->
            connection.prepareStatement("DELETE FROM connector_instances WHERE id = ?").use { prepared ->
                prepared.setString(1, settingId)
                prepared.executeUpdate()
            }
        }
        return deleted > 0
    }

    private fun map(resultSet: java.sql.ResultSet): ConnectorSetting {
        return ConnectorSetting(
            id = resultSet.getString("id"),
            connectorType = resultSet.getString("connector_type"),
            displayName = resultSet.getString("display_name"),
            baseUrl = resultSet.getString("base_url") ?: "",
            defaultModel = resultSet.getString("default_model") ?: "",
            isEnabled = resultSet.getInt("is_enabled") == 1,
            secretRef = resultSet.getString("secret_ref"),
            createdAt = Instant.parse(resultSet.getString("created_at")),
            updatedAt = Instant.parse(resultSet.getString("updated_at")),
            connectionHealth = ConnectionHealth.valueOf(resultSet.getString("connection_health") ?: ConnectionHealth.UNKNOWN.name),
            connectionMessage = resultSet.getString("connection_message") ?: "Not checked",
        )
    }
}