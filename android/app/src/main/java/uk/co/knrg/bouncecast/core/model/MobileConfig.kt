package uk.co.knrg.bouncecast.core.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class MobileConfig(
    val version: Int = 1,
    @SerialName("generated_at") val generatedAt: String = "",
    val app: AppConfig = AppConfig(),
    val branding: BrandingConfig = BrandingConfig(),
    val bouncecast: BounceCastConnection = BounceCastConnection(),
    val features: FeatureFlags = FeatureFlags(),
    val ads: AdConfig = AdConfig(),
    val notifications: NotificationConfig = NotificationConfig(),
    val navigation: List<NavigationItem> = emptyList(),
    val legal: List<LegalPage> = emptyList(),
    val stream: StreamStatus = StreamStatus(),
    val chat: ChatConfig = ChatConfig(),
)

@Serializable
data class AppConfig(
    val name: String = "BounceCast",
    @SerialName("maintenance_mode") val maintenanceMode: Boolean = false,
    @SerialName("maintenance_message") val maintenanceMessage: String = "",
    @SerialName("minimum_supported_version") val minimumSupportedVersion: String = "1.0.0",
    @SerialName("recommended_version") val recommendedVersion: String = "",
    @SerialName("force_update") val forceUpdate: Boolean = false,
    @SerialName("force_update_message") val forceUpdateMessage: String = "",
    @SerialName("homepage_message") val homepageMessage: String = "",
    @SerialName("support_url") val supportUrl: String = "",
)

@Serializable
data class BrandingConfig(
    @SerialName("logo_url") val logoUrl: String = "",
    @SerialName("splash_url") val splashUrl: String = "",
    @SerialName("app_icon_url") val appIconUrl: String = "",
    @SerialName("primary_color") val primaryColor: String = "#080711",
    @SerialName("accent_color") val accentColor: String = "#ff2a8a",
    @SerialName("background_color") val backgroundColor: String = "#05050f",
    @SerialName("theme_mode") val themeMode: String = "system",
)

@Serializable
data class BounceCastConnection(
    @SerialName("base_url") val baseUrl: String = "",
    @SerialName("stream_url") val streamUrl: String = "",
    @SerialName("api_url") val apiUrl: String = "",
    @SerialName("chat_websocket_url") val chatWebSocketUrl: String = "",
    @SerialName("stream_status_url") val streamStatusUrl: String = "",
    @SerialName("chat_history_url") val chatHistoryUrl: String = "",
    @SerialName("register_chat_url") val registerChatUrl: String = "",
)

@Serializable
data class FeatureFlags(
    val chat: Boolean = true,
    @SerialName("gif_picker") val gifPicker: Boolean = true,
    val stickers: Boolean = true,
    val profiles: Boolean = true,
    @SerialName("push_notifications") val pushNotifications: Boolean = false,
    val ads: Boolean = false,
    @SerialName("experimental_features") val experimentalFeatures: Boolean = false,
)

@Serializable
data class AdConfig(
    val enabled: Boolean = false,
    @SerialName("test_mode") val testMode: Boolean = true,
    @SerialName("fallback_enabled") val fallbackEnabled: Boolean = true,
    val priority: List<String> = listOf("google", "unity"),
    @SerialName("google_enabled") val googleEnabled: Boolean = false,
    @SerialName("unity_enabled") val unityEnabled: Boolean = false,
    @SerialName("google_app_id") val googleAppId: String = "",
    @SerialName("google_banner_ad_unit_id") val googleBannerAdUnitId: String = "",
    @SerialName("google_app_open_ad_unit_id") val googleAppOpenAdUnitId: String = "",
    @SerialName("unity_game_id") val unityGameId: String = "",
    @SerialName("unity_banner_placement_id") val unityBannerPlacementId: String = "",
    @SerialName("unity_app_open_placement_id") val unityAppOpenPlacementId: String = "",
    @SerialName("banner_enabled") val bannerEnabled: Boolean = false,
    @SerialName("banner_position") val bannerPosition: String = "bottom",
    @SerialName("app_open_enabled") val appOpenEnabled: Boolean = false,
    @SerialName("app_open_cooldown_minutes") val appOpenCooldownMinutes: Int = 30,
    @SerialName("app_open_show_on_first_launch") val appOpenShowOnFirstLaunch: Boolean = true,
    @SerialName("timeout_ms") val timeoutMs: Int = 5_000,
    @SerialName("retry_limit") val retryLimit: Int = 1,
    @SerialName("consent_required") val consentRequired: Boolean = true,
    @SerialName("personalized_ads_allowed") val personalizedAdsAllowed: Boolean = false,
)

@Serializable
data class NotificationConfig(
    @SerialName("go_live_enabled") val goLiveEnabled: Boolean = false,
    @SerialName("title_template") val titleTemplate: String = "BounceCast is live",
    @SerialName("body_template") val bodyTemplate: String = "{stream_title} is live now. Tap to watch.",
    @SerialName("image_url") val imageUrl: String = "",
    @SerialName("icon_url") val iconUrl: String = "",
    val topic: String = "go-live",
    @SerialName("fcm_project_id") val fcmProjectId: String = "",
)

@Serializable
data class NavigationItem(
    val label: String = "",
    val url: String = "",
    @SerialName("item_type") val itemType: String = "internal",
    @SerialName("display_order") val displayOrder: Int = 0,
    val enabled: Boolean = true,
)

@Serializable
data class LegalPage(
    val slug: String = "",
    val title: String = "",
    val content: String = "",
    val url: String = "",
    @SerialName("display_order") val displayOrder: Int = 0,
    val enabled: Boolean = false,
)

@Serializable
data class StreamStatus(
    val online: Boolean = false,
    @SerialName("stream_title") val streamTitle: String = "",
    @SerialName("viewer_count") val viewerCount: Int = 0,
    @SerialName("server_time") val serverTime: String = "",
    @SerialName("last_live_at") val lastLiveAt: String = "",
    @SerialName("last_offline_at") val lastOfflineAt: String = "",
)

@Serializable
data class ChatConfig(
    val enabled: Boolean = true,
    @SerialName("require_authentication") val requireAuthentication: Boolean = false,
    @SerialName("max_socket_payload_size") val maxSocketPayloadSize: Int = 4096,
    @SerialName("background_image_url") val backgroundImageUrl: String = "",
    @SerialName("background_opacity") val backgroundOpacity: Float = 1f,
    @SerialName("tenor_enabled") val tenorEnabled: Boolean = false,
    @SerialName("tenor_api_key") val tenorApiKey: String = "",
)

