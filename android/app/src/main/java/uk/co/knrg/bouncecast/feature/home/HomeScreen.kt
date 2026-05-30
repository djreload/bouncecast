package uk.co.knrg.bouncecast.feature.home

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.systemBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import androidx.compose.ui.platform.UriHandler
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import uk.co.knrg.bouncecast.core.config.MobileConfigRepository
import uk.co.knrg.bouncecast.core.model.ChatEvent
import uk.co.knrg.bouncecast.core.model.MobileConfig
import uk.co.knrg.bouncecast.core.model.NavigationItem
import uk.co.knrg.bouncecast.core.model.ViewerIdentity
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
    val currentAccessToken by chatRepository.accessToken.collectAsState()
    val currentUserId by chatRepository.userId.collectAsState()
    val currentDisplayName by chatRepository.displayName.collectAsState()
    var viewerIdentity by remember { mutableStateOf(ViewerIdentity()) }
    var identityLoaded by remember { mutableStateOf(false) }
    var accountDialogOpen by remember { mutableStateOf(false) }
    var showRewardsPanel by remember { mutableStateOf(false) }
    var showChatPanel by remember { mutableStateOf(true) }
    val chatAvailable = config?.let { it.features.chat && it.chat.enabled } == true
    val shouldConnectToChat = config?.let {
        it.chat.enabled && (it.features.chat || it.features.starsOverlay || it.features.rewardOverlay)
    } == true
    val shouldRegisterViewer = config?.let { shouldConnectToChat || it.features.rewardWheel } == true

    LaunchedEffect(Unit) {
        val savedIdentity = configRepository.readViewerIdentity()
        viewerIdentity = savedIdentity
        chatRepository.restoreIdentity(savedIdentity)
        identityLoaded = true
    }

    LaunchedEffect(Unit) {
        while (true) {
            runCatching { configRepository.refresh() }
                .onFailure { refreshError = it.message ?: "Unable to refresh mobile config" }
            delay(15_000)
        }
    }

    LaunchedEffect(chatAvailable) {
        showChatPanel = chatAvailable
    }

    LaunchedEffect(identityLoaded, currentUserId, currentAccessToken, currentDisplayName) {
        if (!identityLoaded || currentAccessToken.isBlank()) return@LaunchedEffect
        val nextIdentity = ViewerIdentity(
            userId = currentUserId,
            accessToken = currentAccessToken,
            displayName = currentDisplayName,
        ).normalized()
        if (nextIdentity != viewerIdentity) {
            viewerIdentity = nextIdentity
            configRepository.writeViewerIdentity(nextIdentity)
        }
    }

    LaunchedEffect(
        config?.bouncecast?.registerChatUrl,
        config?.bouncecast?.chatHistoryUrl,
        config?.bouncecast?.chatWebSocketUrl,
        config?.features?.rewardWheel,
        identityLoaded,
        viewerIdentity.accessToken,
        viewerIdentity.displayName,
        shouldConnectToChat,
    ) {
        if (!identityLoaded) return@LaunchedEffect
        val activeConfig = config ?: return@LaunchedEffect
        if (!shouldRegisterViewer) {
            chatRepository.disconnect()
            return@LaunchedEffect
        }
        if (viewerIdentity.accessToken.isNotBlank()) {
            chatRepository.restoreIdentity(viewerIdentity)
        }
        val viewerDisplayName = viewerIdentity.displayName.ifBlank { "Android Viewer" }
        if (shouldConnectToChat) {
            chatRepository.registerAndConnect(
                registerUrl = activeConfig.bouncecast.registerChatUrl,
                historyUrl = activeConfig.bouncecast.chatHistoryUrl,
                webSocketUrl = activeConfig.bouncecast.chatWebSocketUrl,
                displayName = viewerDisplayName,
            )
        } else {
            chatRepository.disconnect()
            runCatching {
                chatRepository.register(activeConfig.bouncecast.registerChatUrl, viewerDisplayName)
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
                            onAccountClick = { accountDialogOpen = true },
                        )
                        MobileFeatureActions(
                            config = config,
                            chatAvailable = chatAvailable,
                            chatOpen = showChatPanel,
                            overlay = true,
                            onChatClick = { showChatPanel = !showChatPanel },
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
                        if (chatAvailable && showChatPanel) {
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
                        if (!showChatPanel) {
                            FooterMenu(
                                config = config,
                                overlay = true,
                                onAccountClick = { accountDialogOpen = true },
                                onRewardsClick = { showRewardsPanel = true },
                                modifier = Modifier
                                    .align(Alignment.BottomCenter)
                                    .fillMaxWidth()
                                    .padding(horizontal = 12.dp, vertical = 14.dp),
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
                            onAccountClick = { accountDialogOpen = true },
                        )
                        MobileFeatureActions(
                            config = config,
                            chatAvailable = chatAvailable,
                            chatOpen = showChatPanel,
                            onChatClick = { showChatPanel = !showChatPanel },
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
                        if (chatAvailable && showChatPanel) {
                            ChatPanel(
                                config = config,
                                repository = chatRepository,
                                fillAvailableHeight = true,
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .weight(1f),
                            )
                        } else {
                            Spacer(Modifier.weight(1f))
                        }
                        FooterMenu(
                            config = config,
                            onAccountClick = { accountDialogOpen = true },
                            onRewardsClick = { showRewardsPanel = true },
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 16.dp, vertical = 12.dp),
                        )
                    }
                }
            }
        }
    }

    config?.let { activeConfig ->
        RewardsWheelPanel(
            visible = showRewardsPanel,
            baseUrl = activeConfig.bouncecast.baseUrl,
            accessToken = currentAccessToken,
            repository = rewardsRepository,
            onDismiss = { showRewardsPanel = false },
        )
        AccountDialog(
            visible = accountDialogOpen,
            identity = viewerIdentity,
            currentUserId = currentUserId,
            currentDisplayName = currentDisplayName,
            onDismiss = { accountDialogOpen = false },
            onSave = { displayName ->
                chatScope.launch {
                    val cleanDisplayName = displayName.ifBlank { "Android Viewer" }
                    runCatching {
                        val response = if (currentAccessToken.isBlank()) {
                            chatRepository.register(activeConfig.bouncecast.registerChatUrl, cleanDisplayName)
                        } else {
                            val profileUrl = "${activeConfig.bouncecast.baseUrl.trimEnd('/')}/api/bouncecast/account/profile"
                            chatRepository.updateProfile(profileUrl, cleanDisplayName)
                        }
                        val nextIdentity = ViewerIdentity(
                            userId = response.id.ifBlank { currentUserId },
                            accessToken = response.accessToken.ifBlank { currentAccessToken },
                            displayName = response.displayName.ifBlank { cleanDisplayName },
                        ).normalized()
                        viewerIdentity = nextIdentity
                        configRepository.writeViewerIdentity(nextIdentity)
                        if (shouldConnectToChat) {
                            chatRepository.loadHistory(activeConfig.bouncecast.chatHistoryUrl)
                            chatRepository.connect(activeConfig.bouncecast.chatWebSocketUrl)
                        }
                    }.onFailure {
                        refreshError = it.message ?: "Unable to save app account"
                    }
                    accountDialogOpen = false
                }
            },
            onReset = {
                chatScope.launch {
                    configRepository.clearViewerIdentity()
                    chatRepository.clearIdentity()
                    viewerIdentity = ViewerIdentity()
                    accountDialogOpen = false
                }
            },
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
    chatAvailable: Boolean,
    chatOpen: Boolean,
    modifier: Modifier = Modifier,
    overlay: Boolean = false,
    onChatClick: () -> Unit,
    onRewardsClick: () -> Unit,
) {
    if (!chatAvailable && !config.features.stars && !config.features.rewardWheel) return
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
        if (chatAvailable) {
            Button(
                onClick = onChatClick,
            ) {
                Text(if (chatOpen) "Hide chat" else "Chat")
            }
        }
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
    onAccountClick: () -> Unit,
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
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
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (config.stream.viewerCount > 0) {
                Text(
                    "${config.stream.viewerCount} watching",
                    style = MaterialTheme.typography.labelLarge,
                    color = if (overlay) Color.White else MaterialTheme.colorScheme.onSurface,
                )
            }
            OutlinedButton(
                onClick = onAccountClick,
                contentPadding = PaddingValues(horizontal = 12.dp, vertical = 0.dp),
            ) {
                Text("Account")
            }
        }
    }
}

