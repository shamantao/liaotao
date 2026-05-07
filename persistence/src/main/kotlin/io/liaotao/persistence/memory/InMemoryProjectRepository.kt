/*
 * InMemoryProjectRepository.kt - in-memory project repository implementation.
 * Responsibilities: provide a deterministic local repository useful for tests
 * and early development before SQLite persistence is integrated.
 */

package io.liaotao.persistence.memory

import io.liaotao.domain.projects.Project
import io.liaotao.domain.projects.ProjectRepository

class InMemoryProjectRepository : ProjectRepository {
    private val projectsById = linkedMapOf<String, Project>()

    override fun create(project: Project): Project {
        check(!projectsById.containsKey(project.id)) { "Project already exists: ${project.id}" }
        projectsById[project.id] = project
        return project
    }

    override fun update(project: Project): Project {
        check(projectsById.containsKey(project.id)) { "Project not found: ${project.id}" }
        projectsById[project.id] = project
        return project
    }

    override fun getById(projectId: String): Project? = projectsById[projectId]

    override fun listAll(): List<Project> = projectsById.values.toList()

    override fun delete(projectId: String): Boolean = projectsById.remove(projectId) != null
}