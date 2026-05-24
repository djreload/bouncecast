package uk.co.knrg.bouncecast.feature.chat

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import uk.co.knrg.bouncecast.core.network.BounceCastApi
import java.net.URLEncoder

data class TenorGif(
    val id: String,
    val description: String,
    val url: String,
    val previewUrl: String,
)

class TenorGifRepository(private val api: BounceCastApi) {
    suspend fun search(
        apiKey: String,
        query: String,
        limit: Int = 12,
    ): List<TenorGif> = withContext(Dispatchers.IO) {
        if (apiKey.isBlank()) return@withContext emptyList()
        val encodedQuery = URLEncoder.encode(query.ifBlank { "dj rave" }, Charsets.UTF_8.name())
        val url = "https://tenor.googleapis.com/v2/search" +
            "?key=${apiKey.urlEncode()}" +
            "&client_key=bouncecast" +
            "&q=$encodedQuery" +
            "&limit=${limit.coerceIn(1, 24)}" +
            "&media_filter=tinygif,gif" +
            "&contentfilter=medium"

        api.get<TenorSearchResponse>(url).results.mapNotNull { result ->
            val preview = result.mediaFormats["tinygif"]?.url ?: result.mediaFormats["gif"]?.url
            val full = result.mediaFormats["gif"]?.url ?: preview
            if (preview.isNullOrBlank() || full.isNullOrBlank()) {
                null
            } else {
                TenorGif(
                    id = result.id,
                    description = result.description.ifBlank { "Tenor GIF" },
                    url = full,
                    previewUrl = preview,
                )
            }
        }
    }
}

@Serializable
private data class TenorSearchResponse(
    val results: List<TenorSearchResult> = emptyList(),
)

@Serializable
private data class TenorSearchResult(
    val id: String = "",
    @SerialName("content_description") val description: String = "",
    @SerialName("media_formats") val mediaFormats: Map<String, TenorMediaFormat> = emptyMap(),
)

@Serializable
private data class TenorMediaFormat(
    val url: String = "",
)

private fun String.urlEncode(): String = URLEncoder.encode(this, Charsets.UTF_8.name())
