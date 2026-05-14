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
	host        string
	port        int
	username    string
	password    string
	fromAddress string
	fromName    string
	startTLS    bool
	subject     string
}

const (
	bounceCastEmailEnabledKey     = "email_enabled"
	bounceCastEmailHostKey        = "email_host"
	bounceCastEmailPortKey        = "email_port"
	bounceCastEmailUsernameKey    = "email_username"
	bounceCastEmailPasswordKey    = "email_password"
	bounceCastEmailFromAddressKey = "email_from_address"
	bounceCastEmailFromNameKey    = "email_from_name"
	bounceCastEmailStartTLSKey    = "email_start_tls"
	bounceCastEmailSubjectKey     = "email_subject"
)

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
	defer rows.Close()

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
			if _, err := db.Exec(`UPDATE bouncecast_streamer_stream_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?`, keyID); err != nil {
				log.Debugln("unable to update BounceCast streamer stream key last_used_at", err)
			}
			if displayName.Valid {
				log.Infoln("Accepted BounceCast streamer key for", displayName.String)
			}
			return &bounceCastStreamerKeyMatch{
				streamerID:  streamerID,
				streamKeyID: keyID,
				displayName: displayName.String,
			}
		}
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
	defer rows.Close()

	queuedCount := 0
	for rows.Next() {
		var subscriberID int64
		var channel string
		var destination string
		if err := rows.Scan(&subscriberID, &channel, &destination); err != nil {
			log.Debugln("unable to scan BounceCast notification subscriber", err)
			continue
		}
		if !enabledChannels[channel] {
			continue
		}
		if _, err := db.Exec(`
			INSERT INTO bouncecast_notification_deliveries(go_live_event_id, subscriber_id, channel, destination)
			VALUES(?, ?, ?, ?)
		`, goLiveEventID, subscriberID, channel, destination); err != nil {
			log.Debugln("unable to queue BounceCast notification delivery", err)
			continue
		}
		queuedCount++
	}

	state := "none"
	if queuedCount > 0 {
		state = "queued"
	}
	if _, err := db.Exec(`UPDATE bouncecast_go_live_events SET notification_state = ? WHERE id = ?`, state, goLiveEventID); err != nil {
		log.Debugln("unable to update BounceCast notification state", err)
	}

	if queuedCount > 0 {
		go sendBounceCastWebhookDeliveries(goLiveEventID)
		go sendBounceCastEmailDeliveries(goLiveEventID)
	}
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
	defer rows.Close()

	for rows.Next() {
		var deliveryID int64
		var destination string
		if err := rows.Scan(&deliveryID, &destination); err != nil {
			log.Debugln("unable to scan BounceCast webhook delivery", err)
			continue
		}
		if err := sendBounceCastWebhook(destination, payload); err != nil {
			if _, updateErr := db.Exec(`
				UPDATE bouncecast_notification_deliveries
				SET status = 'failed', attempt_count = attempt_count + 1, last_error = ?
				WHERE id = ?
			`, err.Error(), deliveryID); updateErr != nil {
				log.Debugln("unable to mark BounceCast webhook delivery failed", updateErr)
			}
			continue
		}
		if _, err := db.Exec(`
			UPDATE bouncecast_notification_deliveries
			SET status = 'sent', attempt_count = attempt_count + 1, sent_at = CURRENT_TIMESTAMP, last_error = NULL
			WHERE id = ?
		`, deliveryID); err != nil {
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
	defer rows.Close()

	for rows.Next() {
		var deliveryID int64
		var destination string
		if err := rows.Scan(&deliveryID, &destination); err != nil {
			log.Debugln("unable to scan BounceCast email delivery", err)
			continue
		}

		if err := sendBounceCastEmail(settings, destination, payload); err != nil {
			if _, updateErr := db.Exec(`
				UPDATE bouncecast_notification_deliveries
				SET status = 'failed', attempt_count = attempt_count + 1, last_error = ?
				WHERE id = ?
			`, err.Error(), deliveryID); updateErr != nil {
				log.Debugln("unable to mark BounceCast email delivery failed", updateErr)
			}
			continue
		}

		if _, err := db.Exec(`
			UPDATE bouncecast_notification_deliveries
			SET status = 'sent', attempt_count = attempt_count + 1, sent_at = CURRENT_TIMESTAMP, last_error = NULL
			WHERE id = ?
		`, deliveryID); err != nil {
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

	from := mail.Address{Name: settings.fromName, Address: settings.fromAddress}
	to := mail.Address{Address: destination}
	subject := strings.ReplaceAll(settings.subject, "{{streamer}}", payload.Streamer)
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
	client, err := smtp.Dial(address)
	if err != nil {
		return err
	}
	defer client.Close()

	if settings.startTLS {
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
	if err := client.Mail(settings.fromAddress); err != nil {
		return err
	}
	if err := client.Rcpt(destination); err != nil {
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

func readBounceCastEmailSettings() bounceCastEmailSettings {
	port, err := strconv.Atoi(getBounceCastNotificationSetting(bounceCastEmailPortKey))
	if err != nil || port == 0 {
		port = 587
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
		host:        getBounceCastNotificationSetting(bounceCastEmailHostKey),
		port:        port,
		username:    getBounceCastNotificationSetting(bounceCastEmailUsernameKey),
		password:    getBounceCastNotificationSetting(bounceCastEmailPasswordKey),
		fromAddress: getBounceCastNotificationSetting(bounceCastEmailFromAddressKey),
		fromName:    fromName,
		startTLS:    getBounceCastNotificationSetting(bounceCastEmailStartTLSKey) == "true",
		subject:     subject,
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
