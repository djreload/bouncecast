package starsrepository

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
)

const (
	orderStatusPending   = "pending"
	orderStatusCompleted = "completed"
	orderStatusFailed    = "failed"
	orderStatusRefunded  = "refunded"
)

type Repository interface {
	GetSettings() (models.StarSettings, error)
	SetSettings(settings models.StarSettings) error
	ListPackages(includeDisabled bool) ([]models.StarPackage, error)
	GetPackage(packageID int64) (models.StarPackage, error)
	UpsertPackage(pkg models.StarPackage) (models.StarPackage, error)
	GetWalletSummary(userID string) (models.StarWalletSummary, error)
	CreatePayPalOrder(userID string, pkg models.StarPackage, paypalOrderID string) (models.StarPayPalOrder, error)
	CompletePayPalOrder(paypalOrderID string, captureID string, payerID string, rawStatus string) (models.StarPayPalOrder, bool, error)
	FailPayPalOrder(paypalOrderID string, rawStatus string) error
	RefundPayPalCapture(captureID string, rawStatus string) error
	SendStars(userID string, displayName string, amount int, message string, effect string) (models.StarSendEvent, error)
	GetLastSendTime(userID string) (*time.Time, error)
	AdminAdjustWallet(userID string, amount int, notes string) (models.StarWallet, error)
	RecordWebhookEvent(eventID string, eventType string, resourceID string, status string, errText string) error
	GetAdminSummary() (models.StarAdminSummary, error)
}

type SqlRepository struct {
	datastore *data.Datastore
}

var temporaryGlobalInstance Repository

func Get() Repository {
	if temporaryGlobalInstance == nil {
		temporaryGlobalInstance = New(data.GetDatastore())
	}
	return temporaryGlobalInstance
}

func New(datastore *data.Datastore) Repository {
	return &SqlRepository{datastore: datastore}
}

func defaultSettings() models.StarSettings {
	return models.StarSettings{
		Enabled:               false,
		PayPalEnvironment:     "sandbox",
		Currency:              "GBP",
		SupportMessage:        "Stars are a fun way to support this site and trigger live on-screen effects. Stars have no cash value and are not paid out to streamers.",
		MinimumSendAmount:     1,
		MaximumSendAmount:     7000,
		SendCooldownSeconds:   10,
		OverlayEffectsEnabled: true,
		SoundEffectsEnabled:   true,
	}
}

func (r *SqlRepository) GetSettings() (models.StarSettings, error) {
	settings := defaultSettings()
	rows, err := r.datastore.DB.Query("SELECT key, value FROM star_settings")
	if err != nil {
		return settings, err
	}
	defer rows.Close()

	values := map[string]string{}
	for rows.Next() {
		var key string
		var value sql.NullString
		if err := rows.Scan(&key, &value); err != nil {
			return settings, err
		}
		values[key] = value.String
	}
	if err := rows.Err(); err != nil {
		return settings, err
	}

	settings.Enabled = parseBool(values["enabled"], settings.Enabled)
	settings.PayPalEnvironment = normalizePayPalEnvironment(valueOrDefault(values["paypal_environment"], settings.PayPalEnvironment))
	settings.PayPalClientID = strings.TrimSpace(values["paypal_client_id"])
	settings.PayPalClientSecret = strings.TrimSpace(values["paypal_client_secret"])
	settings.PayPalWebhookID = strings.TrimSpace(values["paypal_webhook_id"])
	settings.Currency = normalizeCurrency(valueOrDefault(values["currency"], settings.Currency))
	settings.SupportMessage = valueOrDefault(values["support_message"], settings.SupportMessage)
	settings.MinimumSendAmount = parseInt(values["minimum_send_amount"], settings.MinimumSendAmount)
	settings.MaximumSendAmount = parseInt(values["maximum_send_amount"], settings.MaximumSendAmount)
	settings.SendCooldownSeconds = parseInt(values["send_cooldown_seconds"], settings.SendCooldownSeconds)
	settings.OverlayEffectsEnabled = parseBool(values["overlay_effects_enabled"], settings.OverlayEffectsEnabled)
	settings.SoundEffectsEnabled = parseBool(values["sound_effects_enabled"], settings.SoundEffectsEnabled)
	settings.DebugLoggingEnabled = parseBool(values["debug_logging_enabled"], false)
	settings = normalizeSettings(settings)

	return settings, nil
}

