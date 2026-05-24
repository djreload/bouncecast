package uk.co.knrg.bouncecast.feature.notifications

import android.os.Build
import com.google.firebase.messaging.FirebaseMessaging
import kotlinx.coroutines.tasks.await
import uk.co.knrg.bouncecast.core.model.MobileConfig
import uk.co.knrg.bouncecast.core.network.BounceCastApi

class DeviceTokenRegistrar(private val api: BounceCastApi) {
    suspend fun register(config: MobileConfig, notificationsEnabled: Boolean) {
        if (!config.features.pushNotifications || !config.notifications.goLiveEnabled) return
        val token = FirebaseMessaging.getInstance().token.await()
        val url = "${config.bouncecast.baseUrl.trimEnd('/')}/api/mobile/v1/devices/register"
        api.post<DeviceRegistrationRequest, Map<String, Boolean>>(
            url = url,
            body = DeviceRegistrationRequest(
                token = token,
                platform = "android",
                deviceId = Build.MODEL.orEmpty(),
                appVersion = "1.0.0",
                locale = java.util.Locale.getDefault().toLanguageTag(),
                notificationsEnabled = notificationsEnabled,
            ),
        )
    }
}

@kotlinx.serialization.Serializable
data class DeviceRegistrationRequest(
    val token: String,
    val platform: String,
    val deviceId: String,
    val appVersion: String,
    val locale: String,
    val notificationsEnabled: Boolean,
)