@Composable
private fun FooterMenu(
    config: MobileConfig,
    modifier: Modifier = Modifier,
    overlay: Boolean = false,
    onAccountClick: () -> Unit,
    onRewardsClick: () -> Unit,
) {
    val uriHandler = LocalUriHandler.current
    val baseUrl = config.bouncecast.baseUrl.trimEnd('/')
    val navigationItems = config.navigation
        .filter { it.enabled && it.label.isNotBlank() }
        .sortedBy { it.displayOrder }
        .take(5)
    val containerColor = if (overlay) {
        Color.Black.copy(alpha = 0.34f)
    } else {
        MaterialTheme.colorScheme.surface.copy(alpha = 0.82f)
    }

    Row(
        modifier = modifier
            .clip(RoundedCornerShape(999.dp))
            .background(containerColor)
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 10.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        OutlinedButton(onClick = onAccountClick) {
            Text("Account")
        }
        if (config.features.stars) {
            OutlinedButton(onClick = { openBounceCastPath(uriHandler, baseUrl, "/account") }) {
                Text("Stars")
            }
        }
        if (config.features.rewardWheel) {
            OutlinedButton(onClick = onRewardsClick) {
                Text("Rewards")
            }
        }
        navigationItems.forEach { item ->
            OutlinedButton(onClick = { openBounceCastNavigationItem(uriHandler, baseUrl, item) }) {
                Text(item.label)
            }
        }
    }
}

