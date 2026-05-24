package models

import "time"

const (
	RewardCreditSourceChatActivity          = "chat_activity"
	RewardCreditSourceTaskCompleted         = "task_completed"
	RewardCreditSourceAchievementUnlocked   = "achievement_unlocked"
	RewardCreditSourceTopSupporterReward    = "top_supporter_reward"
	RewardCreditSourceAdminGrant            = "admin_grant"
	RewardCreditSourceAdminAdjustment       = "admin_adjustment"
	RewardCreditSourceSpinSpend             = "spin_spend"
	RewardCreditSourceSpinSpendRefund       = "spin_spend_refund"
	RewardAchievementConditionFirstSpin     = "reward_first_spin"
	RewardAchievementConditionPrizeWin      = "reward_prize_win"
	RewardAchievementConditionTaskCompleted = "reward_task_completed"
	RewardAchievementConditionStarsSent     = "stars_sent"
	RewardAchievementConditionTopSupporter  = "top_supporter_reward"
	RewardPrizeTypePhysical                 = "physical"
	RewardPrizeTypeDigital                  = "digital"
	RewardPrizeTypeDiscountFuture           = "discount_future_placeholder"
	RewardPrizeTypeSorry                    = "sorry"
	RewardClaimStatusPendingDetails         = "pending_details"
	RewardClaimStatusSubmitted              = "submitted"
	RewardClaimStatusApproved               = "approved"
	RewardClaimStatusCancelled              = "cancelled"
	RewardOrderStatusAwaitingClaimDetails   = "awaiting_claim_details"
	RewardOrderStatusReadyToFulfil          = "ready_to_fulfil"
	RewardOrderStatusPacked                 = "packed"
	RewardOrderStatusDispatched             = "dispatched"
	RewardOrderStatusDelivered              = "delivered"
	RewardOrderStatusCancelled              = "cancelled"
	RewardAdminMessageTypeWin               = "reward_win"
	RewardUserNotificationOrderDispatched   = "reward_order_dispatched"
	RewardUserNotificationPrizeClaimCreated = "reward_prize_claim_created"
)

// RewardSettings controls the internal/manual Rewards Wheel.
type RewardSettings struct {
	Enabled                    bool   `json:"enabled"`
	SpinCost                   int    `json:"spinCost"`
	ChatRewardsEnabled         bool   `json:"chatRewardsEnabled"`
	ChatValidMessageCount      int    `json:"chatValidMessageCount"`
	ChatCooldownSeconds        int    `json:"chatCooldownSeconds"`
	ChatCreditReward           int    `json:"chatCreditReward"`
	TopSupporterFirstCredits   int    `json:"topSupporterFirstCredits"`
	TopSupporterSecondCredits  int    `json:"topSupporterSecondCredits"`
	TopSupporterThirdCredits   int    `json:"topSupporterThirdCredits"`
	OverlayEnabled             bool   `json:"overlayEnabled"`
	OverlayDurationSeconds     int    `json:"overlayDurationSeconds"`
	OverlaySoundEnabled        bool   `json:"overlaySoundEnabled"`
	OverlayShowImage           bool   `json:"overlayShowImage"`
	OverlayTemplate            string `json:"overlayTemplate"`
	DebugLoggingEnabled        bool   `json:"debugLoggingEnabled,omitempty"`
	FuturePaidRewardsNote      string `json:"futurePaidRewardsNote,omitempty"`
	FutureDiscountRewardsNote  string `json:"futureDiscountRewardsNote,omitempty"`
	FulfilmentPrivacyStatement string `json:"fulfilmentPrivacyStatement,omitempty"`
}

// RewardPrize is a wheel segment managed by admins.
type RewardPrize struct {
	ID                       int64     `json:"id"`
	Name                     string    `json:"name"`
	Description              string    `json:"description,omitempty"`
	Image                    string    `json:"image,omitempty"`
	PrizeType                string    `json:"prizeType"`
	OddsWeight               int       `json:"oddsWeight"`
	StockQuantity            *int      `json:"stockQuantity,omitempty"`
	Active                   bool      `json:"active"`
	DisplayOrder             int       `json:"displayOrder"`
	ClaimRequired            bool      `json:"claimRequired"`
	MarketingConsentRequired bool      `json:"marketingConsentRequired"`
	Terms                    string    `json:"terms,omitempty"`
	FulfilmentNotes          string    `json:"fulfilmentNotes,omitempty"`
	CreatedAt                time.Time `json:"createdAt,omitempty"`
	UpdatedAt                time.Time `json:"updatedAt,omitempty"`
}

type RewardSpinBalance struct {
	UserID          string    `json:"userId"`
	Balance         int       `json:"balance"`
	LifetimeEarned  int       `json:"lifetimeEarned"`
	LifetimeSpent   int       `json:"lifetimeSpent"`
	CreatedAt       time.Time `json:"createdAt,omitempty"`
	UpdatedAt       time.Time `json:"updatedAt,omitempty"`
	DisplayName     string    `json:"displayName,omitempty"`
	Email           string    `json:"email,omitempty"`
	ProfileImageURL string    `json:"profileImageUrl,omitempty"`
}

