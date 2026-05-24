package uk.co.knrg.bouncecast.feature.ads

import android.app.Activity
import android.content.Context
import com.google.android.gms.ads.MobileAds
import com.unity3d.ads.IUnityAdsInitializationListener
import com.unity3d.ads.UnityAds
import uk.co.knrg.bouncecast.core.model.AdConfig

class AdManager(private val context: Context) {
    private var config: AdConfig = AdConfig()
    private var initialized = false

    fun configure(config: AdConfig) {
        this.config = config
        if (!config.enabled || initialized) return

        if (config.googleEnabled) {
            MobileAds.initialize(context)
        }
        if (config.unityEnabled && config.unityGameId.isNotBlank()) {
            UnityAds.initialize(
                context,
                config.unityGameId,
                config.testMode,
                object : IUnityAdsInitializationListener {
                    override fun onInitializationComplete() = Unit
                    override fun onInitializationFailed(
                        error: UnityAds.UnityAdsInitializationError,
                        message: String,
                    ) = Unit
                },
            )
        }
        initialized = true
    }

    fun providersForBanner(): List<AdProvider> {
        if (!config.enabled || !config.bannerEnabled) return emptyList()
        return AdPriorityResolver.providers(config).mapNotNull { provider ->
            when (provider) {
                "google" -> GoogleBannerProvider(config)
                "unity" -> UnityBannerProvider(config)
                else -> null
            }
        }
    }

    fun shouldShowAppOpen(nowMillis: Long, lastShownMillis: Long?): Boolean {
        if (!config.enabled || !config.appOpenEnabled) return false
        if (lastShownMillis == null) return config.appOpenShowOnFirstLaunch
        val cooldownMillis = config.appOpenCooldownMinutes.coerceAtLeast(1) * 60_000L
        return nowMillis - lastShownMillis >= cooldownMillis
    }

    fun showAppOpen(activity: Activity, onFinished: () -> Unit) {
        // Stage 6 will load/show real provider app-open/interstitial objects.
        // This non-blocking placeholder preserves startup behavior until live ad
        // account IDs and consent flows are configured.
        onFinished()
    }
}

object AdPriorityResolver {
    fun providers(config: AdConfig): List<String> {
        if (!config.enabled) return emptyList()
        val enabled = buildSet {
            if (config.googleEnabled) add("google")
            if (config.unityEnabled) add("unity")
        }
        val ordered = config.priority
            .map { it.lowercase().trim() }
            .filter { enabled.contains(it) }
            .distinct()
        return if (config.fallbackEnabled) ordered else ordered.take(1)
    }
}

