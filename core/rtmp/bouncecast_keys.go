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

type bounceCastNotificationSubscriberTarget struct {
	id          int64
	channel     string
	destination string
}

type bounceCastQueuedDelivery struct {
	id          int64
	destination string
}

const (
	bounceCastEmailProviderKey    = "email_provider"
	bounceCastEmailEnabledKey     = "email_enabled"
	bounceCastEmailHostKey        = "email_host"
	bounceCastEmailPortKey        = "email_port"
	bounceCastEmailUsernameKey    = "email_username"
	bounceCastEmailPasswordKey    = "email_password"
	bounceCastEmailFromAddressKey = "email_from_address"
	bounceCastEmailFromNameKey    = "email_from_name"
	bounceCastEmailStartTLSKey    = "email_start_tls"
	bounceCastEmailSubjectKey     = "email_subject"
	bounceCastPushDeliveryChannel = "push"
	bounceCastEmailProviderBrevo  = "brevo"
	bounceCastEmailProviderCustom = "custom"
	bounceCastBrevoSMTPHost       = "smtp-relay.brevo.com"
	bounceCastBrevoSMTPPort       = 587
)

var sendBounceCastQueuedDeliveries = func(goLiveEventID int64) {
	go sendBounceCastWebhookDeliveries(goLiveEventID)
	go sendBounceCastEmailDeliveries(goLiveEventID)
	go sendBounceCastBrowserPushDeliveries(goLiveEventID)
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

	if scheduleID.Valid && enabledChannels["email"] {
		queuedCount += queueBounceCastScheduleReminderDeliveries(db, goLiveEventID, scheduleID.Int64)
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

func queueBounceCastScheduleReminderDeliveries(db *sql.DB, goLiveEventID int64, scheduleID int64) int {
	rows, err := db.Query(`
		SELECT email
		FROM bouncecast_schedule_reminders
		WHERE schedule_id = ?
			AND disabled_at IS NULL
			AND notify_email = 1
			AND COALESCE(email, '') != ''
	`, scheduleID)
	if err != nil {
		log.Debugln("unable to query BounceCast schedule reminders", err)
		return 0
	}

	destinations := []string{}
	for rows.Next() {
		var destination string
		if err := rows.Scan(&destination); err != nil {
			log.Debugln("unable to scan BounceCast schedule reminder", err)
			continue
		}
		destination = strings.TrimSpace(destination)
		if destination == "" {
			continue
		}
		destinations = append(destinations, destination)
	}
	if err := rows.Close(); err != nil {
		log.Debugln("unable to close BounceCast schedule reminder rows", err)
	}
	if err := rows.Err(); err != nil {
		log.Debugln("unable to iterate BounceCast schedule reminders", err)
	}

	queuedCount := 0
	for _, destination := range destinations {
		result, err := db.Exec(`
			INSERT INTO bouncecast_notification_deliveries(go_live_event_id, channel, destination)
			SELECT ?, 'email', ?
			WHERE NOT EXISTS (
				SELECT 1
				FROM bouncecast_notification_deliveries
				WHERE go_live_event_id = ? AND channel = 'email' AND destination = ?
			)
		`, goLiveEventID, destination, goLiveEventID, destination)
		if err != nil {
			log.Debugln("unable to queue BounceCast schedule reminder delivery", err)
			continue
		}
		if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
			queuedCount++
		}
	}
	return queuedCount
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
		if _, err := db.Exec(`
			INSERT INTO bouncecast_notification_deliveries(go_live_event_id, channel, destination)
			VALUES(?, ?, ?)
		`, goLiveEventID, bounceCastPushDeliveryChannel, destination); err != nil {
			log.Debugln("unable to queue BounceCast browser push delivery", err)
			continue
		}
		queuedCount++
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
		SELECT id, destination
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
		if err := rows.Scan(&delivery.id, &delivery.destination); err != nil {
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
			continue
		}
		if _, err := db.Exec(`
			UPDATE bouncecast_notification_deliveries
			SET status = 'sent', attempt_count = attempt_count + 1, sent_at = CURRENT_TIMESTAMP, last_error = NULL
			WHERE id = ?
		`, delivery.id); err != nil {
			log.Debugln("unable to mark BounceCast webhook delivery sent", err)
		}
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
		SELECT id, destination
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
		if err := rows.Scan(&delivery.id, &delivery.destination); err != nil {
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
			continue
		}

		if _, err := db.Exec(`
			UPDATE bouncecast_notification_deliveries
			SET status = 'sent', attempt_count = attempt_count + 1, sent_at = CURRENT_TIMESTAMP, last_error = NULL
			WHERE id = ?
		`, delivery.id); err != nil {
			log.Debugln("unable to mark BounceCast browser push delivery sent", err)
		}
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
		SELECT id, destination
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
		if err := rows.Scan(&delivery.id, &delivery.destination); err != nil {
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
			continue
		}

		if _, err := db.Exec(`
			UPDATE bouncecast_notification_deliveries
			SET status = 'sent', attempt_count = attempt_count + 1, sent_at = CURRENT_TIMESTAMP, last_error = NULL
			WHERE id = ?
		`, delivery.id); err != nil {
			log.Debugln("unable to mark BounceCast email delivery sent", err)
		}
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
