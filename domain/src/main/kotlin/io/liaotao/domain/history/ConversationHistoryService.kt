/*
 * ConversationHistoryService.kt - conversation search and filtering logic.
 * Responsibilities: provide keyword search, project/source filters, and
 * deterministic sorting by last activity for history browsing.
 */

package io.liaotao.domain.history

import io.liaotao.domain.conversations.Conversation

data class ConversationHistoryQuery(
    val keyword: String = "",
    val projectId: String? = null,
    val source: String? = null,
    val includeArchived: Boolean = true,
)

class ConversationHistoryService {
    fun query(conversations: List<Conversation>, request: ConversationHistoryQuery): List<Conversation> {
        val keyword = request.keyword.trim().lowercase()
        val projectFilter = request.projectId?.trim()?.takeIf { it.isNotEmpty() }
        val sourceFilter = request.source?.trim()?.takeIf { it.isNotEmpty() }?.lowercase()

        return conversations
            .asSequence()
            .filter { request.includeArchived || !it.isArchived }
            .filter { projectFilter == null || it.projectId == projectFilter }
            .filter { sourceFilter == null || it.source.lowercase() == sourceFilter }
            .filter {
                keyword.isEmpty() ||
                    it.title.lowercase().contains(keyword) ||
                    it.model.lowercase().contains(keyword) ||
                    it.source.lowercase().contains(keyword)
            }
            .sortedByDescending { it.lastActivityAt }
            .toList()
    }
}