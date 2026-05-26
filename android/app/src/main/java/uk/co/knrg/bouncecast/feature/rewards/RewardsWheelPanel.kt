package uk.co.knrg.bouncecast.feature.rewards

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import uk.co.knrg.bouncecast.core.model.RewardPrize
import uk.co.knrg.bouncecast.core.model.RewardTask

@Composable
fun RewardsWheelPanel(
    visible: Boolean,
    baseUrl: String,
    accessToken: String,
    repository: RewardsRepository,
    onDismiss: () -> Unit,
) {
    if (!visible) return

    val scope = rememberCoroutineScope()
    val wheelData by repository.wheelData.collectAsState()
    val loading by repository.loading.collectAsState()
    val spinning by repository.spinning.collectAsState()
    val error by repository.error.collectAsState()
    val result by repository.lastResult.collectAsState()

    LaunchedEffect(visible, baseUrl, accessToken) {
        if (visible && accessToken.isNotBlank()) {
            repository.load(baseUrl, accessToken)
        }
    }

    val activePrizes = remember(wheelData?.prizes) {
        wheelData?.prizes.orEmpty().filter { it.active && it.oddsWeight > 0 }
    }
    val activeTasks = remember(wheelData?.tasks) {
        wheelData?.tasks.orEmpty().filter { it.active }
    }
    val completedTaskIds = remember(wheelData?.taskCompletions) {
        wheelData?.taskCompletions.orEmpty().map { it.taskId }.toSet()
    }
    val spinCost = wheelData?.settings?.spinCost?.coerceAtLeast(1) ?: 1
    val balance = wheelData?.balance?.balance ?: 0
    val enabled = wheelData?.settings?.enabled == true
    val canSpin = enabled && activePrizes.isNotEmpty() && balance >= spinCost && !spinning && !loading
    val statusMessage = when {
        accessToken.isBlank() -> "Preparing your viewer session..."
        loading && wheelData == null -> "Loading Rewards Wheel..."
        wheelData == null -> "Rewards Wheel data is not available yet."
        !enabled -> "Rewards Wheel is enabled in the app, but it still needs to be switched on in Admin > Rewards Wheel."
        activePrizes.isEmpty() -> "Add at least one active prize with odds above zero in Admin > Rewards Wheel > Prizes."
        balance < spinCost -> "You need $spinCost Spin Credit${if (spinCost == 1) "" else "s"} to spin. Complete a task, chat when rewards are on, or ask an admin to grant credits."
        else -> null
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Rewards Wheel") },
        text = {
            Column(
                modifier = Modifier
                    .widthIn(max = 460.dp)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Column {
                        Text("Spin Credits", style = MaterialTheme.typography.labelLarge)
                        Text(
                            balance.toString(),
                            style = MaterialTheme.typography.headlineSmall,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                    Text(
                        "$spinCost per spin",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }

                if (loading && wheelData != null) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        CircularProgressIndicator(modifier = Modifier.padding(end = 12.dp))
                        Text("Refreshing rewards")
                    }
                }

                statusMessage?.let { message ->
                    Surface(
                        color = MaterialTheme.colorScheme.surfaceVariant,
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(
                            text = message,
                            modifier = Modifier.padding(12.dp),
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }

                error?.let { message ->
                    Surface(
                        color = MaterialTheme.colorScheme.errorContainer,
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(
                            text = message,
                            modifier = Modifier.padding(12.dp),
                            color = MaterialTheme.colorScheme.onErrorContainer,
                        )
                    }
                }

                result?.let { spinResult ->
                    Surface(
                        color = MaterialTheme.colorScheme.primaryContainer,
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Column(modifier = Modifier.padding(12.dp)) {
                            Text(
                                spinResult.message.ifBlank { "Spin complete" },
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onPrimaryContainer,
                            )
                            Text(
                                spinResult.prize.name.ifBlank { spinResult.spin.prizeSnapshot },
                                color = MaterialTheme.colorScheme.onPrimaryContainer,
                            )
                        }
                    }
                }

                if (wheelData != null) {
                    RewardSectionTitle("Prize options")
                    if (activePrizes.isEmpty()) {
                        EmptyRewardText("No active prize options are showing yet.")
                    } else {
                        activePrizes.forEach { prize ->
                            RewardPrizeRow(prize)
                        }
                    }

                    RewardSectionTitle("Ways to earn")
                    if (activeTasks.isEmpty()) {
                        EmptyRewardText("No reward tasks are active yet.")
                    } else {
                        activeTasks.forEach { task ->
                            RewardTaskRow(
                                task = task,
                                completed = completedTaskIds.contains(task.id),
                                onComplete = {
                                    scope.launch {
                                        repository.completeTask(baseUrl, accessToken, task.id)
                                    }
                                },
                            )
                        }
                    }
                }
            }
        },
        confirmButton = {
            Button(
                enabled = canSpin,
                onClick = {
                    scope.launch {
                        repository.spin(baseUrl, accessToken)
                    }
                },
            ) {
                Text(if (spinning) "Spinning..." else "Spin")
            }
        },
        dismissButton = {
            TextButton(
                onClick = {
                    repository.clearError()
                    repository.dismissResult()
                    onDismiss()
                },
            ) {
                Text("Close")
            }
        },
    )
}

@Composable
private fun RewardSectionTitle(title: String) {
    Text(
        text = title,
        style = MaterialTheme.typography.titleMedium,
        fontWeight = FontWeight.Bold,
    )
}

@Composable
private fun RewardPrizeRow(prize: RewardPrize) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceVariant,
        shape = RoundedCornerShape(12.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text(prize.name.ifBlank { "Reward prize" }, fontWeight = FontWeight.Bold)
                Text(prize.prizeType.ifBlank { "prize" }, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            if (prize.description.isNotBlank()) {
                Spacer(Modifier.height(4.dp))
                Text(prize.description, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            prize.stockQuantity?.let { stock ->
                Spacer(Modifier.height(4.dp))
                Text("Stock: $stock", style = MaterialTheme.typography.labelMedium)
            }
        }
    }
}

@Composable
private fun RewardTaskRow(
    task: RewardTask,
    completed: Boolean,
    onComplete: () -> Unit,
) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceVariant,
        shape = RoundedCornerShape(12.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(
                modifier = Modifier
                    .weight(1f)
                    .padding(end = 12.dp),
            ) {
                Text(task.title.ifBlank { "Reward task" }, fontWeight = FontWeight.Bold)
                Text(
                    task.description.ifBlank { "Complete this task to earn Spin Credits." },
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Text("+${task.creditReward} credits", style = MaterialTheme.typography.labelMedium)
            }
            if (completed) {
                Text("Done", color = MaterialTheme.colorScheme.primary)
            } else {
                OutlinedButton(onClick = onComplete) {
                    Text("Complete")
                }
            }
        }
    }
}

@Composable
private fun EmptyRewardText(message: String) {
    Text(
        text = message,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        style = MaterialTheme.typography.bodyMedium,
    )
}
