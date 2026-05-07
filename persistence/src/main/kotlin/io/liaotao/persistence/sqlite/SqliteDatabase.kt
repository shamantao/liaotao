/*
 * SqliteDatabase.kt - SQLite connection and migration engine.
 * Responsibilities: manage JDBC connections, apply versioned migrations,
 * enforce foreign-key behavior, and provide safe transactional execution.
 */

package io.liaotao.persistence.sqlite

import java.nio.file.Files
import java.nio.file.Path
import java.sql.Connection
import java.sql.DriverManager
import java.time.Instant

class SqliteDatabase private constructor(private val jdbcUrl: String) {
    fun migrate() {
        withConnection { connection ->
            connection.createStatement().use { statement ->
                statement.execute(
                    """
                    CREATE TABLE IF NOT EXISTS schema_migrations (
                        version INTEGER PRIMARY KEY,
                        name TEXT NOT NULL,
                        applied_at TEXT NOT NULL
                    )
                    """.trimIndent(),
                )
            }

            val appliedVersions = mutableSetOf<Int>()
            connection.prepareStatement("SELECT version FROM schema_migrations").use { prepared ->
                prepared.executeQuery().use { resultSet ->
                    while (resultSet.next()) {
                        appliedVersions.add(resultSet.getInt(1))
                    }
                }
            }

            SqliteSchema.migrations
                .sortedBy { it.version }
                .filterNot { appliedVersions.contains(it.version) }
                .forEach { migration ->
                    connection.autoCommit = false
                    try {
                        migration.statements.forEach { sql ->
                            connection.createStatement().use { statement ->
                                statement.execute(sql)
                            }
                        }

                        connection.prepareStatement(
                            "INSERT INTO schema_migrations(version, name, applied_at) VALUES(?, ?, ?)",
                        ).use { prepared ->
                            prepared.setInt(1, migration.version)
                            prepared.setString(2, migration.name)
                            prepared.setString(3, Instant.now().toString())
                            prepared.executeUpdate()
                        }

                        connection.commit()
                    } catch (exception: Exception) {
                        connection.rollback()
                        throw exception
                    } finally {
                        connection.autoCommit = true
                    }
                }
        }
    }

    fun <T> withConnection(block: (Connection) -> T): T {
        DriverManager.getConnection(jdbcUrl).use { connection ->
            connection.createStatement().use { statement ->
                statement.execute("PRAGMA foreign_keys = ON")
            }
            return block(connection)
        }
    }

    companion object {
        fun fromPath(databasePath: Path): SqliteDatabase {
            val parent = databasePath.parent
            if (parent != null) {
                Files.createDirectories(parent)
            }
            return SqliteDatabase(jdbcUrl = "jdbc:sqlite:${databasePath.toAbsolutePath()}")
        }
    }
}