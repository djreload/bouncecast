package uk.co.knrg.bouncecast.feature.chat

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.SerialName
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import uk.co.knrg.bouncecast.core.model.ChatEvent
import uk.co.knrg.bouncecast.core.model.ChatRegistrationRequest
import uk.co.knrg.bouncecast.core.model.ChatRegistrationResponse
import uk.co.knrg.bouncecast.core.model.ChatUser
import uk.co.knrg.bouncecast.core.model.ViewerIdentity
import uk.co.knrg.bouncecast.core.network.BounceCastApi
import java.net.URLEncoder

class ChatRepository(
    private val api: BounceCastApi,
    private val scope: CoroutineScope,
) {
    private val _events = MutableStateFlow<List<ChatEvent>>(emptyList())
    val events: StateFlow<List<ChatEvent>> = _events

    private val _overlayEvents = MutableStateFlow<List<ChatEvent>>(emptyList())
    val overlayEvents: StateFlow<List<ChatEvent>> = _overlayEvents

    private val _connected = MutableStateFlow(false)
    val connected: StateFlow<Boolean> = _connected

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error

    private val _accessToken = MutableStateFlow("")
    val accessToken: StateFlow<String> = _accessToken

    private val _userId = MutableStateFlow("")
    val userId: StateFlow<String> = _userId

    private val _displayName = MutableStateFlow("Android Viewer")
    val displayName: StateFlow<String> = _displayName

    private var websocket: WebSocket? = null
    private var currentAccessToken: String = ""
    private var currentUserId: String = ""
    private var currentDisplayName: String = "Android Viewer"

    fun restoreIdentity(identity: ViewerIdentity) {
        val normalized = identity.normalized()
        currentUserId = normalized.userId
        currentAccessToken = normalized.accessToken
        currentDisplayName = normalized.displayName
        _userId.value = normalized.userId
        _accessToken.value = normalized.accessToken
        _displayName.value = normalized.displayName
    }

    fun clearIdentity() {
        disconnect()
        currentUserId = ""
        currentAccessToken = ""
        currentDisplayName = "Android Viewer"
        _userId.value = ""
        _accessToken.value = ""
        _displayName.value = "Android Viewer"
        _events.value = emptyList()
        _overlayEvents.value = emptyList()
    }

    suspend fun register(registerUrl: String, displayName: String): ChatRegistrationResponse {
        if (currentAccessToken.isNotBlank()) {
            return ChatRegistrationResponse(
                id = currentUserId,
                accessToken = currentAccessToken,
                displayName = currentDisplayName.ifBlank { displayName.ifBlank { "Android Viewer" } },
            )
        }

        val requestedDisplayName = displayName.ifBlank { "Android Viewer" }
        return api.post<ChatRegistrationRequest, ChatRegistrationResponse>(
            url = registerUrl,
            body = ChatRegistrationRequest(displayName = requestedDisplayName),
        ).also { response ->
            currentAccessToken = response.accessToken
            currentUserId = response.id
            currentDisplayName = response.displayName.ifBlank { requestedDisplayName }
            _accessToken.value = response.accessToken
            _userId.value = response.id
            _displayName.value = currentDisplayName
        }
    }

    suspend fun updateProfile(profileUrl: String, displayName: String): ChatRegistrationResponse {
        if (currentAccessToken.isBlank()) {
            throw IllegalStateException("Create your app viewer first.")
        }
        val cleanDisplayName = displayName.ifBlank { "Android Viewer" }
        val separator = if (profileUrl.contains("?")) "&" else "?"
        return api.post<ChatProfileUpdateRequest, ChatProfileUpdateResponse>(
            url = "$profileUrl${separator}accessToken=${currentAccessToken.urlEncode()}",
            body = ChatProfileUpdateRequest(displayName = cleanDisplayName, profileImageUrl = ""),
        ).let { response ->
            val user = response.user
            currentUserId = user?.id?.ifBlank { currentUserId } ?: currentUserId
            currentDisplayName = user?.displayName?.ifBlank { cleanDisplayName } ?: cleanDisplayName
            _userId.value = currentUserId
            _displayName.value = currentDisplayName
            ChatRegistrationResponse(
                id = currentUserId,
                accessToken = currentAccessToken,
                displayName = currentDisplayName,
                displayColor = user?.displayColor ?: 0,
            )
        }
    }

    suspend fun loadHistory(historyUrl: String) {
        if (currentAccessToken.isBlank()) return
        val url = "$historyUrl?accessToken=${currentAccessToken.urlEncode()}"
        runCatching { api.get<List<ChatEvent>>(url) }
            .onSuccess { _events.value = it }
    }

    fun connect(webSocketUrl: String) {
        if (currentAccessToken.isBlank()) return
        val separator = if (webSocketUrl.contains("?")) "&" else "?"
        val request = Request.Builder()
            .url("$webSocketUrl${separator}accessToken=${currentAccessToken.urlEncode()}")
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
                                "PONG", "CONNECTED_USER_INFO", "USER_JOINED", "USER_PARTED" -> Unit
                                "VISIBILITY-UPDATE" -> applyVisibility(event)
                                "CHAT_REACTION" -> applyReaction(event)
                                "STARS_SENT", "REWARD_WHEEL_WIN" -> {
                                    _overlayEvents.value = (_overlayEvents.value + event).takeLast(8)
                                }
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
                    _error.value = t.message ?: "Chat connection failed"
                }
            },
        )
    }

    fun sendMessage(message: String) {
        val clean = message.trim()
        if (clean.isBlank()) return
        websocket?.send(api.json.encodeToString(ChatOutboundEvent.serializer(), ChatOutboundEvent(body = clean)))
    }

    fun sendGif(url: String) {
        val clean = url.trim()
        if (clean.isBlank()) return
        sendMessage("![Tenor GIF]($clean)")
    }

    fun sendReaction(messageId: String, reaction: String) {
        val cleanMessageId = messageId.trim()
        val cleanReaction = reaction.trim()
        if (cleanMessageId.isBlank() || cleanReaction.isBlank()) return
        websocket?.send(
            api.json.encodeToString(
                ChatReactionOutboundEvent.serializer(),
                ChatReactionOutboundEvent(messageId = cleanMessageId, reaction = cleanReaction),
            ),
        )
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

    private fun applyReaction(event: ChatEvent) {
        if (event.messageId.isBlank()) return
        _events.value = _events.value.map { chatEvent ->
            if (chatEvent.id == event.messageId) {
                chatEvent.copy(reactions = event.counts)
            } else {
                chatEvent
            }
        }
    }

    fun registerAndConnect(
        registerUrl: String,
        historyUrl: String,
        webSocketUrl: String,
        displayName: String,
    ) {
        scope.launch(Dispatchers.IO) {
            runCatching {
                _error.value = null
                register(registerUrl, displayName)
                loadHistory(historyUrl)
                connect(webSocketUrl)
            }.onFailure {
                _connected.value = false
                _error.value = it.message ?: "Unable to join BounceCast chat"
            }
        }
    }
}

@kotlinx.serialization.Serializable
private data class ChatOutboundEvent(
    val type: String = "CHAT",
    val body: String,
)

@kotlinx.serialization.Serializable
private data class ChatReactionOutboundEvent(
    val type: String = "CHAT_REACTION",
    @SerialName("messageId") val messageId: String,
    val reaction: String,
)

@kotlinx.serialization.Serializable
private data class ChatProfileUpdateRequest(
    @SerialName("displayName") val displayName: String,
    @SerialName("profileImageUrl") val profileImageUrl: String,
)

@kotlinx.serialization.Serializable
private data class ChatProfileUpdateResponse(
    val user: ChatUser? = null,
    val message: String = "",
)

private fun String.urlEncode(): String = URLEncoder.encode(this, Charsets.UTF_8.name())