@Composable
private fun AccountDialog(
    visible: Boolean,
    identity: ViewerIdentity,
    currentUserId: String,
    currentDisplayName: String,
    onDismiss: () -> Unit,
    onSave: (String) -> Unit,
    onReset: () -> Unit,
) {
    if (!visible) return
    var displayName by remember(visible, currentDisplayName, identity.displayName) {
        mutableStateOf(
            currentDisplayName
                .ifBlank { identity.displayName }
                .ifBlank { "Android Viewer" },
        )
    }
    val userId = currentUserId.ifBlank { identity.userId }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("App account") },
        text = {
            Column {
                Text(
                    if (userId.isBlank()) "Viewer ID: not created yet" else "Viewer ID: $userId",
                    style = MaterialTheme.typography.bodyMedium,
                )
                Spacer(Modifier.height(12.dp))
                OutlinedTextField(
                    value = displayName,
                    onValueChange = { displayName = it.take(40) },
                    label = { Text("Display name") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(8.dp))
                Text(
                    "This name is used for chat, Stars, reactions, and Rewards Wheel entries.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        },
        confirmButton = {
            Button(
                onClick = { onSave(displayName) },
                enabled = displayName.isNotBlank(),
            ) {
                Text("Save")
            }
        },
        dismissButton = {
            Row {
                TextButton(onClick = onReset) {
                    Text("Reset")
                }
                Spacer(Modifier.width(8.dp))
                TextButton(onClick = onDismiss) {
                    Text("Cancel")
                }
            }
        },
    )
}

private fun openBounceCastPath(
    uriHandler: UriHandler,
    baseUrl: String,
    path: String,
) {
    if (baseUrl.isBlank()) return
    runCatching { uriHandler.openUri("$baseUrl$path") }
}

private fun openBounceCastNavigationItem(
    uriHandler: UriHandler,
    baseUrl: String,
    item: NavigationItem,
) {
    val url = item.url.trim()
    if (url.isBlank()) return
    val target = if (url.startsWith("http://") || url.startsWith("https://")) {
        url
    } else {
        val path = if (url.startsWith("/")) url else "/$url"
        "$baseUrl$path"
    }
    runCatching { uriHandler.openUri(target) }
}

private fun ChatEvent.isRewardOverlay(): Boolean {
    return type == "REWARD_WHEEL_WIN" || prizeName.isNotBlank()
}
