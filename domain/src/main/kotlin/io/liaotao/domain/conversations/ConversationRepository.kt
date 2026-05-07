/*
 * ConversationRepository.kt - conversation persistence contract for the domain.
 * Responsibilities: define conversation CRUD and project-scoped access methods
 * required by project and conversation orchestration services.
 */

package io.liaotao.domain.conversations

interface ConversationRepository {
    fun create(conversation: Conversation): Conversation
    fun update(conversation: Conversation): Conversation
    fun getById(conversationId: String): Conversation?
    fun listByProject(projectId: String, includeArchived: Boolean): List<Conversation>
    fun delete(conversationId: String): Boolean
    fun deleteByProject(projectId: String): Int
}