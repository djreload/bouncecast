package uk.co.knrg.bouncecast.core.network

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.IOException
import java.util.concurrent.TimeUnit

class BounceCastApi(
    val client: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(20, TimeUnit.SECONDS)
        .build(),
    val json: Json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        encodeDefaults = true
    },
) {
    @PublishedApi
    internal val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    suspend inline fun <reified T> get(url: String): T = withContext(Dispatchers.IO) {
        val request = Request.Builder().url(url).get().build()
        client.newCall(request).execute().use { response ->
            if (!response.isSuccessful) {
                throw IOException("GET $url failed with HTTP ${response.code}")
            }
            json.decodeFromString<T>(response.body?.string().orEmpty())
        }
    }

    suspend inline fun <reified RequestT, reified ResponseT> post(
        url: String,
        body: RequestT,
    ): ResponseT = withContext(Dispatchers.IO) {
        val requestBody = json.encodeToString(body).toRequestBody(jsonMediaType)
        val request = Request.Builder().url(url).post(requestBody).build()
        client.newCall(request).execute().use { response ->
            if (!response.isSuccessful) {
                throw IOException("POST $url failed with HTTP ${response.code}")
            }
            json.decodeFromString<ResponseT>(response.body?.string().orEmpty())
        }
    }
}