func (r *SqlRepository) SetSettings(settings models.StarSettings) error {
	settings = normalizeSettings(settings)
	values := map[string]string{
		"enabled":                 strconv.FormatBool(settings.Enabled),
		"paypal_environment":      settings.PayPalEnvironment,
		"paypal_client_id":        strings.TrimSpace(settings.PayPalClientID),
		"paypal_client_secret":    strings.TrimSpace(settings.PayPalClientSecret),
		"paypal_webhook_id":       strings.TrimSpace(settings.PayPalWebhookID),
		"currency":                settings.Currency,
		"support_message":         settings.SupportMessage,
		"minimum_send_amount":     strconv.Itoa(settings.MinimumSendAmount),
		"maximum_send_amount":     strconv.Itoa(settings.MaximumSendAmount),
		"send_cooldown_seconds":   strconv.Itoa(settings.SendCooldownSeconds),
		"overlay_effects_enabled": strconv.FormatBool(settings.OverlayEffectsEnabled),
		"sound_effects_enabled":   strconv.FormatBool(settings.SoundEffectsEnabled),
		"debug_logging_enabled":   strconv.FormatBool(settings.DebugLoggingEnabled),
	}

	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint

	for key, value := range values {
		if _, err := tx.Exec(
			"INSERT INTO star_settings(key, value, updated_at) VALUES(?, ?, CURRENT_TIMESTAMP) ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=CURRENT_TIMESTAMP",
			key,
			value,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SqlRepository) ListPackages(includeDisabled bool) ([]models.StarPackage, error) {
	query := "SELECT id, name, star_amount, price_cents, currency, enabled, display_order, created_at, updated_at FROM star_packages"
	args := []interface{}{}
	if !includeDisabled {
		query += " WHERE enabled = ?"
		args = append(args, 1)
	}
	query += " ORDER BY display_order ASC, price_cents ASC"

	rows, err := r.datastore.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	packages := []models.StarPackage{}
	for rows.Next() {
		pkg, err := scanPackage(rows)
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}
	return packages, rows.Err()
}

func (r *SqlRepository) GetPackage(packageID int64) (models.StarPackage, error) {
	row := r.datastore.DB.QueryRow("SELECT id, name, star_amount, price_cents, currency, enabled, display_order, created_at, updated_at FROM star_packages WHERE id = ?", packageID)
	return scanPackage(row)
}

func (r *SqlRepository) UpsertPackage(pkg models.StarPackage) (models.StarPackage, error) {
	if err := validatePackage(pkg); err != nil {
		return models.StarPackage{}, err
	}
	pkg.Currency = normalizeCurrency(pkg.Currency)

	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	if pkg.ID > 0 {
		_, err := r.datastore.DB.Exec(
			"UPDATE star_packages SET name=?, star_amount=?, price_cents=?, currency=?, enabled=?, display_order=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
			strings.TrimSpace(pkg.Name),
			pkg.StarAmount,
			pkg.PriceCents,
			pkg.Currency,
			boolToInt(pkg.Enabled),
			pkg.DisplayOrder,
			pkg.ID,
		)
		if err != nil {
			return models.StarPackage{}, err
		}
		return r.GetPackage(pkg.ID)
	}

	result, err := r.datastore.DB.Exec(
		"INSERT INTO star_packages(name, star_amount, price_cents, currency, enabled, display_order) VALUES(?, ?, ?, ?, ?, ?)",
		strings.TrimSpace(pkg.Name),
		pkg.StarAmount,
		pkg.PriceCents,
		pkg.Currency,
		boolToInt(pkg.Enabled),
		pkg.DisplayOrder,
	)
	if err != nil {
		return models.StarPackage{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.StarPackage{}, err
	}
	return r.GetPackage(id)
}

func (r *SqlRepository) GetWalletSummary(userID string) (models.StarWalletSummary, error) {
	wallet, err := r.getOrCreateWallet(userID)
	if err != nil {
		return models.StarWalletSummary{}, err
	}

	rows, err := r.datastore.DB.Query(
		"SELECT id, user_id, transaction_type, amount, balance_after, reference_type, reference_id, notes, created_at FROM star_wallet_transactions WHERE user_id = ? ORDER BY created_at DESC, id DESC LIMIT 50",
		userID,
	)
	if err != nil {
		return models.StarWalletSummary{}, err
	}
	defer rows.Close()

	transactions := []models.StarWalletTransaction{}
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return models.StarWalletSummary{}, err
		}
		transactions = append(transactions, transaction)
	}

	return models.StarWalletSummary{Wallet: wallet, Transactions: transactions}, rows.Err()
}

func (r *SqlRepository) CreatePayPalOrder(userID string, pkg models.StarPackage, paypalOrderID string) (models.StarPayPalOrder, error) {
	if paypalOrderID == "" {
		return models.StarPayPalOrder{}, errors.New("missing PayPal order ID")
	}
	if err := validatePackage(pkg); err != nil {
		return models.StarPayPalOrder{}, err
	}

	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return models.StarPayPalOrder{}, err
	}
	defer tx.Rollback() // nolint

	if _, err := ensureWalletTx(tx, userID); err != nil {
		return models.StarPayPalOrder{}, err
	}

	if _, err = tx.Exec(
		"INSERT INTO star_paypal_orders(user_id, package_id, paypal_order_id, star_amount, amount_cents, currency, status, raw_status) VALUES(?, ?, ?, ?, ?, ?, ?, ?)",
		userID,
		pkg.ID,
		paypalOrderID,
		pkg.StarAmount,
		pkg.PriceCents,
		pkg.Currency,
		orderStatusPending,
		orderStatusPending,
	); err != nil {
		return models.StarPayPalOrder{}, err
	}

	wallet, err := getWalletTx(tx, userID)
	if err != nil {
		return models.StarPayPalOrder{}, err
	}
	if _, err = insertLedgerTx(tx, userID, models.StarTransactionPayPalPurchasePending, 0, wallet.Balance, "paypal_order", paypalOrderID, "PayPal order created"); err != nil {
		return models.StarPayPalOrder{}, err
	}

	if err = tx.Commit(); err != nil {
		return models.StarPayPalOrder{}, err
	}

	return r.getOrderByPayPalID(paypalOrderID)
}

