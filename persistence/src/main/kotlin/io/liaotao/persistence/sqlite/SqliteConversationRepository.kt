/*
 * SqliteConversationRepository.kt - SQLite conversation repository.
 * Responsibilities: persist conversation lifecycle data and support project
 * scoped retrieval with archive filtering and deterministic ordering.
 */

package io.liaotao.persistence.sqlite

import io.liaotao.domain.conversations.Conversation
import io.liaotao.domain.conversations.ConversationRepository
import java.time.Instant

class SqliteConversationRepository(private val database: SqliteDatabase) : ConversationRepository {
    override fun create(conversation: Conversation): Conversation {
        database.withConnection { connection ->
            connection.prepareStatement(
                """
                INSERT INTO conversations(
                    id, project_id, title, source, model,
                    created_at, updated_at, last_activity_at, archived_at
                )
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
                """.trimIndent(),
            ).use { prepared ->
                prepared.setString(1, conversation.id)
                prepared.setString(2, conversation.projectId)
                prepared.setString(3, conversation.title)
                prepared.setString(4, conversation.source)
                prepared.setString(5, conversation.model)
                prepared.setString(6, conversation.createdAt.toString())
                prepared.setString(7, conversation.updatedAt.toString())
                prepared.setString(8, conversation.lastActivityAt.toString())
                prepared.setString(9, conversation.archivedAt?.toString())
                prepared.executeUpdate()
            }
        }

        return conversation
    }

    override fun update(conversation: Conversation): Conversation {
        val updated = database.withConnection { connection ->
            connection.prepareStatement(
                """
                UPDATE conversations
                SET project_id = ?, title = ?, source = ?, model = ?,
                    updated_at = ?, last_activity_at = ?, archived_at = ?
                WHERE id = ?
                """.trimIndent(),
            ).use { prepared ->
                prepared.setString(1, conversation.projectId)
                prepared.setString(2, conversation.title)
                prepared.setString(3, conversation.source)
                prepared.setString(4, conversation.model)
                prepared.setString(5, conversation.updatedAt.toString())
                prepared.setString(6, conversation.lastActivityAt.toString())
                prepared.setString(7, conversation.archivedAt?.toString())
                prepared.setString(8, conversation.id)
                prepared.executeUpdate()
            }
        }

        check(updated == 1) { "Conversation not found: ${conversation.id}" }
        return conversation
    }

    override fun getById(conversationId: String): Conversation? {
        return database.withConnection { connection ->
            connection.prepareStatement(
                """
                SELECT id, project_id, title, source, model,
                       created_at, updated_at, last_activity_at, archived_at
                FROM conversations
                WHERE id = ?
                """.trimIndent(),
            ).use { prepared ->
                prepared.setString(1, conversationId)
                prepared.executeQuery().use { resultSet ->
                    if (!resultSet.next()) {
                        return@withConnection null
                    }
                    mapConversation(resultSet)
                }
            }
        }
    }

    override fun listByProject(projectId: String, includeArchived: Boolean): List<Conversation> {
        return database.withConnection { connection ->
            val sql = if (includeArchived) {
                """
                SELECT id, project_id, title, source, model,
                       created_at, updated_at, last_activity_at, archived_at
                FROM conversations
                WHERE project_id = ?
                ORDER BY last_activity_at DESC
                """.trimIndent()
            } else {
                """
                SELECT id, project_id, title, source, model,
                       created_at, updated_at, last_activity_at, archived_at
                FROM conversations
                WHERE project_id = ? AND archived_at IS NULL
                ORDER BY last_activity_at DESC
                """.trimIndent()
            }

            connection.prepareStatement(sql).use { prepared ->
                prepared.setString(1, projectId)
                prepared.executeQuery().use { resultSet ->
                    val conversations = mutableListOf<Conversation>()
                    while (resultSet.next()) {
                        conversations.add(mapConversation(resultSet))
                    }
                    conversations
                }
            }
        }
    }

    override fun delete(conversationId: String): Boolean {
        val deleted = database.withConnection { connection ->
            connection.prepareStatement("DELETE FROM conversations WHERE id = ?").use { prepared ->
                prepared.setString(1, conversationId)
                prepared.executeUpdate()
            }
        }
        return deleted > 0
    }

    override fun deleteByProject(projectId: String): Int {
        return database.withConnection { connection ->
            connection.prepareStatement("DELETE FROM conversations WHERE project_id = ?").use { prepared ->
                prepared.setString(1, projectId)
                prepared.executeUpdate()
            }
        }
    }

    private fun mapConversation(resultSet: java.sql.ResultSet): Conversation {
        val archivedRaw = resultSet.getString("archived_at")
        return Conversation(
            id = resultSet.getString("id"),
            projectId = resultSet.getString("project_id"),
            title = resultSet.getString("title"),
            source = resultSet.getString("source"),
            model = resultSet.getString("model"),
            createdAt = Instant.parse(resultSet.getString("created_at")),
            updatedAt = Instant.parse(resultSet.getString("updated_at")),
            lastActivityAt = Instant.parse(resultSet.getString("last_activity_at")),
            archivedAt = archivedRaw?.let { Instant.parse(it) },
        )
    }
}