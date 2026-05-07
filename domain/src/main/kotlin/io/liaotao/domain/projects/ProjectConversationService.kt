/*
 * ProjectConversationService.kt - project and conversation use-case service.
 * Responsibilities: implement CRUD, archive/restore, move/duplicate operations,
 * and guarantee lifecycle metadata timestamps are maintained consistently.
 */

package io.liaotao.domain.projects

import io.liaotao.domain.conversations.Conversation
import io.liaotao.domain.conversations.ConversationRepository
import io.liaotao.domain.conversations.CreateConversationRequest
import io.liaotao.domain.conversations.UpdateConversationRequest
import java.time.Instant
import java.util.UUID

class ProjectConversationService(
    private val projectRepository: ProjectRepository,
    private val conversationRepository: ConversationRepository,
    private val nowProvider: () -> Instant = { Instant.now() },
    private val idProvider: () -> String = { UUID.randomUUID().toString() },
) {
    fun createProject(request: CreateProjectRequest): Project {
        val now = nowProvider()
        val sanitizedName = request.name.trim()
        require(sanitizedName.isNotEmpty()) { "Project name cannot be empty" }

        return projectRepository.create(
            Project(
                id = idProvider(),
                name = sanitizedName,
                description = request.description.trim(),
                createdAt = now,
                updatedAt = now,
            ),
        )
    }

    fun updateProject(projectId: String, request: UpdateProjectRequest): Project {
        val current = requireProject(projectId)
        val now = nowProvider()
        val sanitizedName = request.name.trim()
        require(sanitizedName.isNotEmpty()) { "Project name cannot be empty" }

        return projectRepository.update(
            current.copy(
                name = sanitizedName,
                description = request.description.trim(),
                updatedAt = now,
            ),
        )
    }

    fun listProjects(): List<Project> = projectRepository.listAll().sortedByDescending { it.updatedAt }

    fun deleteProject(projectId: String): Boolean {
        requireProject(projectId)
        conversationRepository.deleteByProject(projectId)
        return projectRepository.delete(projectId)
    }

    fun createConversation(request: CreateConversationRequest): Conversation {
        requireProject(request.projectId)
        val now = nowProvider()

        return conversationRepository.create(
            Conversation(
                id = idProvider(),
                projectId = request.projectId,
                title = sanitizeConversationTitle(request.title),
                source = sanitizeRequired(request.source, "Source"),
                model = sanitizeRequired(request.model, "Model"),
                createdAt = now,
                updatedAt = now,
                lastActivityAt = now,
                archivedAt = null,
            ),
        )
    }

    fun updateConversation(conversationId: String, request: UpdateConversationRequest): Conversation {
        val current = requireConversation(conversationId)
        val now = nowProvider()

        return conversationRepository.update(
            current.copy(
                title = sanitizeConversationTitle(request.title),
                source = sanitizeRequired(request.source, "Source"),
                model = sanitizeRequired(request.model, "Model"),
                updatedAt = now,
                lastActivityAt = now,
            ),
        )
    }

    fun archiveConversation(conversationId: String): Conversation {
        val current = requireConversation(conversationId)
        if (current.isArchived) {
            return current
        }
        val now = nowProvider()
        return conversationRepository.update(
            current.copy(
                archivedAt = now,
                updatedAt = now,
            ),
        )
    }

    fun restoreConversation(conversationId: String): Conversation {
        val current = requireConversation(conversationId)
        if (!current.isArchived) {
            return current
        }
        val now = nowProvider()
        return conversationRepository.update(
            current.copy(
                archivedAt = null,
                updatedAt = now,
                lastActivityAt = now,
            ),
        )
    }

    fun moveConversation(conversationId: String, targetProjectId: String): Conversation {
        val current = requireConversation(conversationId)
        requireProject(targetProjectId)
        if (current.projectId == targetProjectId) {
            return current
        }
        val now = nowProvider()

        return conversationRepository.update(
            current.copy(
                projectId = targetProjectId,
                updatedAt = now,
                lastActivityAt = now,
            ),
        )
    }

    fun duplicateConversation(conversationId: String, targetProjectId: String? = null): Conversation {
        val current = requireConversation(conversationId)
        val destinationProjectId = targetProjectId ?: current.projectId
        requireProject(destinationProjectId)
        val now = nowProvider()

        return conversationRepository.create(
            current.copy(
                id = idProvider(),
                projectId = destinationProjectId,
                title = buildDuplicateTitle(current.title),
                createdAt = now,
                updatedAt = now,
                lastActivityAt = now,
                archivedAt = null,
            ),
        )
    }

    fun listConversations(projectId: String, includeArchived: Boolean = false): List<Conversation> {
        requireProject(projectId)
        return conversationRepository
            .listByProject(projectId, includeArchived)
            .sortedByDescending { it.lastActivityAt }
    }

    fun deleteConversation(conversationId: String): Boolean {
        requireConversation(conversationId)
        return conversationRepository.delete(conversationId)
    }

    private fun requireProject(projectId: String): Project {
        return projectRepository.getById(projectId)
            ?: throw IllegalArgumentException("Project not found: $projectId")
    }

    private fun requireConversation(conversationId: String): Conversation {
        return conversationRepository.getById(conversationId)
            ?: throw IllegalArgumentException("Conversation not found: $conversationId")
    }

    private fun sanitizeRequired(value: String, fieldName: String): String {
        val sanitized = value.trim()
        require(sanitized.isNotEmpty()) { "$fieldName cannot be empty" }
        return sanitized
    }

    private fun sanitizeConversationTitle(title: String): String {
        val sanitized = title.trim()
        require(sanitized.isNotEmpty()) { "Conversation title cannot be empty" }
        return sanitized
    }

    private fun buildDuplicateTitle(originalTitle: String): String {
        return if (originalTitle.endsWith(" (copy)")) {
            "$originalTitle 2"
        } else {
            "$originalTitle (copy)"
        }
    }
}