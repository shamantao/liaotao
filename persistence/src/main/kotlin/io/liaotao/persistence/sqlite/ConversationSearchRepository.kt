/*
 * ConversationSearchRepository.kt - conversation search contract.
 * Responsibilities: expose a dedicated query model for indexed/FTS search
 * on persisted conversation messages.
 */

package io.liaotao.persistence.sqlite

data class ConversationSearchResult(
    val conversationId: String,
    val score: Double,
)

interface ConversationSearchRepository {
    fun indexMessage(messageId: String, conversationId: String, content: String)

    fun search(
        projectId: String,
        query: String,
        includeArchived: Boolean,
        limit: Int = 25,
    ): List<ConversationSearchResult>
}