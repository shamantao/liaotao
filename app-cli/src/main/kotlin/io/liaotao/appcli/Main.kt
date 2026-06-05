package io.liaotao.appcli

import com.github.ajalt.clikt.core.CliktCommand
import com.github.ajalt.clikt.core.main
import com.github.ajalt.clikt.core.Context
import com.github.ajalt.clikt.core.subcommands
import com.github.ajalt.mordant.rendering.TextColors
import com.github.ajalt.mordant.rendering.TextStyles
import io.liaotao.appcli.commands.*

fun main(args: Array<String>) {
    val term = com.github.ajalt.mordant.terminal.Terminal()
    val ctx = try {
        createCliContext()
    } catch (e: Exception) {
        term.println(" ${TextColors.brightRed("✗")} ${e.message}")
        kotlin.system.exitProcess(1)
    }

    LiaotaoCli(ctx, term).subcommands(
        ChatCommand(ctx, term),
        ConfigCommand(ctx, term).subcommands(
            ConfigList(ctx, term),
            ConfigAdd(ctx, term),
            ConfigEdit(ctx, term),
            ConfigDelete(ctx, term),
            ConfigTest(ctx, term),
        ),
        ConversationsCommand(ctx, term).subcommands(
            ConversationList(ctx, term),
            ConversationShow(ctx, term),
            ConversationDelete(ctx, term),
        ),
    ).main(args)
}

class LiaotaoCli(
    private val ctx: CliContext,
    private val term: com.github.ajalt.mordant.terminal.Terminal,
) : CliktCommand(name = "liaotao") {
    override val invokeWithoutSubcommand = true
    override fun help(context: Context): String = "AI chat from the terminal"

    override fun run() {
        if (currentContext.invokedSubcommand != null) return

        val enabled = ctx.connectorSettingsRepository.listAll().count { it.isEnabled }
        val projects = ctx.projectService.listProjects().size

        term.println()
        term.println(TextColors.brightGreen(TextStyles.bold(" liaotao v${Version.current}")) +
            TextColors.gray(" — $projects project(s), $enabled provider(s)"))
        term.println()

        if (enabled == 0) {
            term.println(" ${TextColors.brightYellow("!")} No provider configured. Run: liaotao config add")
            term.println()
        }
    }
}
