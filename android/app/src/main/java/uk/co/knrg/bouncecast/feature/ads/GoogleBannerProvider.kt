package uk.co.knrg.bouncecast.feature.ads

import android.view.View
import androidx.compose.runtime.Composable
import androidx.compose.ui.platform.LocalContext
import com.google.android.gms.ads.AdRequest
import com.google.android.gms.ads.AdSize
import com.google.android.gms.ads.AdView
import uk.co.knrg.bouncecast.core.model.AdConfig

class GoogleBannerProvider(private val config: AdConfig) : AdProvider {
    override val name: String = "google"

    override fun isConfigured(): Boolean = config.googleEnabled && config.googleBannerAdUnitId.isNotBlank()

    @Composable
    override fun Banner(onFailed: () -> Unit): View? {
        if (!isConfigured()) {
            onFailed()
            return null
        }
        val context = LocalContext.current
        return AdView(context).apply {
            setAdSize(AdSize.BANNER)
            adUnitId = if (config.testMode) {
                "ca-app-pub-3940256099942544/6300978111"
            } else {
                config.googleBannerAdUnitId
            }
            loadAd(AdRequest.Builder().build())
        }
    }
}

