package uk.co.knrg.bouncecast.feature.chat

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.fromHtml
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import kotlinx.coroutines.delay
import uk.co.knrg.bouncecast.core.model.ChatEvent
import uk.co.knrg.bouncecast.core.model.MobileConfig
import uk.co.knrg.bouncecast.core.network.BounceCastApi

@Composable
fun ChatPanel(
    config: MobileConfig,
    modifier: Modifier = Modifier,
    overlay: Boolean = false,
) {
    val scope = rememberCoroutineScope()
    val api = remember { BounceCastApi() }
    val repository = remember { ChatRepository(api, scope) }
    val gifRepository = remember { TenorGifRepository(api) }
    val events by repository.events.collectAsState()
    val connected by repository.connected.collectAsState()
    val error by repository.error.collectAsState()
    val listState = rememberLazyListState()
    var draft by remember { mutableStateOf("") }
    var gifOpen by remember { mutableStateOf(false) }
    var gifQuery by remember { mutableStateOf("dj rave") }
    var gifResults by remember { mutableStateOf<List<TenorGif>>(emptyList()) }
    var gifLoading by remember { mutableStateOf(false) }
    var gifError by remember { mutableStateOf<String?>(null) }
    val tenorEnabled = config.features.gifPicker &&
        config.chat.tenorEnabled &&
        config.chat.tenorApiKey.isNotBlank()

    LaunchedEffect(config.bouncecast.chatWebSocketUrl) {
        repository.registerAndConnect(
            registerUrl = config.bouncecast.registerChatUrl,
            historyUrl = config.bouncecast.chatHistoryUrl,
            webSocketUrl = config.bouncecast.chatWebSocketUrl,
            displayName = "Android Viewer",
        )
    }

    LaunchedEffect(events.size) {
        if (events.isNotEmpty()) {
            listState.animateScrollToItem(events.lastIndex)
        }
    }

    LaunchedEffect(gifOpen, gifQuery, tenorEnabled, config.chat.tenorApiKey) {
        if (!gifOpen || !tenorEnabled) {
            gifResults = emptyList()
            gifLoading = false
            gifError = null
            return@LaunchedEffect
        }

        delay(300)
        gifLoading = true
        gifError = null
        runCatching {
            gifRepository.search(config.chat.tenorApiKey, gifQuery)
        }.onSuccess {
            gifResults = it
        }.onFailure {
            gifResults = emptyList()
            gifError = it.message ?: "Unable to load GIFs"
        }
        gifLoading = false
    }

    DisposableEffect(Unit) {
        onDispose { repository.disconnect() }
    }

    val panelBackground = if (overlay) {
        Color.Black.copy(alpha = 0.38f)
    } else {
        Color.Transparent
    }
    val foreground = if (overlay) Color.White else MaterialTheme.colorScheme.onSurface

    Column(
        modifier = modifier
            .imePadding()
            .background(panelBackground, RoundedCornerShape(if (overlay) 18.dp else 0.dp))
            .padding(12.dp),
    ) {
        Text(
            text = when {
                connected -> "Live chat"
                error != null -> "Chat unavailable"
                else -> "Connecting chat"
            },
            color = foreground,
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
            state = listState,
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(max = if (overlay) 280.dp else 360.dp),
            contentPadding = PaddingValues(bottom = 4.dp),
        ) {
            items(events, key = { it.id.ifBlank { "${it.timestamp}-${it.body}" } }) { event ->
                ChatRow(event = event, overlay = overlay)
            }
        }
        if (gifOpen && tenorEnabled) {
            Spacer(Modifier.height(8.dp))
            GifPicker(
                query = gifQuery,
                onQueryChange = { gifQuery = it },
                loading = gifLoading,
                error = gifError,
                results = gifResults,
                onGifSelected = {
                    repository.sendGif(it.url)
                    gifOpen = false
                },
                overlay = overlay,
            )
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
            if (tenorEnabled) {
                OutlinedButton(
                    onClick = { gifOpen = !gifOpen },
                    modifier = Modifier.padding(start = 8.dp),
                    contentPadding = PaddingValues(horizontal = 12.dp),
                ) {
                    Text("GIF")
                }
            }
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
private fun GifPicker(
    query: String,
    onQueryChange: (String) -> Unit,
    loading: Boolean,
    error: String?,
    results: List<TenorGif>,
    onGifSelected: (TenorGif) -> Unit,
    overlay: Boolean,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = if (overlay) Color.Black.copy(alpha = 0.46f) else MaterialTheme.colorScheme.surface,
                shape = RoundedCornerShape(12.dp),
            )
            .padding(10.dp),
    ) {
        OutlinedTextField(
            value = query,
            onValueChange = onQueryChange,
            modifier = Modifier.fillMaxWidth(),
            placeholder = { Text("Search Tenor GIFs") },
            singleLine = true,
        )
        Spacer(Modifier.height(8.dp))
        when {
            loading -> CircularProgressIndicator(modifier = Modifier.size(28.dp))
            error != null -> Text(error, color = MaterialTheme.colorScheme.error)
            results.isEmpty() -> Text(
                "Search for a GIF",
                color = if (overlay) Color.White else MaterialTheme.colorScheme.onSurfaceVariant,
            )
            else -> LazyRow {
                items(results, key = { it.id }) { gif ->
                    AsyncImage(
                        model = gif.previewUrl,
                        contentDescription = gif.description,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier
                            .padding(end = 8.dp)
                            .size(92.dp)
                            .clip(RoundedCornerShape(10.dp))
                            .clickable { onGifSelected(gif) },
                    )
                }
            }
        }
    }
}

@Composable
private fun ChatRow(event: ChatEvent, overlay: Boolean) {
    val name = event.user?.displayName?.ifBlank { "Viewer" } ?: "System"
    val body = event.body.ifBlank { event.type }
    val gifUrl = extractTenorGifUrl(body)
    val textBody = stripTenorGifMarkup(body)
    val foreground = if (overlay) Color.White else MaterialTheme.colorScheme.onSurface

    Column(modifier = Modifier.padding(vertical = 5.dp)) {
        Text(name, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary)
        if (textBody.isNotBlank()) {
            Text(
                AnnotatedString.fromHtml(textBody),
                style = MaterialTheme.typography.bodyMedium,
                color = foreground,
            )
        }
        gifUrl?.let {
            AsyncImage(
                model = it,
                contentDescription = "Tenor GIF",
                contentScale = ContentScale.Crop,
                modifier = Modifier
                    .padding(top = 4.dp)
                    .fillMaxWidth(0.72f)
                    .heightIn(min = 96.dp, max = 180.dp)
                    .clip(RoundedCornerShape(12.dp)),
            )
        }
    }
}

private const val tenorGifUrlPattern = """https://media\d*\.tenor\.com/[^\s<>()"]+\.gif(?:\?[^\s<>()"]*)?"""
private val tenorGifUrlRegex = Regex(tenorGifUrlPattern, RegexOption.IGNORE_CASE)
private val tenorMarkdownRegex = Regex("""!\[[^\]]*]\(($tenorGifUrlPattern)\)""", RegexOption.IGNORE_CASE)
private val tenorAnchorRegex = Regex(
    """<a\b[^>]*href="($tenorGifUrlPattern)"[^>]*>[\s\S]*?</a>""",
    RegexOption.IGNORE_CASE,
)

private fun extractTenorGifUrl(body: String): String? {
    return tenorMarkdownRegex.find(body)?.groupValues?.getOrNull(1)
        ?: tenorAnchorRegex.find(body)?.groupValues?.getOrNull(1)
        ?: tenorGifUrlRegex.find(body)?.value
}

private fun stripTenorGifMarkup(body: String): String {
    return tenorGifUrlRegex.replace(
        tenorMarkdownRegex.replace(
            tenorAnchorRegex.replace(body, ""),
            "",
        ),
        "",
    ).trim()
}
