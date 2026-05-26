package uk.co.knrg.bouncecast.feature.home

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.systemBarsPadding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import kotlinx.coroutines.delay
import uk.co.knrg.bouncecast.core.config.MobileConfigRepository
import uk.co.knrg.bouncecast.core.model.ChatEvent
import uk.co.knrg.bouncecast.core.model.MobileConfig
import uk.co.knrg.bouncecast.core.network.BounceCastApi
import uk.co.knrg.bouncecast.feature.ads.AdBanner
import uk.co.knrg.bouncecast.feature.ads.AdManager
import uk.co.knrg.bouncecast.feature.chat.ChatPanel
import uk.co.knrg.bouncecast.feature.chat.ChatRepository
import uk.co.knrg.bouncecast.feature.rewards.RewardsRepository
import uk.co.knrg.bouncecast.feature.rewards.RewardsWheelPanel
import uk.co.knrg.bouncecast.feature.stream.StreamPlayer

@Composable
fun HomeScreen(
    config: MobileConfig?,
    configRepository: MobileConfigRepository,
    adManager: AdManager,
) {
    var refreshError by remember { mutableStateOf<String?>(null) }
    val chatScope = rememberCoroutineScope()
    val chatApi = remember { BounceCastApi() }
    val chatRepository = remember { ChatRepository(chatApi, chatScope) }
    val rewardsRepository = remember { RewardsRepository(chatApi) }
    val overlayEvents by chatRepository.overlayEvents.collectAsState()
    val rewardAccessToken by chatRepository.accessToken.collectAsState()
    var showRewardsPanel by remember { mutableStateOf(false) }
    val shouldConnectToChat = config?.let {
        it.chat.enabled && (it.features.chat || it.features.starsOverlay || it.features.rewardOverlay)
    } == true
    val shouldRegisterViewer = config?.let { shouldConnectToChat || it.features.rewardWheel } == true

    LaunchedEffect(Unit) {
        while (true) {
            runCatching { configRepository.refresh() }
                .onFailure { refreshError = it.message ?: "Unable to refresh mobile config" }
            delay(15_000)
        }
    }

    LaunchedEffect(
        config?.bouncecast?.registerChatUrl,
        config?.bouncecast?.chatHistoryUrl,
        config?.bouncecast?.chatWebSocketUrl,
        config?.features?.rewardWheel,
        shouldConnectToChat,
    ) {
        val activeConfig = config ?: return@LaunchedEffect
        if (!shouldRegisterViewer) {
            chatRepository.disconnect()
            return@LaunchedEffect
        }
        if (shouldConnectToChat) {
            chatRepository.registerAndConnect(
                registerUrl = activeConfig.bouncecast.registerChatUrl,
                historyUrl = activeConfig.bouncecast.chatHistoryUrl,
                webSocketUrl = activeConfig.bouncecast.chatWebSocketUrl,
                displayName = "Android Viewer",
            )
        } else {
            chatRepository.disconnect()
            runCatching {
                chatRepository.register(activeConfig.bouncecast.registerChatUrl, "Android Viewer")
            }.onFailure {
                refreshError = it.message ?: "Unable to prepare Rewards Wheel"
            }
        }
    }

    DisposableEffect(chatRepository) {
        onDispose { chatRepository.disconnect() }
    }

    Surface(
        color = MaterialTheme.colorScheme.background,
        modifier = Modifier
            .fillMaxSize()
            .systemBarsPadding(),
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            config?.branding?.appBackgroundUrl
                ?.takeIf { it.isNotBlank() }
                ?.let { backgroundUrl ->
                    AsyncImage(
                        model = backgroundUrl,
                        contentDescription = null,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier.fillMaxSize(),
                    )
                }

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
            } else {
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

                if (config.stream.online) {
                    Box(modifier = Modifier.fillMaxSize()) {
                        StreamPlayer(
                            streamUrl = config.bouncecast.streamUrl,
                            online = true,
                            title = config.stream.streamTitle,
                            modifier = Modifier.fillMaxSize(),
                        )
                        Header(
                            config = config,
                            modifier = Modifier
                                .align(Alignment.TopCenter)
                                .fillMaxWidth()
                                .padding(horizontal = 16.dp, vertical = 12.dp),
                            overlay = true,
                        )
                        MobileFeatureActions(
                            config = config,
                            overlay = true,
                            onRewardsClick = { showRewardsPanel = true },
                            modifier = Modifier
                                .align(Alignment.TopEnd)
                                .padding(top = 78.dp, end = 12.dp),
                        )
                        MobileLiveEventOverlay(
                            events = overlayEvents,
                            starsEnabled = config.features.starsOverlay,
                            rewardsEnabled = config.features.rewardOverlay,
                            modifier = Modifier.align(Alignment.Center),
                        )
                        if (config.features.chat && config.chat.enabled) {
                            ChatPanel(
                                config = config,
                                overlay = true,
                                repository = chatRepository,
                                modifier = Modifier
                                    .align(Alignment.BottomCenter)
                                    .fillMaxWidth()
                                    .padding(10.dp),
                            )
                        }
                    }
                } else {
                    Column(modifier = Modifier.fillMaxSize()) {
                        Header(
                            config = config,
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 16.dp, vertical = 12.dp),
                        )
                        MobileFeatureActions(
                            config = config,
                            onRewardsClick = { showRewardsPanel = true },
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 16.dp),
                        )
                        StreamPlayer(
                            streamUrl = config.bouncecast.streamUrl,
                            online = false,
                            title = config.stream.streamTitle,
                        )
                        if (config.ads.enabled && config.ads.bannerEnabled) {
                            AdBanner(adManager = adManager)
                        }
                        if (config.features.chat && config.chat.enabled) {
                            ChatPanel(
                                config = config,
                                repository = chatRepository,
                            )
                        }
                    }
                }
            }
        }
    }

    config?.let { activeConfig ->
        RewardsWheelPanel(
            visible = showRewardsPanel,
            baseUrl = activeConfig.bouncecast.baseUrl,
            accessToken = rewardAccessToken,
            repository = rewardsRepository,
            onDismiss = { showRewardsPanel = false },
        )
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
private fun MobileFeatureActions(
    config: MobileConfig,
    modifier: Modifier = Modifier,
    overlay: Boolean = false,
    onRewardsClick: () -> Unit,
) {
    if (!config.features.stars && !config.features.rewardWheel) return
    val uriHandler = LocalUriHandler.current
    val baseUrl = config.bouncecast.baseUrl.trimEnd('/')
    val containerModifier = if (overlay) {
        modifier
            .clip(RoundedCornerShape(999.dp))
            .background(Color.Black.copy(alpha = 0.28f))
            .padding(4.dp)
    } else {
        modifier
    }

    Row(
        modifier = containerModifier,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        if (config.features.stars) {
            Button(
                onClick = { openBounceCastPath(uriHandler, baseUrl, "/account") },
            ) {
                Text("Stars")
            }
        }
        if (config.features.rewardWheel) {
            Button(
                onClick = onRewardsClick,
            ) {
                Text("Rewards")
            }
        }
    }
}

@Composable
private fun MobileLiveEventOverlay(
    events: List<ChatEvent>,
    starsEnabled: Boolean,
    rewardsEnabled: Boolean,
    modifier: Modifier = Modifier,
) {
    var activeEvent by remember { mutableStateOf<ChatEvent?>(null) }
    val latestEvent = events.lastOrNull()

    LaunchedEffect(latestEvent?.id, latestEvent?.timestamp, starsEnabled, rewardsEnabled) {
        val event = latestEvent ?: return@LaunchedEffect
        val isReward = event.isRewardOverlay()
        if ((isReward && !rewardsEnabled) || (!isReward && !starsEnabled)) return@LaunchedEffect
        activeEvent = event
        delay(4_200)
        if (activeEvent == event) {
            activeEvent = null
        }
    }

    activeEvent?.let { event ->
        val isReward = event.isRewardOverlay()
        val displayName = event.displayName
            .ifBlank { event.user?.displayName.orEmpty() }
            .ifBlank { "Viewer" }
        Column(
            modifier = modifier
                .widthIn(max = 320.dp)
                .clip(RoundedCornerShape(18.dp))
                .background(Color.Black.copy(alpha = 0.72f))
                .padding(horizontal = 18.dp, vertical = 14.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                if (isReward) {
                    event.prizeName.ifBlank { "Rewards Wheel win" }
                } else {
                    "${event.amount} Stars"
                },
                color = Color.White,
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                displayName,
                color = Color.White.copy(alpha = 0.86f),
                style = MaterialTheme.typography.bodyMedium,
            )
            event.message.ifBlank { event.body }.takeIf { it.isNotBlank() }?.let { message ->
                Text(
                    message,
                    color = Color.White,
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.padding(top = 6.dp),
                )
            }
        }
    }
}

@Composable
private fun Header(
    config: MobileConfig,
    modifier: Modifier = Modifier,
    overlay: Boolean = false,
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Column {
            Text(
                config.app.name,
                style = MaterialTheme.typography.titleLarge,
                color = if (overlay) Color.White else MaterialTheme.colorScheme.onSurface,
            )
            Text(
                if (config.stream.online) "Live now" else "Offline",
                color = if (config.stream.online) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        if (config.stream.viewerCount > 0) {
            Text(
                "${config.stream.viewerCount} watching",
                style = MaterialTheme.typography.labelLarge,
                color = if (overlay) Color.White else MaterialTheme.colorScheme.onSurface,
            )
        }
    }
}

private fun openBounceCastPath(
    uriHandler: androidx.compose.ui.platform.UriHandler,
    baseUrl: String,
    path: String,
) {
    if (baseUrl.isBlank()) return
    runCatching { uriHandler.openUri("$baseUrl$path") }
}

private fun ChatEvent.isRewardOverlay(): Boolean {
    return type == "REWARD_WHEEL_WIN" || prizeName.isNotBlank()
}
