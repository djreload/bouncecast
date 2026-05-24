package uk.co.knrg.bouncecast.feature.ads

import android.view.View
import androidx.compose.runtime.Composable
import androidx.compose.ui.platform.LocalContext
import com.unity3d.services.banners.BannerView
import com.unity3d.services.banners.UnityBannerSize
import uk.co.knrg.bouncecast.core.model.AdConfig

class UnityBannerProvider(private val config: AdConfig) : AdProvider {
    override val name: String = "unity"

    override fun isConfigured(): Boolean = config.unityEnabled && config.unityBannerPlacementId.isNotBlank()

    @Composable
    override fun Banner(onFailed: () -> Unit): View? {
        if (!isConfigured()) {
            onFailed()
            return null
        }
        val activity = LocalContext.current as? android.app.Activity ?: return null
        return BannerView(activity, config.unityBannerPlacementId, UnityBannerSize(320, 50)).apply {
            load()
        }
    }
}
