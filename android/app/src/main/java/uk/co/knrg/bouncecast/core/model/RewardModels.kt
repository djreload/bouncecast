package uk.co.knrg.bouncecast.core.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class RewardSettings(
    val enabled: Boolean = false,
    @SerialName("spinCost") val spinCost: Int = 1,
    @SerialName("chatRewardsEnabled") val chatRewardsEnabled: Boolean = false,
    @SerialName("chatValidMessageCount") val chatValidMessageCount: Int = 10,
    @SerialName("chatCooldownSeconds") val chatCooldownSeconds: Int = 300,
    @SerialName("chatCreditReward") val chatCreditReward: Int = 1,
    @SerialName("topSupporterFirstCredits") val topSupporterFirstCredits: Int = 25,
    @SerialName("topSupporterSecondCredits") val topSupporterSecondCredits: Int = 15,
    @SerialName("topSupporterThirdCredits") val topSupporterThirdCredits: Int = 10,
    @SerialName("overlayEnabled") val overlayEnabled: Boolean = true,
    @SerialName("overlayDurationSeconds") val overlayDurationSeconds: Int = 5,
    @SerialName("overlaySoundEnabled") val overlaySoundEnabled: Boolean = true,
    @SerialName("overlayShowImage") val overlayShowImage: Boolean = true,
    @SerialName("overlayTemplate") val overlayTemplate: String = "",
)

@Serializable
data class RewardPrize(
    val id: Long = 0,
    val name: String = "",
    val description: String = "",
    val image: String = "",
    @SerialName("prizeType") val prizeType: String = "",
    @SerialName("oddsWeight") val oddsWeight: Int = 0,
    @SerialName("stockQuantity") val stockQuantity: Int? = null,
    val active: Boolean = false,
    @SerialName("displayOrder") val displayOrder: Int = 0,
    @SerialName("claimRequired") val claimRequired: Boolean = false,
    @SerialName("marketingConsentRequired") val marketingConsentRequired: Boolean = false,
    val terms: String = "",
)

@Serializable
data class RewardSpinBalance(
    @SerialName("userId") val userId: String = "",
    @SerialName("displayName") val displayName: String = "",
    val balance: Int = 0,
    @SerialName("lifetimeEarned") val lifetimeEarned: Int = 0,
    @SerialName("lifetimeSpent") val lifetimeSpent: Int = 0,
)

@Serializable
data class RewardTask(
    val id: Long = 0,
    val title: String = "",
    val description: String = "",
    @SerialName("creditReward") val creditReward: Int = 0,
    val active: Boolean = false,
)

@Serializable
data class RewardTaskCompletion(
    val id: Long = 0,
    @SerialName("userId") val userId: String = "",
    @SerialName("taskId") val taskId: Long = 0,
    @SerialName("taskTitle") val taskTitle: String = "",
    @SerialName("ledgerId") val ledgerId: Long? = null,
    val status: String = "",
    @SerialName("completedAt") val completedAt: String = "",
)

@Serializable
data class RewardWheelData(
    val settings: RewardSettings = RewardSettings(),
    val balance: RewardSpinBalance = RewardSpinBalance(),
    val prizes: List<RewardPrize> = emptyList(),
    val tasks: List<RewardTask> = emptyList(),
    @SerialName("taskCompletions") val taskCompletions: List<RewardTaskCompletion> = emptyList(),
)

@Serializable
data class RewardSpin(
    val id: Long = 0,
    @SerialName("userId") val userId: String = "",
    @SerialName("prizeId") val prizeId: Long? = null,
    @SerialName("prizeSnapshot") val prizeSnapshot: String = "",
    @SerialName("imageSnapshot") val imageSnapshot: String = "",
    @SerialName("typeSnapshot") val typeSnapshot: String = "",
    @SerialName("oddsSnapshot") val oddsSnapshot: Int = 0,
    @SerialName("ledgerId") val ledgerId: Long = 0,
    @SerialName("resultType") val resultType: String = "",
    @SerialName("createdAt") val createdAt: String = "",
)

@Serializable
data class RewardWinner(
    val id: Long = 0,
    @SerialName("userId") val userId: String = "",
    @SerialName("usernameSnapshot") val usernameSnapshot: String = "",
    @SerialName("prizeSnapshot") val prizeSnapshot: String = "",
    @SerialName("imageSnapshot") val imageSnapshot: String = "",
    @SerialName("typeSnapshot") val typeSnapshot: String = "",
    @SerialName("spinId") val spinId: Long = 0,
    @SerialName("notificationStatus") val notificationStatus: String = "",
    @SerialName("createdAt") val createdAt: String = "",
)

@Serializable
data class RewardClaim(
    val id: Long = 0,
    @SerialName("userId") val userId: String = "",
    @SerialName("spinId") val spinId: Long = 0,
    val status: String = "",
    @SerialName("prizeSnapshot") val prizeSnapshot: String = "",
    @SerialName("createdAt") val createdAt: String = "",
    @SerialName("updatedAt") val updatedAt: String = "",
)

@Serializable
data class RewardOrder(
    val id: Long = 0,
    @SerialName("winnerUserId") val winnerUserId: String = "",
    @SerialName("usernameSnapshot") val usernameSnapshot: String = "",
    @SerialName("prizeSnapshot") val prizeSnapshot: String = "",
    @SerialName("spinId") val spinId: Long = 0,
    @SerialName("orderStatus") val orderStatus: String = "",
    @SerialName("dispatchStatus") val dispatchStatus: String = "",
    @SerialName("createdAt") val createdAt: String = "",
    @SerialName("updatedAt") val updatedAt: String = "",
)

@Serializable
data class RewardSpinResult(
    val spin: RewardSpin = RewardSpin(),
    val prize: RewardPrize = RewardPrize(),
    val balance: Int = 0,
    val winner: RewardWinner? = null,
    val claim: RewardClaim? = null,
    val order: RewardOrder? = null,
    @SerialName("overlaySent") val overlaySent: Boolean = false,
    val message: String = "",
)

@Serializable
data class RewardTaskCompleteRequest(
    @SerialName("taskId") val taskId: Long,
)

@Serializable
data class RewardTaskCompleteResponse(
    val completion: RewardTaskCompletion = RewardTaskCompletion(),
    val balance: RewardSpinBalance = RewardSpinBalance(),
    val awarded: Boolean = false,
)

@Serializable
data class EmptyRewardRequest(
    val request: String = "mobile",
)
