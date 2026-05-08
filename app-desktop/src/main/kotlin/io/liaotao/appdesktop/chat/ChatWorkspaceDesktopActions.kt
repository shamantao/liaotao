/*
 * ChatWorkspaceDesktopActions.kt - desktop file actions for chat workspace.
 * Responsibilities: handle file chooser flows for export and attachment in
 * Compose Desktop screens.
 */

package io.liaotao.appdesktop.chat

import io.liaotao.persistence.files.ConversationImportExportService
import io.liaotao.persistence.files.ConversationTranscript
import java.awt.FileDialog
import java.awt.Frame
import java.io.File
import java.nio.charset.StandardCharsets
import java.nio.file.Files
import java.nio.file.Path

internal object ChatWorkspaceDesktopActions {
    private val exportService = ConversationImportExportService()

    fun chooseAttachment(): Path? {
        val dialog = FileDialog(null as Frame?, "Attach file", FileDialog.LOAD)
        dialog.isVisible = true
        val fileName = dialog.file ?: return null
        return File(dialog.directory, fileName).toPath()
    }

    fun chooseExportTarget(defaultFileName: String): Path? {
        val dialog = FileDialog(null as Frame?, "Export conversations", FileDialog.SAVE)
        dialog.file = defaultFileName
        dialog.isVisible = true
        val fileName = dialog.file ?: return null
        return File(dialog.directory, fileName).toPath()
    }

    fun exportAllConversations(transcripts: List<ConversationTranscript>, outputPath: Path): Result<Path> {
        return runCatching {
            val payload = exportService.exportAllConversations(transcripts)
            Files.writeString(outputPath, payload, StandardCharsets.UTF_8)
            outputPath
        }
    }

    fun exportProjectConversations(transcripts: List<ConversationTranscript>, projectId: String, outputPath: Path): Result<Path> {
        return runCatching {
            val payload = exportService.exportProjectConversations(transcripts, projectId)
            Files.writeString(outputPath, payload, StandardCharsets.UTF_8)
            outputPath
        }
    }

    fun readAttachmentPreview(path: Path, maxChars: Int = 8000): String {
        val content = Files.readString(path, StandardCharsets.UTF_8)
        return if (content.length <= maxChars) content else content.take(maxChars)
    }
}
