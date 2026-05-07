/*
 * ChatWorkspaceScreen.kt - chat experience with routing feedback.
 * Responsibilities: provide composer, message timeline, provider switching,
 * streaming rendering, retry/regenerate action, and execution status surface.
 */

package io.liaotao.appdesktop.chat

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import io.liaotao.connectors.core.ConnectorType
import io.liaotao.domain.history.ConversationHistoryQuery
import io.liaotao.domain.history.ConversationHistoryService
import io.liaotao.domain.routing.ExecutionAttempt
import java.time.Instant
import java.util.UUID

@Composable
fun ChatWorkspaceScreen() {
    val controller = remember { ChatWorkspaceController() }
    var prompt by remember { mutableStateOf("") }
    var selectedProvider by remember { mutableStateOf(ConnectorType.OLLAMA) }
    var search by remember { mutableStateOf("") }
    var projectFilter by remember { mutableStateOf("") }
    var sourceFilter by remember { mutableStateOf("") }

    val visibleConversations = remember(search, projectFilter, sourceFilter, controller.conversations.toList()) {
        ConversationHistoryService().query(
            conversations = controller.conversations.toList(),
            request = ConversationHistoryQuery(
                keyword = search,
                projectId = projectFilter.ifBlank { null },
                source = sourceFilter.ifBlank { null },
            ),
        )
    }

    Column(modifier = Modifier.fillMaxSize(), verticalArrangement = Arrangement.spacedBy(10.dp)) {
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            ConnectorType.entries.forEach { type ->
                Button(onClick = { selectedProvider = type }) {
                    Text(if (type == selectedProvider) "* ${type.name}" else type.name)
                }
            }
        }

        OutlinedTextField(
            value = prompt,
            onValueChange = { prompt = it },
            label = { Text("Message") },
            modifier = Modifier.fillMaxWidth(),
        )

        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Button(onClick = {
                val text = prompt.trim()
                if (text.isNotEmpty()) {
                    controller.send(text, selectedProvider)
                    prompt = ""
                }
            }) {
                Text("Send")
            }
            Button(onClick = { controller.retryLast(selectedProvider) }) {
                Text("Retry / Regenerate")
            }
        }

        Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.25f))) {
            Column(modifier = Modifier.fillMaxWidth().padding(10.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text("Execution Attempts", style = MaterialTheme.typography.titleSmall)
                controller.executionAttempts.takeLast(8).forEach { attempt ->
                    val label = "${attempt.providerId} | ${attempt.status} | retry=${attempt.retryIndex}" +
                        (attempt.errorMessage?.let { " | $it" } ?: "")
                    Text(label, style = MaterialTheme.typography.bodySmall)
                }
            }
        }

        Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.25f))) {
            Column(modifier = Modifier.fillMaxWidth().padding(10.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text("History Search", style = MaterialTheme.typography.titleSmall)
                OutlinedTextField(value = search, onValueChange = { search = it }, label = { Text("Keyword") }, modifier = Modifier.fillMaxWidth())
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(value = projectFilter, onValueChange = { projectFilter = it }, label = { Text("Project") }, modifier = Modifier.weight(1f))
                    OutlinedTextField(value = sourceFilter, onValueChange = { sourceFilter = it }, label = { Text("Source") }, modifier = Modifier.weight(1f))
                }
                Text("Visible conversations: ${visibleConversations.size}", style = MaterialTheme.typography.bodySmall)
            }
        }

        Card(modifier = Modifier.fillMaxWidth().weight(1f)) {
            LazyColumn(
                modifier = Modifier.fillMaxSize().padding(10.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                items(controller.messages) { message ->
                    Card(
                        modifier = Modifier.fillMaxWidth().heightIn(min = 56.dp),
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.2f)),
                    ) {
                        Column(modifier = Modifier.fillMaxWidth().padding(10.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text("${message.role} - ${message.source}/${message.model}", style = MaterialTheme.typography.labelSmall)
                            Text(message.content)
                        }
                    }
                }
            }
        }
    }
}

internal class ChatWorkspaceController {
    private val chatService = ChatWorkspaceService()

    val messages = mutableStateListOf<ChatUiMessage>()
    val executionAttempts = mutableStateListOf<ExecutionAttempt>()
    val conversations = mutableStateListOf<io.liaotao.domain.conversations.Conversation>()

    private var lastPrompt: String? = null
    private var lastProvider: ConnectorType = ConnectorType.OLLAMA

    fun send(prompt: String, provider: ConnectorType) {
        val userMessage = ChatUiMessage(
            id = UUID.randomUUID().toString(),
            role = "user",
            content = prompt,
            source = provider.name,
            model = "user-input",
        )
        messages.add(userMessage)

        val draftId = UUID.randomUUID().toString()
        messages.add(
            ChatUiMessage(
                id = draftId,
                role = "assistant",
                content = "",
                source = provider.name,
                model = "streaming",
            ),
        )

        val result = chatService.execute(prompt, provider) { chunk ->
            val index = messages.indexOfFirst { it.id == draftId }
            if (index >= 0) {
                val current = messages[index]
                messages[index] = current.copy(content = current.content + chunk)
            }
        }
        executionAttempts.addAll(result.attempts)

        val draftIndex = messages.indexOfFirst { it.id == draftId }
        if (draftIndex >= 0) {
            val draft = messages[draftIndex]
            val finalContent = if (draft.content.isNotBlank()) draft.content else result.reply
            messages[draftIndex] = draft.copy(
                content = finalContent,
                source = result.source,
                model = result.model,
            )
        }

        conversations.clear()
        conversations.addAll(chatService.conversationHistory())

        lastPrompt = prompt
        lastProvider = provider
    }

    fun retryLast(provider: ConnectorType) {
        val prompt = lastPrompt ?: return
        send(prompt, provider.takeIf { it != ConnectorType.OPENAI_COMPAT } ?: lastProvider)
    }
}

internal data class ChatUiMessage(
    val id: String,
    val role: String,
    val content: String,
    val source: String,
    val model: String,
)