/*
 * ConversationImportExportServiceTest.kt - tests for JSON import/export.
 * Responsibilities: validate schema version enforcement, single/project export,
 * and partial import error reporting.
 */

package io.liaotao.persistence.files

import io.liaotao.domain.conversations.Conversation
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.time.Instant
import java.nio.file.Files
import java.nio.file.Paths
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class ConversationImportExportServiceTest {
    @Test
    fun `export single conversation uses schema version and no secret fields`() {
        val service = ConversationImportExportService(nowProvider = { Instant.parse("2026-05-07T12:30:00Z") })
        val payload = service.exportSingleConversation(sampleTranscript("p1"))

        val node = Json.parseToJsonElement(payload).jsonObject
        assertEquals(1, node["schemaVersion"]?.jsonPrimitive?.content?.toInt())
        assertTrue("secrets" !in node.keys)
        assertTrue("apiKey" !in payload)

        val snapshotPath = Paths.get("src/test/resources/snapshots/export_v1_single_conversation.json")
        val expected = Files.readString(snapshotPath).trim()
        assertEquals(expected, payload.trim())
    }

    @Test
    fun `export project conversations filters by project id`() {
        val service = ConversationImportExportService()
        val payload = service.exportProjectConversations(
            transcripts = listOf(sampleTranscript("a"), sampleTranscript("b")),
            projectId = "a",
        )

        val imported = service.importPackage(payload).first
        assertEquals(1, imported.size)
        assertEquals("a", imported.first().conversation.projectId)
    }

    @Test
    fun `export all conversations keeps transcripts from multiple projects`() {
        val service = ConversationImportExportService()
        val payload = service.exportAllConversations(
            transcripts = listOf(sampleTranscript("a"), sampleTranscript("b")),
        )

        val imported = service.importPackage(payload).first
        assertEquals(2, imported.size)
        assertTrue(imported.map { it.conversation.projectId }.containsAll(listOf("a", "b")))
    }

    @Test
    fun `import reports invalid schema version`() {
        val service = ConversationImportExportService()
        val invalid = """
            {
              "schemaVersion": 99,
              "exportedAt": "2026-05-07T12:00:00Z",
              "conversations": []
            }
        """.trimIndent()

        val (_, report) = service.importPackage(invalid)
        assertEquals(0, report.importedConversations)
        assertTrue(report.partialErrors.isNotEmpty())
    }

    private fun sampleTranscript(projectId: String): ConversationTranscript {
        val now = Instant.parse("2026-05-07T12:00:00Z")
        return ConversationTranscript(
            conversation = Conversation(
                id = "conv-$projectId",
                projectId = projectId,
                title = "Sample",
                source = "OLLAMA",
                model = "qwen3",
                createdAt = now,
                updatedAt = now,
                lastActivityAt = now,
                archivedAt = null,
            ),
            messages = listOf(
                TranscriptMessage(role = "user", content = "hello", createdAt = now),
                TranscriptMessage(role = "assistant", content = "hi", createdAt = now),
            ),
        )
    }
}