type RewardSpinLedgerEntry struct {
	ID           int64     `json:"id"`
	UserID       string    `json:"userId"`
	Amount       int       `json:"amount"`
	BalanceAfter int       `json:"balanceAfter"`
	Source       string    `json:"source"`
	ReferenceID  string    `json:"referenceId,omitempty"`
	Note         string    `json:"note,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type RewardSpin struct {
	ID            int64     `json:"id"`
	UserID        string    `json:"userId"`
	PrizeID       int64     `json:"prizeId,omitempty"`
	PrizeSnapshot string    `json:"prizeSnapshot"`
	ImageSnapshot string    `json:"imageSnapshot,omitempty"`
	TypeSnapshot  string    `json:"typeSnapshot"`
	OddsSnapshot  int       `json:"oddsSnapshot"`
	LedgerID      int64     `json:"ledgerId"`
	ResultType    string    `json:"resultType"`
	CreatedAt     time.Time `json:"createdAt"`
}

type RewardWinner struct {
	ID                 int64     `json:"id"`
	UserID             string    `json:"userId"`
	UsernameSnapshot   string    `json:"usernameSnapshot"`
	Email              string    `json:"email,omitempty"`
	PrizeID            int64     `json:"prizeId,omitempty"`
	PrizeSnapshot      string    `json:"prizeSnapshot"`
	ImageSnapshot      string    `json:"imageSnapshot,omitempty"`
	TypeSnapshot       string    `json:"typeSnapshot"`
	SpinID             int64     `json:"spinId"`
	ClaimID            int64     `json:"claimId,omitempty"`
	OrderID            int64     `json:"orderId,omitempty"`
	NotificationStatus string    `json:"notificationStatus"`
	CreatedAt          time.Time `json:"createdAt"`
}

type RewardClaim struct {
	ID                       int64      `json:"id"`
	UserID                   string     `json:"userId"`
	SpinID                   int64      `json:"spinId"`
	PrizeID                  int64      `json:"prizeId,omitempty"`
	Status                   string     `json:"status"`
	FullName                 string     `json:"fullName,omitempty"`
	AddressLine1             string     `json:"addressLine1,omitempty"`
	AddressLine2             string     `json:"addressLine2,omitempty"`
	TownCity                 string     `json:"townCity,omitempty"`
	CountyState              string     `json:"countyState,omitempty"`
	Postcode                 string     `json:"postcode,omitempty"`
	Country                  string     `json:"country,omitempty"`
	Email                    string     `json:"email,omitempty"`
	Phone                    string     `json:"phone,omitempty"`
	DeliveryNotes            string     `json:"deliveryNotes,omitempty"`
	MarketingConsent         bool       `json:"marketingConsent"`
	ConsentAt                *time.Time `json:"consentAt,omitempty"`
	ConsentText              string     `json:"consentText,omitempty"`
	IPAddress                string     `json:"-"`
	UserAgent                string     `json:"-"`
	CreatedAt                time.Time  `json:"createdAt"`
	UpdatedAt                time.Time  `json:"updatedAt"`
	PrizeSnapshot            string     `json:"prizeSnapshot,omitempty"`
	ImageSnapshot            string     `json:"imageSnapshot,omitempty"`
	MarketingConsentRequired bool       `json:"marketingConsentRequired,omitempty"`
}

type RewardClaimSubmission struct {
	ClaimID          int64  `json:"claimId"`
	FullName         string `json:"fullName"`
	AddressLine1     string `json:"addressLine1"`
	AddressLine2     string `json:"addressLine2"`
	TownCity         string `json:"townCity"`
	CountyState      string `json:"countyState"`
	Postcode         string `json:"postcode"`
	Country          string `json:"country"`
	Email            string `json:"email"`
	Phone            string `json:"phone"`
	DeliveryNotes    string `json:"deliveryNotes"`
	MarketingConsent bool   `json:"marketingConsent"`
	ConsentText      string `json:"consentText"`
}

type RewardOrder struct {
	ID                  int64      `json:"id"`
	WinnerUserID        string     `json:"winnerUserId"`
	UsernameSnapshot    string     `json:"usernameSnapshot"`
	PrizeID             int64      `json:"prizeId,omitempty"`
	PrizeSnapshot       string     `json:"prizeSnapshot"`
	ImageSnapshot       string     `json:"imageSnapshot,omitempty"`
	SpinID              int64      `json:"spinId"`
	ClaimID             int64      `json:"claimId,omitempty"`
	AddressSnapshot     string     `json:"addressSnapshot,omitempty"`
	OrderStatus         string     `json:"orderStatus"`
	DispatchStatus      string     `json:"dispatchStatus"`
	AdminNotes          string     `json:"adminNotes,omitempty"`
	Courier             string     `json:"courier,omitempty"`
	TrackingReference   string     `json:"trackingReference,omitempty"`
	TrackingURL         string     `json:"trackingUrl,omitempty"`
	DispatchNote        string     `json:"dispatchNote,omitempty"`
	DispatchDate        *time.Time `json:"dispatchDate,omitempty"`
	DeliveredDate       *time.Time `json:"deliveredDate,omitempty"`
	DispatchedByAdminID string     `json:"dispatchedByAdminId,omitempty"`
	DispatchedAt        *time.Time `json:"dispatchedAt,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type RewardAdminMessage struct {
	ID        int64      `json:"id"`
	Type      string     `json:"type"`
	Severity  string     `json:"severity"`
	Title     string     `json:"title"`
	Body      string     `json:"body,omitempty"`
	UserID    string     `json:"userId,omitempty"`
	PrizeID   int64      `json:"prizeId,omitempty"`
	SpinID    int64      `json:"spinId,omitempty"`
	ClaimID   int64      `json:"claimId,omitempty"`
	OrderID   int64      `json:"orderId,omitempty"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type RewardUserNotification struct {
	ID         int64      `json:"id"`
	UserID     string     `json:"userId"`
	Type       string     `json:"type"`
	OrderID    int64      `json:"orderId,omitempty"`
	PrizeID    int64      `json:"prizeId,omitempty"`
	Title      string     `json:"title"`
	Message    string     `json:"message,omitempty"`
	EmailSent  bool       `json:"emailSent"`
	EmailError string     `json:"emailError,omitempty"`
	ReadAt     *time.Time `json:"readAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type RewardTask struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	CreditReward int       `json:"creditReward"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"createdAt,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
}

