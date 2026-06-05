package io.liaotao.appcli.commands

import com.github.ajalt.clikt.core.CliktCommand
import com.github.ajalt.clikt.parameters.arguments.argument
import com.github.ajalt.mordant.terminal.Terminal
import io.liaotao.appcli.CliContext
import io.liaotao.appcli.ui.*
import io.liaotao.appcli.toDisplayName
import io.liaotao.appcli.toDefaultUrl
import io.liaotao.connectors.core.ConnectorType
import io.liaotao.connectors.core.ConnectorExecutionConfig
import io.liaotao.connectors.core.ConnectorRegistry
import io.liaotao.shared.settings.ConnectorSetting
import java.io.BufferedReader
import java.io.InputStreamReader
import java.time.Instant
import java.util.UUID

class ConfigCommand(
    private val ctx: CliContext,
    private val term: Terminal,
) : CliktCommand(name = "config") {
    override val invokeWithoutSubcommand = true
    override fun help(context: com.github.ajalt.clikt.core.Context): String = "Manage provider configurations"

    override fun run() {
        if (currentContext.invokedSubcommand != null) return
        ctx.connectorSettingsRepository.listAll().let { providers ->
            if (providers.isEmpty()) {
                term.info("No providers configured. Use: liaotao config add")
                return
            }
            term.header("Providers", "${providers.size} configured")
            term.renderTable(
                columns = listOf("ID", "Type", "Name", "Model", "Status"),
                rows = providers.map { p ->
                    listOf(p.id.take(8), p.connectorType.lowercase(), p.displayName, p.defaultModel,
                        if (p.isEnabled) "enabled" else "disabled")
                },
            )
        }
    }
}

class ConfigList(
    private val ctx: CliContext,
    private val term: Terminal,
) : CliktCommand(name = "list") {
    override fun help(context: com.github.ajalt.clikt.core.Context): String = "List configured providers"

    override fun run() {
        val providers = ctx.connectorSettingsRepository.listAll()
        if (providers.isEmpty()) {
            term.info("No providers configured")
            return
        }
        term.header("Providers", "${providers.size} configured")
        term.renderTable(
            columns = listOf("ID", "Type", "Name", "Model", "Status"),
            rows = providers.map { p ->
                listOf(p.id.take(8), p.connectorType.lowercase(), p.displayName, p.defaultModel,
                    if (p.isEnabled) "enabled" else "disabled")
            },
        )
    }
}

class ConfigAdd(
    private val ctx: CliContext,
    private val term: Terminal,
) : CliktCommand(name = "add") {
    override fun help(context: com.github.ajalt.clikt.core.Context): String = "Add a new provider"

    override fun run() {
        val reader = BufferedReader(InputStreamReader(System.`in`))

        term.header("Add Provider", "")
        term.info("Select type:")
        ConnectorType.entries.forEachIndexed { i, type ->
            println("  ${i + 1}) ${type.toDisplayName()}")
        }
        print("Type [1-${ConnectorType.entries.size}]: ")
        val typeIdx = (reader.readLine()?.trim()?.toIntOrNull() ?: 1).coerceIn(1, ConnectorType.entries.size) - 1
        val connectorType = ConnectorType.entries[typeIdx]

        print("Display name [${connectorType.toDisplayName()}]: ")
        val displayName = reader.readLine()?.trim()?.ifBlank { connectorType.toDisplayName() } ?: connectorType.toDisplayName()

        print("Base URL [${connectorType.toDefaultUrl()}]: ")
        val baseUrl = reader.readLine()?.trim()?.ifBlank { connectorType.toDefaultUrl() } ?: connectorType.toDefaultUrl()

        print("Default model: ")
        val defaultModel = reader.readLine()?.trim() ?: ""
        if (defaultModel.isBlank()) {
            term.error("Model is required")
            return
        }

        print("API key (optional): ")
        val apiKey = reader.readLine()?.trim() ?: ""

        val id = UUID.randomUUID().toString()
        val secretRef = if (apiKey.isNotBlank()) {
            val ref = "connector:$id"
            ctx.secretStore.putSecret(ref, apiKey)
            ref
        } else null

        ctx.connectorSettingsRepository.create(
            ConnectorSetting(
                id = id,
                connectorType = connectorType.name,
                displayName = displayName,
                baseUrl = baseUrl,
                defaultModel = defaultModel,
                isEnabled = true,
                secretRef = secretRef,
                createdAt = Instant.now(),
                updatedAt = Instant.now(),
            ),
        )

        term.success("Provider '$displayName' added")
    }
}

