package facebookmessenger

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/owncast/owncast/persistence/migrations"
)

func setupMessengerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := migrations.Run(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"object":"page"}`)
	mac := hmac.New(sha256.New, []byte("app-secret"))
	_, _ = mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !VerifySignature("app-secret", body, signature) {
		t.Fatal("expected valid signature")
	}
	if VerifySignature("wrong-secret", body, signature) {
		t.Fatal("expected invalid signature")
	}
	if VerifySignature("app-secret", body, "sha1=bad") {
		t.Fatal("expected unsupported signature prefix to fail")
	}
}

func TestHandleIncomingMessageKeywords(t *testing.T) {
	db := setupMessengerTestDB(t)

	action, response, err := HandleIncomingMessage(db, "psid-1", "LIVE", "page_message")
	if err != nil {
		t.Fatalf("opt in: %v", err)
	}
	if action != "opt_in" || !strings.Contains(response, "subscribed") {
		t.Fatalf("unexpected opt-in response %q %q", action, response)
	}

	var optedIn bool
	var status string
	if err := db.QueryRow(`SELECT opted_in, status FROM bouncecast_messenger_alert_subscribers WHERE psid = 'psid-1'`).Scan(&optedIn, &status); err != nil {
		t.Fatalf("read subscriber: %v", err)
	}
	if !optedIn || status != StatusActive {
		t.Fatalf("subscriber not active opt-in: optedIn=%t status=%q", optedIn, status)
	}

	action, response, err = HandleIncomingMessage(db, "psid-1", "STOP", "page_message")
	if err != nil {
		t.Fatalf("opt out: %v", err)
	}
	if action != "opt_out" || !strings.Contains(response, "unsubscribed") {
		t.Fatalf("unexpected opt-out response %q %q", action, response)
	}
	if err := db.QueryRow(`SELECT opted_in, status FROM bouncecast_messenger_alert_subscribers WHERE psid = 'psid-1'`).Scan(&optedIn, &status); err != nil {
		t.Fatalf("read subscriber after opt out: %v", err)
	}
	if optedIn || status != StatusOptedOut {
		t.Fatalf("subscriber not opted out: optedIn=%t status=%q", optedIn, status)
	}
}

func TestRenderTemplateVariables(t *testing.T) {
	message := RenderTemplate("{site_name}: {stream_title} at {stream_url} with {channel_name}", TemplateContext{
		SiteName:    "BounceCast",
		StreamTitle: "Friday Frequencies",
		StreamURL:   "https://k-nrg.co.uk",
		ChannelName: "DJ Pulse",
	})
	if message != "BounceCast: Friday Frequencies at https://k-nrg.co.uk with DJ Pulse" {
		t.Fatalf("message = %q", message)
	}
}

func TestSendGoLiveCampaignIdempotencyCooldownAndPolicySkips(t *testing.T) {
	db := setupMessengerTestDB(t)
	fixedNow := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	previousNow := Now
	Now = func() time.Time { return fixedNow }
	t.Cleanup(func() { Now = previousNow })

	var sendCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/v25.0/me/messages") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		sendCount++
		_, _ = w.Write([]byte(`{"recipient_id":"psid-active","message_id":"mid"}`))
	}))
	defer server.Close()
	previousBaseURL := GraphBaseURL
	GraphBaseURL = server.URL
	t.Cleanup(func() { GraphBaseURL = previousBaseURL })

	settings := Settings{
		Enabled:           true,
		PageID:            "page-1",
		PageAccessToken:   "token",
		GraphAPIVersion:   DefaultGraphAPIVersion,
		LiveURLOverride:   "https://k-nrg.co.uk",
		CooldownSeconds:   21600,
		MessageTemplate:   "{stream_title} is live at {stream_url}",
		ButtonLabel:       "Watch Live",
		ValidateAppSecret: true,
	}
	if err := SaveSettings(db, settings, false); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	streamerID, eventID := insertMessengerGoLiveFixture(t, db, "Friday Frequencies")
	if _, err := db.Exec(`
		INSERT INTO bouncecast_messenger_alert_subscribers(psid, source, opted_in, opt_in_at, last_interaction_at, status)
		VALUES
			('psid-active', 'page_message', 1, ?, ?, 'active'),
			('psid-old', 'page_message', 1, ?, ?, 'active'),
			('psid-opted-out', 'page_message', 0, ?, ?, 'opted_out')
	`, fixedNow, fixedNow, fixedNow.Add(-48*time.Hour), fixedNow.Add(-48*time.Hour), fixedNow, fixedNow); err != nil {
		t.Fatalf("insert subscribers: %v", err)
	}

	campaign, err := SendGoLiveCampaign(db, eventID)
	if err != nil {
		t.Fatalf("send campaign: %v", err)
	}
	if campaign.SentCount != 1 || campaign.SkippedCount != 1 || campaign.FailedCount != 0 {
		t.Fatalf("unexpected campaign counts: %+v", campaign)
	}
	if sendCount != 1 {
		t.Fatalf("send count = %d, want 1", sendCount)
	}

	duplicate, err := SendGoLiveCampaign(db, eventID)
	if err != nil {
		t.Fatalf("duplicate campaign: %v", err)
	}
	if duplicate.ID != campaign.ID || sendCount != 1 {
		t.Fatalf("duplicate was not idempotent: first=%d duplicate=%d sends=%d", campaign.ID, duplicate.ID, sendCount)
	}

	secondEventID := insertMessengerGoLiveEvent(t, db, streamerID, "Second Set")
	cooldownCampaign, err := SendGoLiveCampaign(db, secondEventID)
	if err != nil {
		t.Fatalf("cooldown campaign: %v", err)
	}
	if cooldownCampaign.Status != "skipped" || cooldownCampaign.SentCount != 0 {
		t.Fatalf("cooldown campaign = %+v, want skipped with zero sends", cooldownCampaign)
	}
}

func TestSendGoLiveCampaignLogsInvalidTokenError(t *testing.T) {
	db := setupMessengerTestDB(t)
	fixedNow := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	previousNow := Now
	Now = func() time.Time { return fixedNow }
	t.Cleanup(func() { Now = previousNow })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid OAuth access token","type":"OAuthException","code":190}}`))
	}))
	defer server.Close()
	previousBaseURL := GraphBaseURL
	GraphBaseURL = server.URL
	t.Cleanup(func() { GraphBaseURL = previousBaseURL })

	if err := SaveSettings(db, Settings{
		Enabled:         true,
		PageID:          "page-1",
		PageAccessToken: "bad-token",
		LiveURLOverride: "https://k-nrg.co.uk",
	}, false); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	_, eventID := insertMessengerGoLiveFixture(t, db, "Bad Token Set")
	if _, err := db.Exec(`
		INSERT INTO bouncecast_messenger_alert_subscribers(psid, source, opted_in, opt_in_at, last_interaction_at, status)
		VALUES('psid-active', 'page_message', 1, ?, ?, 'active')
	`, fixedNow, fixedNow); err != nil {
		t.Fatalf("insert subscriber: %v", err)
	}

	campaign, err := SendGoLiveCampaign(db, eventID)
	if err != nil {
		t.Fatalf("send campaign returned hard error: %v", err)
	}
	if campaign.Status != "failed" || campaign.FailedCount != 1 {
		t.Fatalf("campaign = %+v, want failed", campaign)
	}
	if !strings.Contains(ReadSettings(db).LastError, "Invalid OAuth access token") {
		t.Fatalf("last error was not logged: %q", ReadSettings(db).LastError)
	}
}

