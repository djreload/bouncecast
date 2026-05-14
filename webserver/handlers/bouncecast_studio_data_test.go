package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/owncast/owncast/core/data"
)

func authenticatedStudioRequest(method string, path string, token string, body string) (*httptest.ResponseRecorder, *http.Request) {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	return httptest.NewRecorder(), request
}

func TestBounceCastStudioDataIsScopedToLoggedInStreamer(t *testing.T) {
	resetBounceCastStudioAuthTestTables(t)
	streamerID := insertBounceCastStudioAuthStreamer(t, "scoped-dj", "scoped@example.com", "active", "correct-password")
	otherStreamerID := insertBounceCastStudioAuthStreamer(t, "other-dj", "other@example.com", "active", "correct-password")

	recorder, loginResponse := loginBounceCastStudioStreamer(t, `{"login":"scoped-dj","password":"correct-password"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	db := data.GetDatabase()
	if db == nil {
		t.Fatal("expected test database")
	}
	if _, err := db.Exec(`
		INSERT INTO bouncecast_stream_schedule(streamer_id, title, starts_at, notify_email, notify_push, notify_webhook)
		VALUES
			(?, 'Scoped Set', datetime('now', '+1 hour'), 1, 1, 1),
			(?, 'Other Set', datetime('now', '+1 hour'), 1, 1, 1)
	`, streamerID, otherStreamerID); err != nil {
		t.Fatalf("insert schedule: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO bouncecast_go_live_events(streamer_id, started_at, status, notification_state)
		VALUES
			(?, datetime('now', '-1 hour'), 'ended', 'sent'),
			(?, datetime('now', '-1 hour'), 'ended', 'sent')
	`, streamerID, otherStreamerID); err != nil {
		t.Fatalf("insert events: %v", err)
	}

	keysRecorder, keysRequest := authenticatedStudioRequest(http.MethodPost, "/api/bouncecast/studio/streamkeys", loginResponse.Token, `{"label":"Main OBS"}`)
	BounceCastStudioCreateStreamKey(keysRecorder, keysRequest)
	if keysRecorder.Code != http.StatusOK {
		t.Fatalf("create stream key status = %d, want 200: %s", keysRecorder.Code, keysRecorder.Body.String())
	}
	var createKeyResponse map[string]interface{}
	if err := json.NewDecoder(keysRecorder.Body).Decode(&createKeyResponse); err != nil {
		t.Fatalf("decode create key response: %v", err)
	}
	if createKeyResponse["streamKey"] == "" {
		t.Fatal("expected raw stream key in create response")
	}

	scheduleRecorder, scheduleRequest := authenticatedStudioRequest(http.MethodGet, "/api/bouncecast/studio/schedule", loginResponse.Token, "")
	BounceCastStudioSchedule(scheduleRecorder, scheduleRequest)
	if scheduleRecorder.Code != http.StatusOK {
		t.Fatalf("schedule status = %d, want 200: %s", scheduleRecorder.Code, scheduleRecorder.Body.String())
	}
	var schedule []bounceCastStudioScheduleItem
	if err := json.NewDecoder(scheduleRecorder.Body).Decode(&schedule); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}
	if len(schedule) != 1 || schedule[0].Title != "Scoped Set" {
		t.Fatalf("schedule = %+v, want only Scoped Set", schedule)
	}

	listKeysRecorder, listKeysRequest := authenticatedStudioRequest(http.MethodGet, "/api/bouncecast/studio/streamkeys", loginResponse.Token, "")
	BounceCastStudioStreamKeys(listKeysRecorder, listKeysRequest)
	if listKeysRecorder.Code != http.StatusOK {
		t.Fatalf("list keys status = %d, want 200: %s", listKeysRecorder.Code, listKeysRecorder.Body.String())
	}
	var keys []bounceCastStudioStreamKey
	if err := json.NewDecoder(listKeysRecorder.Body).Decode(&keys); err != nil {
		t.Fatalf("decode stream keys: %v", err)
	}
	if len(keys) != 1 || keys[0].Label != "Main OBS" {
		t.Fatalf("keys = %+v, want only current DJ key", keys)
	}

	eventsRecorder, eventsRequest := authenticatedStudioRequest(http.MethodGet, "/api/bouncecast/studio/live-events", loginResponse.Token, "")
	BounceCastStudioLiveEvents(eventsRecorder, eventsRequest)
	if eventsRecorder.Code != http.StatusOK {
		t.Fatalf("events status = %d, want 200: %s", eventsRecorder.Code, eventsRecorder.Body.String())
	}
	var events []bounceCastStudioGoLiveEvent
	if err := json.NewDecoder(eventsRecorder.Body).Decode(&events); err != nil {
		t.Fatalf("decode events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want only current DJ event", len(events))
	}
}

