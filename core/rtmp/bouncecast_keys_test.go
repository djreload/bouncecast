package rtmp

import (
	"database/sql"
	"net"
	"strings"
	"testing"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/utils"
)

func setupBounceCastRTMPTestDB(t *testing.T) *sql.DB {
	t.Helper()

	if err := data.SetupPersistence(":memory:"); err != nil {
		t.Fatalf("setup persistence: %v", err)
	}

	db := data.GetDatabase()
	if db == nil {
		t.Fatal("expected test database")
	}

	previousDispatcher := sendBounceCastQueuedDeliveries
	sendBounceCastQueuedDeliveries = func(int64) {}
	t.Cleanup(func() {
		sendBounceCastQueuedDeliveries = previousDispatcher
		_ = db.Close()
	})

	return db
}

func insertBounceCastStreamerWithKey(t *testing.T, db *sql.DB, displayName string, handle string, status string, rawKey string) (int64, int64) {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO bouncecast_streamer_accounts(display_name, handle, status)
		VALUES(?, ?, ?)
	`, displayName, handle, status)
	if err != nil {
		t.Fatalf("insert streamer: %v", err)
	}
	streamerID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read streamer id: %v", err)
	}

	hashedKey, err := utils.HashPassword(rawKey)
	if err != nil {
		t.Fatalf("hash stream key: %v", err)
	}
	result, err = db.Exec(`
		INSERT INTO bouncecast_streamer_stream_keys(streamer_id, key_hash, label)
		VALUES(?, ?, ?)
	`, streamerID, hashedKey, "Primary key")
	if err != nil {
		t.Fatalf("insert stream key: %v", err)
	}
	streamKeyID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read stream key id: %v", err)
	}

	return streamerID, streamKeyID
}

func TestValidateBounceCastStreamerKeyMatchesActiveStreamer(t *testing.T) {
	db := setupBounceCastRTMPTestDB(t)
	streamerID, streamKeyID := insertBounceCastStreamerWithKey(t, db, "DJ Echo", "dj-echo", "active", "dj-secret")
	insertBounceCastStreamerWithKey(t, db, "Inactive DJ", "inactive-dj", "disabled", "inactive-secret")

	match := validateBounceCastStreamerKey("/live/dj-secret")
	if match == nil {
		t.Fatal("expected active streamer key match")
	}
	if match.streamerID != streamerID {
		t.Fatalf("streamerID = %d, want %d", match.streamerID, streamerID)
	}
	if match.streamKeyID != streamKeyID {
		t.Fatalf("streamKeyID = %d, want %d", match.streamKeyID, streamKeyID)
	}
	if match.displayName != "DJ Echo" {
		t.Fatalf("displayName = %q, want DJ Echo", match.displayName)
	}

	var lastUsedAt sql.NullString
	if err := db.QueryRow(`SELECT CAST(last_used_at AS TEXT) FROM bouncecast_streamer_stream_keys WHERE id = ?`, streamKeyID).Scan(&lastUsedAt); err != nil {
		t.Fatalf("read last_used_at: %v", err)
	}
	if !lastUsedAt.Valid || strings.TrimSpace(lastUsedAt.String) == "" {
		t.Fatal("expected last_used_at to be updated")
	}

	if got := validateBounceCastStreamerKey("/live/inactive-secret"); got != nil {
		t.Fatal("expected inactive streamer key to be rejected")
	}
	if got := validateBounceCastStreamerKey("/live/not-the-key"); got != nil {
		t.Fatal("expected unknown streamer key to be rejected")
	}
}

func TestBeginAndEndBounceCastGoLiveEventLinksSchedule(t *testing.T) {
	db := setupBounceCastRTMPTestDB(t)
	streamerID, streamKeyID := insertBounceCastStreamerWithKey(t, db, "DJ Tempo", "dj-tempo", "active", "tempo-secret")

	result, err := db.Exec(`
		INSERT INTO bouncecast_stream_schedule(streamer_id, title, starts_at, ends_at, notify_email, notify_push, notify_webhook)
		VALUES(?, 'Late Night Set', datetime('now', '-5 minutes'), datetime('now', '+1 hour'), 0, 0, 0)
	`, streamerID)
	if err != nil {
		t.Fatalf("insert schedule: %v", err)
	}
	scheduleID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read schedule id: %v", err)
	}

	match := &bounceCastStreamerKeyMatch{
		streamerID:  streamerID,
		streamKeyID: streamKeyID,
		displayName: "DJ Tempo",
	}
	beginBounceCastGoLiveEvent(match, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1935})
	if match.goLiveEventID == 0 {
		t.Fatal("expected go-live event id to be set")
	}

	var eventStreamerID int64
	var eventStreamKeyID int64
	var eventScheduleID sql.NullInt64
	var remoteAddr string
	var eventStatus string
	var notificationState string
	if err := db.QueryRow(`
		SELECT streamer_id, stream_key_id, schedule_id, COALESCE(remote_addr, ''), status, notification_state
		FROM bouncecast_go_live_events
		WHERE id = ?
	`, match.goLiveEventID).Scan(&eventStreamerID, &eventStreamKeyID, &eventScheduleID, &remoteAddr, &eventStatus, &notificationState); err != nil {
		t.Fatalf("read go-live event: %v", err)
	}
	if eventStreamerID != streamerID || eventStreamKeyID != streamKeyID {
		t.Fatalf("event streamer/key = %d/%d, want %d/%d", eventStreamerID, eventStreamKeyID, streamerID, streamKeyID)
	}
	if !eventScheduleID.Valid || eventScheduleID.Int64 != scheduleID {
		t.Fatalf("event schedule id = %+v, want %d", eventScheduleID, scheduleID)
	}
	if !strings.Contains(remoteAddr, "127.0.0.1") {
		t.Fatalf("remoteAddr = %q, want loopback address", remoteAddr)
	}
	if eventStatus != "live" {
		t.Fatalf("event status = %q, want live", eventStatus)
	}
	if notificationState != "none" {
		t.Fatalf("notification state = %q, want none", notificationState)
	}

	var scheduleStatus string
	if err := db.QueryRow(`SELECT status FROM bouncecast_stream_schedule WHERE id = ?`, scheduleID).Scan(&scheduleStatus); err != nil {
		t.Fatalf("read schedule status: %v", err)
	}
	if scheduleStatus != "live" {
		t.Fatalf("schedule status = %q, want live", scheduleStatus)
	}

	endBounceCastGoLiveEvent(match)

	var endedAt sql.NullString
	if err := db.QueryRow(`
		SELECT status, CAST(ended_at AS TEXT)
		FROM bouncecast_go_live_events
		WHERE id = ?
	`, match.goLiveEventID).Scan(&eventStatus, &endedAt); err != nil {
		t.Fatalf("read ended go-live event: %v", err)
	}
	if eventStatus != "ended" {
		t.Fatalf("event status after end = %q, want ended", eventStatus)
	}
	if !endedAt.Valid || strings.TrimSpace(endedAt.String) == "" {
		t.Fatal("expected ended_at to be set")
	}
	if err := db.QueryRow(`SELECT status FROM bouncecast_stream_schedule WHERE id = ?`, scheduleID).Scan(&scheduleStatus); err != nil {
		t.Fatalf("read completed schedule status: %v", err)
	}
	if scheduleStatus != "completed" {
		t.Fatalf("schedule status after end = %q, want completed", scheduleStatus)
	}
}

func TestQueueBounceCastGoLiveNotificationsRespectsScheduleChannels(t *testing.T) {
	db := setupBounceCastRTMPTestDB(t)
	streamerID, streamKeyID := insertBounceCastStreamerWithKey(t, db, "DJ Signal", "dj-signal", "active", "signal-secret")

	result, err := db.Exec(`
		INSERT INTO bouncecast_stream_schedule(streamer_id, title, starts_at, notify_email, notify_push, notify_webhook)
		VALUES(?, 'Signal Hour', datetime('now'), 1, 0, 1)
	`, streamerID)
	if err != nil {
		t.Fatalf("insert schedule: %v", err)
	}
	scheduleID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read schedule id: %v", err)
	}

	result, err = db.Exec(`
		INSERT INTO bouncecast_go_live_events(streamer_id, schedule_id, stream_key_id)
		VALUES(?, ?, ?)
	`, streamerID, scheduleID, streamKeyID)
	if err != nil {
		t.Fatalf("insert go-live event: %v", err)
	}
	eventID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read event id: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO bouncecast_notification_subscribers(channel, destination, display_name)
		VALUES
			('email', 'alerts@example.com', 'Email crew'),
			('webhook', 'https://example.com/hooks/live', 'Webhook crew'),
			('push', '{"endpoint":"https://push.example.com/subscription","keys":{"p256dh":"key","auth":"auth"}}', 'Push crew')
	`); err != nil {
		t.Fatalf("insert subscribers: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO bouncecast_notification_subscribers(channel, destination, display_name, disabled_at)
		VALUES('webhook', 'https://example.com/hooks/disabled', 'Disabled webhook', CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert disabled subscriber: %v", err)
	}

	queueBounceCastGoLiveNotifications(eventID, sql.NullInt64{Int64: scheduleID, Valid: true})

	rows, err := db.Query(`
		SELECT channel, COUNT(*)
		FROM bouncecast_notification_deliveries
		WHERE go_live_event_id = ?
		GROUP BY channel
	`, eventID)
	if err != nil {
		t.Fatalf("read delivery counts: %v", err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var channel string
		var count int
		if err := rows.Scan(&channel, &count); err != nil {
			t.Fatalf("scan delivery count: %v", err)
		}
		counts[channel] = count
	}

	if counts["email"] != 1 {
		t.Fatalf("email delivery count = %d, want 1", counts["email"])
	}
	if counts["webhook"] != 1 {
		t.Fatalf("webhook delivery count = %d, want 1", counts["webhook"])
	}
	if counts["push"] != 0 {
		t.Fatalf("push delivery count = %d, want 0 because schedule disabled push", counts["push"])
	}

	var notificationState string
	if err := db.QueryRow(`SELECT notification_state FROM bouncecast_go_live_events WHERE id = ?`, eventID).Scan(&notificationState); err != nil {
		t.Fatalf("read notification state: %v", err)
	}
	if notificationState != "queued" {
		t.Fatalf("notification state = %q, want queued", notificationState)
	}
}

func TestBuildBounceCastBrowserPushMessage(t *testing.T) {
	title, body := buildBounceCastBrowserPushMessage(bounceCastWebhookPayload{})
	if title != "A DJ is live on BounceCast" {
		t.Fatalf("title = %q, want fallback DJ title", title)
	}
	if body != "Tune in now for the live DJ stream." {
		t.Fatalf("body = %q, want default body", body)
	}

	title, body = buildBounceCastBrowserPushMessage(bounceCastWebhookPayload{
		Streamer:      "DJ Pulse",
		ScheduleTitle: "Friday Frequencies",
	})
	if title != "DJ Pulse is live on BounceCast" {
		t.Fatalf("title = %q, want streamer title", title)
	}
	if body != "Friday Frequencies" {
		t.Fatalf("body = %q, want schedule title", body)
	}
}
