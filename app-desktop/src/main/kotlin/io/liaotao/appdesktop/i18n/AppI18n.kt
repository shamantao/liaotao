/*
 * AppI18n.kt - lightweight localization contract and message catalog.
 * Responsibilities: provide centralized UI labels and default language
 * values for desktop screens.
 */

package io.liaotao.appdesktop.i18n

import androidx.compose.runtime.staticCompositionLocalOf

enum class AppLanguage {
    ENGLISH,
    FRENCH,
}

data class AppStrings(
    val appName: String,
    val settings: String,
    val backToChat: String,
    val language: String,
    val english: String,
    val french: String,
    val folders: String,
    val conversations: String,
    val executionAttempts: String,
    val askYourQuestion: String,
    val noProviderEnabled: String,
    val attachFile: String,
    val attachFileDialog: String,
    val exportDialog: String,
    val send: String,
    val retry: String,
    val providerSettings: String,
    val createProvider: String,
    val displayName: String,
    val type: String,
    val baseUrl: String,
    val defaultModel: String,
    val secretRef: String,
    val secretRefHint: String,
    val addProvider: String,
    val enabled: String,
    val save: String,
    val validate: String,
    val delete: String,
    val status: String,
    val mcpServerSettings: String,
    val serverName: String,
    val serverUrl: String,
    val notChecked: String,
) {
    fun modelLabel(providerLabel: String): String = "Model: $providerLabel"
    fun attached(fileName: String): String = "Attached $fileName"
    fun foldersExported(path: String): String = "Folders exported to $path"
    fun conversationsExported(path: String): String = "Conversations exported to $path"
    fun exportFailed(reason: String?): String = "Export failed: ${reason ?: "Unknown error"}"
    fun attachmentReadFailed(reason: String?): String = "Attachment read failed: ${reason ?: "Unknown error"}"
    fun providerAutoName(index: Int): String = "Provider $index"
    fun folderLabel(index: Int): String = "Folder #$index"
    fun healthLatency(ms: Long): String = "Healthy ($ms ms)"
    fun degraded(message: String): String = "Degraded: $message"
    fun statusLabel(value: String): String = "Status: $value"
}

object AppI18n {
    fun strings(language: AppLanguage): AppStrings {
        return when (language) {
            AppLanguage.ENGLISH -> englishStrings()
            AppLanguage.FRENCH -> frenchStrings()
        }
    }

    private fun englishStrings(): AppStrings {
        return AppStrings(
            appName = "Liaotao",
            settings = "Settings",
            backToChat = "Back to chat",
            language = "Language",
            english = "English",
            french = "French",
            folders = "Folders",
            conversations = "Conversations",
            executionAttempts = "Execution attempts",
            askYourQuestion = "Ask your question",
            noProviderEnabled = "No provider enabled",
            attachFile = "Attach file",
            attachFileDialog = "Attach file",
            exportDialog = "Export conversations",
            send = "Send",
            retry = "Retry",
            providerSettings = "Provider Settings",
            createProvider = "Create provider",
            displayName = "Display name",
            type = "Type",
            baseUrl = "Base URL",
            defaultModel = "Default model",
            secretRef = "Secret Ref",
            secretRefHint = "Secret Ref (stored in OS keychain)",
            addProvider = "Add provider",
            enabled = "Enabled",
            save = "Save",
            validate = "Validate",
            delete = "Delete",
            status = "Status",
            mcpServerSettings = "MCP Server Settings",
            serverName = "Server Name",
            serverUrl = "Server URL",
            notChecked = "Not checked",
        )
    }

    private fun frenchStrings(): AppStrings {
        return AppStrings(
            appName = "Liaotao",
            settings = "Parametres",
            backToChat = "Retour au chat",
            language = "Langue",
            english = "Anglais",
            french = "Francais",
            folders = "Dossiers",
            conversations = "Conversations",
            executionAttempts = "Tentatives d'execution",
            askYourQuestion = "Pose ta question",
            noProviderEnabled = "Aucun provider active",
            attachFile = "Joindre un fichier",
            attachFileDialog = "Joindre un fichier",
            exportDialog = "Exporter les conversations",
            send = "Envoyer",
            retry = "Relancer",
            providerSettings = "Parametres provider",
            createProvider = "Creer un provider",
            displayName = "Nom affiche",
            type = "Type",
            baseUrl = "URL de base",
            defaultModel = "Modele par defaut",
            secretRef = "Reference secret",
            secretRefHint = "Reference secret (stockee dans le trousseau OS)",
            addProvider = "Ajouter provider",
            enabled = "Active",
            save = "Enregistrer",
            validate = "Valider",
            delete = "Supprimer",
            status = "Statut",
            mcpServerSettings = "Parametres serveur MCP",
            serverName = "Nom du serveur",
            serverUrl = "URL du serveur",
            notChecked = "Non verifie",
        )
    }
}

val LocalAppStrings = staticCompositionLocalOf { AppI18n.strings(AppLanguage.ENGLISH) }