func insertMessengerGoLiveFixture(t *testing.T, db *sql.DB, title string) (int64, int64) {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO bouncecast_streamer_accounts(display_name, handle, status)
		VALUES('DJ Messenger', 'dj-messenger', 'active')
	`)
	if err != nil {
		t.Fatalf("insert streamer: %v", err)
	}
	streamerID, _ := result.LastInsertId()
	return streamerID, insertMessengerGoLiveEvent(t, db, streamerID, title)
}

func insertMessengerGoLiveEvent(t *testing.T, db *sql.DB, streamerID int64, title string) int64 {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO bouncecast_stream_schedule(streamer_id, title, starts_at, status, visibility)
		VALUES(?, ?, datetime('now'), 'live', 'public')
	`, streamerID, title)
	if err != nil {
		t.Fatalf("insert schedule: %v", err)
	}
	scheduleID, _ := result.LastInsertId()
	result, err = db.Exec(`
		INSERT INTO bouncecast_go_live_events(streamer_id, schedule_id, started_at, status)
		VALUES(?, ?, datetime('now'), 'live')
	`, streamerID, scheduleID)
	if err != nil {
		t.Fatalf("insert go-live event: %v", err)
	}
	eventID, _ := result.LastInsertId()
	if eventID == 0 {
		t.Fatalf("empty event id for %s", fmt.Sprint(title))
	}
	return eventID
}