func (r *SqlRepository) CompletePayPalOrder(paypalOrderID string, captureID string, payerID string, rawStatus string) (models.StarPayPalOrder, bool, error) {
	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return models.StarPayPalOrder{}, false, err
	}
	defer tx.Rollback() // nolint

	order, err := getOrderByPayPalIDTx(tx, paypalOrderID)
	if err != nil {
		return models.StarPayPalOrder{}, false, err
	}
	if order.Status == orderStatusCompleted {
		return order, false, tx.Commit()
	}
	if order.Status == orderStatusRefunded {
		return order, false, errors.New("cannot complete refunded order")
	}

	wallet, err := ensureWalletTx(tx, order.UserID)
	if err != nil {
		return models.StarPayPalOrder{}, false, err
	}
	nextBalance := wallet.Balance + order.StarAmount
	if _, err = tx.Exec(
		"UPDATE star_wallets SET balance=?, lifetime_purchased=lifetime_purchased+?, updated_at=CURRENT_TIMESTAMP WHERE user_id=?",
		nextBalance,
		order.StarAmount,
		order.UserID,
	); err != nil {
		return models.StarPayPalOrder{}, false, err
	}
	if _, err = insertLedgerTx(tx, order.UserID, models.StarTransactionPayPalPurchaseCompleted, order.StarAmount, nextBalance, "paypal_order", paypalOrderID, "PayPal payment completed"); err != nil {
		return models.StarPayPalOrder{}, false, err
	}
	if _, err = tx.Exec(
		"UPDATE star_paypal_orders SET paypal_capture_id=?, paypal_payer_id=?, status=?, raw_status=?, completed_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE paypal_order_id=?",
		nullableString(captureID),
		nullableString(payerID),
		orderStatusCompleted,
		rawStatus,
		paypalOrderID,
	); err != nil {
		return models.StarPayPalOrder{}, false, err
	}

	if err = tx.Commit(); err != nil {
		return models.StarPayPalOrder{}, false, err
	}
	completedOrder, err := r.getOrderByPayPalID(paypalOrderID)
	return completedOrder, true, err
}

