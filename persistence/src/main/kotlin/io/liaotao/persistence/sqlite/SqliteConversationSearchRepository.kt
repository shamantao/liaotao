/*
 * SqliteConversationSearchRepository.kt - SQLite FTS-based message search.
 * Responsibilities: maintain search index entries and execute ranked search
 * queries constrained by project and archive visibility.
 */

package io.liaotao.persistence.sqlite

class SqliteConversationSearchRepository(private val database: SqliteDatabase) : ConversationSearchRepository {
    override fun indexMessage(messageId: String, conversationId: String, content: String) {
        database.withConnection { connection ->
            connection.prepareStatement(
                """
                INSERT INTO messages(id, conversation_id, role, content, created_at)
                VALUES (?, ?, 'user', ?, datetime('now'))
                ON CONFLICT(id) DO UPDATE SET
                    conversation_id = excluded.conversation_id,
                    content = excluded.content
                """.trimIndent(),
            ).use { prepared ->
                prepared.setString(1, messageId)
                prepared.setString(2, conversationId)
                prepared.setString(3, content)
                prepared.executeUpdate()
            }
        }
    }

    override fun search(
        projectId: String,
        query: String,
        includeArchived: Boolean,
        limit: Int,
    ): List<ConversationSearchResult> {
        val sql = if (includeArchived) {
            """
                        SELECT c.id AS conversation_id, COUNT(*) AS rank_score
                        FROM message_fts f
                        JOIN conversations c ON c.id = f.conversation_id
            WHERE c.project_id = ?
                            AND message_fts MATCH ?
            GROUP BY c.id
                        ORDER BY rank_score DESC, c.last_activity_at DESC
            LIMIT ?
            """.trimIndent()
        } else {
            """
                        SELECT c.id AS conversation_id, COUNT(*) AS rank_score
                        FROM message_fts f
                        JOIN conversations c ON c.id = f.conversation_id
            WHERE c.project_id = ?
                            AND message_fts MATCH ?
              AND c.archived_at IS NULL
            GROUP BY c.id
                        ORDER BY rank_score DESC, c.last_activity_at DESC
            LIMIT ?
            """.trimIndent()
        }

        return database.withConnection { connection ->
            connection.prepareStatement(sql).use { prepared ->
                prepared.setString(1, projectId)
                prepared.setString(2, query)
                prepared.setInt(3, limit)
                prepared.executeQuery().use { resultSet ->
                    val rows = mutableListOf<ConversationSearchResult>()
                    while (resultSet.next()) {
                        rows.add(
                            ConversationSearchResult(
                                conversationId = resultSet.getString("conversation_id"),
                                score = resultSet.getDouble("rank_score"),
                            ),
                        )
                    }
                    rows
                }
            }
        }
    }
}