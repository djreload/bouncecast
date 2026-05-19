package rtmp

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/core/facebookmessenger"
	"github.com/owncast/owncast/notifications/browser"
	"github.com/owncast/owncast/persistence/configrepository"
	"github.com/owncast/owncast/persistence/notificationsrepository"
	"github.com/owncast/owncast/utils"
	log "github.com/sirupsen/logrus"
)

type bounceCastStreamerKeyMatch struct {
	streamerID    int64
	streamKeyID   int64
	displayName   string
	goLiveEventID int64
}

type bounceCastEmailSettings struct {
	enabled     bool
	provider    string
	host        string
	port        int
	username    string
	password    string
	fromAddress string
	fromName    string
	startTLS    bool
	subject     string
}

type bounceCastMessengerSettings struct {
	enabled         bool
	graphAPIVersion string
	pageAccessToken string
	messageTemplate string
}

type bounceCastNotificationSubscriberTarget struct {
	id          int64
	channel     string
	destination string
}

type bounceCastQueuedDelivery struct {
	id          int64
	destination string
	reminderID  sql.NullInt64
}

const (
	bounceCastEmailProviderKey            = "email_provider"
	bounceCastEmailEnabledKey             = "email_enabled"
	bounceCastEmailHostKey                = "email_host"
	bounceCastEmailPortKey                = "email_port"
	bounceCastEmailUsernameKey            = "email_username"
	bounceCastEmailPasswordKey            = "email_password"
	bounceCastEmailFromAddressKey         = "email_from_address"
	bounceCastEmailFromNameKey            = "email_from_name"
	bounceCastEmailStartTLSKey            = "email_start_tls"
	bounceCastEmailSubjectKey             = "email_subject"
	bounceCastPushDeliveryChannel         = "push"
	bounceCastEmailProviderBrevo          = "brevo"
	bounceCastEmailProviderCustom         = "custom"
	bounceCastBrevoSMTPHost               = "smtp-relay.brevo.com"
	bounceCastBrevoSMTPPort               = 587
	bounceCastMessengerEnabledKey         = "messenger_enabled"
	bounceCastMessengerAPIVersionKey      = "messenger_graph_api_version"
	bounceCastMessengerPageAccessTokenKey = "messenger_page_access_token"
	bounceCastMessengerMessageTemplateKey = "messenger_message_template"
)

var sendBounceCastQueuedDeliveries = func(goLiveEventID int64) {
	go sendBounceCastWebhookDeliveries(goLiveEventID)
	go sendBounceCastEmailDeliveries(goLiveEventID)
	go sendBounceCastBrowserPushDeliveries(goLiveEventID)
	go sendBounceCastMessengerDeliveries(goLiveEventID)
}

// RetryBounceCastFailedNotificationDeliveries requeues failed BounceCast
// go-live notification deliveries and dispatches the affected go-live events.
func RetryBounceCastFailedNotificationDeliveries(goLiveEventID int64, channel string) (int64, error) {
	db := data.GetDatabase()
	if db == nil {
		return 0, fmt.Errorf("database unavailable")
	}

	channel = strings.TrimSpace(strings.ToLower(channel))
	args := []interface{}{}
	query := `
		SELECT DISTINCT go_live_event_id
		FROM bouncecast_notification_deliveries
		WHERE status = 'failed'
			AND go_live_event_id IS NOT NULL
	`
	if goLiveEventID > 0 {
		query += " AND go_live_event_id = ?"
		args = append(args, goLiveEventID)
	}
	if channel != "" && channel != "all" {
		query += " AND channel = ?"
		args = append(args, channel)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return 0, err
	}
	eventIDs := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return 0, err
		}
		eventIDs = append(eventIDs, id)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	updateArgs := []interface{}{}
	updateQuery := `
		UPDATE bouncecast_notification_deliveries
		SET status = 'queued',
			last_error = NULL
		WHERE status = 'failed'
	`
	if goLiveEventID > 0 {
		updateQuery += " AND go_live_event_id = ?"
		updateArgs = append(updateArgs, goLiveEventID)
	}
	if channel != "" && channel != "all" {
		updateQuery += " AND channel = ?"
		updateArgs = append(updateArgs, channel)
	}

	result, err := db.Exec(updateQuery, updateArgs...)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	for _, id := range eventIDs {
		sendBounceCastQueuedDeliveries(id)
	}
	return count, nil
}

