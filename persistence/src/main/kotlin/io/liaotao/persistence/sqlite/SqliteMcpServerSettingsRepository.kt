/*
 * SqliteMcpServerSettingsRepository.kt - SQLite MCP settings repository.
 * Responsibilities: persist MCP server metadata and auth references while
 * keeping credential material outside plain settings storage.
 */

package io.liaotao.persistence.sqlite

import io.liaotao.shared.settings.ConnectionHealth
import io.liaotao.shared.settings.McpServerSetting
import io.liaotao.shared.settings.McpServerSettingsRepository
import java.time.Instant

class SqliteMcpServerSettingsRepository(
    private val database: SqliteDatabase,
) : McpServerSettingsRepository {
    override fun create(setting: McpServerSetting): McpServerSetting {
        database.withConnection { connection ->
            connection.prepareStatement(
                """
                INSERT INTO mcp_servers(id, name, url, auth_ref, is_enabled, created_at, updated_at)
                VALUES (?, ?, ?, ?, ?, ?, ?)
                """.trimIndent(),
            ).use { prepared ->
                prepared.setString(1, setting.id)
                prepared.setString(2, setting.name)
                prepared.setString(3, setting.url)
                prepared.setString(4, setting.authRef)
                prepared.setInt(5, if (setting.isEnabled) 1 else 0)
                prepared.setString(6, setting.createdAt.toString())
                prepared.setString(7, setting.updatedAt.toString())
                prepared.executeUpdate()
            }
        }
        return setting
    }

    override fun update(setting: McpServerSetting): McpServerSetting {
        val updated = database.withConnection { connection ->
            connection.prepareStatement(
                """
                UPDATE mcp_servers
                SET name = ?, url = ?, auth_ref = ?, is_enabled = ?, updated_at = ?
                WHERE id = ?
                """.trimIndent(),
            ).use { prepared ->
                prepared.setString(1, setting.name)
                prepared.setString(2, setting.url)
                prepared.setString(3, setting.authRef)
                prepared.setInt(4, if (setting.isEnabled) 1 else 0)
                prepared.setString(5, setting.updatedAt.toString())
                prepared.setString(6, setting.id)
                prepared.executeUpdate()
            }
        }

        check(updated == 1) { "MCP setting not found: ${setting.id}" }
        return setting
    }

    override fun getById(settingId: String): McpServerSetting? {
        return database.withConnection { connection ->
            connection.prepareStatement(
                """
                SELECT id, name, url, auth_ref, is_enabled, created_at, updated_at
                FROM mcp_servers
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

    override fun listAll(): List<McpServerSetting> {
        return database.withConnection { connection ->
            connection.prepareStatement(
                """
                SELECT id, name, url, auth_ref, is_enabled, created_at, updated_at
                FROM mcp_servers
                ORDER BY updated_at DESC
                """.trimIndent(),
            ).use { prepared ->
                prepared.executeQuery().use { resultSet ->
                    val items = mutableListOf<McpServerSetting>()
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
            connection.prepareStatement("DELETE FROM mcp_servers WHERE id = ?").use { prepared ->
                prepared.setString(1, settingId)
                prepared.executeUpdate()
            }
        }
        return deleted > 0
    }

    private fun map(resultSet: java.sql.ResultSet): McpServerSetting {
        return McpServerSetting(
            id = resultSet.getString("id"),
            name = resultSet.getString("name"),
            url = resultSet.getString("url"),
            authRef = resultSet.getString("auth_ref"),
            isEnabled = resultSet.getInt("is_enabled") == 1,
            createdAt = Instant.parse(resultSet.getString("created_at")),
            updatedAt = Instant.parse(resultSet.getString("updated_at")),
            connectionHealth = ConnectionHealth.UNKNOWN,
            connectionMessage = "Not checked",
        )
    }
}