package uk.co.knrg.bouncecast.feature.rewards

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import uk.co.knrg.bouncecast.core.model.EmptyRewardRequest
import uk.co.knrg.bouncecast.core.model.RewardSpinResult
import uk.co.knrg.bouncecast.core.model.RewardTaskCompleteRequest
import uk.co.knrg.bouncecast.core.model.RewardTaskCompleteResponse
import uk.co.knrg.bouncecast.core.model.RewardWheelData
import uk.co.knrg.bouncecast.core.network.BounceCastApi
import java.net.URLEncoder

class RewardsRepository(
    private val api: BounceCastApi,
) {
    private val _wheelData = MutableStateFlow<RewardWheelData?>(null)
    val wheelData: StateFlow<RewardWheelData?> = _wheelData

    private val _loading = MutableStateFlow(false)
    val loading: StateFlow<Boolean> = _loading

    private val _spinning = MutableStateFlow(false)
    val spinning: StateFlow<Boolean> = _spinning

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error

    private val _lastResult = MutableStateFlow<RewardSpinResult?>(null)
    val lastResult: StateFlow<RewardSpinResult?> = _lastResult

    suspend fun load(baseUrl: String, accessToken: String) {
        if (baseUrl.isBlank() || accessToken.isBlank()) return
        _loading.value = true
        runCatching {
            api.get<RewardWheelData>(rewardUrl(baseUrl, "wheel", accessToken))
        }.onSuccess { data ->
            _wheelData.value = data
            _error.value = null
        }.onFailure { throwable ->
            _error.value = throwable.message ?: "Unable to load Rewards Wheel"
        }
        _loading.value = false
    }

    suspend fun spin(baseUrl: String, accessToken: String) {
        if (baseUrl.isBlank() || accessToken.isBlank()) return
        _spinning.value = true
        _lastResult.value = null
        runCatching {
            api.post<EmptyRewardRequest, RewardSpinResult>(
                url = rewardUrl(baseUrl, "spin", accessToken),
                body = EmptyRewardRequest(),
            )
        }.onSuccess { result ->
            _lastResult.value = result
            _wheelData.value = _wheelData.value?.let { current ->
                current.copy(balance = current.balance.copy(balance = result.balance))
            }
            _error.value = null
        }.onFailure { throwable ->
            _error.value = throwable.message ?: "Unable to spin the Rewards Wheel"
        }
        _spinning.value = false
    }

    suspend fun completeTask(baseUrl: String, accessToken: String, taskId: Long) {
        if (baseUrl.isBlank() || accessToken.isBlank() || taskId <= 0) return
        runCatching {
            api.post<RewardTaskCompleteRequest, RewardTaskCompleteResponse>(
                url = rewardUrl(baseUrl, "tasks/complete", accessToken),
                body = RewardTaskCompleteRequest(taskId),
            )
        }.onSuccess { response ->
            _wheelData.value = _wheelData.value?.let { current ->
                val completions = if (current.taskCompletions.any { it.taskId == taskId }) {
                    current.taskCompletions
                } else {
                    current.taskCompletions + response.completion
                }
                current.copy(balance = response.balance, taskCompletions = completions)
            }
            _error.value = null
        }.onFailure { throwable ->
            _error.value = throwable.message ?: "Unable to complete reward task"
        }
    }

    fun dismissResult() {
        _lastResult.value = null
    }

    fun clearError() {
        _error.value = null
    }
}

private fun rewardUrl(baseUrl: String, path: String, accessToken: String): String {
    val cleanBaseUrl = baseUrl.trimEnd('/')
    return "$cleanBaseUrl/api/rewards/$path?accessToken=${accessToken.urlEncode()}"
}

private fun String.urlEncode(): String = URLEncoder.encode(this, Charsets.UTF_8.name())