func validateBounceCastStreamerKey(path string) *bounceCastStreamerKeyMatch {
	streamingKey, ok := getStreamKeyFromPath(path)
	if !ok {
		return nil
	}

	db := data.GetDatabase()
	if db == nil {
		return nil
	}

	rows, err := db.Query(`
		SELECT k.id, k.streamer_id, k.key_hash, a.display_name
		FROM bouncecast_streamer_stream_keys k
		INNER JOIN bouncecast_streamer_accounts a ON a.id = k.streamer_id
		WHERE k.enabled = 1 AND k.revoked_at IS NULL AND a.status = 'active'
	`)
	if err != nil {
		log.Debugln("unable to query BounceCast streamer stream keys", err)
		return nil
	}

	var match *bounceCastStreamerKeyMatch
	for rows.Next() {
		var keyID int64
		var streamerID int64
		var keyHash string
		var displayName sql.NullString
		if err := rows.Scan(&keyID, &streamerID, &keyHash, &displayName); err != nil {
			log.Debugln("unable to scan BounceCast streamer stream key", err)
			continue
		}

		if utils.CompareHash(keyHash, streamingKey) == nil {
			match = &bounceCastStreamerKeyMatch{
				streamerID:  streamerID,
				streamKeyID: keyID,
				displayName: displayName.String,
			}
			break
		}
	}
	if err := rows.Close(); err != nil {
		log.Debugln("unable to close BounceCast streamer stream key rows", err)
	}
	if err := rows.Err(); err != nil {
		log.Debugln("unable to iterate BounceCast streamer stream keys", err)
	}

	if match != nil {
		if _, err := db.Exec(`UPDATE bouncecast_streamer_stream_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?`, match.streamKeyID); err != nil {
			log.Debugln("unable to update BounceCast streamer stream key last_used_at", err)
		}
		if match.displayName != "" {
			log.Infoln("Accepted BounceCast streamer key for", match.displayName)
		}
		return match
	}

	return nil
}

func beginBounceCastGoLiveEvent(match *bounceCastStreamerKeyMatch, remoteAddr net.Addr) {
	if match == nil {
		return
	}

	db := data.GetDatabase()
	if db == nil {
		return
	}

	var scheduleID sql.NullInt64
	err := db.QueryRow(`
		SELECT id
		FROM bouncecast_stream_schedule
		WHERE streamer_id = ?
			AND status IN ('planned', 'live')
			AND starts_at <= datetime('now', '+30 minutes')
			AND (ends_at IS NULL OR ends_at >= datetime('now', '-30 minutes'))
		ORDER BY starts_at DESC
		LIMIT 1
	`, match.streamerID).Scan(&scheduleID)
	if err != nil && err != sql.ErrNoRows {
		log.Debugln("unable to find BounceCast schedule for go-live event", err)
	}

	var remoteAddrString string
	if remoteAddr != nil {
		remoteAddrString = remoteAddr.String()
	}

	result, err := db.Exec(`
		INSERT INTO bouncecast_go_live_events(streamer_id, schedule_id, stream_key_id, remote_addr)
		VALUES(?, ?, ?, NULLIF(?, ''))
	`, match.streamerID, scheduleID, match.streamKeyID, remoteAddrString)
	if err != nil {
		log.Debugln("unable to create BounceCast go-live event", err)
		return
	}

	eventID, err := result.LastInsertId()
	if err == nil {
		match.goLiveEventID = eventID
	}

	if scheduleID.Valid {
		if _, err := db.Exec(`UPDATE bouncecast_stream_schedule SET status = 'live', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, scheduleID.Int64); err != nil {
			log.Debugln("unable to mark BounceCast schedule live", err)
		}
	}

	queueBounceCastGoLiveNotifications(eventID, scheduleID)
	facebookmessenger.QueueGoLiveAlert(eventID)
}

func endBounceCastGoLiveEvent(match *bounceCastStreamerKeyMatch) {
	if match == nil || match.goLiveEventID == 0 {
		return
	}

	db := data.GetDatabase()
	if db == nil {
		return
	}

	var scheduleID sql.NullInt64
	if err := db.QueryRow(`SELECT schedule_id FROM bouncecast_go_live_events WHERE id = ?`, match.goLiveEventID).Scan(&scheduleID); err != nil {
		log.Debugln("unable to find BounceCast go-live event schedule", err)
	}

	if _, err := db.Exec(`
		UPDATE bouncecast_go_live_events
		SET ended_at = CURRENT_TIMESTAMP, status = 'ended'
		WHERE id = ?
	`, match.goLiveEventID); err != nil {
		log.Debugln("unable to end BounceCast go-live event", err)
	}

	if scheduleID.Valid {
		if _, err := db.Exec(`UPDATE bouncecast_stream_schedule SET status = 'completed', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, scheduleID.Int64); err != nil {
			log.Debugln("unable to complete BounceCast schedule", err)
		}
	}
}

