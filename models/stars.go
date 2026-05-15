package models

import "time"

const (
	StarTransactionPayPalPurchasePending   = "paypal_purchase_pending"
	StarTransactionPayPalPurchaseCompleted = "paypal_purchase_completed"
	StarTransactionPayPalPurchaseFailed    = "paypal_purchase_failed"
	StarTransactionPayPalRefund            = "paypal_refund"
	StarTransactionStarsSent               = "stars_sent"
	StarTransactionAdminAdjustment         = "admin_adjustment"
)

// StarSettings contains site-owner Stars configuration. PayPalSecret is admin-only.
type StarSettings struct {
	Enabled               bool   `json:"enabled"`
	PayPalEnvironment     string `json:"paypalEnvironment"`
	PayPalClientID        string `json:"paypalClientId"`
	PayPalClientSecret    string `json:"paypalClientSecret,omitempty"`
	PayPalWebhookID       string `json:"paypalWebhookId,omitempty"`
	Currency              string `json:"currency"`
	SupportMessage        string `json:"supportMessage"`
	MinimumSendAmount     int    `json:"minimumSendAmount"`
	MaximumSendAmount     int    `json:"maximumSendAmount"`
	SendCooldownSeconds   int    `json:"sendCooldownSeconds"`
	OverlayEffectsEnabled bool   `json:"overlayEffectsEnabled"`
	SoundEffectsEnabled   bool   `json:"soundEffectsEnabled"`
	DebugLoggingEnabled   bool   `json:"debugLoggingEnabled,omitempty"`
}

// PublicStarSettings is safe to expose to viewers.
type PublicStarSettings struct {
	Enabled               bool          `json:"enabled"`
	PayPalEnvironment     string        `json:"paypalEnvironment"`
	PayPalClientID        string        `json:"paypalClientId"`
	Currency              string        `json:"currency"`
	SupportMessage        string        `json:"supportMessage"`
	MinimumSendAmount     int           `json:"minimumSendAmount"`
	MaximumSendAmount     int           `json:"maximumSendAmount"`
	SendCooldownSeconds   int           `json:"sendCooldownSeconds"`
	OverlayEffectsEnabled bool          `json:"overlayEffectsEnabled"`
	SoundEffectsEnabled   bool          `json:"soundEffectsEnabled"`
	Packages              []StarPackage `json:"packages"`
}

// StarPackage is a purchasable bundle resolved server-side before checkout.
type StarPackage struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	StarAmount   int       `json:"starAmount"`
	PriceCents   int       `json:"priceCents"`
	Currency     string    `json:"currency"`
	Enabled      bool      `json:"enabled"`
	DisplayOrder int       `json:"displayOrder"`
	CreatedAt    time.Time `json:"createdAt,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
}

// StarWallet is a cached view over the ledger totals for a chat user.
type StarWallet struct {
	UserID            string    `json:"userId"`
	Balance           int       `json:"balance"`
	LifetimePurchased int       `json:"lifetimePurchased"`
	LifetimeSent      int       `json:"lifetimeSent"`
	CreatedAt         time.Time `json:"createdAt,omitempty"`
	UpdatedAt         time.Time `json:"updatedAt,omitempty"`
}

// StarWalletTransaction is the ledger entry for any Stars balance mutation.
type StarWalletTransaction struct {
	ID              int64     `json:"id"`
	UserID          string    `json:"userId"`
	TransactionType string    `json:"transactionType"`
	Amount          int       `json:"amount"`
	BalanceAfter    int       `json:"balanceAfter"`
	ReferenceType   string    `json:"referenceType,omitempty"`
	ReferenceID     string    `json:"referenceId,omitempty"`
	Notes           string    `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

// StarPayPalOrder stores the server-side package resolution and PayPal state.
type StarPayPalOrder struct {
	ID              int64      `json:"id"`
	UserID          string     `json:"userId"`
	PackageID       int64      `json:"packageId"`
	PayPalOrderID   string     `json:"paypalOrderId"`
	PayPalCaptureID string     `json:"paypalCaptureId,omitempty"`
	PayPalPayerID   string     `json:"paypalPayerId,omitempty"`
	StarAmount      int        `json:"starAmount"`
	AmountCents     int        `json:"amountCents"`
	Currency        string     `json:"currency"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
	RawStatus       string     `json:"rawStatus,omitempty"`
}

// StarSendEvent represents a public Stars send shown in chat and overlays.
type StarSendEvent struct {
	ID          int64     `json:"id"`
	UserID      string    `json:"userId"`
	DisplayName string    `json:"displayName"`
	Amount      int       `json:"amount"`
	Message     string    `json:"message,omitempty"`
	Effect      string    `json:"effect"`
	CreatedAt   time.Time `json:"createdAt"`
}

// StarWalletSummary combines wallet balance and recent transactions.
type StarWalletSummary struct {
	Wallet       StarWallet              `json:"wallet"`
	Transactions []StarWalletTransaction `json:"transactions"`
}

// StarAdminSummary is the admin Stars dashboard payload.
type StarAdminSummary struct {
	Settings     StarSettings            `json:"settings"`
	Packages     []StarPackage           `json:"packages"`
	Orders       []StarPayPalOrder       `json:"orders"`
	SendEvents   []StarSendEvent         `json:"sendEvents"`
	Transactions []StarWalletTransaction `json:"transactions"`
	Wallets      []StarWallet            `json:"wallets"`
}
