/*
 * InMemoryConversationRepository.kt - in-memory conversation repository.
 * Responsibilities: provide CRUD and project-scoped conversation retrieval for
 * early delivery and deterministic tests before SQLite is introduced.
 */

package io.liaotao.persistence.memory

import io.liaotao.domain.conversations.Conversation
import io.liaotao.domain.conversations.ConversationRepository

class InMemoryConversationRepository : ConversationRepository {
    private val conversationsById = linkedMapOf<String, Conversation>()

    override fun create(conversation: Conversation): Conversation {
        check(!conversationsById.containsKey(conversation.id)) { "Conversation already exists: ${conversation.id}" }
        conversationsById[conversation.id] = conversation
        return conversation
    }

    override fun update(conversation: Conversation): Conversation {
        check(conversationsById.containsKey(conversation.id)) { "Conversation not found: ${conversation.id}" }
        conversationsById[conversation.id] = conversation
        return conversation
    }

    override fun getById(conversationId: String): Conversation? = conversationsById[conversationId]

    override fun listByProject(projectId: String, includeArchived: Boolean): List<Conversation> {
        return conversationsById.values
            .asSequence()
            .filter { it.projectId == projectId }
            .filter { includeArchived || !it.isArchived }
            .toList()
    }

    override fun delete(conversationId: String): Boolean = conversationsById.remove(conversationId) != null

    override fun deleteByProject(projectId: String): Int {
        val toDelete = conversationsById.values.filter { it.projectId == projectId }.map { it.id }
        toDelete.forEach { conversationsById.remove(it) }
        return toDelete.size
    }
}