func queueBounceCastGoLiveNotifications(goLiveEventID int64, scheduleID sql.NullInt64) {
	if goLiveEventID == 0 {
		return
	}

	db := data.GetDatabase()
	if db == nil {
		return
	}

	enabledChannels := map[string]bool{
		"email":   true,
		"push":    true,
		"webhook": true,
	}

	if scheduleID.Valid {
		var notifyEmail bool
		var notifyPush bool
		var notifyWebhook bool
		if err := db.QueryRow(`
			SELECT notify_email, notify_push, notify_webhook
			FROM bouncecast_stream_schedule
			WHERE id = ?
		`, scheduleID.Int64).Scan(&notifyEmail, &notifyPush, &notifyWebhook); err != nil {
			log.Debugln("unable to read BounceCast schedule notification settings", err)
		} else {
			enabledChannels["email"] = notifyEmail
			enabledChannels["push"] = notifyPush
			enabledChannels["webhook"] = notifyWebhook
		}
	}

	rows, err := db.Query(`
		SELECT id, channel, destination
		FROM bouncecast_notification_subscribers
		WHERE disabled_at IS NULL
	`)
	if err != nil {
		log.Debugln("unable to query BounceCast notification subscribers", err)
		return
	}

	subscribers := []bounceCastNotificationSubscriberTarget{}
	for rows.Next() {
		var subscriber bounceCastNotificationSubscriberTarget
		if err := rows.Scan(&subscriber.id, &subscriber.channel, &subscriber.destination); err != nil {
			log.Debugln("unable to scan BounceCast notification subscriber", err)
			continue
		}
		if !enabledChannels[subscriber.channel] {
			continue
		}
		subscribers = append(subscribers, subscriber)
	}
	if err := rows.Close(); err != nil {
		log.Debugln("unable to close BounceCast notification subscriber rows", err)
	}
	if err := rows.Err(); err != nil {
		log.Debugln("unable to iterate BounceCast notification subscribers", err)
	}

	queuedCount := 0
	for _, subscriber := range subscribers {
		if _, err := db.Exec(`
			INSERT INTO bouncecast_notification_deliveries(go_live_event_id, subscriber_id, channel, destination)
			VALUES(?, ?, ?, ?)
		`, goLiveEventID, subscriber.id, subscriber.channel, subscriber.destination); err != nil {
			log.Debugln("unable to queue BounceCast notification delivery", err)
			continue
		}
		queuedCount++
	}

	if scheduleID.Valid {
		queuedCount += queueBounceCastScheduleReminderDeliveries(db, goLiveEventID, scheduleID.Int64, enabledChannels)
	}

	if enabledChannels[bounceCastPushDeliveryChannel] {
		queuedCount += queueBounceCastBrowserPushDeliveries(db, goLiveEventID)
	}

	state := "none"
	if queuedCount > 0 {
		state = "queued"
	}
	if _, err := db.Exec(`UPDATE bouncecast_go_live_events SET notification_state = ? WHERE id = ?`, state, goLiveEventID); err != nil {
		log.Debugln("unable to update BounceCast notification state", err)
	}

	if queuedCount > 0 {
		sendBounceCastQueuedDeliveries(goLiveEventID)
	}
}

func queueBounceCastScheduleReminderDeliveries(db *sql.DB, goLiveEventID int64, scheduleID int64, enabledChannels map[string]bool) int {
	rows, err := db.Query(`
		SELECT r.id, COALESCE(r.email, ''), r.notify_email, r.notify_push, r.notify_messenger,
			COALESCE(r.messenger_destination, ''),
			COALESCE(r.browser_push_endpoint, ''),
			COALESCE((
				SELECT p.subscription_json
				FROM bouncecast_account_push_subscriptions p
				WHERE p.user_id = r.user_id AND p.enabled = 1 AND p.disabled_at IS NULL
				ORDER BY p.last_seen_at DESC
				LIMIT 1
			), '')
		FROM bouncecast_schedule_reminders
		r
		WHERE schedule_id = ?
			AND disabled_at IS NULL
	`, scheduleID)
	if err != nil {
		log.Debugln("unable to query BounceCast schedule reminders", err)
		return 0
	}

	type reminderTarget struct {
		id                   int64
		email                string
		notifyEmail          bool
		notifyPush           bool
		notifyMessenger      bool
		messengerDestination string
		browserPushEndpoint  string
		storedPushEndpoint   string
	}
	targets := []reminderTarget{}
	for rows.Next() {
		var target reminderTarget
		if err := rows.Scan(&target.id, &target.email, &target.notifyEmail, &target.notifyPush, &target.notifyMessenger, &target.messengerDestination, &target.browserPushEndpoint, &target.storedPushEndpoint); err != nil {
			log.Debugln("unable to scan BounceCast schedule reminder", err)
			continue
		}
		targets = append(targets, target)
	}
	if err := rows.Close(); err != nil {
		log.Debugln("unable to close BounceCast schedule reminder rows", err)
	}
	if err := rows.Err(); err != nil {
		log.Debugln("unable to iterate BounceCast schedule reminders", err)
	}

	queuedCount := 0
	for _, target := range targets {
		if target.notifyEmail && enabledChannels["email"] && strings.TrimSpace(target.email) != "" {
			if queueBounceCastReminderDelivery(db, goLiveEventID, target.id, "email", target.email) {
				queuedCount++
			}
		}
		if target.notifyPush && enabledChannels[bounceCastPushDeliveryChannel] {
			pushDestination := strings.TrimSpace(target.browserPushEndpoint)
			if pushDestination == "" {
				pushDestination = strings.TrimSpace(target.storedPushEndpoint)
			}
			if pushDestination != "" && queueBounceCastReminderDelivery(db, goLiveEventID, target.id, bounceCastPushDeliveryChannel, pushDestination) {
				queuedCount++
			}
		}
		if target.notifyMessenger && strings.TrimSpace(target.messengerDestination) != "" {
			if queueBounceCastReminderDelivery(db, goLiveEventID, target.id, "messenger", target.messengerDestination) {
				queuedCount++
			}
		}
	}
	return queuedCount
}

