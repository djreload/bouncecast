package uk.co.knrg.bouncecast.feature.chat

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import uk.co.knrg.bouncecast.core.model.ChatEvent
import uk.co.knrg.bouncecast.core.model.ChatRegistrationRequest
import uk.co.knrg.bouncecast.core.model.ChatRegistrationResponse
import uk.co.knrg.bouncecast.core.network.BounceCastApi
import java.net.URLEncoder

class ChatRepository(
    private val api: BounceCastApi,
    private val scope: CoroutineScope,
) {
    private val _events = MutableStateFlow<List<ChatEvent>>(emptyList())
    val events: StateFlow<List<ChatEvent>> = _events

    private val _connected = MutableStateFlow(false)
    val connected: StateFlow<Boolean> = _connected

    private var websocket: WebSocket? = null
    private var accessToken: String = ""

    suspend fun register(registerUrl: String, displayName: String): ChatRegistrationResponse {
        return api.post<ChatRegistrationRequest, ChatRegistrationResponse>(
            url = registerUrl,
            body = ChatRegistrationRequest(displayName = displayName),
        ).also { response ->
            accessToken = response.accessToken
        }
    }

    suspend fun loadHistory(historyUrl: String) {
        if (accessToken.isBlank()) return
        val url = "$historyUrl?accessToken=${accessToken.urlEncode()}"
        runCatching { api.get<List<ChatEvent>>(url) }
            .onSuccess { _events.value = it }
    }

    fun connect(webSocketUrl: String) {
        if (accessToken.isBlank()) return
        val separator = if (webSocketUrl.contains("?")) "&" else "?"
        val request = Request.Builder()
            .url("$webSocketUrl${separator}accessToken=${accessToken.urlEncode()}")
            .build()

        websocket?.cancel()
        websocket = api.client.newWebSocket(
            request,
            object : WebSocketListener() {
                override fun onOpen(webSocket: WebSocket, response: Response) {
                    _connected.value = true
                }

                override fun onMessage(webSocket: WebSocket, text: String) {
                    text.lineSequence()
                        .filter { it.isNotBlank() }
                        .forEach { line ->
                            val event = runCatching { api.json.decodeFromString<ChatEvent>(line) }.getOrNull()
                            when (event?.type) {
                                "PING" -> webSocket.send("""{"type":"PONG"}""")
                                "VISIBILITY-UPDATE" -> applyVisibility(event)
                                null -> Unit
                                else -> _events.value = (_events.value + event).takeLast(200)
                            }
                        }
                }

                override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                    _connected.value = false
                }

                override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                    _connected.value = false
                }
            },
        )
    }

    fun sendMessage(message: String) {
        val clean = message.trim()
        if (clean.isBlank()) return
        websocket?.send(api.json.encodeToString(ChatOutboundEvent.serializer(), ChatOutboundEvent(body = clean)))
    }

    fun disconnect() {
        websocket?.close(1000, "Leaving BounceCast chat")
        websocket = null
        _connected.value = false
    }

    private fun applyVisibility(event: ChatEvent) {
        if (event.ids.isEmpty()) return
        val hidden = event.ids.toSet()
        _events.value = if (event.visible) {
            _events.value
        } else {
            _events.value.filterNot { hidden.contains(it.id) }
        }
    }

    fun registerAndConnect(
        registerUrl: String,
        historyUrl: String,
        webSocketUrl: String,
        displayName: String,
    ) {
        scope.launch(Dispatchers.IO) {
            register(registerUrl, displayName)
            loadHistory(historyUrl)
            connect(webSocketUrl)
        }
    }
}

@kotlinx.serialization.Serializable
private data class ChatOutboundEvent(
    val type: String = "CHAT",
    val body: String,
)

private fun String.urlEncode(): String = URLEncoder.encode(this, Charsets.UTF_8.name())
