package uk.co.knrg.bouncecast.core.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ChatUser(
    val id: String = "",
    @SerialName("displayName") val displayName: String = "",
    @SerialName("displayColor") val displayColor: Int = 0,
    @SerialName("profileImage") val profileImage: String = "",
    val scopes: List<String> = emptyList(),
)

@Serializable
data class ChatEvent(
    val id: String = "",
    val timestamp: String = "",
    val type: String = "",
    val body: String = "",
    val user: ChatUser? = null,
    val visible: Boolean = true,
    val ids: List<String> = emptyList(),
    @SerialName("messageId") val messageId: String = "",
    val counts: Map<String, Int> = emptyMap(),
)

@Serializable
data class ChatRegistrationRequest(
    @SerialName("displayName") val displayName: String,
)

@Serializable
data class ChatRegistrationResponse(
    val id: String = "",
    @SerialName("accessToken") val accessToken: String = "",
    @SerialName("displayName") val displayName: String = "",
    @SerialName("displayColor") val displayColor: Int = 0,
)

