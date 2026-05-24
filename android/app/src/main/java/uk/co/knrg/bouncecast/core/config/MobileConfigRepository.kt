package uk.co.knrg.bouncecast.core.config

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.serialization.decodeFromString
import uk.co.knrg.bouncecast.core.model.MobileConfig
import uk.co.knrg.bouncecast.core.network.BounceCastApi
import uk.co.knrg.bouncecast.core.storage.ConfigStore

class MobileConfigRepository(
    private val api: BounceCastApi,
    private val configStore: ConfigStore,
    private val configUrl: String,
) {
    private val _config = MutableStateFlow<MobileConfig?>(null)
    val config: StateFlow<MobileConfig?> = _config

    suspend fun refresh() {
        val cached = configStore.readConfigJson()?.let { runCatching { api.json.decodeFromString<MobileConfig>(it) }.getOrNull() }
        if (cached != null) {
            _config.value = cached
        }

        val fresh = api.get<MobileConfig>(configUrl)
        validateConfig(fresh)
        configStore.writeConfigJson(api.json.encodeToString(MobileConfig.serializer(), fresh))
        _config.value = fresh
    }

    private fun validateConfig(config: MobileConfig) {
        require(config.bouncecast.streamUrl.startsWith("http")) { "Missing stream URL" }
        require(config.bouncecast.chatWebSocketUrl.startsWith("ws")) { "Missing chat websocket URL" }
        require(config.app.minimumSupportedVersion.isNotBlank()) { "Missing minimum app version" }
    }
}
