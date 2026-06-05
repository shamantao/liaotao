package io.liaotao.appcli.commands

import com.github.ajalt.clikt.core.CliktCommand
import com.github.ajalt.clikt.parameters.arguments.argument
import com.github.ajalt.clikt.parameters.arguments.optional
import com.github.ajalt.mordant.rendering.TextColors
import com.github.ajalt.mordant.rendering.TextStyles
import com.github.ajalt.mordant.terminal.Terminal
import io.liaotao.appcli.CliContext
import io.liaotao.appcli.ui.error
import io.liaotao.appcli.ui.header
import io.liaotao.appcli.ui.info
import io.liaotao.appcli.ui.success
import io.liaotao.connectors.core.ConnectorChatRequest
import io.liaotao.connectors.core.ConnectorExecutionConfig
import io.liaotao.connectors.core.ConnectorMessage
import io.liaotao.connectors.core.ConnectorStreamResult
import io.liaotao.connectors.core.ConnectorRegistry
import io.liaotao.domain.conversations.CreateConversationRequest
import io.liaotao.domain.conversations.UpdateConversationRequest
import io.liaotao.shared.settings.ConnectorSetting
import java.io.BufferedReader
import java.io.InputStreamReader

class ChatCommand(
    private val ctx: CliContext,
    private val term: Terminal,
) : CliktCommand(name = "chat") {
    override fun help(context: com.github.ajalt.clikt.core.Context): String = "Start an interactive chat session"

    private val prompt by argument("prompt", help = "Optional prompt for one-shot mode").optional()

    override fun run() {
        val provider = activeProvider() ?: run {
            term.error("No provider configured. Run: liaotao config add")
            return
        }

        if (prompt != null) {
            oneShot(provider, prompt!!)
        } else {
            interactive(provider)
        }
    }

    private fun activeProvider(): ConnectorSetting? {
        val all = ctx.connectorSettingsRepository.listAll()
        return all.firstOrNull { it.isEnabled } ?: all.firstOrNull()
    }

    private fun oneShot(provider: ConnectorSetting, message: String) {
        term.info("Using ${provider.displayName} (${provider.defaultModel})")
        streamAndPrint(provider, listOf(ConnectorMessage(role = "user", content = message))) { chunk ->
            print(chunk)
        }
        println()
        term.success("Done")
    }

    private fun interactive(provider: ConnectorSetting) {
        val reader = BufferedReader(InputStreamReader(System.`in`))
        val messages = mutableListOf<ConnectorMessage>()
        var conversationId: String? = null

        term.header("liaotao chat", "${provider.displayName} :: ${provider.defaultModel}")
        term.info("Type your message. Commands: /exit /help /new")
        println()

        while (true) {
            print(TextStyles.bold("You: "))
            val input = reader.readLine()?.trim() ?: break

            when {
                input.startsWith("/exit") || input.startsWith("/quit") -> {
                    term.success("Bye!")
                    break
                }
                input.startsWith("/help") -> showHelp()
                input.startsWith("/new") -> {
                    messages.clear()
                    conversationId = null
                    term.success("New conversation started")
                    continue
                }
                input.isBlank() -> continue
            }

            if (conversationId == null) {
                val defaultProject = ctx.projectService.listProjects().first()
                conversationId = ctx.projectService.createConversation(
                    CreateConversationRequest(
                        projectId = defaultProject.id,
                        title = input.take(60),
                        source = provider.displayName,
                        model = provider.defaultModel,
                    ),
                ).id
            }

            messages.add(ConnectorMessage(role = "user", content = input))
            print(TextStyles.bold("Assistant: "))

            val reply = StringBuilder()
            val ok = streamAndPrint(provider, messages.toList()) { chunk ->
                print(chunk)
                reply.append(chunk)
            }
            println()

            if (ok) {
                messages.add(ConnectorMessage(role = "assistant", content = reply.toString()))
                ctx.projectService.updateConversation(
                    conversationId!!,
                    UpdateConversationRequest(
                        title = messages.firstOrNull { it.role == "user" }?.content?.take(60) ?: "Chat",
                        source = provider.displayName,
                        model = provider.defaultModel,
                    ),
                )
            } else {
                messages.removeLast()
                term.error("No response from provider")
            }
        }
    }

    private fun showHelp() {
        println()
        term.info("Commands:")
        println("  /exit       Exit")
        println("  /help       Show this help")
        println("  /new        New conversation")
    }

    private fun streamAndPrint(
        provider: ConnectorSetting,
        messages: List<ConnectorMessage>,
        onChunk: (String) -> Unit,
    ): Boolean {
        val connector = ConnectorRegistry.create(provider.connectorType)
            ?: return false

        val config = ConnectorExecutionConfig(
            baseUrl = provider.baseUrl,
            apiKey = provider.secretRef?.let { ctx.secretStore.getSecret(it) },
        )

        val request = ConnectorChatRequest(
            model = provider.defaultModel,
            messages = messages,
        )

        return when (val stream = connector.streamChat(config, request)) {
            is ConnectorStreamResult.Success -> {
                stream.chunks.forEach { chunk ->
                    if (chunk.content.isNotEmpty()) onChunk(chunk.content)
                }
                true
            }
            is ConnectorStreamResult.Failure -> {
                when (val result = connector.chat(config, request)) {
                    is io.liaotao.connectors.core.ConnectorChatResult.Success -> {
                        onChunk(result.response.content)
                        true
                    }
                    is io.liaotao.connectors.core.ConnectorChatResult.Failure -> false
                }
            }
        }
    }
}
