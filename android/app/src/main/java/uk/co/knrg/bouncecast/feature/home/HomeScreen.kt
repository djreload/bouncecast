package uk.co.knrg.bouncecast.feature.home

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import uk.co.knrg.bouncecast.core.config.MobileConfigRepository
import uk.co.knrg.bouncecast.core.model.MobileConfig
import uk.co.knrg.bouncecast.feature.ads.AdBanner
import uk.co.knrg.bouncecast.feature.ads.AdManager
import uk.co.knrg.bouncecast.feature.chat.ChatPanel
import uk.co.knrg.bouncecast.feature.stream.StreamPlayer

@Composable
fun HomeScreen(
    config: MobileConfig?,
    configRepository: MobileConfigRepository,
    adManager: AdManager,
) {
    var refreshError by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) {
        runCatching { configRepository.refresh() }
            .onFailure { refreshError = it.message ?: "Unable to refresh mobile config" }
    }

    Surface(
        color = MaterialTheme.colorScheme.background,
        modifier = Modifier.fillMaxSize(),
    ) {
        if (config == null) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(24.dp),
                verticalArrangement = Arrangement.Center,
            ) {
                CircularProgressIndicator()
                Spacer(Modifier.height(16.dp))
                Text("Loading BounceCast")
            }
            return@Surface
        }

        if (config.app.maintenanceMode || config.app.forceUpdate) {
            AlertDialog(
                onDismissRequest = {},
                confirmButton = {},
                title = { Text(if (config.app.forceUpdate) "Update required" else "Maintenance") },
                text = {
                    Text(
                        if (config.app.forceUpdate) {
                            config.app.forceUpdateMessage.ifBlank { "Please update the BounceCast app." }
                        } else {
                            config.app.maintenanceMessage.ifBlank { "BounceCast is temporarily unavailable." }
                        },
                    )
                },
            )
        }

        Column(modifier = Modifier.fillMaxSize()) {
            Header(config)
            StreamPlayer(
                streamUrl = config.bouncecast.streamUrl,
                online = config.stream.online,
                title = config.stream.streamTitle,
            )
            if (config.ads.enabled && config.ads.bannerEnabled) {
                AdBanner(adManager = adManager)
            }
            if (config.features.chat && config.chat.enabled) {
                ChatPanel(config = config)
            }
        }
    }

    refreshError?.let { error ->
        AlertDialog(
            onDismissRequest = { refreshError = null },
            confirmButton = {
                Button(onClick = { refreshError = null }) {
                    Text("OK")
                }
            },
            title = { Text("Config warning") },
            text = { Text(error) },
        )
    }
}

@Composable
private fun Header(config: MobileConfig) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Column {
            Text(config.app.name, style = MaterialTheme.typography.titleLarge)
            Text(
                if (config.stream.online) "Live now" else "Offline",
                color = if (config.stream.online) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        if (config.stream.viewerCount > 0) {
            Text("${config.stream.viewerCount} watching", style = MaterialTheme.typography.labelLarge)
        }
    }
}