func queueBounceCastReminderDelivery(db *sql.DB, goLiveEventID int64, reminderID int64, channel string, destination string) bool {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return false
	}
	result, err := db.Exec(`
			INSERT INTO bouncecast_notification_deliveries(go_live_event_id, channel, destination, reminder_id)
			SELECT ?, ?, ?, ?
			WHERE NOT EXISTS (
				SELECT 1
				FROM bouncecast_notification_deliveries
				WHERE go_live_event_id = ? AND channel = ? AND destination = ?
			)
		`, goLiveEventID, channel, destination, reminderID, goLiveEventID, channel, destination)
	if err != nil {
		log.Debugln("unable to queue BounceCast schedule reminder delivery", err)
		return false
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		return false
	}
	if _, err := db.Exec(`
		UPDATE bouncecast_schedule_reminders
		SET last_queued_at = CURRENT_TIMESTAMP,
			last_delivery_status = 'queued',
			last_delivery_error = NULL,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, reminderID); err != nil {
		log.Debugln("unable to update BounceCast schedule reminder queued status", err)
	}
	return true
}

func queueBounceCastBrowserPushDeliveries(db *sql.DB, goLiveEventID int64) int {
	rows, err := db.Query(`
		SELECT destination
		FROM notifications
		WHERE channel = ?
	`, notificationsrepository.BrowserPushNotification)
	if err != nil {
		log.Debugln("unable to query BounceCast browser push subscribers", err)
		return 0
	}

	destinations := []string{}
	for rows.Next() {
		var destination string
		if err := rows.Scan(&destination); err != nil {
			log.Debugln("unable to scan BounceCast browser push subscriber", err)
			continue
		}
		if strings.TrimSpace(destination) == "" {
			continue
		}
		destinations = append(destinations, destination)
	}
	if err := rows.Close(); err != nil {
		log.Debugln("unable to close BounceCast browser push subscriber rows", err)
	}
	if err := rows.Err(); err != nil {
		log.Debugln("unable to iterate BounceCast browser push subscribers", err)
	}

	queuedCount := 0
	for _, destination := range destinations {
		result, err := db.Exec(`
			INSERT INTO bouncecast_notification_deliveries(go_live_event_id, channel, destination)
			SELECT ?, ?, ?
			WHERE NOT EXISTS (
				SELECT 1
				FROM bouncecast_notification_deliveries
				WHERE go_live_event_id = ? AND channel = ? AND destination = ?
			)
		`, goLiveEventID, bounceCastPushDeliveryChannel, destination, goLiveEventID, bounceCastPushDeliveryChannel, destination)
		if err != nil {
			log.Debugln("unable to queue BounceCast browser push delivery", err)
			continue
		}
		if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
			queuedCount++
		}
	}

	return queuedCount
}

type bounceCastWebhookPayload struct {
	EventID       int64  `json:"eventId"`
	StreamerID    int64  `json:"streamerId"`
	Streamer      string `json:"streamer"`
	ScheduleID    *int64 `json:"scheduleId,omitempty"`
	ScheduleTitle string `json:"scheduleTitle,omitempty"`
	StartedAt     string `json:"startedAt"`
	Status        string `json:"status"`
	Type          string `json:"type"`
}

func sendBounceCastWebhookDeliveries(goLiveEventID int64) {
	db := data.GetDatabase()
	if db == nil {
		return
	}

	payload, err := getBounceCastWebhookPayload(goLiveEventID)
	if err != nil {
		log.Debugln("unable to build BounceCast webhook payload", err)
		return
	}

	rows, err := db.Query(`
		SELECT id, destination, reminder_id
		FROM bouncecast_notification_deliveries
		WHERE go_live_event_id = ? AND channel = 'webhook' AND status = 'queued'
	`, goLiveEventID)
	if err != nil {
		log.Debugln("unable to query BounceCast webhook deliveries", err)
		return
	}

	deliveries := []bounceCastQueuedDelivery{}
	for rows.Next() {
		var delivery bounceCastQueuedDelivery
		if err := rows.Scan(&delivery.id, &delivery.destination, &delivery.reminderID); err != nil {
			log.Debugln("unable to scan BounceCast webhook delivery", err)
			continue
		}
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Close(); err != nil {
		log.Debugln("unable to close BounceCast webhook delivery rows", err)
	}
	if err := rows.Err(); err != nil {
		log.Debugln("unable to iterate BounceCast webhook deliveries", err)
	}

	for _, delivery := range deliveries {
		if err := sendBounceCastWebhook(delivery.destination, payload); err != nil {
			if _, updateErr := db.Exec(`
				UPDATE bouncecast_notification_deliveries
				SET status = 'failed', attempt_count = attempt_count + 1, last_error = ?
				WHERE id = ?
			`, err.Error(), delivery.id); updateErr != nil {
				log.Debugln("unable to mark BounceCast webhook delivery failed", updateErr)
			}
			markBounceCastReminderDeliveryStatus(db, delivery.reminderID, "failed", err.Error())
			continue
		}
		if _, err := db.Exec(`
			UPDATE bouncecast_notification_deliveries
			SET status = 'sent', attempt_count = attempt_count + 1, sent_at = CURRENT_TIMESTAMP, last_error = NULL
			WHERE id = ?
		`, delivery.id); err != nil {
			log.Debugln("unable to mark BounceCast webhook delivery sent", err)
		}
		markBounceCastReminderDeliveryStatus(db, delivery.reminderID, "sent", "")
	}
}

func getBounceCastWebhookPayload(goLiveEventID int64) (bounceCastWebhookPayload, error) {
	db := data.GetDatabase()
	if db == nil {
		return bounceCastWebhookPayload{}, fmt.Errorf("database unavailable")
	}

	var payload bounceCastWebhookPayload
	var scheduleID sql.NullInt64
	var scheduleTitle sql.NullString
	var startedAt time.Time
	if err := db.QueryRow(`
		SELECT e.id, e.streamer_id, COALESCE(a.display_name, ''), e.schedule_id, s.title, e.started_at, e.status
		FROM bouncecast_go_live_events e
		LEFT JOIN bouncecast_streamer_accounts a ON a.id = e.streamer_id
		LEFT JOIN bouncecast_stream_schedule s ON s.id = e.schedule_id
		WHERE e.id = ?
	`, goLiveEventID).Scan(
		&payload.EventID,
		&payload.StreamerID,
		&payload.Streamer,
		&scheduleID,
		&scheduleTitle,
		&startedAt,
		&payload.Status,
	); err != nil {
		return bounceCastWebhookPayload{}, err
	}

	if scheduleID.Valid {
		payload.ScheduleID = &scheduleID.Int64
	}
	if scheduleTitle.Valid {
		payload.ScheduleTitle = scheduleTitle.String
	}
	payload.StartedAt = startedAt.Format(time.RFC3339)
	payload.Type = "bouncecast.go_live"

	return payload, nil
}

func sendBounceCastWebhook(destination string, payload bounceCastWebhookPayload) error {
	parsedURL, err := url.Parse(destination)
	if err != nil {
		return err
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("webhook destination must be http or https")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest(http.MethodPost, destination, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "BounceCast")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned HTTP %d", response.StatusCode)
	}
	return nil
}

func sendBounceCastBrowserPushDeliveries(goLiveEventID int64) {
	db := data.GetDatabase()
	if db == nil {
		return
	}

	payload, err := getBounceCastWebhookPayload(goLiveEventID)
	if err != nil {
		log.Debugln("unable to build BounceCast browser push payload", err)
		return
	}

	notifier, err := newBounceCastBrowserNotifier()
	if err != nil {
		markQueuedBounceCastDeliveriesFailed(db, goLiveEventID, bounceCastPushDeliveryChannel, err)
		return
	}

	rows, err := db.Query(`
		SELECT id, destination, reminder_id
		FROM bouncecast_notification_deliveries
		WHERE go_live_event_id = ? AND channel = ? AND status = 'queued'
	`, goLiveEventID, bounceCastPushDeliveryChannel)
	if err != nil {
		log.Debugln("unable to query BounceCast browser push deliveries", err)
		return
	}

	deliveries := []bounceCastQueuedDelivery{}
	for rows.Next() {
		var delivery bounceCastQueuedDelivery
		if err := rows.Scan(&delivery.id, &delivery.destination, &delivery.reminderID); err != nil {
			log.Debugln("unable to scan BounceCast browser push delivery", err)
			continue
		}
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Close(); err != nil {
		log.Debugln("unable to close BounceCast browser push delivery rows", err)
	}
	if err := rows.Err(); err != nil {
		log.Debugln("unable to iterate BounceCast browser push deliveries", err)
	}

	title, body := buildBounceCastBrowserPushMessage(payload)
	for _, delivery := range deliveries {
		unsubscribed, err := notifier.Send(delivery.destination, title, body)
		if unsubscribed {
			if removeErr := notificationsrepository.Get().RemoveNotificationForChannel(notificationsrepository.BrowserPushNotification, delivery.destination); removeErr != nil {
				log.Debugln("unable to remove expired BounceCast browser push subscriber", removeErr)
			}
			err = fmt.Errorf("browser push subscription expired")
		}
		if err != nil {
			if _, updateErr := db.Exec(`
				UPDATE bouncecast_notification_deliveries
				SET status = 'failed', attempt_count = attempt_count + 1, last_error = ?
				WHERE id = ?
			`, err.Error(), delivery.id); updateErr != nil {
				log.Debugln("unable to mark BounceCast browser push delivery failed", updateErr)
			}
			markBounceCastReminderDeliveryStatus(db, delivery.reminderID, "failed", err.Error())
			continue
		}

		if _, err := db.Exec(`
			UPDATE bouncecast_notification_deliveries
			SET status = 'sent', attempt_count = attempt_count + 1, sent_at = CURRENT_TIMESTAMP, last_error = NULL
			WHERE id = ?
		`, delivery.id); err != nil {
			log.Debugln("unable to mark BounceCast browser push delivery sent", err)
		}
		markBounceCastReminderDeliveryStatus(db, delivery.reminderID, "sent", "")
	}
}

func newBounceCastBrowserNotifier() (*browser.Browser, error) {
	configRepository := configrepository.Get()
	if !configRepository.GetBrowserPushConfig().Enabled {
		return nil, fmt.Errorf("browser push notifications are disabled")
	}

	publicKey, err := configRepository.GetBrowserPushPublicKey()
	if err != nil {
		return nil, fmt.Errorf("unable to read browser push public key: %w", err)
	}
	if publicKey == "" {
		return nil, fmt.Errorf("browser push public key is not configured")
	}

	privateKey, err := configRepository.GetBrowserPushPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("unable to read browser push private key: %w", err)
	}
	if privateKey == "" {
		return nil, fmt.Errorf("browser push private key is not configured")
	}

	return browser.New(data.GetDatastore(), publicKey, privateKey)
}

func buildBounceCastBrowserPushMessage(payload bounceCastWebhookPayload) (string, string) {
	streamer := payload.Streamer
	if streamer == "" {
		streamer = "A DJ"
	}

	title := fmt.Sprintf("%s is live on BounceCast", streamer)
	body := "Tune in now for the live DJ stream."
	if payload.ScheduleTitle != "" {
		body = payload.ScheduleTitle
	}
	return title, body
}

func markQueuedBounceCastDeliveriesFailed(db *sql.DB, goLiveEventID int64, channel string, deliveryErr error) {
	if deliveryErr == nil {
		return
	}
	if _, err := db.Exec(`
		UPDATE bouncecast_notification_deliveries
		SET status = 'failed', attempt_count = attempt_count + 1, last_error = ?
		WHERE go_live_event_id = ? AND channel = ? AND status = 'queued'
	`, deliveryErr.Error(), goLiveEventID, channel); err != nil {
		log.Debugln("unable to mark BounceCast notification deliveries failed", err)
	}
	if _, err := db.Exec(`
		UPDATE bouncecast_schedule_reminders
		SET last_delivery_status = 'failed',
			last_delivery_error = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id IN (
			SELECT reminder_id
			FROM bouncecast_notification_deliveries
			WHERE go_live_event_id = ? AND channel = ? AND reminder_id IS NOT NULL
		)
	`, deliveryErr.Error(), goLiveEventID, channel); err != nil {
		log.Debugln("unable to mark BounceCast reminder deliveries failed", err)
	}
}

func markBounceCastReminderDeliveryStatus(db *sql.DB, reminderID sql.NullInt64, status string, lastError string) {
	if !reminderID.Valid || reminderID.Int64 == 0 {
		return
	}
	if _, err := db.Exec(`
		UPDATE bouncecast_schedule_reminders
		SET last_delivery_status = ?,
			last_delivery_error = NULLIF(?, ''),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, strings.TrimSpace(lastError), reminderID.Int64); err != nil {
		log.Debugln("unable to update BounceCast reminder delivery status", err)
	}
}

