/*
 * ExportPackageModels.kt - versioned export/import package schema.
 * Responsibilities: define portable JSON payload for conversations/projects
 * without embedding secrets or credential material.
 */

package io.liaotao.shared.exportimport

import kotlinx.serialization.Serializable

@Serializable
data class LiaotaoExportPackage(
    val schemaVersion: Int,
    val exportedAt: String,
    val conversations: List<ExportConversation>,
)

@Serializable
data class ExportConversation(
    val id: String,
    val projectId: String,
    val title: String,
    val source: String,
    val model: String,
    val createdAt: String,
    val updatedAt: String,
    val lastActivityAt: String,
    val archivedAt: String? = null,
    val messages: List<ExportMessage>,
)

@Serializable
data class ExportMessage(
    val role: String,
    val content: String,
    val createdAt: String,
)

@Serializable
data class ImportReport(
    val importedConversations: Int,
    val partialErrors: List<String>,
)

object ExportSchema {
    const val CURRENT_VERSION: Int = 1
}