func (r *SqlRepository) FailPayPalOrder(paypalOrderID string, rawStatus string) error {
	_, err := r.datastore.DB.Exec(
		"UPDATE star_paypal_orders SET status=?, raw_status=?, updated_at=CURRENT_TIMESTAMP WHERE paypal_order_id=? AND status != ?",
		orderStatusFailed,
		rawStatus,
		paypalOrderID,
		orderStatusCompleted,
	)
	return err
}

func (r *SqlRepository) RefundPayPalCapture(captureID string, rawStatus string) error {
	if captureID == "" {
		return errors.New("missing PayPal capture ID")
	}

	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint

	order, err := getOrderByCaptureIDTx(tx, captureID)
	if err != nil {
		return err
	}
	if order.Status == orderStatusRefunded {
		return tx.Commit()
	}

	wallet, err := ensureWalletTx(tx, order.UserID)
	if err != nil {
		return err
	}
	refundAmount := order.StarAmount
	if wallet.Balance < refundAmount {
		refundAmount = wallet.Balance
	}
	nextBalance := wallet.Balance - refundAmount
	if _, err = tx.Exec(
		"UPDATE star_wallets SET balance=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=?",
		nextBalance,
		order.UserID,
	); err != nil {
		return err
	}
	if _, err = insertLedgerTx(tx, order.UserID, models.StarTransactionPayPalRefund, -refundAmount, nextBalance, "paypal_capture", captureID, "PayPal refund or reversal"); err != nil {
		return err
	}
	if _, err = tx.Exec(
		"UPDATE star_paypal_orders SET status=?, raw_status=?, updated_at=CURRENT_TIMESTAMP WHERE paypal_capture_id=?",
		orderStatusRefunded,
		rawStatus,
		captureID,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SqlRepository) SendStars(userID string, displayName string, amount int, message string, effect string) (models.StarSendEvent, error) {
	if amount <= 0 {
		return models.StarSendEvent{}, errors.New("amount must be positive")
	}
	if effect == "" {
		effect = "sparkle"
	}
	createdAt := time.Now().UTC()

	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return models.StarSendEvent{}, err
	}
	defer tx.Rollback() // nolint

	wallet, err := ensureWalletTx(tx, userID)
	if err != nil {
		return models.StarSendEvent{}, err
	}
	if wallet.Balance < amount {
		return models.StarSendEvent{}, errors.New("insufficient star balance")
	}
	nextBalance := wallet.Balance - amount

	result, err := tx.Exec(
		"INSERT INTO star_send_events(user_id, display_name, amount, message, effect, created_at) VALUES(?, ?, ?, ?, ?, ?)",
		userID,
		displayName,
		amount,
		nullableString(message),
		effect,
		createdAt,
	)
	if err != nil {
		return models.StarSendEvent{}, err
	}
	sendID, err := result.LastInsertId()
	if err != nil {
		return models.StarSendEvent{}, err
	}
	if _, err = tx.Exec(
		"UPDATE star_wallets SET balance=?, lifetime_sent=lifetime_sent+?, updated_at=CURRENT_TIMESTAMP WHERE user_id=?",
		nextBalance,
		amount,
		userID,
	); err != nil {
		return models.StarSendEvent{}, err
	}
	if _, err = insertLedgerTx(tx, userID, models.StarTransactionStarsSent, -amount, nextBalance, "star_send", strconv.FormatInt(sendID, 10), message); err != nil {
		return models.StarSendEvent{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.StarSendEvent{}, err
	}

	return models.StarSendEvent{
		ID:          sendID,
		UserID:      userID,
		DisplayName: displayName,
		Amount:      amount,
		Message:     message,
		Effect:      effect,
		CreatedAt:   createdAt,
	}, nil
}

func (r *SqlRepository) GetLastSendTime(userID string) (*time.Time, error) {
	var createdAt time.Time
	err := r.datastore.DB.QueryRow("SELECT created_at FROM star_send_events WHERE user_id = ? ORDER BY created_at DESC LIMIT 1", userID).Scan(&createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &createdAt, nil
}

func (r *SqlRepository) AdminAdjustWallet(userID string, amount int, notes string) (models.StarWallet, error) {
	updatedAt := time.Now().UTC()

	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return models.StarWallet{}, err
	}
	defer tx.Rollback() // nolint

	wallet, err := ensureWalletTx(tx, userID)
	if err != nil {
		return models.StarWallet{}, err
	}
	nextBalance := wallet.Balance + amount
	if nextBalance < 0 {
		return models.StarWallet{}, errors.New("wallet balance cannot be negative")
	}
	lifetimePurchased := wallet.LifetimePurchased
	if amount > 0 {
		lifetimePurchased += amount
	}
	if _, err := tx.Exec(
		"UPDATE star_wallets SET balance=?, lifetime_purchased=?, updated_at=? WHERE user_id=?",
		nextBalance,
		lifetimePurchased,
		updatedAt,
		userID,
	); err != nil {
		return models.StarWallet{}, err
	}
	if _, err = insertLedgerTx(tx, userID, models.StarTransactionAdminAdjustment, amount, nextBalance, "admin", "", notes); err != nil {
		return models.StarWallet{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.StarWallet{}, err
	}

	return models.StarWallet{
		UserID:            userID,
		Balance:           nextBalance,
		LifetimePurchased: lifetimePurchased,
		LifetimeSent:      wallet.LifetimeSent,
		CreatedAt:         wallet.CreatedAt,
		UpdatedAt:         updatedAt,
	}, nil
}

func (r *SqlRepository) RecordWebhookEvent(eventID string, eventType string, resourceID string, status string, errText string) error {
	if eventID == "" {
		return nil
	}
	_, err := r.datastore.DB.Exec(
		"INSERT INTO star_paypal_webhook_events(paypal_event_id, event_type, resource_id, status, error, processed_at) VALUES(?, ?, ?, ?, ?, CURRENT_TIMESTAMP) ON CONFLICT(paypal_event_id) DO UPDATE SET status=excluded.status, error=excluded.error, processed_at=CURRENT_TIMESTAMP",
		eventID,
		eventType,
		nullableString(resourceID),
		status,
		nullableString(errText),
	)
	return err
}

func (r *SqlRepository) GetAdminSummary() (models.StarAdminSummary, error) {
	settings, err := r.GetSettings()
	if err != nil {
		return models.StarAdminSummary{}, err
	}
	packages, err := r.ListPackages(true)
	if err != nil {
		return models.StarAdminSummary{}, err
	}
	orders, err := r.listOrders()
	if err != nil {
		return models.StarAdminSummary{}, err
	}
	sends, err := r.listSendEvents()
	if err != nil {
		return models.StarAdminSummary{}, err
	}
	transactions, err := r.listTransactions()
	if err != nil {
		return models.StarAdminSummary{}, err
	}
	wallets, err := r.listWallets()
	if err != nil {
		return models.StarAdminSummary{}, err
	}

	return models.StarAdminSummary{
		Settings:     settings,
		Packages:     packages,
		Orders:       orders,
		SendEvents:   sends,
		Transactions: transactions,
		Wallets:      wallets,
	}, nil
}

func (r *SqlRepository) getOrCreateWallet(userID string) (models.StarWallet, error) {
	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return models.StarWallet{}, err
	}
	defer tx.Rollback() // nolint

	wallet, err := ensureWalletTx(tx, userID)
	if err != nil {
		return models.StarWallet{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.StarWallet{}, err
	}
	return wallet, nil
}

func ensureWalletTx(tx *sql.Tx, userID string) (models.StarWallet, error) {
	if _, err := tx.Exec("INSERT INTO star_wallets(user_id) VALUES(?) ON CONFLICT(user_id) DO NOTHING", userID); err != nil {
		return models.StarWallet{}, err
	}
	return getWalletTx(tx, userID)
}

func getWalletTx(tx *sql.Tx, userID string) (models.StarWallet, error) {
	row := tx.QueryRow("SELECT user_id, balance, lifetime_purchased, lifetime_sent, created_at, updated_at FROM star_wallets WHERE user_id = ?", userID)
	return scanWallet(row)
}

func insertLedgerTx(tx *sql.Tx, userID string, transactionType string, amount int, balanceAfter int, referenceType string, referenceID string, notes string) (int64, error) {
	if balanceAfter < 0 {
		return 0, errors.New("wallet balance cannot be negative")
	}
	result, err := tx.Exec(
		"INSERT INTO star_wallet_transactions(user_id, transaction_type, amount, balance_after, reference_type, reference_id, notes) VALUES(?, ?, ?, ?, ?, ?, ?)",
		userID,
		transactionType,
		amount,
		balanceAfter,
		nullableString(referenceType),
		nullableString(referenceID),
		nullableString(notes),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *SqlRepository) getOrderByPayPalID(paypalOrderID string) (models.StarPayPalOrder, error) {
	row := r.datastore.DB.QueryRow("SELECT id, user_id, package_id, paypal_order_id, paypal_capture_id, paypal_payer_id, star_amount, amount_cents, currency, status, created_at, updated_at, completed_at, raw_status FROM star_paypal_orders WHERE paypal_order_id = ?", paypalOrderID)
	return scanOrder(row)
}

func getOrderByPayPalIDTx(tx *sql.Tx, paypalOrderID string) (models.StarPayPalOrder, error) {
	row := tx.QueryRow("SELECT id, user_id, package_id, paypal_order_id, paypal_capture_id, paypal_payer_id, star_amount, amount_cents, currency, status, created_at, updated_at, completed_at, raw_status FROM star_paypal_orders WHERE paypal_order_id = ?", paypalOrderID)
	return scanOrder(row)
}

func getOrderByCaptureIDTx(tx *sql.Tx, captureID string) (models.StarPayPalOrder, error) {
	row := tx.QueryRow("SELECT id, user_id, package_id, paypal_order_id, paypal_capture_id, paypal_payer_id, star_amount, amount_cents, currency, status, created_at, updated_at, completed_at, raw_status FROM star_paypal_orders WHERE paypal_capture_id = ?", captureID)
	return scanOrder(row)
}

func (r *SqlRepository) getSendEvent(sendID int64) (models.StarSendEvent, error) {
	row := r.datastore.DB.QueryRow("SELECT id, user_id, display_name, amount, message, effect, created_at FROM star_send_events WHERE id = ?", sendID)
	return scanSendEvent(row)
}

func (r *SqlRepository) listOrders() ([]models.StarPayPalOrder, error) {
	rows, err := r.datastore.DB.Query("SELECT id, user_id, package_id, paypal_order_id, paypal_capture_id, paypal_payer_id, star_amount, amount_cents, currency, status, created_at, updated_at, completed_at, raw_status FROM star_paypal_orders ORDER BY created_at DESC LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []models.StarPayPalOrder{}
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (r *SqlRepository) listSendEvents() ([]models.StarSendEvent, error) {
	rows, err := r.datastore.DB.Query("SELECT id, user_id, display_name, amount, message, effect, created_at FROM star_send_events ORDER BY created_at DESC LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []models.StarSendEvent{}
	for rows.Next() {
		event, err := scanSendEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *SqlRepository) listTransactions() ([]models.StarWalletTransaction, error) {
	rows, err := r.datastore.DB.Query("SELECT id, user_id, transaction_type, amount, balance_after, reference_type, reference_id, notes, created_at FROM star_wallet_transactions ORDER BY created_at DESC, id DESC LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []models.StarWalletTransaction{}
	for rows.Next() {
		transaction, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	return transactions, rows.Err()
}

func (r *SqlRepository) listWallets() ([]models.StarWallet, error) {
	rows, err := r.datastore.DB.Query("SELECT user_id, balance, lifetime_purchased, lifetime_sent, created_at, updated_at FROM star_wallets ORDER BY balance DESC, updated_at DESC LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	wallets := []models.StarWallet{}
	for rows.Next() {
		wallet, err := scanWallet(rows)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, wallet)
	}
	return wallets, rows.Err()
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanPackage(row scanner) (models.StarPackage, error) {
	var pkg models.StarPackage
	var enabled int
	if err := row.Scan(&pkg.ID, &pkg.Name, &pkg.StarAmount, &pkg.PriceCents, &pkg.Currency, &enabled, &pkg.DisplayOrder, &pkg.CreatedAt, &pkg.UpdatedAt); err != nil {
		return models.StarPackage{}, err
	}
	pkg.Enabled = enabled == 1
	return pkg, nil
}

func scanWallet(row scanner) (models.StarWallet, error) {
	var wallet models.StarWallet
	if err := row.Scan(&wallet.UserID, &wallet.Balance, &wallet.LifetimePurchased, &wallet.LifetimeSent, &wallet.CreatedAt, &wallet.UpdatedAt); err != nil {
		return models.StarWallet{}, err
	}
	return wallet, nil
}

func scanTransaction(row scanner) (models.StarWalletTransaction, error) {
	var transaction models.StarWalletTransaction
	var referenceType sql.NullString
	var referenceID sql.NullString
	var notes sql.NullString
	if err := row.Scan(&transaction.ID, &transaction.UserID, &transaction.TransactionType, &transaction.Amount, &transaction.BalanceAfter, &referenceType, &referenceID, &notes, &transaction.CreatedAt); err != nil {
		return models.StarWalletTransaction{}, err
	}
	transaction.ReferenceType = referenceType.String
	transaction.ReferenceID = referenceID.String
	transaction.Notes = notes.String
	return transaction, nil
}

func scanOrder(row scanner) (models.StarPayPalOrder, error) {
	var order models.StarPayPalOrder
	var captureID sql.NullString
	var payerID sql.NullString
	var completedAt sql.NullTime
	var rawStatus sql.NullString
	if err := row.Scan(&order.ID, &order.UserID, &order.PackageID, &order.PayPalOrderID, &captureID, &payerID, &order.StarAmount, &order.AmountCents, &order.Currency, &order.Status, &order.CreatedAt, &order.UpdatedAt, &completedAt, &rawStatus); err != nil {
		return models.StarPayPalOrder{}, err
	}
	order.PayPalCaptureID = captureID.String
	order.PayPalPayerID = payerID.String
	if completedAt.Valid {
		order.CompletedAt = &completedAt.Time
	}
	order.RawStatus = rawStatus.String
	return order, nil
}

func scanSendEvent(row scanner) (models.StarSendEvent, error) {
	var event models.StarSendEvent
	var message sql.NullString
	if err := row.Scan(&event.ID, &event.UserID, &event.DisplayName, &event.Amount, &message, &event.Effect, &event.CreatedAt); err != nil {
		return models.StarSendEvent{}, err
	}
	event.Message = message.String
	return event, nil
}

func validatePackage(pkg models.StarPackage) error {
	if strings.TrimSpace(pkg.Name) == "" {
		return errors.New("package name is required")
	}
	if pkg.StarAmount <= 0 {
		return errors.New("star amount must be positive")
	}
	if pkg.PriceCents <= 0 {
		return errors.New("price must be positive")
	}
	if normalizeCurrency(pkg.Currency) == "" {
		return errors.New("currency is required")
	}
	return nil
}

func normalizeSettings(settings models.StarSettings) models.StarSettings {
	settings.PayPalEnvironment = normalizePayPalEnvironment(settings.PayPalEnvironment)
	settings.Currency = normalizeCurrency(settings.Currency)
	if settings.Currency == "" {
		settings.Currency = "GBP"
	}
	if settings.MinimumSendAmount < 1 {
		settings.MinimumSendAmount = 1
	}
	if settings.MaximumSendAmount < settings.MinimumSendAmount {
		settings.MaximumSendAmount = settings.MinimumSendAmount
	}
	if settings.SendCooldownSeconds < 0 {
		settings.SendCooldownSeconds = 0
	}
	settings.PayPalClientID = strings.TrimSpace(settings.PayPalClientID)
	settings.PayPalClientSecret = strings.TrimSpace(settings.PayPalClientSecret)
	settings.PayPalWebhookID = strings.TrimSpace(settings.PayPalWebhookID)
	if strings.TrimSpace(settings.SupportMessage) == "" {
		settings.SupportMessage = defaultSettings().SupportMessage
	}
	return settings
}

func normalizePayPalEnvironment(value string) string {
	if strings.EqualFold(value, "live") {
		return "live"
	}
	return "sandbox"
}

func normalizeCurrency(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return ""
	}
	return value
}

func valueOrDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func parseBool(value string, fallback bool) bool {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullableString(value string) interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func centsToDecimal(cents int) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