func sendBounceCastMessengerDeliveries(goLiveEventID int64) {
	db := data.GetDatabase()
	if db == nil {
		return
	}

	payload, err := getBounceCastWebhookPayload(goLiveEventID)
	if err != nil {
		log.Debugln("unable to build BounceCast Messenger payload", err)
		return
	}

	settings := readBounceCastMessengerSettings()
	rows, err := db.Query(`
		SELECT id, destination, reminder_id
		FROM bouncecast_notification_deliveries
		WHERE go_live_event_id = ? AND channel = 'messenger' AND status = 'queued'
	`, goLiveEventID)
	if err != nil {
		log.Debugln("unable to query BounceCast Messenger deliveries", err)
		return
	}

	deliveries := []bounceCastQueuedDelivery{}
	for rows.Next() {
		var delivery bounceCastQueuedDelivery
		if err := rows.Scan(&delivery.id, &delivery.destination, &delivery.reminderID); err != nil {
			log.Debugln("unable to scan BounceCast Messenger delivery", err)
			continue
		}
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Close(); err != nil {
		log.Debugln("unable to close BounceCast Messenger delivery rows", err)
	}
	if err := rows.Err(); err != nil {
		log.Debugln("unable to iterate BounceCast Messenger deliveries", err)
	}

	for _, delivery := range deliveries {
		if err := sendBounceCastMessenger(settings, delivery.destination, payload); err != nil {
			if _, updateErr := db.Exec(`
				UPDATE bouncecast_notification_deliveries
				SET status = 'failed', attempt_count = attempt_count + 1, last_error = ?
				WHERE id = ?
			`, err.Error(), delivery.id); updateErr != nil {
				log.Debugln("unable to mark BounceCast Messenger delivery failed", updateErr)
			}
			markBounceCastReminderDeliveryStatus(db, delivery.reminderID, "failed", err.Error())
			continue
		}

		if _, err := db.Exec(`
			UPDATE bouncecast_notification_deliveries
			SET status = 'sent', attempt_count = attempt_count + 1, sent_at = CURRENT_TIMESTAMP, last_error = NULL
			WHERE id = ?
		`, delivery.id); err != nil {
			log.Debugln("unable to mark BounceCast Messenger delivery sent", err)
		}
		markBounceCastReminderDeliveryStatus(db, delivery.reminderID, "sent", "")
	}
}

func sendBounceCastMessenger(settings bounceCastMessengerSettings, destination string, payload bounceCastWebhookPayload) error {
	if !settings.enabled {
		return fmt.Errorf("Messenger delivery is not enabled")
	}
	if strings.TrimSpace(settings.pageAccessToken) == "" {
		return fmt.Errorf("Messenger page access token is not configured")
	}
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return fmt.Errorf("Messenger destination is required")
	}

	version := strings.TrimSpace(settings.graphAPIVersion)
	if version == "" {
		version = "v20.0"
	}
	message := buildBounceCastMessengerMessage(settings, payload)
	requestBody, err := json.Marshal(map[string]interface{}{
		"recipient": map[string]string{"id": destination},
		"message":   map[string]string{"text": message},
	})
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/me/messages?access_token=%s", version, url.QueryEscape(settings.pageAccessToken))
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Messenger API returned HTTP %d", response.StatusCode)
	}
	return nil
}

