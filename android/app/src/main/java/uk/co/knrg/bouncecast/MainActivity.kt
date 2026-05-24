package uk.co.knrg.bouncecast

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import uk.co.knrg.bouncecast.core.config.MobileConfigRepository
import uk.co.knrg.bouncecast.core.network.BounceCastApi
import uk.co.knrg.bouncecast.core.storage.ConfigStore
import uk.co.knrg.bouncecast.core.ui.BounceCastTheme
import uk.co.knrg.bouncecast.feature.ads.AdManager
import uk.co.knrg.bouncecast.feature.home.HomeScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val api = BounceCastApi()
        val configStore = ConfigStore(this)
        val configRepository = MobileConfigRepository(
            api = api,
            configStore = configStore,
            configUrl = BuildConfig.DEFAULT_CONFIG_URL,
        )
        val adManager = AdManager(this)

        setContent {
            BounceCastAndroidApp(configRepository, adManager)
        }
    }
}

@Composable
private fun BounceCastAndroidApp(
    configRepository: MobileConfigRepository,
    adManager: AdManager,
) {
    val config by configRepository.config.collectAsState()

    LaunchedEffect(config?.ads) {
        config?.ads?.let { adManager.configure(it) }
    }

    BounceCastTheme(config = config) {
        HomeScreen(
            config = config,
            configRepository = remember { configRepository },
            adManager = adManager,
        )
    }
}
