package io.liaotao.appcli

import io.liaotao.connectors.core.ConnectorRegistry
import io.liaotao.connectors.core.ConnectorType
import io.liaotao.domain.conversations.ConversationRepository
import io.liaotao.domain.projects.ProjectConversationService
import io.liaotao.domain.projects.ProjectRepository
import io.liaotao.domain.projects.CreateProjectRequest
import io.liaotao.domain.routing.InMemoryExecutionHistoryRepository
import io.liaotao.domain.routing.RoutingPolicyService
import io.liaotao.persistence.secrets.InMemorySecretStore
import io.liaotao.persistence.secrets.SecretStore
import io.liaotao.persistence.sqlite.SqliteConnectorSettingsRepository
import io.liaotao.persistence.sqlite.SqliteConversationRepository
import io.liaotao.persistence.sqlite.SqliteDatabase
import io.liaotao.persistence.sqlite.SqliteProjectRepository
import io.liaotao.shared.logging.Logger
import io.liaotao.shared.logging.logger
import io.liaotao.shared.paths.PathManager
import io.liaotao.shared.settings.ConnectorSetting
import io.liaotao.shared.settings.ConnectorSettingsRepository

data class CliContext(
    val projectService: ProjectConversationService,
    val connectorSettingsRepository: ConnectorSettingsRepository,
    val conversationRepository: ConversationRepository,
    val projectRepository: ProjectRepository,
    val connectorRegistry: ConnectorRegistry,
    val routingService: RoutingPolicyService,
    val secretStore: SecretStore,
    val pathManager: PathManager,
    val logger: Logger,
)

fun createCliContext(
    pathManager: PathManager = CliPathManager,
    log: Logger = logger("liaotao-cli"),
): CliContext {
    val dbPath = pathManager.appDatabaseFile("liaotao.db")
    log.info("Initializing database at $dbPath")

    val database = SqliteDatabase.fromPath(dbPath).also { it.migrate() }
    log.debug("Database migration complete")

    val projectRepository = SqliteProjectRepository(database)
    val conversationRepository = SqliteConversationRepository(database)
    val connectorSettingsRepository = SqliteConnectorSettingsRepository(database)
    val secretStore = InMemorySecretStore()

    val executionHistoryRepo = InMemoryExecutionHistoryRepository()
    val routingService = RoutingPolicyService(historyRepository = executionHistoryRepo)

    val projectService = ProjectConversationService(
        projectRepository = projectRepository,
        conversationRepository = conversationRepository,
    )

    val projects = projectService.listProjects()
    if (projects.isEmpty()) {
        log.info("No projects found, creating default project")
        projectService.createProject(CreateProjectRequest(name = "Default"))
    }

    log.info("Ready — ${projects.size} project(s), ${connectorSettingsRepository.listAll().size} provider(s)")

    return CliContext(
        projectService = projectService,
        connectorSettingsRepository = connectorSettingsRepository,
        conversationRepository = conversationRepository,
        projectRepository = projectRepository,
        connectorRegistry = ConnectorRegistry,
        routingService = routingService,
        secretStore = secretStore,
        pathManager = pathManager,
        logger = log,
    )
}

fun ConnectorType.toDisplayName(): String = when (this) {
    ConnectorType.OPENAI_COMPAT -> "OpenAI Compatible"
    ConnectorType.OLLAMA -> "Ollama"
    ConnectorType.LITELLM -> "LiteLLM"
    ConnectorType.AITAO -> "Aitao"
}

fun ConnectorType.toDefaultUrl(): String = when (this) {
    ConnectorType.OPENAI_COMPAT -> "https://api.openai.com/v1"
    ConnectorType.OLLAMA -> "http://localhost:11434"
    ConnectorType.LITELLM -> "http://localhost:4000"
    ConnectorType.AITAO -> "http://localhost:8080"
}

fun firstEnabledProvider(settingsRepo: ConnectorSettingsRepository): ConnectorSetting? {
    return settingsRepo.listAll().firstOrNull { it.isEnabled }
}
