package rewardsrepository

import (
	crand "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
)

type Repository interface {
	GetSettings() (models.RewardSettings, error)
	SetSettings(settings models.RewardSettings) error
	ListPrizes(includeInactive bool) ([]models.RewardPrize, error)
	UpsertPrize(prize models.RewardPrize) (models.RewardPrize, error)
	AwardSpinCredits(userID string, amount int, source string, referenceID string, note string) (models.RewardSpinBalance, error)
	AdminAdjustSpinCredits(userID string, amount int, note string) (models.RewardSpinBalance, error)
	GetBalance(userID string) (models.RewardSpinBalance, error)
	GetWheelData(userID string) (models.RewardWheelData, error)
	RecordChatActivity(userID string, messageBody string, referenceID string) (models.RewardSpinBalance, bool, error)
	CompleteTask(userID string, taskID int64) (models.RewardTaskCompletion, models.RewardSpinBalance, bool, error)
	SpinWheel(user models.User) (models.RewardSpinResult, error)
	ListUserSpins(userID string) ([]models.RewardSpin, error)
	ListUserClaims(userID string) ([]models.RewardClaim, error)
	SubmitClaim(userID string, submission models.RewardClaimSubmission, ipAddress string, userAgent string) (models.RewardClaim, error)
	ListUserNotifications(userID string) ([]models.RewardUserNotification, error)
	MarkUserNotificationRead(userID string, notificationID int64) error
	GetAdminSummary() (models.RewardAdminSummary, error)
	UpdateOrder(adminUserID string, order models.RewardOrder) (models.RewardOrder, error)
	MarkOrderDispatched(adminUserID string, orderID int64, courier string, trackingReference string, trackingURL string, dispatchNote string) (models.RewardOrder, models.RewardUserNotification, error)
	MarkAdminMessageRead(messageID int64) error
	UpsertTask(task models.RewardTask) (models.RewardTask, error)
	UpsertAchievement(achievement models.RewardAchievement) (models.RewardAchievement, error)
}

const repeatedChatMessageWindow = 2 * time.Minute

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

func defaultSettings() models.RewardSettings {
	return models.RewardSettings{
		Enabled:                    false,
		SpinCost:                   1,
		ChatValidMessageCount:      10,
		ChatCooldownSeconds:        300,
		ChatCreditReward:           1,
		TopSupporterFirstCredits:   25,
		TopSupporterSecondCredits:  15,
		TopSupporterThirdCredits:   10,
		OverlayEnabled:             true,
		OverlayDurationSeconds:     5,
		OverlaySoundEnabled:        true,
		OverlayShowImage:           true,
		OverlayTemplate:            "{viewer} just won {prize} on the Rewards Wheel!",
		FuturePaidRewardsNote:      "Paid spins and checkout are intentionally not implemented yet.",
		FutureDiscountRewardsNote:  "Discount prizes are placeholders only until a future fulfilment flow is approved.",
		FulfilmentPrivacyStatement: "Delivery details are collected only to fulfil prizes. Marketing consent is optional and never assumed.",
	}
}

func (r *SqlRepository) GetSettings() (models.RewardSettings, error) {
	settings := defaultSettings()
	rows, err := r.datastore.DB.Query("SELECT key, value FROM reward_config")
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
	settings.SpinCost = parseInt(values["spin_cost"], settings.SpinCost)
	settings.ChatRewardsEnabled = parseBool(values["chat_rewards_enabled"], settings.ChatRewardsEnabled)
	settings.ChatValidMessageCount = parseInt(values["chat_valid_message_count"], settings.ChatValidMessageCount)
	settings.ChatCooldownSeconds = parseInt(values["chat_cooldown_seconds"], settings.ChatCooldownSeconds)
	settings.ChatCreditReward = parseInt(values["chat_credit_reward"], settings.ChatCreditReward)
	settings.TopSupporterFirstCredits = parseInt(values["top_supporter_first_credits"], settings.TopSupporterFirstCredits)
	settings.TopSupporterSecondCredits = parseInt(values["top_supporter_second_credits"], settings.TopSupporterSecondCredits)
	settings.TopSupporterThirdCredits = parseInt(values["top_supporter_third_credits"], settings.TopSupporterThirdCredits)
	settings.OverlayEnabled = parseBool(values["overlay_enabled"], settings.OverlayEnabled)
	settings.OverlayDurationSeconds = parseInt(values["overlay_duration_seconds"], settings.OverlayDurationSeconds)
	settings.OverlaySoundEnabled = parseBool(values["overlay_sound_enabled"], settings.OverlaySoundEnabled)
	settings.OverlayShowImage = parseBool(values["overlay_show_image"], settings.OverlayShowImage)
	settings.OverlayTemplate = valueOrDefault(values["overlay_template"], settings.OverlayTemplate)
	settings.DebugLoggingEnabled = parseBool(values["debug_logging_enabled"], settings.DebugLoggingEnabled)
	return normalizeSettings(settings), nil
}

