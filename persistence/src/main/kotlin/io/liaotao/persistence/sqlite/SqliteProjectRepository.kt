/*
 * SqliteProjectRepository.kt - SQLite implementation of project repository.
 * Responsibilities: persist and retrieve project records while preserving
 * creation/update timestamps and deterministic listing behavior.
 */

package io.liaotao.persistence.sqlite

import io.liaotao.domain.projects.Project
import io.liaotao.domain.projects.ProjectRepository
import java.time.Instant

class SqliteProjectRepository(private val database: SqliteDatabase) : ProjectRepository {
    override fun create(project: Project): Project {
        database.withConnection { connection ->
            connection.prepareStatement(
                """
                INSERT INTO projects(id, name, description, created_at, updated_at)
                VALUES (?, ?, ?, ?, ?)
                """.trimIndent(),
            ).use { prepared ->
                prepared.setString(1, project.id)
                prepared.setString(2, project.name)
                prepared.setString(3, project.description)
                prepared.setString(4, project.createdAt.toString())
                prepared.setString(5, project.updatedAt.toString())
                prepared.executeUpdate()
            }
        }
        return project
    }

    override fun update(project: Project): Project {
        val updated = database.withConnection { connection ->
            connection.prepareStatement(
                """
                UPDATE projects
                SET name = ?, description = ?, updated_at = ?
                WHERE id = ?
                """.trimIndent(),
            ).use { prepared ->
                prepared.setString(1, project.name)
                prepared.setString(2, project.description)
                prepared.setString(3, project.updatedAt.toString())
                prepared.setString(4, project.id)
                prepared.executeUpdate()
            }
        }

        check(updated == 1) { "Project not found: ${project.id}" }
        return project
    }

    override fun getById(projectId: String): Project? {
        return database.withConnection { connection ->
            connection.prepareStatement(
                "SELECT id, name, description, created_at, updated_at FROM projects WHERE id = ?",
            ).use { prepared ->
                prepared.setString(1, projectId)
                prepared.executeQuery().use { resultSet ->
                    if (!resultSet.next()) {
                        return@withConnection null
                    }
                    mapProject(resultSet)
                }
            }
        }
    }

    override fun listAll(): List<Project> {
        return database.withConnection { connection ->
            connection.prepareStatement(
                "SELECT id, name, description, created_at, updated_at FROM projects ORDER BY updated_at DESC",
            ).use { prepared ->
                prepared.executeQuery().use { resultSet ->
                    val projects = mutableListOf<Project>()
                    while (resultSet.next()) {
                        projects.add(mapProject(resultSet))
                    }
                    projects
                }
            }
        }
    }

    override fun delete(projectId: String): Boolean {
        val deleted = database.withConnection { connection ->
            connection.prepareStatement("DELETE FROM projects WHERE id = ?").use { prepared ->
                prepared.setString(1, projectId)
                prepared.executeUpdate()
            }
        }
        return deleted > 0
    }

    private fun mapProject(resultSet: java.sql.ResultSet): Project {
        return Project(
            id = resultSet.getString("id"),
            name = resultSet.getString("name"),
            description = resultSet.getString("description"),
            createdAt = Instant.parse(resultSet.getString("created_at")),
            updatedAt = Instant.parse(resultSet.getString("updated_at")),
        )
    }
}