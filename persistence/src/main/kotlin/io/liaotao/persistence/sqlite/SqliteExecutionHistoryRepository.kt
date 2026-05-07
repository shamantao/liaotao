/*
 * SqliteExecutionHistoryRepository.kt - SQLite execution history repository.
 * Responsibilities: persist execution attempts and provide recent diagnostics
 * reads for routing fallback visibility in UI and support workflows.
 */

package io.liaotao.persistence.sqlite

import io.liaotao.domain.routing.AttemptStatus
import io.liaotao.domain.routing.ExecutionAttempt
import io.liaotao.domain.routing.ExecutionHistoryRepository
import java.time.Instant

class SqliteExecutionHistoryRepository(
    private val database: SqliteDatabase,
) : ExecutionHistoryRepository {
    override fun recordAttempt(attempt: ExecutionAttempt) {
        database.withConnection { connection ->
            connection.prepareStatement(
                """
                INSERT INTO execution_runs(
                    id, conversation_id, connector_instance_id, status, started_at, finished_at, error_message
                ) VALUES (?, ?, ?, ?, ?, ?, ?)
                """.trimIndent(),
            ).use { prepared ->
                prepared.setString(1, attempt.id)
                prepared.setNull(2, java.sql.Types.VARCHAR)
                prepared.setString(3, attempt.providerId)
                prepared.setString(4, attempt.status.name)
                prepared.setString(5, attempt.startedAt.toString())
                prepared.setString(6, attempt.finishedAt.toString())
                prepared.setString(7, attempt.errorMessage)
                prepared.executeUpdate()
            }
        }
    }

    override fun listRecent(limit: Int): List<ExecutionAttempt> {
        return database.withConnection { connection ->
            connection.prepareStatement(
                """
                SELECT id, connector_instance_id, status, started_at, finished_at, error_message
                FROM execution_runs
                ORDER BY started_at DESC
                LIMIT ?
                """.trimIndent(),
            ).use { prepared ->
                prepared.setInt(1, limit)
                prepared.executeQuery().use { resultSet ->
                    val rows = mutableListOf<ExecutionAttempt>()
                    while (resultSet.next()) {
                        rows.add(
                            ExecutionAttempt(
                                id = resultSet.getString("id"),
                                providerId = resultSet.getString("connector_instance_id") ?: "unknown",
                                status = AttemptStatus.valueOf(resultSet.getString("status")),
                                startedAt = Instant.parse(resultSet.getString("started_at")),
                                finishedAt = Instant.parse(resultSet.getString("finished_at")),
                                retryIndex = 0,
                                errorMessage = resultSet.getString("error_message"),
                            ),
                        )
                    }
                    rows
                }
            }
        }
    }
}