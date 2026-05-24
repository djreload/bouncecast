package uk.co.knrg.bouncecast.feature.ads

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView

@Composable
fun AdBanner(adManager: AdManager) {
    val providers = remember { adManager.providersForBanner() }
    var providerIndex by remember { mutableIntStateOf(0) }
    val provider = providers.getOrNull(providerIndex) ?: return

    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(56.dp),
    ) {
        provider.Banner(onFailed = { providerIndex += 1 })?.let { banner ->
            AndroidView(
                factory = { banner },
                modifier = Modifier.fillMaxWidth(),
            )
        }
    }
}

