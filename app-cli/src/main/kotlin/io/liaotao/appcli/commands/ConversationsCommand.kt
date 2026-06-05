package io.liaotao.appcli.commands

import com.github.ajalt.clikt.core.CliktCommand
import com.github.ajalt.clikt.parameters.arguments.argument
import com.github.ajalt.mordant.terminal.Terminal
import io.liaotao.appcli.CliContext
import io.liaotao.appcli.ui.*
import java.io.BufferedReader
import java.io.InputStreamReader

class ConversationsCommand(
    private val ctx: CliContext,
    private val term: Terminal,
) : CliktCommand(name = "conversations") {
    override val invokeWithoutSubcommand = true
    override fun help(context: com.github.ajalt.clikt.core.Context): String = "Manage conversations"

    override fun run() {
        if (currentContext.invokedSubcommand != null) return
        ctx.projectService.listProjects().firstOrNull()?.let { project ->
            val convs = ctx.projectService.listConversations(project.id)
            if (convs.isEmpty()) {
                term.info("No conversations yet")
                return
            }
            term.header("Conversations", "${convs.size} total")
            term.renderTable(
                columns = listOf("ID", "Title", "Source", "Date"),
                rows = convs.take(20).map { c ->
                    listOf(c.id.take(8), c.title.take(30), c.source,
                        c.lastActivityAt.toString().take(10))
                },
            )
        } ?: term.info("No projects found")
    }
}

class ConversationList(
    private val ctx: CliContext,
    private val term: Terminal,
) : CliktCommand(name = "list") {
    override fun help(context: com.github.ajalt.clikt.core.Context): String = "List conversations"

    override fun run() {
        val project = ctx.projectService.listProjects().firstOrNull()
            ?: run { term.info("No projects"); return }

        val convs = ctx.projectService.listConversations(project.id)
        if (convs.isEmpty()) { term.info("No conversations"); return }

        term.header("Conversations", "${convs.size} total")
        term.renderTable(
            columns = listOf("ID", "Title", "Source", "Model", "Messages"),
            rows = convs.take(20).map { c ->
                listOf(c.id.take(8), c.title.take(28), c.source, c.model, "0")
            },
        )
    }
}

class ConversationShow(
    private val ctx: CliContext,
    private val term: Terminal,
) : CliktCommand(name = "show") {
    override fun help(context: com.github.ajalt.clikt.core.Context): String = "Show conversation details"

    private val conversationId by argument("id", help = "Conversation ID")

    override fun run() {
        val conv = ctx.conversationRepository.getById(conversationId)
            ?: run { term.error("Conversation not found"); return }

        term.header(conv.title, "${conv.source} · ${conv.model}")
        term.info("Created: ${conv.createdAt}")
        if (conv.isArchived) term.warn("Archived")
    }
}

class ConversationDelete(
    private val ctx: CliContext,
    private val term: Terminal,
) : CliktCommand(name = "delete") {
    override fun help(context: com.github.ajalt.clikt.core.Context): String = "Delete a conversation"

    private val conversationId by argument("id", help = "Conversation ID")

    override fun run() {
        val conv = ctx.conversationRepository.getById(conversationId)
            ?: run { term.error("Conversation not found"); return }

        term.warn("Delete '${conv.title}'? (y/N)")
        val confirm = BufferedReader(InputStreamReader(System.`in`)).readLine()?.trim()?.lowercase()
        if (confirm != "y" && confirm != "yes") { term.info("Cancelled"); return }

        ctx.projectService.deleteConversation(conversationId)
        term.success("Conversation deleted")
    }
}
