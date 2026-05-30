package uk.co.knrg.bouncecast.core.model

data class ViewerIdentity(
    val userId: String = "",
    val accessToken: String = "",
    val displayName: String = "Android Viewer",
) {
    fun normalized(): ViewerIdentity {
        return copy(displayName = displayName.ifBlank { "Android Viewer" })
    }
}
