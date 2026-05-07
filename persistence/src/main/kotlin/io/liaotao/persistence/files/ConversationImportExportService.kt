/*
 * ConversationImportExportService.kt - JSON export and import service.
 * Responsibilities: export single/project conversation bundles, validate
 * schema version on import, and report partial import errors safely.
 */

package io.liaotao.persistence.files

import io.liaotao.domain.conversations.Conversation
import io.liaotao.shared.exportimport.ExportConversation
import io.liaotao.shared.exportimport.ExportMessage
import io.liaotao.shared.exportimport.ExportSchema
import io.liaotao.shared.exportimport.ImportReport
import io.liaotao.shared.exportimport.LiaotaoExportPackage
import kotlinx.serialization.json.Json
import java.time.Instant

data class ConversationTranscript(
    val conversation: Conversation,
    val messages: List<TranscriptMessage>,
)

data class TranscriptMessage(
    val role: String,
    val content: String,
    val createdAt: Instant,
)

class ConversationImportExportService(
    private val json: Json = Json { prettyPrint = true; ignoreUnknownKeys = true },
    private val nowProvider: () -> Instant = { Instant.now() },
) {
    fun exportSingleConversation(transcript: ConversationTranscript): String {
        return exportPackage(listOf(transcript))
    }

    fun exportProjectConversations(transcripts: List<ConversationTranscript>, projectId: String): String {
        val filtered = transcripts.filter { it.conversation.projectId == projectId }
        return exportPackage(filtered)
    }

    fun importPackage(payload: String): Pair<List<ConversationTranscript>, ImportReport> {
        val parsed = json.parseToJsonElement(payload)
        val packageModel = runCatching {
            json.decodeFromJsonElement(LiaotaoExportPackage.serializer(), parsed)
        }.getOrElse {
            return emptyList<ConversationTranscript>() to ImportReport(
                importedConversations = 0,
                partialErrors = listOf("Invalid JSON package: ${it.message}"),
            )
        }

        if (packageModel.schemaVersion != ExportSchema.CURRENT_VERSION) {
            return emptyList<ConversationTranscript>() to ImportReport(
                importedConversations = 0,
                partialErrors = listOf(
                    "Unsupported schema version ${packageModel.schemaVersion}; expected ${ExportSchema.CURRENT_VERSION}",
                ),
            )
        }

        val partialErrors = mutableListOf<String>()
        val imported = mutableListOf<ConversationTranscript>()

        packageModel.conversations.forEachIndexed { index, conversation ->
            val model = runCatching {
                Conversation(
                    id = conversation.id,
                    projectId = conversation.projectId,
                    title = conversation.title,
                    source = conversation.source,
                    model = conversation.model,
                    createdAt = Instant.parse(conversation.createdAt),
                    updatedAt = Instant.parse(conversation.updatedAt),
                    lastActivityAt = Instant.parse(conversation.lastActivityAt),
                    archivedAt = conversation.archivedAt?.let { Instant.parse(it) },
                )
            }.getOrElse {
                partialErrors.add("Conversation[$index] skipped: invalid metadata")
                return@forEachIndexed
            }

            val messages = conversation.messages.mapNotNull { message ->
                runCatching {
                    TranscriptMessage(
                        role = message.role,
                        content = message.content,
                        createdAt = Instant.parse(message.createdAt),
                    )
                }.getOrElse {
                    partialErrors.add("Conversation[$index] contains malformed message and it was skipped")
                    null
                }
            }

            imported.add(ConversationTranscript(conversation = model, messages = messages))
        }

        return imported to ImportReport(
            importedConversations = imported.size,
            partialErrors = partialErrors,
        )
    }

    private fun exportPackage(transcripts: List<ConversationTranscript>): String {
        val conversations = transcripts.map { transcript ->
            ExportConversation(
                id = transcript.conversation.id,
                projectId = transcript.conversation.projectId,
                title = transcript.conversation.title,
                source = transcript.conversation.source,
                model = transcript.conversation.model,
                createdAt = transcript.conversation.createdAt.toString(),
                updatedAt = transcript.conversation.updatedAt.toString(),
                lastActivityAt = transcript.conversation.lastActivityAt.toString(),
                archivedAt = transcript.conversation.archivedAt?.toString(),
                messages = transcript.messages.map { message ->
                    ExportMessage(
                        role = message.role,
                        content = message.content,
                        createdAt = message.createdAt.toString(),
                    )
                },
            )
        }

        val packageModel = LiaotaoExportPackage(
            schemaVersion = ExportSchema.CURRENT_VERSION,
            exportedAt = nowProvider().toString(),
            conversations = conversations,
        )

        return json.encodeToString(LiaotaoExportPackage.serializer(), packageModel)
    }
}