func sendBounceCastEmailDeliveries(goLiveEventID int64) {
	db := data.GetDatabase()
	if db == nil {
		return
	}

	payload, err := getBounceCastWebhookPayload(goLiveEventID)
	if err != nil {
		log.Debugln("unable to build BounceCast email payload", err)
		return
	}

	settings := readBounceCastEmailSettings()
	rows, err := db.Query(`
		SELECT id, destination, reminder_id
		FROM bouncecast_notification_deliveries
		WHERE go_live_event_id = ? AND channel = 'email' AND status = 'queued'
	`, goLiveEventID)
	if err != nil {
		log.Debugln("unable to query BounceCast email deliveries", err)
		return
	}

	deliveries := []bounceCastQueuedDelivery{}
	for rows.Next() {
		var delivery bounceCastQueuedDelivery
		if err := rows.Scan(&delivery.id, &delivery.destination, &delivery.reminderID); err != nil {
			log.Debugln("unable to scan BounceCast email delivery", err)
			continue
		}
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Close(); err != nil {
		log.Debugln("unable to close BounceCast email delivery rows", err)
	}
	if err := rows.Err(); err != nil {
		log.Debugln("unable to iterate BounceCast email deliveries", err)
	}

	for _, delivery := range deliveries {
		if err := sendBounceCastEmail(settings, delivery.destination, payload); err != nil {
			if _, updateErr := db.Exec(`
				UPDATE bouncecast_notification_deliveries
				SET status = 'failed', attempt_count = attempt_count + 1, last_error = ?
				WHERE id = ?
			`, err.Error(), delivery.id); updateErr != nil {
				log.Debugln("unable to mark BounceCast email delivery failed", updateErr)
			}
			markBounceCastReminderDeliveryStatus(db, delivery.reminderID, "failed", err.Error())
			continue
		}

		if _, err := db.Exec(`
			UPDATE bouncecast_notification_deliveries
			SET status = 'sent', attempt_count = attempt_count + 1, sent_at = CURRENT_TIMESTAMP, last_error = NULL
			WHERE id = ?
		`, delivery.id); err != nil {
			log.Debugln("unable to mark BounceCast email delivery sent", err)
		}
		markBounceCastReminderDeliveryStatus(db, delivery.reminderID, "sent", "")
	}
}

func sendBounceCastEmail(settings bounceCastEmailSettings, destination string, payload bounceCastWebhookPayload) error {
	if !settings.enabled {
		return fmt.Errorf("SMTP email notifications are not enabled")
	}
	if settings.host == "" || settings.fromAddress == "" {
		return fmt.Errorf("SMTP host and from address are required")
	}

	fromAddress := strings.TrimSpace(settings.fromAddress)
	if _, err := mail.ParseAddress(fromAddress); err != nil {
		return fmt.Errorf("invalid SMTP from address: %w", err)
	}
	to, err := mail.ParseAddress(strings.TrimSpace(destination))
	if err != nil {
		return fmt.Errorf("invalid email destination: %w", err)
	}

	from := mail.Address{Name: sanitizeBounceCastEmailHeader(settings.fromName), Address: fromAddress}
	subject := sanitizeBounceCastEmailHeader(strings.ReplaceAll(settings.subject, "{{streamer}}", payload.Streamer))
	body := fmt.Sprintf("%s is live on BounceCast.\n\n", payload.Streamer)
	if payload.ScheduleTitle != "" {
		body += fmt.Sprintf("Set: %s\n", payload.ScheduleTitle)
	}
	body += fmt.Sprintf("Started: %s\n", payload.StartedAt)

	var message strings.Builder
	message.WriteString(fmt.Sprintf("From: %s\r\n", from.String()))
	message.WriteString(fmt.Sprintf("To: %s\r\n", to.String()))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	message.WriteString("\r\n")
	message.WriteString(body)

	address := net.JoinHostPort(settings.host, strconv.Itoa(settings.port))
	dialer := net.Dialer{Timeout: 10 * time.Second}
	var connection net.Conn
	var connectionErr error
	if settings.port == 465 {
		connection, connectionErr = tls.DialWithDialer(&dialer, "tcp", address, &tls.Config{ServerName: settings.host, MinVersion: tls.VersionTLS12})
	} else {
		connection, connectionErr = dialer.Dial("tcp", address)
	}
	if connectionErr != nil {
		return connectionErr
	}
	defer connection.Close()

	client, err := smtp.NewClient(connection, settings.host)
	if err != nil {
		return err
	}
	defer client.Close()

	if settings.startTLS && settings.port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: settings.host, MinVersion: tls.VersionTLS12}); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("SMTP server does not support STARTTLS")
		}
	}

	if settings.username != "" {
		if err := client.Auth(smtp.PlainAuth("", settings.username, settings.password, settings.host)); err != nil {
			return err
		}
	}
	if err := client.Mail(fromAddress); err != nil {
		return err
	}
	if err := client.Rcpt(to.Address); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write([]byte(message.String())); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func sanitizeBounceCastEmailHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func readBounceCastEmailSettings() bounceCastEmailSettings {
	provider := normalizeBounceCastEmailProvider(getBounceCastNotificationSetting(bounceCastEmailProviderKey))
	port, err := strconv.Atoi(getBounceCastNotificationSetting(bounceCastEmailPortKey))
	if err != nil || port == 0 {
		port = 587
	}
	host := getBounceCastNotificationSetting(bounceCastEmailHostKey)
	startTLS := getBounceCastNotificationSetting(bounceCastEmailStartTLSKey) == "true"
	if provider == bounceCastEmailProviderBrevo {
		host = bounceCastBrevoSMTPHost
		if port == 0 {
			port = bounceCastBrevoSMTPPort
		}
		startTLS = true
	}

	fromName := getBounceCastNotificationSetting(bounceCastEmailFromNameKey)
	if fromName == "" {
		fromName = "BounceCast"
	}
	subject := getBounceCastNotificationSetting(bounceCastEmailSubjectKey)
	if subject == "" {
		subject = "{{streamer}} is live on BounceCast"
	}

	return bounceCastEmailSettings{
		enabled:     getBounceCastNotificationSetting(bounceCastEmailEnabledKey) == "true",
		provider:    provider,
		host:        host,
		port:        port,
		username:    getBounceCastNotificationSetting(bounceCastEmailUsernameKey),
		password:    getBounceCastNotificationSetting(bounceCastEmailPasswordKey),
		fromAddress: getBounceCastNotificationSetting(bounceCastEmailFromAddressKey),
		fromName:    fromName,
		startTLS:    startTLS,
		subject:     subject,
	}
}

