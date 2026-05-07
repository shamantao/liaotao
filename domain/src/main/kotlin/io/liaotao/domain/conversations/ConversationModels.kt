/*
 * ConversationModels.kt - conversation domain entities for Liaotao.
 * Responsibilities: define conversation metadata, state transitions, and
 * lifecycle timestamps used by project-level orchestration.
 */

package io.liaotao.domain.conversations

import java.time.Instant

data class Conversation(
    val id: String,
    val projectId: String,
    val title: String,
    val source: String,
    val model: String,
    val createdAt: Instant,
    val updatedAt: Instant,
    val lastActivityAt: Instant,
    val archivedAt: Instant?,
) {
    val isArchived: Boolean
        get() = archivedAt != null
}

data class CreateConversationRequest(
    val projectId: String,
    val title: String,
    val source: String,
    val model: String,
)

data class UpdateConversationRequest(
    val title: String,
    val source: String,
    val model: String,
)