/*
 * ProjectConversationServiceTest.kt - tests for project and conversation flows.
 * Responsibilities: verify CRUD operations, archive lifecycle, move/duplicate
 * behavior, and timestamp updates required by P1.2.
 */

package io.liaotao.persistence.memory

import io.liaotao.domain.conversations.CreateConversationRequest
import io.liaotao.domain.conversations.UpdateConversationRequest
import io.liaotao.domain.projects.CreateProjectRequest
import io.liaotao.domain.projects.ProjectConversationService
import io.liaotao.domain.projects.UpdateProjectRequest
import java.time.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNotEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

class ProjectConversationServiceTest {
    @Test
    fun `project and conversation CRUD keeps metadata consistent`() {
        val clock = FakeClock()
        val id = FakeIds()
        val service = newService(clock, id)

        val project = service.createProject(CreateProjectRequest(name = "Roadmap", description = "2026"))
        assertEquals("Roadmap", project.name)
        assertEquals(project.createdAt, project.updatedAt)

        clock.tickSeconds(30)
        val updatedProject = service.updateProject(project.id, UpdateProjectRequest(name = "Roadmap V2", description = "2027"))
        assertEquals("Roadmap V2", updatedProject.name)
        assertTrue(updatedProject.updatedAt.isAfter(updatedProject.createdAt))

        val conversation = service.createConversation(
            CreateConversationRequest(
                projectId = project.id,
                title = "Kickoff",
                source = "Ollama",
                model = "qwen3",
            ),
        )
        assertEquals(project.id, conversation.projectId)
        assertNull(conversation.archivedAt)

        clock.tickSeconds(20)
        val updatedConversation = service.updateConversation(
            conversation.id,
            UpdateConversationRequest(title = "Kickoff 2", source = "LiteLLM", model = "gpt-oss"),
        )
        assertEquals("Kickoff 2", updatedConversation.title)
        assertTrue(updatedConversation.lastActivityAt.isAfter(conversation.lastActivityAt))
    }

    @Test
    fun `archive restore move and duplicate work end to end`() {
        val clock = FakeClock()
        val id = FakeIds()
        val service = newService(clock, id)

        val p1 = service.createProject(CreateProjectRequest(name = "A"))
        val p2 = service.createProject(CreateProjectRequest(name = "B"))
        val c1 = service.createConversation(
            CreateConversationRequest(
                projectId = p1.id,
                title = "Spec",
                source = "Aitao",
                model = "m1",
            ),
        )

        clock.tickSeconds(10)
        val archived = service.archiveConversation(c1.id)
        assertTrue(archived.isArchived)
        assertNotNull(archived.archivedAt)

        val visibleWithoutArchive = service.listConversations(p1.id, includeArchived = false)
        assertTrue(visibleWithoutArchive.isEmpty())

        clock.tickSeconds(10)
        val restored = service.restoreConversation(c1.id)
        assertFalse(restored.isArchived)
        assertNull(restored.archivedAt)

        clock.tickSeconds(10)
        val moved = service.moveConversation(c1.id, p2.id)
        assertEquals(p2.id, moved.projectId)

        val p1Conversations = service.listConversations(p1.id, includeArchived = true)
        assertTrue(p1Conversations.isEmpty())

        val p2Conversations = service.listConversations(p2.id, includeArchived = true)
        assertEquals(1, p2Conversations.size)

        clock.tickSeconds(10)
        val duplicate = service.duplicateConversation(c1.id, targetProjectId = p1.id)
        assertNotEquals(c1.id, duplicate.id)
        assertEquals(p1.id, duplicate.projectId)
        assertTrue(duplicate.title.contains("copy"))

        val deleted = service.deleteConversation(c1.id)
        assertTrue(deleted)
    }

    @Test
    fun `deleting project removes related conversations`() {
        val service = newService(FakeClock(), FakeIds())
        val project = service.createProject(CreateProjectRequest(name = "Project X"))
        service.createConversation(
            CreateConversationRequest(
                projectId = project.id,
                title = "Thread 1",
                source = "MCP",
                model = "tooling",
            ),
        )

        assertTrue(service.deleteProject(project.id))
        assertTrue(service.listProjects().isEmpty())
    }

    private fun newService(clock: FakeClock, ids: FakeIds): ProjectConversationService {
        return ProjectConversationService(
            projectRepository = InMemoryProjectRepository(),
            conversationRepository = InMemoryConversationRepository(),
            nowProvider = { clock.now() },
            idProvider = { ids.next() },
        )
    }
}

private class FakeClock {
    private var current = Instant.parse("2026-05-07T12:00:00Z")

    fun now(): Instant = current

    fun tickSeconds(seconds: Long) {
        current = current.plusSeconds(seconds)
    }
}

private class FakeIds {
    private var value = 0

    fun next(): String {
        value += 1
        return "id-$value"
    }
}