func readBounceCastMessengerSettings() bounceCastMessengerSettings {
	graphVersion := getBounceCastNotificationSetting(bounceCastMessengerAPIVersionKey)
	if graphVersion == "" {
		graphVersion = "v20.0"
	}
	template := getBounceCastNotificationSetting(bounceCastMessengerMessageTemplateKey)
	if template == "" {
		template = "{{streamer}} is live on BounceCast: {{schedule}}"
	}
	return bounceCastMessengerSettings{
		enabled:         getBounceCastNotificationSetting(bounceCastMessengerEnabledKey) == "true",
		graphAPIVersion: graphVersion,
		pageAccessToken: getBounceCastNotificationSetting(bounceCastMessengerPageAccessTokenKey),
		messageTemplate: template,
	}
}

func buildBounceCastMessengerMessage(settings bounceCastMessengerSettings, payload bounceCastWebhookPayload) string {
	streamer := strings.TrimSpace(payload.Streamer)
	if streamer == "" {
		streamer = "A DJ"
	}
	schedule := strings.TrimSpace(payload.ScheduleTitle)
	if schedule == "" {
		schedule = "the live stream"
	}
	message := strings.TrimSpace(settings.messageTemplate)
	if message == "" {
		message = "{{streamer}} is live on BounceCast: {{schedule}}"
	}
	message = strings.ReplaceAll(message, "{{streamer}}", streamer)
	message = strings.ReplaceAll(message, "{{schedule}}", schedule)
	if len(message) > 240 {
		message = message[:240]
	}
	return message
}

func normalizeBounceCastEmailProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case bounceCastEmailProviderBrevo:
		return bounceCastEmailProviderBrevo
	default:
		return bounceCastEmailProviderCustom
	}
}

func getBounceCastNotificationSetting(key string) string {
	db := data.GetDatabase()
	if db == nil {
		return ""
	}

	var value sql.NullString
	if err := db.QueryRow(`SELECT value FROM bouncecast_notification_settings WHERE key = ?`, key).Scan(&value); err != nil {
		return ""
	}
	if !value.Valid {
		return ""
	}
	return value.String
}