type RewardTaskCompletion struct {
	ID          int64     `json:"id"`
	UserID      string    `json:"userId"`
	TaskID      int64     `json:"taskId"`
	TaskTitle   string    `json:"taskTitle,omitempty"`
	LedgerID    int64     `json:"ledgerId,omitempty"`
	Status      string    `json:"status"`
	CompletedAt time.Time `json:"completedAt"`
}

type RewardAchievement struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	ConditionKey string    `json:"conditionKey"`
	RewardAmount int       `json:"rewardAmount"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"createdAt,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
}

type RewardAchievementUnlock struct {
	ID              int64     `json:"id"`
	UserID          string    `json:"userId"`
	AchievementID   int64     `json:"achievementId"`
	AchievementName string    `json:"achievementName,omitempty"`
	ConditionKey    string    `json:"conditionKey"`
	ReferenceID     string    `json:"referenceId,omitempty"`
	LedgerID        int64     `json:"ledgerId,omitempty"`
	UnlockedAt      time.Time `json:"unlockedAt"`
}

type RewardTopSupporterAwardResult struct {
	Rank         int    `json:"rank"`
	UserID       string `json:"userId,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	TotalSent    int    `json:"totalSent,omitempty"`
	Credits      int    `json:"credits"`
	BalanceAfter int    `json:"balanceAfter,omitempty"`
	Awarded      bool   `json:"awarded"`
	Reason       string `json:"reason,omitempty"`
}

type RewardWheelData struct {
	Settings        RewardSettings         `json:"settings"`
	Balance         RewardSpinBalance      `json:"balance"`
	Prizes          []RewardPrize          `json:"prizes"`
	Tasks           []RewardTask           `json:"tasks,omitempty"`
	TaskCompletions []RewardTaskCompletion `json:"taskCompletions,omitempty"`
}

type RewardSpinResult struct {
	Spin        RewardSpin    `json:"spin"`
	Prize       RewardPrize   `json:"prize"`
	Balance     int           `json:"balance"`
	Winner      *RewardWinner `json:"winner,omitempty"`
	Claim       *RewardClaim  `json:"claim,omitempty"`
	Order       *RewardOrder  `json:"order,omitempty"`
	OverlaySent bool          `json:"overlaySent"`
	Message     string        `json:"message"`
	CreatedAt   time.Time     `json:"createdAt"`
}

type RewardAdminSummary struct {
	Settings           RewardSettings            `json:"settings"`
	Prizes             []RewardPrize             `json:"prizes"`
	Balances           []RewardSpinBalance       `json:"balances"`
	Ledger             []RewardSpinLedgerEntry   `json:"ledger"`
	Spins              []RewardSpin              `json:"spins"`
	Winners            []RewardWinner            `json:"winners"`
	Claims             []RewardClaim             `json:"claims"`
	Orders             []RewardOrder             `json:"orders"`
	AdminMessages      []RewardAdminMessage      `json:"adminMessages"`
	Notifications      []RewardUserNotification  `json:"notifications"`
	Tasks              []RewardTask              `json:"tasks"`
	TaskCompletions    []RewardTaskCompletion    `json:"taskCompletions"`
	Achievements       []RewardAchievement       `json:"achievements"`
	AchievementUnlocks []RewardAchievementUnlock `json:"achievementUnlocks"`
	UnreadCount        int                       `json:"unreadCount"`
}
