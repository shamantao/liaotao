/*
 * ProjectRepository.kt - project persistence contract for the domain.
 * Responsibilities: define project CRUD operations independently from storage
 * technology so services can be tested and reused.
 */

package io.liaotao.domain.projects

interface ProjectRepository {
    fun create(project: Project): Project
    fun update(project: Project): Project
    fun getById(projectId: String): Project?
    fun listAll(): List<Project>
    fun delete(projectId: String): Boolean
}