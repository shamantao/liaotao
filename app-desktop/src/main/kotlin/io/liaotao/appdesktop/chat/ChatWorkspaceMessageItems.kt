/*
 * ChatWorkspaceMessageItems.kt - message-level UI components.
 * Responsibilities: render user/assistant bubbles and assistant metadata tags
 * for the chat timeline.
 */

package io.liaotao.appdesktop.chat

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

@Composable
internal fun ChatBubble(message: ChatUiMessage, onEditUserMessage: (String) -> Unit) {
    val isUser = message.role == "user"
    val container = if (isUser) {
        MaterialTheme.colorScheme.primary.copy(alpha = 0.25f)
    } else {
        MaterialTheme.colorScheme.secondary.copy(alpha = 0.22f)
    }

    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = if (isUser) Arrangement.End else Arrangement.Start,
    ) {
        Card(
            modifier = Modifier
                .fillMaxWidth(0.82f)
                .heightIn(min = 72.dp),
            colors = CardDefaults.cardColors(containerColor = container),
        ) {
            Column(modifier = Modifier.fillMaxWidth().padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(message.content.ifBlank { "..." })

                if (isUser) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.End,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(formatDateTime(message.createdAt), style = MaterialTheme.typography.labelSmall)
                        Spacer(modifier = Modifier.size(8.dp))
                        TextButton(onClick = { onEditUserMessage(message.content) }) { Text("✎") }
                    }
                } else {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                        AssistantMetaTag("Model", message.model)
                        AssistantMetaTag("Tokens", (message.tokensUsed ?: 0).toString())
                        AssistantMetaTag("Done", formatDateTime(message.completedAt ?: message.createdAt))
                    }
                }
            }
        }
    }
}

@Composable
private fun AssistantMetaTag(label: String, value: String) {
    Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.65f))) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text("$label:", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.SemiBold)
            Text(value, style = MaterialTheme.typography.labelSmall)
        }
    }
}

private val dateTimeFormatter: DateTimeFormatter = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")
private val localZone: ZoneId = ZoneId.systemDefault()

private fun formatDateTime(instant: Instant): String = dateTimeFormatter.format(instant.atZone(localZone))