func (r *SqlRepository) SetSettings(settings models.RewardSettings) error {
	settings = normalizeSettings(settings)
	values := map[string]string{
		"enabled":                      strconv.FormatBool(settings.Enabled),
		"spin_cost":                    strconv.Itoa(settings.SpinCost),
		"chat_rewards_enabled":         strconv.FormatBool(settings.ChatRewardsEnabled),
		"chat_valid_message_count":     strconv.Itoa(settings.ChatValidMessageCount),
		"chat_cooldown_seconds":        strconv.Itoa(settings.ChatCooldownSeconds),
		"chat_credit_reward":           strconv.Itoa(settings.ChatCreditReward),
		"top_supporter_first_credits":  strconv.Itoa(settings.TopSupporterFirstCredits),
		"top_supporter_second_credits": strconv.Itoa(settings.TopSupporterSecondCredits),
		"top_supporter_third_credits":  strconv.Itoa(settings.TopSupporterThirdCredits),
		"overlay_enabled":              strconv.FormatBool(settings.OverlayEnabled),
		"overlay_duration_seconds":     strconv.Itoa(settings.OverlayDurationSeconds),
		"overlay_sound_enabled":        strconv.FormatBool(settings.OverlaySoundEnabled),
		"overlay_show_image":           strconv.FormatBool(settings.OverlayShowImage),
		"overlay_template":             settings.OverlayTemplate,
		"debug_logging_enabled":        strconv.FormatBool(settings.DebugLoggingEnabled),
	}

	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint

	for key, value := range values {
		if _, err := tx.Exec(`INSERT INTO reward_config(key, value, updated_at) VALUES(?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=CURRENT_TIMESTAMP`, key, value); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *SqlRepository) ListPrizes(includeInactive bool) ([]models.RewardPrize, error) {
	query := `SELECT id, name, description, image, prize_type, odds_weight, stock_quantity, active, display_order,
		claim_required, marketing_consent_required, terms, fulfilment_notes, created_at, updated_at
		FROM reward_prizes`
	if !includeInactive {
		query += ` WHERE active = 1 AND odds_weight > 0 AND (prize_type = 'sorry' OR stock_quantity IS NULL OR stock_quantity > 0)`
	}
	query += ` ORDER BY display_order ASC, id ASC`

	rows, err := r.datastore.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prizes := []models.RewardPrize{}
	for rows.Next() {
		prize, err := scanPrize(rows)
		if err != nil {
			return nil, err
		}
		prizes = append(prizes, prize)
	}
	return prizes, rows.Err()
}

func (r *SqlRepository) UpsertPrize(prize models.RewardPrize) (models.RewardPrize, error) {
	prize = normalizePrize(prize)
	if err := validatePrize(prize); err != nil {
		return models.RewardPrize{}, err
	}

	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	if prize.ID > 0 {
		if _, err := r.datastore.DB.Exec(`UPDATE reward_prizes
			SET name=?, description=?, image=?, prize_type=?, odds_weight=?, stock_quantity=?, active=?, display_order=?,
				claim_required=?, marketing_consent_required=?, terms=?, fulfilment_notes=?, updated_at=CURRENT_TIMESTAMP
			WHERE id=?`,
			prize.Name, nullableString(prize.Description), nullableString(prize.Image), prize.PrizeType, prize.OddsWeight,
			nullableInt(prize.StockQuantity), boolInt(prize.Active), prize.DisplayOrder, boolInt(prize.ClaimRequired),
			boolInt(prize.MarketingConsentRequired), nullableString(prize.Terms), nullableString(prize.FulfilmentNotes), prize.ID); err != nil {
			return models.RewardPrize{}, err
		}
		return r.getPrize(prize.ID)
	}

	result, err := r.datastore.DB.Exec(`INSERT INTO reward_prizes(name, description, image, prize_type, odds_weight, stock_quantity,
		active, display_order, claim_required, marketing_consent_required, terms, fulfilment_notes)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		prize.Name, nullableString(prize.Description), nullableString(prize.Image), prize.PrizeType, prize.OddsWeight,
		nullableInt(prize.StockQuantity), boolInt(prize.Active), prize.DisplayOrder, boolInt(prize.ClaimRequired),
		boolInt(prize.MarketingConsentRequired), nullableString(prize.Terms), nullableString(prize.FulfilmentNotes))
	if err != nil {
		return models.RewardPrize{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.RewardPrize{}, err
	}
	return r.getPrize(id)
}

func (r *SqlRepository) AwardSpinCredits(userID string, amount int, source string, referenceID string, note string) (models.RewardSpinBalance, error) {
	if amount <= 0 {
		return models.RewardSpinBalance{}, errors.New("credit amount must be positive")
	}
	source = strings.TrimSpace(source)
	if !isAwardSource(source) {
		return models.RewardSpinBalance{}, errors.New("invalid reward source")
	}
	return r.adjustCredits(userID, amount, source, referenceID, note, true)
}

func (r *SqlRepository) AdminAdjustSpinCredits(userID string, amount int, note string) (models.RewardSpinBalance, error) {
	if amount == 0 {
		return models.RewardSpinBalance{}, errors.New("adjustment amount is required")
	}
	return r.adjustCredits(userID, amount, models.RewardCreditSourceAdminAdjustment, "", note, false)
}

func (r *SqlRepository) GetBalance(userID string) (models.RewardSpinBalance, error) {
	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return models.RewardSpinBalance{}, err
	}
	defer tx.Rollback() //nolint

	balance, err := ensureBalanceTx(tx, userID)
	if err != nil {
		return models.RewardSpinBalance{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.RewardSpinBalance{}, err
	}
	return balance, nil
}

func (r *SqlRepository) GetWheelData(userID string) (models.RewardWheelData, error) {
	settings, err := r.GetSettings()
	if err != nil {
		return models.RewardWheelData{}, err
	}
	balance, err := r.GetBalance(userID)
	if err != nil {
		return models.RewardWheelData{}, err
	}
	prizes, err := r.ListPrizes(false)
	if err != nil {
		return models.RewardWheelData{}, err
	}
	tasks, err := r.listActiveTasks()
	if err != nil {
		return models.RewardWheelData{}, err
	}
	taskCompletions, err := r.listTaskCompletionsForUser(userID)
	if err != nil {
		return models.RewardWheelData{}, err
	}
	return models.RewardWheelData{Settings: settings, Balance: balance, Prizes: prizes, Tasks: tasks, TaskCompletions: taskCompletions}, nil
}

func (r *SqlRepository) RecordChatActivity(userID string, messageBody string, referenceID string) (models.RewardSpinBalance, bool, error) {
	userID = strings.TrimSpace(userID)
	messageBody = strings.TrimSpace(messageBody)
	if userID == "" || messageBody == "" {
		return models.RewardSpinBalance{}, false, nil
	}

	settings, err := r.GetSettings()
	if err != nil {
		return models.RewardSpinBalance{}, false, err
	}
	if !settings.Enabled || !settings.ChatRewardsEnabled {
		return models.RewardSpinBalance{}, false, nil
	}
	settings = normalizeSettings(settings)

	now := time.Now().UTC()
	messageHash := hashChatRewardMessage(messageBody)
	referenceID = strings.TrimSpace(referenceID)
	if referenceID == "" {
		referenceID = fmt.Sprintf("chat:%s:%d", userID, now.UnixNano())
	} else {
		referenceID = "chat:" + referenceID
	}

	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return models.RewardSpinBalance{}, false, err
	}
	defer tx.Rollback() //nolint

	if err := ensureUserExistsTx(tx, userID); err != nil {
		return models.RewardSpinBalance{}, false, err
	}

	var messageCount int
	var lastHash sql.NullString
	var lastMessageAt sql.NullTime
	var lastAwardedAt sql.NullTime
	err = tx.QueryRow(`SELECT message_count_window, last_message_body_hash, last_message_at, last_awarded_at
		FROM reward_chat_activity WHERE user_id=?`, userID).Scan(&messageCount, &lastHash, &lastMessageAt, &lastAwardedAt)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err = tx.Exec(`INSERT INTO reward_chat_activity(user_id, message_count_window, last_message_body_hash, last_message_at, updated_at)
			VALUES(?, 0, ?, ?, CURRENT_TIMESTAMP)`, userID, messageHash, now); err != nil {
			return models.RewardSpinBalance{}, false, err
		}
		messageCount = 0
		lastHash = sql.NullString{}
		lastMessageAt = sql.NullTime{}
	} else if err != nil {
		return models.RewardSpinBalance{}, false, err
	}

	if lastHash.Valid && lastHash.String == messageHash && lastMessageAt.Valid && now.Sub(lastMessageAt.Time) <= repeatedChatMessageWindow {
		if _, err = tx.Exec(`UPDATE reward_chat_activity
			SET last_message_at=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=?`, now, userID); err != nil {
			return models.RewardSpinBalance{}, false, err
		}
		return models.RewardSpinBalance{}, false, tx.Commit()
	}

	nextCount := messageCount + 1
	onCooldown := settings.ChatCooldownSeconds > 0 && lastAwardedAt.Valid && now.Sub(lastAwardedAt.Time) < time.Duration(settings.ChatCooldownSeconds)*time.Second
	if onCooldown && nextCount > settings.ChatValidMessageCount {
		nextCount = settings.ChatValidMessageCount
	}

	if !onCooldown && nextCount >= settings.ChatValidMessageCount {
		balance, err := ensureBalanceTx(tx, userID)
		if err != nil {
			return models.RewardSpinBalance{}, false, err
		}
		nextBalance := balance.Balance + settings.ChatCreditReward
		lifetimeEarned := balance.LifetimeEarned + settings.ChatCreditReward
		if _, err := insertLedgerTx(tx, userID, settings.ChatCreditReward, nextBalance, models.RewardCreditSourceChatActivity, referenceID, "Chat activity reward"); err != nil {
			return models.RewardSpinBalance{}, false, err
		}
		if _, err = tx.Exec(`UPDATE reward_spin_balances
			SET balance=?, lifetime_earned=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=?`, nextBalance, lifetimeEarned, userID); err != nil {
			return models.RewardSpinBalance{}, false, err
		}
		if _, err = tx.Exec(`UPDATE reward_chat_activity
			SET message_count_window=0, last_message_body_hash=?, last_message_at=?, last_awarded_at=?, updated_at=CURRENT_TIMESTAMP
			WHERE user_id=?`, messageHash, now, now, userID); err != nil {
			return models.RewardSpinBalance{}, false, err
		}
		if err = tx.Commit(); err != nil {
			return models.RewardSpinBalance{}, false, err
		}
		balance.Balance = nextBalance
		balance.LifetimeEarned = lifetimeEarned
		balance.UpdatedAt = now
		return balance, true, nil
	}

	if _, err = tx.Exec(`UPDATE reward_chat_activity
		SET message_count_window=?, last_message_body_hash=?, last_message_at=?, updated_at=CURRENT_TIMESTAMP
		WHERE user_id=?`, nextCount, messageHash, now, userID); err != nil {
		return models.RewardSpinBalance{}, false, err
	}
	return models.RewardSpinBalance{}, false, tx.Commit()
}

func (r *SqlRepository) CompleteTask(userID string, taskID int64) (models.RewardTaskCompletion, models.RewardSpinBalance, bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" || taskID <= 0 {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, errors.New("task and user are required")
	}

	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, err
	}
	defer tx.Rollback() //nolint

	if err := ensureUserExistsTx(tx, userID); err != nil {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, err
	}
	task, err := getTaskTx(tx, taskID)
	if err != nil {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, err
	}
	if !task.Active {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, errors.New("reward task is not active")
	}

	completion, err := getTaskCompletionTx(tx, userID, taskID)
	if err == nil {
		balance, balanceErr := ensureBalanceTx(tx, userID)
		if balanceErr != nil {
			return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, balanceErr
		}
		return completion, balance, false, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, err
	}

	balance, err := ensureBalanceTx(tx, userID)
	if err != nil {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, err
	}
	nextBalance := balance.Balance + task.CreditReward
	lifetimeEarned := balance.LifetimeEarned + task.CreditReward
	ledgerID, err := insertLedgerTx(tx, userID, task.CreditReward, nextBalance, models.RewardCreditSourceTaskCompleted, fmt.Sprintf("task:%d:%s", task.ID, userID), "Reward task completed: "+task.Title)
	if err != nil {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, err
	}
	if _, err = tx.Exec(`UPDATE reward_spin_balances SET balance=?, lifetime_earned=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=?`, nextBalance, lifetimeEarned, userID); err != nil {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, err
	}
	result, err := tx.Exec(`INSERT INTO reward_task_completions(user_id, task_id, ledger_id, status) VALUES(?, ?, ?, 'completed')`, userID, task.ID, ledgerID)
	if err != nil {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, err
	}
	completionID, err := result.LastInsertId()
	if err != nil {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, err
	}
	completion, err = getTaskCompletionByIDTx(tx, completionID)
	if err != nil {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return models.RewardTaskCompletion{}, models.RewardSpinBalance{}, false, err
	}
	balance.Balance = nextBalance
	balance.LifetimeEarned = lifetimeEarned
	balance.UpdatedAt = time.Now().UTC()
	return completion, balance, true, nil
}

func (r *SqlRepository) SpinWheel(user models.User) (models.RewardSpinResult, error) {
	settings, err := r.GetSettings()
	if err != nil {
		return models.RewardSpinResult{}, err
	}
	if !settings.Enabled {
		return models.RewardSpinResult{}, errors.New("Rewards Wheel is not enabled")
	}
	if settings.SpinCost <= 0 {
		settings.SpinCost = 1
	}

	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return models.RewardSpinResult{}, err
	}
	defer tx.Rollback() //nolint

	if err := ensureUserExistsTx(tx, user.ID); err != nil {
		return models.RewardSpinResult{}, err
	}

	prizes, err := eligiblePrizesTx(tx)
	if err != nil {
		return models.RewardSpinResult{}, err
	}
	if len(prizes) == 0 {
		return models.RewardSpinResult{}, errors.New("no active rewards are available")
	}

	balance, err := ensureBalanceTx(tx, user.ID)
	if err != nil {
		return models.RewardSpinResult{}, err
	}
	if balance.Balance < settings.SpinCost {
		return models.RewardSpinResult{}, errors.New("not enough Spin Credits")
	}
	prize := selectWeightedPrize(prizes)
	nextBalance := balance.Balance - settings.SpinCost
	ledgerID, err := insertLedgerTx(tx, user.ID, -settings.SpinCost, nextBalance, models.RewardCreditSourceSpinSpend, "", fmt.Sprintf("Rewards Wheel spin for %s", prize.Name))
	if err != nil {
		return models.RewardSpinResult{}, err
	}
	if _, err = tx.Exec(`UPDATE reward_spin_balances SET balance=?, lifetime_spent=lifetime_spent+?, updated_at=CURRENT_TIMESTAMP WHERE user_id=?`, nextBalance, settings.SpinCost, user.ID); err != nil {
		return models.RewardSpinResult{}, err
	}

	resultType := "sorry"
	if isRealPrize(prize) {
		resultType = "win"
	} else if prize.PrizeType == models.RewardPrizeTypeDiscountFuture {
		resultType = "discount_future_placeholder"
	}

	spinResult, err := tx.Exec(`INSERT INTO reward_spins(user_id, prize_id, prize_snapshot, image_snapshot, type_snapshot, odds_snapshot, ledger_id, result_type)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, nullableInt64(prize.ID), prize.Name, nullableString(prize.Image), prize.PrizeType, prize.OddsWeight, ledgerID, resultType)
	if err != nil {
		return models.RewardSpinResult{}, err
	}
	spinID, err := spinResult.LastInsertId()
	if err != nil {
		return models.RewardSpinResult{}, err
	}

	var winner *models.RewardWinner
	var claim *models.RewardClaim
	var order *models.RewardOrder
	if isRealPrize(prize) {
		if prize.StockQuantity != nil {
			if *prize.StockQuantity <= 0 {
				return models.RewardSpinResult{}, errors.New("selected prize is out of stock")
			}
			if _, err = tx.Exec(`UPDATE reward_prizes SET stock_quantity=stock_quantity-1, updated_at=CURRENT_TIMESTAMP WHERE id=? AND stock_quantity > 0`, prize.ID); err != nil {
				return models.RewardSpinResult{}, err
			}
		}

		claimStatus := models.RewardClaimStatusPendingDetails
		orderStatus := models.RewardOrderStatusAwaitingClaimDetails
		if !prize.ClaimRequired {
			claimStatus = models.RewardClaimStatusSubmitted
			orderStatus = models.RewardOrderStatusReadyToFulfil
		}

		claimResult, err := tx.Exec(`INSERT INTO reward_claims(user_id, spin_id, prize_id, status, consent_text)
			VALUES(?, ?, ?, ?, ?)`, user.ID, spinID, nullableInt64(prize.ID), claimStatus, defaultConsentText(prize))
		if err != nil {
			return models.RewardSpinResult{}, err
		}
		claimID, err := claimResult.LastInsertId()
		if err != nil {
			return models.RewardSpinResult{}, err
		}

		orderResult, err := tx.Exec(`INSERT INTO reward_orders(winner_user_id, username_snapshot, prize_id, prize_snapshot, image_snapshot, spin_id, claim_id, order_status)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?)`, user.ID, user.DisplayName, nullableInt64(prize.ID), prize.Name, nullableString(prize.Image), spinID, claimID, orderStatus)
		if err != nil {
			return models.RewardSpinResult{}, err
		}
		orderID, err := orderResult.LastInsertId()
		if err != nil {
			return models.RewardSpinResult{}, err
		}

		winnerResult, err := tx.Exec(`INSERT INTO reward_winners(user_id, username_snapshot, email, prize_id, prize_snapshot, image_snapshot, type_snapshot, spin_id, claim_id, order_id, notification_status)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			user.ID, user.DisplayName, nullableString(user.Email), nullableInt64(prize.ID), prize.Name, nullableString(prize.Image), prize.PrizeType, spinID, claimID, orderID, "admin_message_created")
		if err != nil {
			return models.RewardSpinResult{}, err
		}
		winnerID, err := winnerResult.LastInsertId()
		if err != nil {
			return models.RewardSpinResult{}, err
		}

		if _, err = tx.Exec(`INSERT INTO reward_admin_messages(type, severity, title, body, user_id, prize_id, spin_id, claim_id, order_id)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			models.RewardAdminMessageTypeWin, "success", "Rewards Wheel win",
			fmt.Sprintf("%s won %s. Order #%d is waiting in fulfilment.", user.DisplayName, prize.Name, orderID),
			user.ID, nullableInt64(prize.ID), spinID, claimID, orderID); err != nil {
			return models.RewardSpinResult{}, err
		}
		if _, err = tx.Exec(`INSERT INTO reward_user_notifications(user_id, type, order_id, prize_id, title, message)
			VALUES(?, ?, ?, ?, ?, ?)`,
			user.ID, models.RewardUserNotificationPrizeClaimCreated, orderID, nullableInt64(prize.ID), "You won on the Rewards Wheel", fmt.Sprintf("You won %s. Please submit any required claim details so admins can fulfil it.", prize.Name)); err != nil {
			return models.RewardSpinResult{}, err
		}

		scannedWinner, err := getWinnerTx(tx, winnerID)
		if err != nil {
			return models.RewardSpinResult{}, err
		}
		scannedClaim, err := getClaimTx(tx, claimID)
		if err != nil {
			return models.RewardSpinResult{}, err
		}
		scannedOrder, err := getOrderTx(tx, orderID)
		if err != nil {
			return models.RewardSpinResult{}, err
		}
		winner = &scannedWinner
		claim = &scannedClaim
		order = &scannedOrder
	}

	spin, err := getSpinTx(tx, spinID)
	if err != nil {
		return models.RewardSpinResult{}, err
	}

	if err = tx.Commit(); err != nil {
		return models.RewardSpinResult{}, err
	}

	message := "Try again"
	if isRealPrize(prize) {
		message = fmt.Sprintf("You won %s!", prize.Name)
	} else if prize.PrizeType == models.RewardPrizeTypeDiscountFuture {
		message = "Discount prize placeholder selected. No fulfilment is attached yet."
	}

	return models.RewardSpinResult{
		Spin:      spin,
		Prize:     prize,
		Balance:   nextBalance,
		Winner:    winner,
		Claim:     claim,
		Order:     order,
		Message:   message,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (r *SqlRepository) ListUserSpins(userID string) ([]models.RewardSpin, error) {
	rows, err := r.datastore.DB.Query(`SELECT id, user_id, prize_id, prize_snapshot, image_snapshot, type_snapshot, odds_snapshot, ledger_id, result_type, created_at
		FROM reward_spins WHERE user_id=? ORDER BY created_at DESC, id DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	spins := []models.RewardSpin{}
	for rows.Next() {
		spin, err := scanSpin(rows)
		if err != nil {
			return nil, err
		}
		spins = append(spins, spin)
	}
	return spins, rows.Err()
}

func (r *SqlRepository) ListUserClaims(userID string) ([]models.RewardClaim, error) {
	rows, err := r.datastore.DB.Query(claimListQuery()+` WHERE c.user_id=? ORDER BY c.created_at DESC, c.id DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanClaims(rows)
}

func (r *SqlRepository) SubmitClaim(userID string, submission models.RewardClaimSubmission, ipAddress string, userAgent string) (models.RewardClaim, error) {
	submission = normalizeClaimSubmission(submission)
	if err := validateClaimSubmission(submission); err != nil {
		return models.RewardClaim{}, err
	}

	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return models.RewardClaim{}, err
	}
	defer tx.Rollback() //nolint

	claim, err := getClaimTx(tx, submission.ClaimID)
	if err != nil {
		return models.RewardClaim{}, err
	}
	if claim.UserID != userID {
		return models.RewardClaim{}, errors.New("claim does not belong to this user")
	}
	if claim.Status == models.RewardClaimStatusCancelled {
		return models.RewardClaim{}, errors.New("cancelled claims cannot be submitted")
	}
	if claim.MarketingConsentRequired && !submission.MarketingConsent {
		return models.RewardClaim{}, errors.New("marketing consent is required for this prize")
	}

	var consentAt interface{}
	if submission.MarketingConsent {
		consentAt = time.Now().UTC()
	}
	if _, err = tx.Exec(`UPDATE reward_claims
		SET status=?, full_name=?, address_line_1=?, address_line_2=?, town_city=?, county_state=?, postcode=?, country=?,
			email=?, phone=?, delivery_notes=?, marketing_consent=?, consent_at=?, consent_text=?, ip_address=?, user_agent=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=? AND user_id=?`,
		models.RewardClaimStatusSubmitted, submission.FullName, submission.AddressLine1, nullableString(submission.AddressLine2),
		submission.TownCity, nullableString(submission.CountyState), submission.Postcode, submission.Country,
		submission.Email, nullableString(submission.Phone), nullableString(submission.DeliveryNotes), boolInt(submission.MarketingConsent),
		consentAt, submission.ConsentText, nullableString(ipAddress), nullableString(userAgent), submission.ClaimID, userID); err != nil {
		return models.RewardClaim{}, err
	}

	addressSnapshot, _ := json.Marshal(submission)
	if _, err = tx.Exec(`UPDATE reward_orders SET order_status=?, address_snapshot=?, updated_at=CURRENT_TIMESTAMP
		WHERE claim_id=? AND order_status=?`,
		models.RewardOrderStatusReadyToFulfil, string(addressSnapshot), submission.ClaimID, models.RewardOrderStatusAwaitingClaimDetails); err != nil {
		return models.RewardClaim{}, err
	}

	claim, err = getClaimTx(tx, submission.ClaimID)
	if err != nil {
		return models.RewardClaim{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.RewardClaim{}, err
	}
	return claim, nil
}

func (r *SqlRepository) ListUserNotifications(userID string) ([]models.RewardUserNotification, error) {
	rows, err := r.datastore.DB.Query(`SELECT id, user_id, type, order_id, prize_id, title, message, email_sent, email_error, read_at, created_at
		FROM reward_user_notifications WHERE user_id=? ORDER BY created_at DESC, id DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	notifications := []models.RewardUserNotification{}
	for rows.Next() {
		notification, err := scanUserNotification(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}

func (r *SqlRepository) MarkUserNotificationRead(userID string, notificationID int64) error {
	_, err := r.datastore.DB.Exec(`UPDATE reward_user_notifications SET read_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?`, notificationID, userID)
	return err
}

func (r *SqlRepository) GetAdminSummary() (models.RewardAdminSummary, error) {
	settings, err := r.GetSettings()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	prizes, err := r.ListPrizes(true)
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	balances, err := r.listBalances()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	ledger, err := r.listLedger()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	spins, err := r.listSpins()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	winners, err := r.listWinners()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	claims, err := r.listClaims()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	orders, err := r.listOrders()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	messages, err := r.listAdminMessages()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	notifications, err := r.listNotifications()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	tasks, err := r.listTasks()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	taskCompletions, err := r.listTaskCompletions()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	achievements, err := r.listAchievements()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	achievementUnlocks, err := r.listAchievementUnlocks()
	if err != nil {
		return models.RewardAdminSummary{}, err
	}
	unread := 0
	for _, msg := range messages {
		if msg.ReadAt == nil {
			unread++
		}
	}
	return models.RewardAdminSummary{
		Settings:           settings,
		Prizes:             prizes,
		Balances:           balances,
		Ledger:             ledger,
		Spins:              spins,
		Winners:            winners,
		Claims:             claims,
		Orders:             orders,
		AdminMessages:      messages,
		Notifications:      notifications,
		Tasks:              tasks,
		TaskCompletions:    taskCompletions,
		Achievements:       achievements,
		AchievementUnlocks: achievementUnlocks,
		UnreadCount:        unread,
	}, nil
}

func (r *SqlRepository) UpdateOrder(adminUserID string, order models.RewardOrder) (models.RewardOrder, error) {
	if order.ID <= 0 {
		return models.RewardOrder{}, errors.New("order ID is required")
	}
	order.OrderStatus = normalizeOrderStatus(order.OrderStatus)
	if order.OrderStatus == "" {
		return models.RewardOrder{}, errors.New("invalid order status")
	}
	if _, err := r.datastore.DB.Exec(`UPDATE reward_orders
		SET order_status=?, admin_notes=?, courier=?, tracking_reference=?, tracking_url=?, dispatch_note=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=?`,
		order.OrderStatus, nullableString(order.AdminNotes), nullableString(order.Courier), nullableString(order.TrackingReference),
		nullableString(order.TrackingURL), nullableString(order.DispatchNote), order.ID); err != nil {
		return models.RewardOrder{}, err
	}
	return r.getOrder(order.ID)
}

func (r *SqlRepository) MarkOrderDispatched(adminUserID string, orderID int64, courier string, trackingReference string, trackingURL string, dispatchNote string) (models.RewardOrder, models.RewardUserNotification, error) {
	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return models.RewardOrder{}, models.RewardUserNotification{}, err
	}
	defer tx.Rollback() //nolint

	order, err := getOrderTx(tx, orderID)
	if err != nil {
		return models.RewardOrder{}, models.RewardUserNotification{}, err
	}
	if order.OrderStatus == models.RewardOrderStatusCancelled {
		return models.RewardOrder{}, models.RewardUserNotification{}, errors.New("cancelled orders cannot be dispatched")
	}
	now := time.Now().UTC()
	if _, err = tx.Exec(`UPDATE reward_orders
		SET order_status=?, dispatch_status='dispatched', courier=?, tracking_reference=?, tracking_url=?, dispatch_note=?,
			dispatch_date=?, dispatched_by_admin_id=?, dispatched_at=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=?`,
		models.RewardOrderStatusDispatched, nullableString(courier), nullableString(trackingReference), nullableString(trackingURL),
		nullableString(dispatchNote), now, nullableString(adminUserID), now, orderID); err != nil {
		return models.RewardOrder{}, models.RewardUserNotification{}, err
	}

	message := fmt.Sprintf("Your %s prize has been dispatched.", order.PrizeSnapshot)
	if courier != "" {
		message += " Courier: " + courier + "."
	}
	if trackingReference != "" {
		message += " Tracking: " + trackingReference + "."
	}
	if trackingURL != "" {
		message += " Track it here: " + trackingURL
	}
	if dispatchNote != "" {
		message += " " + dispatchNote
	}
	notificationResult, err := tx.Exec(`INSERT INTO reward_user_notifications(user_id, type, order_id, prize_id, title, message)
		VALUES(?, ?, ?, ?, ?, ?)`,
		order.WinnerUserID, models.RewardUserNotificationOrderDispatched, order.ID, nullableInt64(order.PrizeID), "Reward order dispatched", message)
	if err != nil {
		return models.RewardOrder{}, models.RewardUserNotification{}, err
	}
	notificationID, err := notificationResult.LastInsertId()
	if err != nil {
		return models.RewardOrder{}, models.RewardUserNotification{}, err
	}

	order, err = getOrderTx(tx, orderID)
	if err != nil {
		return models.RewardOrder{}, models.RewardUserNotification{}, err
	}
	notification, err := getNotificationTx(tx, notificationID)
	if err != nil {
		return models.RewardOrder{}, models.RewardUserNotification{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.RewardOrder{}, models.RewardUserNotification{}, err
	}
	return order, notification, nil
}

func (r *SqlRepository) MarkAdminMessageRead(messageID int64) error {
	_, err := r.datastore.DB.Exec(`UPDATE reward_admin_messages SET read_at=CURRENT_TIMESTAMP WHERE id=?`, messageID)
	return err
}

func (r *SqlRepository) UpsertTask(task models.RewardTask) (models.RewardTask, error) {
	task.Title = strings.TrimSpace(task.Title)
	task.Description = strings.TrimSpace(task.Description)
	if task.Title == "" {
		return models.RewardTask{}, errors.New("task title is required")
	}
	if task.CreditReward <= 0 {
		task.CreditReward = 1
	}
	if task.ID > 0 {
		if _, err := r.datastore.DB.Exec(`UPDATE reward_tasks SET title=?, description=?, credit_reward=?, active=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			task.Title, nullableString(task.Description), task.CreditReward, boolInt(task.Active), task.ID); err != nil {
			return models.RewardTask{}, err
		}
		return r.getTask(task.ID)
	}
	result, err := r.datastore.DB.Exec(`INSERT INTO reward_tasks(title, description, credit_reward, active) VALUES(?, ?, ?, ?)`,
		task.Title, nullableString(task.Description), task.CreditReward, boolInt(task.Active))
	if err != nil {
		return models.RewardTask{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.RewardTask{}, err
	}
	return r.getTask(id)
}

func (r *SqlRepository) UpsertAchievement(achievement models.RewardAchievement) (models.RewardAchievement, error) {
	achievement.Name = strings.TrimSpace(achievement.Name)
	achievement.ConditionKey = strings.TrimSpace(achievement.ConditionKey)
	if achievement.Name == "" || achievement.ConditionKey == "" {
		return models.RewardAchievement{}, errors.New("achievement name and condition are required")
	}
	if achievement.RewardAmount <= 0 {
		achievement.RewardAmount = 1
	}
	if achievement.ID > 0 {
		if _, err := r.datastore.DB.Exec(`UPDATE reward_achievements SET name=?, condition_key=?, reward_amount=?, active=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			achievement.Name, achievement.ConditionKey, achievement.RewardAmount, boolInt(achievement.Active), achievement.ID); err != nil {
			return models.RewardAchievement{}, err
		}
		return r.getAchievement(achievement.ID)
	}
	result, err := r.datastore.DB.Exec(`INSERT INTO reward_achievements(name, condition_key, reward_amount, active) VALUES(?, ?, ?, ?)`,
		achievement.Name, achievement.ConditionKey, achievement.RewardAmount, boolInt(achievement.Active))
	if err != nil {
		return models.RewardAchievement{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.RewardAchievement{}, err
	}
	return r.getAchievement(id)
}

func (r *SqlRepository) adjustCredits(userID string, amount int, source string, referenceID string, note string, idempotent bool) (models.RewardSpinBalance, error) {
	r.datastore.DbLock.Lock()
	defer r.datastore.DbLock.Unlock()

	tx, err := r.datastore.DB.Begin()
	if err != nil {
		return models.RewardSpinBalance{}, err
	}
	defer tx.Rollback() //nolint

	if err := ensureUserExistsTx(tx, userID); err != nil {
		return models.RewardSpinBalance{}, err
	}
	if idempotent && strings.TrimSpace(referenceID) != "" {
		var existingID int64
		err := tx.QueryRow(`SELECT id FROM reward_spin_ledger WHERE source=? AND reference_id=?`, source, referenceID).Scan(&existingID)
		if err == nil {
			balance, balanceErr := getBalanceTx(tx, userID)
			if balanceErr != nil {
				return models.RewardSpinBalance{}, balanceErr
			}
			return balance, tx.Commit()
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return models.RewardSpinBalance{}, err
		}
	}

	balance, err := ensureBalanceTx(tx, userID)
	if err != nil {
		return models.RewardSpinBalance{}, err
	}
	nextBalance := balance.Balance + amount
	if nextBalance < 0 {
		return models.RewardSpinBalance{}, errors.New("Spin Credit balance cannot be negative")
	}
	lifetimeEarned := balance.LifetimeEarned
	lifetimeSpent := balance.LifetimeSpent
	if amount > 0 {
		lifetimeEarned += amount
	} else {
		lifetimeSpent += -amount
	}
	if _, err = tx.Exec(`UPDATE reward_spin_balances SET balance=?, lifetime_earned=?, lifetime_spent=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=?`,
		nextBalance, lifetimeEarned, lifetimeSpent, userID); err != nil {
		return models.RewardSpinBalance{}, err
	}
	if _, err = insertLedgerTx(tx, userID, amount, nextBalance, source, referenceID, note); err != nil {
		return models.RewardSpinBalance{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.RewardSpinBalance{}, err
	}
	balance.Balance = nextBalance
	balance.LifetimeEarned = lifetimeEarned
	balance.LifetimeSpent = lifetimeSpent
	balance.UpdatedAt = time.Now().UTC()
	return balance, nil
}

func ensureUserExistsTx(tx *sql.Tx, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return errors.New("user ID is required")
	}
	var exists int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM users WHERE id=?`, userID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return errors.New("user not found")
	}
	return nil
}

func ensureBalanceTx(tx *sql.Tx, userID string) (models.RewardSpinBalance, error) {
	if err := ensureUserExistsTx(tx, userID); err != nil {
		return models.RewardSpinBalance{}, err
	}
	if _, err := tx.Exec(`INSERT INTO reward_spin_balances(user_id) VALUES(?) ON CONFLICT(user_id) DO NOTHING`, userID); err != nil {
		return models.RewardSpinBalance{}, err
	}
	return getBalanceTx(tx, userID)
}

func getBalanceTx(tx *sql.Tx, userID string) (models.RewardSpinBalance, error) {
	row := tx.QueryRow(`SELECT b.user_id, b.balance, b.lifetime_earned, b.lifetime_spent, b.created_at, b.updated_at,
		COALESCE(u.display_name, ''), COALESCE(u.email, ''), COALESCE(u.profile_image_url, '')
		FROM reward_spin_balances b
		LEFT JOIN users u ON u.id = b.user_id
		WHERE b.user_id=?`, userID)
	return scanBalance(row)
}

func insertLedgerTx(tx *sql.Tx, userID string, amount int, balanceAfter int, source string, referenceID string, note string) (int64, error) {
	if balanceAfter < 0 {
		return 0, errors.New("Spin Credit balance cannot be negative")
	}
	result, err := tx.Exec(`INSERT INTO reward_spin_ledger(user_id, amount, balance_after, source, reference_id, note)
		VALUES(?, ?, ?, ?, ?, ?)`, userID, amount, balanceAfter, source, nullableString(referenceID), nullableString(note))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func eligiblePrizesTx(tx *sql.Tx) ([]models.RewardPrize, error) {
	rows, err := tx.Query(`SELECT id, name, description, image, prize_type, odds_weight, stock_quantity, active, display_order,
		claim_required, marketing_consent_required, terms, fulfilment_notes, created_at, updated_at
		FROM reward_prizes
		WHERE active = 1 AND odds_weight > 0 AND (prize_type = 'sorry' OR stock_quantity IS NULL OR stock_quantity > 0)
		ORDER BY display_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	prizes := []models.RewardPrize{}
	for rows.Next() {
		prize, err := scanPrize(rows)
		if err != nil {
			return nil, err
		}
		prizes = append(prizes, prize)
	}
	return prizes, rows.Err()
}

func selectWeightedPrize(prizes []models.RewardPrize) models.RewardPrize {
	total := 0
	for _, prize := range prizes {
		if prize.OddsWeight > 0 {
			total += prize.OddsWeight
		}
	}
	if total <= 0 {
		return prizes[0]
	}
	roll := secureWeightedRoll(total)
	running := 0
	for _, prize := range prizes {
		if prize.OddsWeight <= 0 {
			continue
		}
		running += prize.OddsWeight
		if roll <= running {
			return prize
		}
	}
	return prizes[len(prizes)-1]
}

func secureWeightedRoll(total int) int {
	roll, err := crand.Int(crand.Reader, big.NewInt(int64(total)))
	if err != nil {
		return int(time.Now().UnixNano()%int64(total)) + 1
	}
	return int(roll.Int64()) + 1
}

func (r *SqlRepository) getPrize(id int64) (models.RewardPrize, error) {
	row := r.datastore.DB.QueryRow(`SELECT id, name, description, image, prize_type, odds_weight, stock_quantity, active, display_order,
		claim_required, marketing_consent_required, terms, fulfilment_notes, created_at, updated_at
		FROM reward_prizes WHERE id=?`, id)
	return scanPrize(row)
}

func getSpinTx(tx *sql.Tx, id int64) (models.RewardSpin, error) {
	row := tx.QueryRow(`SELECT id, user_id, prize_id, prize_snapshot, image_snapshot, type_snapshot, odds_snapshot, ledger_id, result_type, created_at
		FROM reward_spins WHERE id=?`, id)
	return scanSpin(row)
}

func getWinnerTx(tx *sql.Tx, id int64) (models.RewardWinner, error) {
	row := tx.QueryRow(`SELECT id, user_id, username_snapshot, email, prize_id, prize_snapshot, image_snapshot, type_snapshot, spin_id, claim_id, order_id, notification_status, created_at
		FROM reward_winners WHERE id=?`, id)
	return scanWinner(row)
}

func getClaimTx(tx *sql.Tx, id int64) (models.RewardClaim, error) {
	row := tx.QueryRow(claimListQuery()+` WHERE c.id=?`, id)
	return scanClaim(row)
}

func (r *SqlRepository) getOrder(id int64) (models.RewardOrder, error) {
	row := r.datastore.DB.QueryRow(orderSelectQuery()+` WHERE id=?`, id)
	return scanOrder(row)
}

func getOrderTx(tx *sql.Tx, id int64) (models.RewardOrder, error) {
	row := tx.QueryRow(orderSelectQuery()+` WHERE id=?`, id)
	return scanOrder(row)
}

func getNotificationTx(tx *sql.Tx, id int64) (models.RewardUserNotification, error) {
	row := tx.QueryRow(`SELECT id, user_id, type, order_id, prize_id, title, message, email_sent, email_error, read_at, created_at
		FROM reward_user_notifications WHERE id=?`, id)
	return scanUserNotification(row)
}

func (r *SqlRepository) getTask(id int64) (models.RewardTask, error) {
	row := r.datastore.DB.QueryRow(`SELECT id, title, description, credit_reward, active, created_at, updated_at FROM reward_tasks WHERE id=?`, id)
	return scanTask(row)
}

func getTaskTx(tx *sql.Tx, id int64) (models.RewardTask, error) {
	row := tx.QueryRow(`SELECT id, title, description, credit_reward, active, created_at, updated_at FROM reward_tasks WHERE id=?`, id)
	return scanTask(row)
}

func getTaskCompletionTx(tx *sql.Tx, userID string, taskID int64) (models.RewardTaskCompletion, error) {
	row := tx.QueryRow(taskCompletionSelectQuery()+` WHERE c.user_id=? AND c.task_id=?`, userID, taskID)
	return scanTaskCompletion(row)
}

func getTaskCompletionByIDTx(tx *sql.Tx, id int64) (models.RewardTaskCompletion, error) {
	row := tx.QueryRow(taskCompletionSelectQuery()+` WHERE c.id=?`, id)
	return scanTaskCompletion(row)
}

func (r *SqlRepository) getAchievement(id int64) (models.RewardAchievement, error) {
	row := r.datastore.DB.QueryRow(`SELECT id, name, condition_key, reward_amount, active, created_at, updated_at FROM reward_achievements WHERE id=?`, id)
	return scanAchievement(row)
}

func (r *SqlRepository) listBalances() ([]models.RewardSpinBalance, error) {
	rows, err := r.datastore.DB.Query(`SELECT b.user_id, b.balance, b.lifetime_earned, b.lifetime_spent, b.created_at, b.updated_at,
		COALESCE(u.display_name, ''), COALESCE(u.email, ''), COALESCE(u.profile_image_url, '')
		FROM reward_spin_balances b
		LEFT JOIN users u ON u.id = b.user_id
		ORDER BY b.balance DESC, b.updated_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []models.RewardSpinBalance{}
	for rows.Next() {
		item, err := scanBalance(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func (r *SqlRepository) listLedger() ([]models.RewardSpinLedgerEntry, error) {
	rows, err := r.datastore.DB.Query(`SELECT id, user_id, amount, balance_after, source, reference_id, note, created_at
		FROM reward_spin_ledger ORDER BY created_at DESC, id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []models.RewardSpinLedgerEntry{}
	for rows.Next() {
		item, err := scanLedger(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func (r *SqlRepository) listSpins() ([]models.RewardSpin, error) {
	rows, err := r.datastore.DB.Query(`SELECT id, user_id, prize_id, prize_snapshot, image_snapshot, type_snapshot, odds_snapshot, ledger_id, result_type, created_at
		FROM reward_spins ORDER BY created_at DESC, id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []models.RewardSpin{}
	for rows.Next() {
		item, err := scanSpin(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func (r *SqlRepository) listWinners() ([]models.RewardWinner, error) {
	rows, err := r.datastore.DB.Query(`SELECT id, user_id, username_snapshot, email, prize_id, prize_snapshot, image_snapshot, type_snapshot, spin_id, claim_id, order_id, notification_status, created_at
		FROM reward_winners ORDER BY created_at DESC, id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []models.RewardWinner{}
	for rows.Next() {
		item, err := scanWinner(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func (r *SqlRepository) listClaims() ([]models.RewardClaim, error) {
	rows, err := r.datastore.DB.Query(claimListQuery() + ` ORDER BY c.created_at DESC, c.id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanClaims(rows)
}

func (r *SqlRepository) listOrders() ([]models.RewardOrder, error) {
	rows, err := r.datastore.DB.Query(orderSelectQuery() + ` ORDER BY created_at DESC, id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []models.RewardOrder{}
	for rows.Next() {
		item, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func (r *SqlRepository) listAdminMessages() ([]models.RewardAdminMessage, error) {
	rows, err := r.datastore.DB.Query(`SELECT id, type, severity, title, body, user_id, prize_id, spin_id, claim_id, order_id, read_at, created_at
		FROM reward_admin_messages ORDER BY created_at DESC, id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []models.RewardAdminMessage{}
	for rows.Next() {
		item, err := scanAdminMessage(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func (r *SqlRepository) listNotifications() ([]models.RewardUserNotification, error) {
	rows, err := r.datastore.DB.Query(`SELECT id, user_id, type, order_id, prize_id, title, message, email_sent, email_error, read_at, created_at
		FROM reward_user_notifications ORDER BY created_at DESC, id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []models.RewardUserNotification{}
	for rows.Next() {
		item, err := scanUserNotification(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func (r *SqlRepository) listTasks() ([]models.RewardTask, error) {
	rows, err := r.datastore.DB.Query(`SELECT id, title, description, credit_reward, active, created_at, updated_at FROM reward_tasks ORDER BY created_at DESC, id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []models.RewardTask{}
	for rows.Next() {
		item, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func (r *SqlRepository) listActiveTasks() ([]models.RewardTask, error) {
	rows, err := r.datastore.DB.Query(`SELECT id, title, description, credit_reward, active, created_at, updated_at
		FROM reward_tasks WHERE active=1 ORDER BY created_at DESC, id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []models.RewardTask{}
	for rows.Next() {
		item, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func (r *SqlRepository) listTaskCompletions() ([]models.RewardTaskCompletion, error) {
	rows, err := r.datastore.DB.Query(taskCompletionSelectQuery() + ` ORDER BY c.completed_at DESC, c.id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskCompletions(rows)
}

func (r *SqlRepository) listTaskCompletionsForUser(userID string) ([]models.RewardTaskCompletion, error) {
	rows, err := r.datastore.DB.Query(taskCompletionSelectQuery()+` WHERE c.user_id=? ORDER BY c.completed_at DESC, c.id DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskCompletions(rows)
}

func (r *SqlRepository) listAchievements() ([]models.RewardAchievement, error) {
	rows, err := r.datastore.DB.Query(`SELECT id, name, condition_key, reward_amount, active, created_at, updated_at FROM reward_achievements ORDER BY created_at DESC, id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []models.RewardAchievement{}
	for rows.Next() {
		item, err := scanAchievement(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func (r *SqlRepository) listAchievementUnlocks() ([]models.RewardAchievementUnlock, error) {
	rows, err := r.datastore.DB.Query(achievementUnlockSelectQuery() + ` ORDER BY u.unlocked_at DESC, u.id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAchievementUnlocks(rows)
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanPrize(row scanner) (models.RewardPrize, error) {
	var prize models.RewardPrize
	var description sql.NullString
	var image sql.NullString
	var stock sql.NullInt64
	var active int
	var claimRequired int
	var marketingConsentRequired int
	var terms sql.NullString
	var notes sql.NullString
	if err := row.Scan(&prize.ID, &prize.Name, &description, &image, &prize.PrizeType, &prize.OddsWeight, &stock, &active, &prize.DisplayOrder,
		&claimRequired, &marketingConsentRequired, &terms, &notes, &prize.CreatedAt, &prize.UpdatedAt); err != nil {
		return models.RewardPrize{}, err
	}
	prize.Description = description.String
	prize.Image = image.String
	if stock.Valid {
		value := int(stock.Int64)
		prize.StockQuantity = &value
	}
	prize.Active = active == 1
	prize.ClaimRequired = claimRequired == 1
	prize.MarketingConsentRequired = marketingConsentRequired == 1
	prize.Terms = terms.String
	prize.FulfilmentNotes = notes.String
	return prize, nil
}

func scanBalance(row scanner) (models.RewardSpinBalance, error) {
	var balance models.RewardSpinBalance
	if err := row.Scan(&balance.UserID, &balance.Balance, &balance.LifetimeEarned, &balance.LifetimeSpent, &balance.CreatedAt, &balance.UpdatedAt,
		&balance.DisplayName, &balance.Email, &balance.ProfileImageURL); err != nil {
		return models.RewardSpinBalance{}, err
	}
	return balance, nil
}

func scanLedger(row scanner) (models.RewardSpinLedgerEntry, error) {
	var entry models.RewardSpinLedgerEntry
	var referenceID sql.NullString
	var note sql.NullString
	if err := row.Scan(&entry.ID, &entry.UserID, &entry.Amount, &entry.BalanceAfter, &entry.Source, &referenceID, &note, &entry.CreatedAt); err != nil {
		return models.RewardSpinLedgerEntry{}, err
	}
	entry.ReferenceID = referenceID.String
	entry.Note = note.String
	return entry, nil
}

func scanSpin(row scanner) (models.RewardSpin, error) {
	var spin models.RewardSpin
	var prizeID sql.NullInt64
	var image sql.NullString
	if err := row.Scan(&spin.ID, &spin.UserID, &prizeID, &spin.PrizeSnapshot, &image, &spin.TypeSnapshot, &spin.OddsSnapshot, &spin.LedgerID, &spin.ResultType, &spin.CreatedAt); err != nil {
		return models.RewardSpin{}, err
	}
	if prizeID.Valid {
		spin.PrizeID = prizeID.Int64
	}
	spin.ImageSnapshot = image.String
	return spin, nil
}

func scanWinner(row scanner) (models.RewardWinner, error) {
	var winner models.RewardWinner
	var email sql.NullString
	var prizeID sql.NullInt64
	var image sql.NullString
	var claimID sql.NullInt64
	var orderID sql.NullInt64
	if err := row.Scan(&winner.ID, &winner.UserID, &winner.UsernameSnapshot, &email, &prizeID, &winner.PrizeSnapshot, &image, &winner.TypeSnapshot,
		&winner.SpinID, &claimID, &orderID, &winner.NotificationStatus, &winner.CreatedAt); err != nil {
		return models.RewardWinner{}, err
	}
	winner.Email = email.String
	if prizeID.Valid {
		winner.PrizeID = prizeID.Int64
	}
	winner.ImageSnapshot = image.String
	if claimID.Valid {
		winner.ClaimID = claimID.Int64
	}
	if orderID.Valid {
		winner.OrderID = orderID.Int64
	}
	return winner, nil
}

func scanClaim(row scanner) (models.RewardClaim, error) {
	var claim models.RewardClaim
	var prizeID sql.NullInt64
	var fullName sql.NullString
	var line1 sql.NullString
	var line2 sql.NullString
	var town sql.NullString
	var county sql.NullString
	var postcode sql.NullString
	var country sql.NullString
	var email sql.NullString
	var phone sql.NullString
	var deliveryNotes sql.NullString
	var consentAt sql.NullTime
	var consentText sql.NullString
	var ipAddress sql.NullString
	var userAgent sql.NullString
	var prizeSnapshot sql.NullString
	var imageSnapshot sql.NullString
	var marketingConsent int
	var marketingConsentRequired int
	if err := row.Scan(&claim.ID, &claim.UserID, &claim.SpinID, &prizeID, &claim.Status, &fullName, &line1, &line2, &town, &county,
		&postcode, &country, &email, &phone, &deliveryNotes, &marketingConsent, &consentAt, &consentText, &ipAddress, &userAgent,
		&claim.CreatedAt, &claim.UpdatedAt, &prizeSnapshot, &imageSnapshot, &marketingConsentRequired); err != nil {
		return models.RewardClaim{}, err
	}
	if prizeID.Valid {
		claim.PrizeID = prizeID.Int64
	}
	claim.FullName = fullName.String
	claim.AddressLine1 = line1.String
	claim.AddressLine2 = line2.String
	claim.TownCity = town.String
	claim.CountyState = county.String
	claim.Postcode = postcode.String
	claim.Country = country.String
	claim.Email = email.String
	claim.Phone = phone.String
	claim.DeliveryNotes = deliveryNotes.String
	claim.MarketingConsent = marketingConsent == 1
	if consentAt.Valid {
		claim.ConsentAt = &consentAt.Time
	}
	claim.ConsentText = consentText.String
	claim.IPAddress = ipAddress.String
	claim.UserAgent = userAgent.String
	claim.PrizeSnapshot = prizeSnapshot.String
	claim.ImageSnapshot = imageSnapshot.String
	claim.MarketingConsentRequired = marketingConsentRequired == 1
	return claim, nil
}

func scanClaims(rows *sql.Rows) ([]models.RewardClaim, error) {
	results := []models.RewardClaim{}
	for rows.Next() {
		item, err := scanClaim(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func scanOrder(row scanner) (models.RewardOrder, error) {
	var order models.RewardOrder
	var prizeID sql.NullInt64
	var claimID sql.NullInt64
	var image sql.NullString
	var address sql.NullString
	var notes sql.NullString
	var courier sql.NullString
	var trackingRef sql.NullString
	var trackingURL sql.NullString
	var dispatchNote sql.NullString
	var dispatchDate sql.NullTime
	var deliveredDate sql.NullTime
	var dispatchedBy sql.NullString
	var dispatchedAt sql.NullTime
	if err := row.Scan(&order.ID, &order.WinnerUserID, &order.UsernameSnapshot, &prizeID, &order.PrizeSnapshot, &image,
		&order.SpinID, &claimID, &address, &order.OrderStatus, &order.DispatchStatus, &notes, &courier, &trackingRef, &trackingURL,
		&dispatchNote, &dispatchDate, &deliveredDate, &dispatchedBy, &dispatchedAt, &order.CreatedAt, &order.UpdatedAt); err != nil {
		return models.RewardOrder{}, err
	}
	if prizeID.Valid {
		order.PrizeID = prizeID.Int64
	}
	order.ImageSnapshot = image.String
	if claimID.Valid {
		order.ClaimID = claimID.Int64
	}
	order.AddressSnapshot = address.String
	order.AdminNotes = notes.String
	order.Courier = courier.String
	order.TrackingReference = trackingRef.String
	order.TrackingURL = trackingURL.String
	order.DispatchNote = dispatchNote.String
	if dispatchDate.Valid {
		order.DispatchDate = &dispatchDate.Time
	}
	if deliveredDate.Valid {
		order.DeliveredDate = &deliveredDate.Time
	}
	order.DispatchedByAdminID = dispatchedBy.String
	if dispatchedAt.Valid {
		order.DispatchedAt = &dispatchedAt.Time
	}
	return order, nil
}

func scanAdminMessage(row scanner) (models.RewardAdminMessage, error) {
	var msg models.RewardAdminMessage
	var body sql.NullString
	var userID sql.NullString
	var prizeID sql.NullInt64
	var spinID sql.NullInt64
	var claimID sql.NullInt64
	var orderID sql.NullInt64
	var readAt sql.NullTime
	if err := row.Scan(&msg.ID, &msg.Type, &msg.Severity, &msg.Title, &body, &userID, &prizeID, &spinID, &claimID, &orderID, &readAt, &msg.CreatedAt); err != nil {
		return models.RewardAdminMessage{}, err
	}
	msg.Body = body.String
	msg.UserID = userID.String
	if prizeID.Valid {
		msg.PrizeID = prizeID.Int64
	}
	if spinID.Valid {
		msg.SpinID = spinID.Int64
	}
	if claimID.Valid {
		msg.ClaimID = claimID.Int64
	}
	if orderID.Valid {
		msg.OrderID = orderID.Int64
	}
	if readAt.Valid {
		msg.ReadAt = &readAt.Time
	}
	return msg, nil
}

func scanUserNotification(row scanner) (models.RewardUserNotification, error) {
	var notification models.RewardUserNotification
	var orderID sql.NullInt64
	var prizeID sql.NullInt64
	var msg sql.NullString
	var emailError sql.NullString
	var readAt sql.NullTime
	var emailSent int
	if err := row.Scan(&notification.ID, &notification.UserID, &notification.Type, &orderID, &prizeID, &notification.Title, &msg, &emailSent, &emailError, &readAt, &notification.CreatedAt); err != nil {
		return models.RewardUserNotification{}, err
	}
	if orderID.Valid {
		notification.OrderID = orderID.Int64
	}
	if prizeID.Valid {
		notification.PrizeID = prizeID.Int64
	}
	notification.Message = msg.String
	notification.EmailSent = emailSent == 1
	notification.EmailError = emailError.String
	if readAt.Valid {
		notification.ReadAt = &readAt.Time
	}
	return notification, nil
}

func scanTask(row scanner) (models.RewardTask, error) {
	var task models.RewardTask
	var description sql.NullString
	var active int
	if err := row.Scan(&task.ID, &task.Title, &description, &task.CreditReward, &active, &task.CreatedAt, &task.UpdatedAt); err != nil {
		return models.RewardTask{}, err
	}
	task.Description = description.String
	task.Active = active == 1
	return task, nil
}

func scanTaskCompletion(row scanner) (models.RewardTaskCompletion, error) {
	var completion models.RewardTaskCompletion
	var ledgerID sql.NullInt64
	var taskTitle sql.NullString
	if err := row.Scan(&completion.ID, &completion.UserID, &completion.TaskID, &taskTitle, &ledgerID, &completion.Status, &completion.CompletedAt); err != nil {
		return models.RewardTaskCompletion{}, err
	}
	completion.TaskTitle = taskTitle.String
	if ledgerID.Valid {
		completion.LedgerID = ledgerID.Int64
	}
	return completion, nil
}

func scanTaskCompletions(rows *sql.Rows) ([]models.RewardTaskCompletion, error) {
	results := []models.RewardTaskCompletion{}
	for rows.Next() {
		item, err := scanTaskCompletion(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func scanAchievement(row scanner) (models.RewardAchievement, error) {
	var achievement models.RewardAchievement
	var active int
	if err := row.Scan(&achievement.ID, &achievement.Name, &achievement.ConditionKey, &achievement.RewardAmount, &active, &achievement.CreatedAt, &achievement.UpdatedAt); err != nil {
		return models.RewardAchievement{}, err
	}
	achievement.Active = active == 1
	return achievement, nil
}

func scanAchievementUnlock(row scanner) (models.RewardAchievementUnlock, error) {
	var unlock models.RewardAchievementUnlock
	var achievementName sql.NullString
	var referenceID sql.NullString
	var ledgerID sql.NullInt64
	if err := row.Scan(&unlock.ID, &unlock.UserID, &unlock.AchievementID, &achievementName, &unlock.ConditionKey, &referenceID, &ledgerID, &unlock.UnlockedAt); err != nil {
		return models.RewardAchievementUnlock{}, err
	}
	unlock.AchievementName = achievementName.String
	unlock.ReferenceID = referenceID.String
	if ledgerID.Valid {
		unlock.LedgerID = ledgerID.Int64
	}
	return unlock, nil
}

func scanAchievementUnlocks(rows *sql.Rows) ([]models.RewardAchievementUnlock, error) {
	results := []models.RewardAchievementUnlock{}
	for rows.Next() {
		item, err := scanAchievementUnlock(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

func claimListQuery() string {
	return `SELECT c.id, c.user_id, c.spin_id, c.prize_id, c.status, c.full_name, c.address_line_1, c.address_line_2,
		c.town_city, c.county_state, c.postcode, c.country, c.email, c.phone, c.delivery_notes, c.marketing_consent,
		c.consent_at, c.consent_text, c.ip_address, c.user_agent, c.created_at, c.updated_at,
		COALESCE(s.prize_snapshot, p.name, ''), COALESCE(s.image_snapshot, p.image, ''), COALESCE(p.marketing_consent_required, 0)
		FROM reward_claims c
		LEFT JOIN reward_spins s ON s.id = c.spin_id
		LEFT JOIN reward_prizes p ON p.id = c.prize_id`
}

func orderSelectQuery() string {
	return `SELECT id, winner_user_id, username_snapshot, prize_id, prize_snapshot, image_snapshot, spin_id, claim_id,
		address_snapshot, order_status, dispatch_status, admin_notes, courier, tracking_reference, tracking_url,
		dispatch_note, dispatch_date, delivered_date, dispatched_by_admin_id, dispatched_at, created_at, updated_at
		FROM reward_orders`
}

func taskCompletionSelectQuery() string {
	return `SELECT c.id, c.user_id, c.task_id, COALESCE(t.title, ''), c.ledger_id, c.status, c.completed_at
		FROM reward_task_completions c
		LEFT JOIN reward_tasks t ON t.id = c.task_id`
}

func achievementUnlockSelectQuery() string {
	return `SELECT u.id, u.user_id, u.achievement_id, COALESCE(a.name, ''), u.condition_key, u.reference_id, u.ledger_id, u.unlocked_at
		FROM reward_achievement_unlocks u
		LEFT JOIN reward_achievements a ON a.id = u.achievement_id`
}

func normalizeSettings(settings models.RewardSettings) models.RewardSettings {
	if settings.SpinCost <= 0 {
		settings.SpinCost = 1
	}
	if settings.ChatValidMessageCount <= 0 {
		settings.ChatValidMessageCount = 10
	}
	if settings.ChatCooldownSeconds < 0 {
		settings.ChatCooldownSeconds = 0
	}
	if settings.ChatCreditReward <= 0 {
		settings.ChatCreditReward = 1
	}
	if settings.OverlayDurationSeconds <= 0 {
		settings.OverlayDurationSeconds = 5
	}
	if strings.TrimSpace(settings.OverlayTemplate) == "" {
		settings.OverlayTemplate = defaultSettings().OverlayTemplate
	}
	settings.OverlayTemplate = strings.TrimSpace(settings.OverlayTemplate)
	settings.FuturePaidRewardsNote = defaultSettings().FuturePaidRewardsNote
	settings.FutureDiscountRewardsNote = defaultSettings().FutureDiscountRewardsNote
	settings.FulfilmentPrivacyStatement = defaultSettings().FulfilmentPrivacyStatement
	return settings
}

func normalizePrize(prize models.RewardPrize) models.RewardPrize {
	prize.Name = strings.TrimSpace(prize.Name)
	prize.Description = strings.TrimSpace(prize.Description)
	prize.Image = strings.TrimSpace(prize.Image)
	prize.PrizeType = normalizePrizeType(prize.PrizeType)
	if prize.OddsWeight < 0 {
		prize.OddsWeight = 0
	}
	prize.Terms = strings.TrimSpace(prize.Terms)
	prize.FulfilmentNotes = strings.TrimSpace(prize.FulfilmentNotes)
	if prize.PrizeType == models.RewardPrizeTypeSorry {
		prize.StockQuantity = nil
		prize.ClaimRequired = false
		prize.MarketingConsentRequired = false
	}
	return prize
}

func validatePrize(prize models.RewardPrize) error {
	if prize.Name == "" {
		return errors.New("prize name is required")
	}
	if prize.OddsWeight < 0 {
		return errors.New("odds weight cannot be negative")
	}
	if prize.StockQuantity != nil && *prize.StockQuantity < 0 {
		return errors.New("stock quantity cannot be negative")
	}
	return nil
}

func normalizePrizeType(value string) string {
	switch strings.TrimSpace(value) {
	case models.RewardPrizeTypePhysical:
		return models.RewardPrizeTypePhysical
	case models.RewardPrizeTypeDigital:
		return models.RewardPrizeTypeDigital
	case models.RewardPrizeTypeDiscountFuture:
		return models.RewardPrizeTypeDiscountFuture
	default:
		return models.RewardPrizeTypeSorry
	}
}

func normalizeOrderStatus(value string) string {
	switch strings.TrimSpace(value) {
	case models.RewardOrderStatusAwaitingClaimDetails, models.RewardOrderStatusReadyToFulfil, models.RewardOrderStatusPacked,
		models.RewardOrderStatusDispatched, models.RewardOrderStatusDelivered, models.RewardOrderStatusCancelled:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func normalizeClaimSubmission(submission models.RewardClaimSubmission) models.RewardClaimSubmission {
	submission.FullName = strings.TrimSpace(submission.FullName)
	submission.AddressLine1 = strings.TrimSpace(submission.AddressLine1)
	submission.AddressLine2 = strings.TrimSpace(submission.AddressLine2)
	submission.TownCity = strings.TrimSpace(submission.TownCity)
	submission.CountyState = strings.TrimSpace(submission.CountyState)
	submission.Postcode = strings.TrimSpace(submission.Postcode)
	submission.Country = strings.TrimSpace(submission.Country)
	submission.Email = strings.TrimSpace(submission.Email)
	submission.Phone = strings.TrimSpace(submission.Phone)
	submission.DeliveryNotes = strings.TrimSpace(submission.DeliveryNotes)
	submission.ConsentText = strings.TrimSpace(submission.ConsentText)
	if submission.ConsentText == "" {
		submission.ConsentText = "I agree that BounceCast can store these details to fulfil my prize. Marketing contact is optional."
	}
	return submission
}

func validateClaimSubmission(submission models.RewardClaimSubmission) error {
	if submission.ClaimID <= 0 {
		return errors.New("claim ID is required")
	}
	required := map[string]string{
		"full name":      submission.FullName,
		"address line 1": submission.AddressLine1,
		"town/city":      submission.TownCity,
		"postcode":       submission.Postcode,
		"country":        submission.Country,
		"email":          submission.Email,
	}
	for label, value := range required {
		if value == "" {
			return fmt.Errorf("%s is required", label)
		}
	}
	if _, err := mail.ParseAddress(submission.Email); err != nil {
		return errors.New("email must be valid")
	}
	return nil
}

func isAwardSource(source string) bool {
	switch source {
	case models.RewardCreditSourceChatActivity, models.RewardCreditSourceTaskCompleted, models.RewardCreditSourceAchievementUnlocked,
		models.RewardCreditSourceTopSupporterReward, models.RewardCreditSourceAdminGrant, models.RewardCreditSourceAdminAdjustment:
		return true
	default:
		return false
	}
}

func isRealPrize(prize models.RewardPrize) bool {
	return prize.PrizeType == models.RewardPrizeTypePhysical || prize.PrizeType == models.RewardPrizeTypeDigital
}

func defaultConsentText(prize models.RewardPrize) string {
	if prize.MarketingConsentRequired {
		return "I consent to BounceCast storing my prize fulfilment details and marketing consent for this prize."
	}
	return "I consent to BounceCast storing my prize fulfilment details for delivery. Marketing contact remains optional."
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
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

func valueOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func hashChatRewardMessage(value string) string {
	normalized := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
	sum := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", sum[:])
}

func nullableString(value string) interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableInt(value *int) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt64(value int64) interface{} {
	if value <= 0 {
		return nil
	}
	return value
}
