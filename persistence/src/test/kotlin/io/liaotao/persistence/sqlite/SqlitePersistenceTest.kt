/*
 * SqlitePersistenceTest.kt - integration tests for SQLite persistence layer.
 * Responsibilities: validate migrations, repository CRUD parity, and FTS
 * search behavior with project and archive filtering.
 */

package io.liaotao.persistence.sqlite

import io.liaotao.domain.conversations.Conversation
import io.liaotao.domain.projects.Project
import java.nio.file.Files
import java.time.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

class SqlitePersistenceTest {
    @Test
    fun `migration creates schema and version tracking`() {
        val database = newDatabase()

        val versions = database.withConnection { connection ->
            connection.prepareStatement("SELECT version FROM schema_migrations ORDER BY version").use { prepared ->
                prepared.executeQuery().use { resultSet ->
                    val collected = mutableListOf<Int>()
                    while (resultSet.next()) {
                        collected.add(resultSet.getInt(1))
                    }
                    collected
                }
            }
        }

        assertEquals(listOf(1, 2), versions)
    }

    @Test
    fun `project and conversation repositories persist and query data`() {
        val database = newDatabase()
        val projectRepository = SqliteProjectRepository(database)
        val conversationRepository = SqliteConversationRepository(database)
        val now = Instant.parse("2026-05-07T12:00:00Z")

        val project = Project(
            id = "project-1",
            name = "Roadmap",
            description = "MVP",
            createdAt = now,
            updatedAt = now,
        )
        projectRepository.create(project)

        val conversation = Conversation(
            id = "conv-1",
            projectId = project.id,
            title = "Kickoff",
            source = "Ollama",
            model = "qwen3",
            createdAt = now,
            updatedAt = now,
            lastActivityAt = now,
            archivedAt = null,
        )
        conversationRepository.create(conversation)

        val loadedProject = projectRepository.getById(project.id)
        val loadedConversation = conversationRepository.getById(conversation.id)

        assertNotNull(loadedProject)
        assertNotNull(loadedConversation)
        assertEquals("Roadmap", loadedProject.name)
        assertEquals("Kickoff", loadedConversation.title)

        val archivedConversation = loadedConversation.copy(archivedAt = now.plusSeconds(60), updatedAt = now.plusSeconds(60))
        conversationRepository.update(archivedConversation)

        assertTrue(conversationRepository.listByProject(project.id, includeArchived = false).isEmpty())
        assertEquals(1, conversationRepository.listByProject(project.id, includeArchived = true).size)
    }

    @Test
    fun `fts search returns matching conversation and applies archive filter`() {
        val database = newDatabase()
        val projectRepository = SqliteProjectRepository(database)
        val conversationRepository = SqliteConversationRepository(database)
        val searchRepository = SqliteConversationSearchRepository(database)
        val now = Instant.parse("2026-05-07T12:00:00Z")

        val project = Project(
            id = "project-search",
            name = "Search",
            description = "history",
            createdAt = now,
            updatedAt = now,
        )
        projectRepository.create(project)

        val activeConversation = Conversation(
            id = "conv-active",
            projectId = project.id,
            title = "Budget thread",
            source = "LiteLLM",
            model = "gpt-oss",
            createdAt = now,
            updatedAt = now,
            lastActivityAt = now,
            archivedAt = null,
        )
        val archivedConversation = Conversation(
            id = "conv-archived",
            projectId = project.id,
            title = "Archive thread",
            source = "LiteLLM",
            model = "gpt-oss",
            createdAt = now,
            updatedAt = now.plusSeconds(1),
            lastActivityAt = now.plusSeconds(1),
            archivedAt = now.plusSeconds(1),
        )

        conversationRepository.create(activeConversation)
        conversationRepository.create(archivedConversation)

        searchRepository.indexMessage("m-1", activeConversation.id, "budget planning 2026")
        searchRepository.indexMessage("m-2", archivedConversation.id, "budget risks and fallback")

        val visible = searchRepository.search(project.id, "budget", includeArchived = false)
        val all = searchRepository.search(project.id, "budget", includeArchived = true)

        assertEquals(listOf("conv-active"), visible.map { it.conversationId })
        assertEquals(setOf("conv-active", "conv-archived"), all.map { it.conversationId }.toSet())
    }

    private fun newDatabase(): SqliteDatabase {
        val dbPath = Files.createTempFile("liaotao-persistence-", ".db")
        val database = SqliteDatabase.fromPath(dbPath)
        database.migrate()
        return database
    }
}