func TestBounceCastStudioCannotRevokeAnotherStreamerKey(t *testing.T) {
	resetBounceCastStudioAuthTestTables(t)
	insertBounceCastStudioAuthStreamer(t, "owner-dj", "", "active", "correct-password")
	otherStreamerID := insertBounceCastStudioAuthStreamer(t, "locked-dj", "", "active", "correct-password")

	recorder, loginResponse := loginBounceCastStudioStreamer(t, `{"login":"owner-dj","password":"correct-password"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	db := data.GetDatabase()
	if db == nil {
		t.Fatal("expected test database")
	}
	result, err := db.Exec(`
		INSERT INTO bouncecast_streamer_stream_keys(streamer_id, key_hash, label)
		VALUES(?, 'not-a-real-hash', 'Other DJ key')
	`, otherStreamerID)
	if err != nil {
		t.Fatalf("insert other stream key: %v", err)
	}
	otherKeyID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read other key id: %v", err)
	}

	revokeRecorder, revokeRequest := authenticatedStudioRequest(http.MethodPost, "/api/bouncecast/studio/streamkeys/revoke", loginResponse.Token, `{"id":`+strconv.FormatInt(otherKeyID, 10)+`}`)
	BounceCastStudioRevokeStreamKey(revokeRecorder, revokeRequest)
	if revokeRecorder.Code != http.StatusBadRequest {
		t.Fatalf("revoke status = %d, want 400", revokeRecorder.Code)
	}

	var enabled bool
	if err := db.QueryRow(`SELECT enabled FROM bouncecast_streamer_stream_keys WHERE id = ?`, otherKeyID).Scan(&enabled); err != nil {
		t.Fatalf("read other stream key: %v", err)
	}
	if !enabled {
		t.Fatal("other streamer's key should remain enabled")
	}
}

func TestBounceCastStudioCanCreateUpdateAndCancelOwnSchedule(t *testing.T) {
	resetBounceCastStudioAuthTestTables(t)
	streamerID := insertBounceCastStudioAuthStreamer(t, "schedule-dj", "schedule@example.com", "active", "correct-password")
	otherStreamerID := insertBounceCastStudioAuthStreamer(t, "other-schedule-dj", "other-schedule@example.com", "active", "correct-password")

	recorder, loginResponse := loginBounceCastStudioStreamer(t, `{"login":"schedule-dj","password":"correct-password"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	startsAt := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	endsAt := startsAt.Add(2 * time.Hour)
	createBody := fmt.Sprintf(
		`{"title":"Opening Set","description":"Warmup hour","startsAt":%q,"endsAt":%q,"timezone":"Europe/London","notifyEmail":true,"notifyPush":true,"notifyWebhook":false}`,
		startsAt.Format(time.RFC3339),
		endsAt.Format(time.RFC3339),
	)

	createRecorder, createRequest := authenticatedStudioRequest(http.MethodPost, "/api/bouncecast/studio/schedule", loginResponse.Token, createBody)
	BounceCastStudioCreateSchedule(createRecorder, createRequest)
	if createRecorder.Code != http.StatusOK {
		t.Fatalf("create schedule status = %d, want 200: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var createResponse struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(createRecorder.Body).Decode(&createResponse); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResponse.ID == 0 {
		t.Fatal("expected created schedule id")
	}

	db := data.GetDatabase()
	if db == nil {
		t.Fatal("expected test database")
	}
	var createdStreamerID int64
	var createdTitle string
	var createdStatus string
	var notifyEmail bool
	var notifyPush bool
	var notifyWebhook bool
	if err := db.QueryRow(`
		SELECT streamer_id, title, status, notify_email, notify_push, notify_webhook
		FROM bouncecast_stream_schedule
		WHERE id = ?
	`, createResponse.ID).Scan(&createdStreamerID, &createdTitle, &createdStatus, &notifyEmail, &notifyPush, &notifyWebhook); err != nil {
		t.Fatalf("read created schedule: %v", err)
	}
	if createdStreamerID != streamerID || createdTitle != "Opening Set" || createdStatus != "planned" || !notifyEmail || !notifyPush || notifyWebhook {
		t.Fatalf("created schedule has unexpected values: streamerID=%d title=%q status=%q email=%t push=%t webhook=%t", createdStreamerID, createdTitle, createdStatus, notifyEmail, notifyPush, notifyWebhook)
	}

	updatedStartsAt := startsAt.Add(24 * time.Hour)
	updatedEndsAt := updatedStartsAt.Add(90 * time.Minute)
	updateBody := fmt.Sprintf(
		`{"id":%d,"title":"Peak Time Set","description":"Main room","startsAt":%q,"endsAt":%q,"timezone":"UTC","notifyEmail":false,"notifyPush":true,"notifyWebhook":true}`,
		createResponse.ID,
		updatedStartsAt.Format(time.RFC3339),
		updatedEndsAt.Format(time.RFC3339),
	)
	updateRecorder, updateRequest := authenticatedStudioRequest(http.MethodPost, "/api/bouncecast/studio/schedule/update", loginResponse.Token, updateBody)
	BounceCastStudioUpdateSchedule(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("update schedule status = %d, want 200: %s", updateRecorder.Code, updateRecorder.Body.String())
	}

	if err := db.QueryRow(`
		SELECT title, status, notify_email, notify_push, notify_webhook
		FROM bouncecast_stream_schedule
		WHERE id = ?
	`, createResponse.ID).Scan(&createdTitle, &createdStatus, &notifyEmail, &notifyPush, &notifyWebhook); err != nil {
		t.Fatalf("read updated schedule: %v", err)
	}
	if createdTitle != "Peak Time Set" || createdStatus != "planned" || notifyEmail || !notifyPush || !notifyWebhook {
		t.Fatalf("updated schedule has unexpected values: title=%q status=%q email=%t push=%t webhook=%t", createdTitle, createdStatus, notifyEmail, notifyPush, notifyWebhook)
	}

	result, err := db.Exec(`
		INSERT INTO bouncecast_stream_schedule(streamer_id, title, starts_at, notify_email, notify_push, notify_webhook)
		VALUES(?, 'Other DJ Set', ?, 1, 1, 1)
	`, otherStreamerID, startsAt)
	if err != nil {
		t.Fatalf("insert other schedule: %v", err)
	}
	otherScheduleID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read other schedule id: %v", err)
	}

	otherUpdateBody := fmt.Sprintf(
		`{"id":%d,"title":"Hijacked Set","startsAt":%q,"timezone":"UTC","notifyEmail":false,"notifyPush":false,"notifyWebhook":false}`,
		otherScheduleID,
		updatedStartsAt.Format(time.RFC3339),
	)
	otherUpdateRecorder, otherUpdateRequest := authenticatedStudioRequest(http.MethodPost, "/api/bouncecast/studio/schedule/update", loginResponse.Token, otherUpdateBody)
	BounceCastStudioUpdateSchedule(otherUpdateRecorder, otherUpdateRequest)
	if otherUpdateRecorder.Code != http.StatusBadRequest {
		t.Fatalf("other update status = %d, want 400", otherUpdateRecorder.Code)
	}

	cancelRecorder, cancelRequest := authenticatedStudioRequest(http.MethodPost, "/api/bouncecast/studio/schedule/cancel", loginResponse.Token, fmt.Sprintf(`{"id":%d}`, createResponse.ID))
	BounceCastStudioCancelSchedule(cancelRecorder, cancelRequest)
	if cancelRecorder.Code != http.StatusOK {
		t.Fatalf("cancel schedule status = %d, want 200: %s", cancelRecorder.Code, cancelRecorder.Body.String())
	}

	if err := db.QueryRow(`SELECT status FROM bouncecast_stream_schedule WHERE id = ?`, createResponse.ID).Scan(&createdStatus); err != nil {
		t.Fatalf("read cancelled schedule: %v", err)
	}
	if createdStatus != "cancelled" {
		t.Fatalf("cancelled schedule status = %q, want cancelled", createdStatus)
	}

	otherCancelRecorder, otherCancelRequest := authenticatedStudioRequest(http.MethodPost, "/api/bouncecast/studio/schedule/cancel", loginResponse.Token, fmt.Sprintf(`{"id":%d}`, otherScheduleID))
	BounceCastStudioCancelSchedule(otherCancelRecorder, otherCancelRequest)
	if otherCancelRecorder.Code != http.StatusBadRequest {
		t.Fatalf("other cancel status = %d, want 400", otherCancelRecorder.Code)
	}

	var otherStatus string
	var otherTitle string
	if err := db.QueryRow(`SELECT title, status FROM bouncecast_stream_schedule WHERE id = ?`, otherScheduleID).Scan(&otherTitle, &otherStatus); err != nil {
		t.Fatalf("read other schedule: %v", err)
	}
	if otherTitle != "Other DJ Set" || otherStatus != "planned" {
		t.Fatalf("other schedule was changed: title=%q status=%q", otherTitle, otherStatus)
	}
}
