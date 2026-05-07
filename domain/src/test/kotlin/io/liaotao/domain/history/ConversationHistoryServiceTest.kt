/*
 * ConversationHistoryServiceTest.kt - tests for conversation history querying.
 * Responsibilities: validate keyword search, project/source filtering, and
 * sort by descending last activity.
 */

package io.liaotao.domain.history

import io.liaotao.domain.conversations.Conversation
import java.time.Instant
import kotlin.test.Test
import kotlin.test.assertEquals

class ConversationHistoryServiceTest {
    private val service = ConversationHistoryService()

    @Test
    fun `filters by project source and keyword then sorts by activity`() {
        val conversations = listOf(
            conversation(id = "1", project = "a", source = "OLLAMA", title = "Budget draft", offset = 20),
            conversation(id = "2", project = "a", source = "LITELLM", title = "Roadmap", offset = 10),
            conversation(id = "3", project = "b", source = "OLLAMA", title = "Budget final", offset = 30),
        )

        val result = service.query(
            conversations = conversations,
            request = ConversationHistoryQuery(
                keyword = "budget",
                projectId = "a",
                source = "OLLAMA",
            ),
        )

        assertEquals(listOf("1"), result.map { it.id })
    }

    @Test
    fun `sorts by descending last activity`() {
        val conversations = listOf(
            conversation(id = "older", project = "a", source = "OLLAMA", title = "T1", offset = 5),
            conversation(id = "newer", project = "a", source = "OLLAMA", title = "T2", offset = 50),
        )

        val result = service.query(conversations, ConversationHistoryQuery())
        assertEquals(listOf("newer", "older"), result.map { it.id })
    }

    private fun conversation(id: String, project: String, source: String, title: String, offset: Long): Conversation {
        val base = Instant.parse("2026-05-07T12:00:00Z")
        return Conversation(
            id = id,
            projectId = project,
            title = title,
            source = source,
            model = "m1",
            createdAt = base,
            updatedAt = base.plusSeconds(offset),
            lastActivityAt = base.plusSeconds(offset),
            archivedAt = null,
        )
    }
}