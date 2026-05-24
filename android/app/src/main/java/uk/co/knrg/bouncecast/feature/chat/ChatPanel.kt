package uk.co.knrg.bouncecast.feature.chat

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.fromHtml
import androidx.compose.ui.unit.dp
import uk.co.knrg.bouncecast.core.model.ChatEvent
import uk.co.knrg.bouncecast.core.model.MobileConfig
import uk.co.knrg.bouncecast.core.network.BounceCastApi

@Composable
fun ChatPanel(config: MobileConfig) {
    val scope = rememberCoroutineScope()
    val repository = remember { ChatRepository(BounceCastApi(), scope) }
    val events by repository.events.collectAsState()
    val connected by repository.connected.collectAsState()
    val error by repository.error.collectAsState()
    var draft by remember { mutableStateOf("") }

    LaunchedEffect(config.bouncecast.chatWebSocketUrl) {
        repository.registerAndConnect(
            registerUrl = config.bouncecast.registerChatUrl,
            historyUrl = config.bouncecast.chatHistoryUrl,
            webSocketUrl = config.bouncecast.chatWebSocketUrl,
            displayName = "Android Viewer",
        )
    }

    DisposableEffect(Unit) {
        onDispose { repository.disconnect() }
    }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(12.dp),
    ) {
        Text(
            text = when {
                connected -> "Live chat"
                error != null -> "Chat unavailable"
                else -> "Connecting chat"
            },
            style = MaterialTheme.typography.titleMedium,
        )
        error?.let {
            Text(
                text = it,
                color = MaterialTheme.colorScheme.error,
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Spacer(Modifier.height(8.dp))
        LazyColumn(
            modifier = Modifier
                .fillMaxWidth()
                .height(260.dp),
        ) {
            items(events, key = { it.id.ifBlank { "${it.timestamp}-${it.body}" } }) { event ->
                ChatRow(event)
            }
        }
        Spacer(Modifier.height(8.dp))
        Row {
            OutlinedTextField(
                value = draft,
                onValueChange = { draft = it.take(config.chat.maxSocketPayloadSize.coerceAtMost(500)) },
                modifier = Modifier.weight(1f),
                placeholder = { Text("Say something") },
                keyboardActions = KeyboardActions(
                    onDone = {
                        repository.sendMessage(draft)
                        draft = ""
                    },
                ),
            )
            Button(
                onClick = {
                    repository.sendMessage(draft)
                    draft = ""
                },
                enabled = draft.isNotBlank(),
                modifier = Modifier.padding(start = 8.dp),
            ) {
                Text("Send")
            }
        }
    }
}

@Composable
private fun ChatRow(event: ChatEvent) {
    val name = event.user?.displayName?.ifBlank { "Viewer" } ?: "System"
    val body = event.body.ifBlank { event.type }
    Column(modifier = Modifier.padding(vertical = 4.dp)) {
        Text(name, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary)
        Text(AnnotatedString.fromHtml(body), style = MaterialTheme.typography.bodyMedium)
    }
}