class ConfigEdit(
    private val ctx: CliContext,
    private val term: Terminal,
) : CliktCommand(name = "edit") {
    override fun help(context: com.github.ajalt.clikt.core.Context): String = "Edit a provider"

    private val providerId by argument("id", help = "Provider ID")

    override fun run() {
        val setting = ctx.connectorSettingsRepository.getById(providerId)
            ?: run { term.error("Provider not found: $providerId"); return }

        val reader = BufferedReader(InputStreamReader(System.`in`))
        println("Editing ${setting.displayName} (leave blank to keep current)")
        println()

        print("Display name [${setting.displayName}]: ")
        val displayName = reader.readLine()?.trim()?.ifBlank { setting.displayName } ?: setting.displayName

        print("Base URL [${setting.baseUrl}]: ")
        val baseUrl = reader.readLine()?.trim()?.ifBlank { setting.baseUrl } ?: setting.baseUrl

        print("Default model [${setting.defaultModel}]: ")
        val defaultModel = reader.readLine()?.trim()?.ifBlank { setting.defaultModel } ?: setting.defaultModel

        print("API key (blank to keep, 'clear' to remove): ")
        val apiKey = reader.readLine()?.trim()
        val secretRef = when {
            apiKey == null || apiKey.isBlank() -> setting.secretRef
            apiKey == "clear" -> {
                setting.secretRef?.let { ctx.secretStore.deleteSecret(it) }
                null
            }
            else -> {
                setting.secretRef?.let { ctx.secretStore.deleteSecret(it) }
                val ref = "connector:${setting.id}"
                ctx.secretStore.putSecret(ref, apiKey)
                ref
            }
        }

        ctx.connectorSettingsRepository.update(
            setting.copy(displayName = displayName, baseUrl = baseUrl, defaultModel = defaultModel,
                secretRef = secretRef, updatedAt = Instant.now()),
        )
        term.success("Provider '$displayName' updated")
    }
}

class ConfigDelete(
    private val ctx: CliContext,
    private val term: Terminal,
) : CliktCommand(name = "delete") {
    override fun help(context: com.github.ajalt.clikt.core.Context): String = "Delete a provider"

    private val providerId by argument("id", help = "Provider ID")

    override fun run() {
        val setting = ctx.connectorSettingsRepository.getById(providerId)
            ?: run { term.error("Provider not found: $providerId"); return }

        term.warn("Delete '${setting.displayName}'? (y/N)")
        val confirm = BufferedReader(InputStreamReader(System.`in`)).readLine()?.trim()?.lowercase()
        if (confirm != "y" && confirm != "yes") { term.info("Cancelled"); return }

        setting.secretRef?.let { ctx.secretStore.deleteSecret(it) }
        ctx.connectorSettingsRepository.delete(providerId)
        term.success("Provider deleted")
    }
}

class ConfigTest(
    private val ctx: CliContext,
    private val term: Terminal,
) : CliktCommand(name = "test") {
    override fun help(context: com.github.ajalt.clikt.core.Context): String = "Test a provider connection"

    private val providerId by argument("id", help = "Provider ID")

    override fun run() {
        val setting = ctx.connectorSettingsRepository.getById(providerId)
            ?: run { term.error("Provider not found: $providerId"); return }

        term.info("Testing ${setting.displayName}...")
        val connector = ConnectorRegistry.create(setting.connectorType)
            ?: run { term.error("Unknown connector type"); return }

        val result = connector.validateConfiguration(
            ConnectorExecutionConfig(
                baseUrl = setting.baseUrl,
                apiKey = setting.secretRef?.let { ctx.secretStore.getSecret(it) },
            ),
        )

        if (result.isValid) term.success("${result.latencyMs}ms — reachable")
        else term.error(result.message)
    }
}
