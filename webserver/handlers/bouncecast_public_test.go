package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
)

func TestBounceCastPublicDJsAndScheduleExposeOnlyActivePublicLineup(t *testing.T) {
	resetBounceCastStudioAuthTestTables(t)

	db := data.GetDatabase()
	if _, err := db.Exec(`
		INSERT INTO bouncecast_streamer_accounts(id, display_name, handle, email, role, status, avatar_url)
		VALUES
			(9001, 'DJ Public', 'dj-public', 'public@lineup.example', 'streamer', 'active', '/public/profiles/dj.png'),
			(9002, 'DJ Hidden', 'dj-hidden', 'hidden@lineup.example', 'streamer', 'inactive', '/public/profiles/hidden.png')
	`); err != nil {
		t.Fatalf("insert streamers: %v", err)
	}
	startsAt := time.Now().UTC().Add(2 * time.Hour)
	if _, err := db.Exec(`
		INSERT INTO bouncecast_stream_schedule(streamer_id, title, description, starts_at, timezone, status, visibility)
		VALUES
			(9001, 'Public Set', 'Main room', ?, 'Europe/London', 'planned', 'public'),
			(9002, 'Hidden Set', 'Back room', ?, 'Europe/London', 'planned', 'public'),
			(9001, 'Private Set', 'VIP', ?, 'Europe/London', 'planned', 'private')
	`, startsAt, startsAt, startsAt); err != nil {
		t.Fatalf("insert schedule: %v", err)
	}

	djsRecorder := httptest.NewRecorder()
	GetBounceCastPublicDJs(djsRecorder, httptest.NewRequest(http.MethodGet, "/api/bouncecast/djs", nil))
	if djsRecorder.Code != http.StatusOK {
		t.Fatalf("djs status = %d, want 200: %s", djsRecorder.Code, djsRecorder.Body.String())
	}
	var djs []bounceCastPublicDJ
	if err := json.NewDecoder(djsRecorder.Body).Decode(&djs); err != nil {
		t.Fatalf("decode djs: %v", err)
	}
	if len(djs) != 1 || djs[0].Handle != "dj-public" || djs[0].UpcomingSet != "Public Set" {
		t.Fatalf("unexpected public DJs: %+v", djs)
	}

	scheduleRecorder := httptest.NewRecorder()
	GetBounceCastPublicSchedule(scheduleRecorder, httptest.NewRequest(http.MethodGet, "/api/bouncecast/schedule", nil))
	if scheduleRecorder.Code != http.StatusOK {
		t.Fatalf("schedule status = %d, want 200: %s", scheduleRecorder.Code, scheduleRecorder.Body.String())
	}
	var schedule []bounceCastPublicScheduleItem
	if err := json.NewDecoder(scheduleRecorder.Body).Decode(&schedule); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}
	if len(schedule) != 1 || schedule[0].Title != "Public Set" || schedule[0].Handle != "dj-public" {
		t.Fatalf("unexpected public schedule: %+v", schedule)
	}

	router := chi.NewRouter()
	router.Get("/api/bouncecast/djs/{handle}", GetBounceCastPublicDJProfile)
	profileRecorder := httptest.NewRecorder()
	router.ServeHTTP(profileRecorder, httptest.NewRequest(http.MethodGet, "/api/bouncecast/djs/dj-public", nil))
	if profileRecorder.Code != http.StatusOK {
		t.Fatalf("profile status = %d, want 200: %s", profileRecorder.Code, profileRecorder.Body.String())
	}
	var profile bounceCastPublicDJProfile
	if err := json.NewDecoder(profileRecorder.Body).Decode(&profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	if profile.DJ.Handle != "dj-public" || len(profile.Schedule) != 1 {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestBounceCastAccountHubCombinesRolesStarsAndDJContext(t *testing.T) {
	resetBounceCastStudioAuthTestTables(t)
	db := data.GetDatabase()
	_, _ = db.Exec(`DELETE FROM star_wallet_transactions WHERE user_id = 'hub-test-user'`)
	_, _ = db.Exec(`DELETE FROM star_wallets WHERE user_id = 'hub-test-user'`)
	_, _ = db.Exec(`DELETE FROM users WHERE id = 'hub-test-user'`)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM star_wallet_transactions WHERE user_id = 'hub-test-user'`)
		_, _ = db.Exec(`DELETE FROM star_wallets WHERE user_id = 'hub-test-user'`)
		_, _ = db.Exec(`DELETE FROM users WHERE id = 'hub-test-user'`)
	})

	if _, err := db.Exec(`
		INSERT INTO users(id, display_name, display_color, previous_names, created_at, authenticated_at, scopes, email, registered_at)
		VALUES('hub-test-user', 'Hub Test DJ', 1, 'Hub Test DJ', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'DJ,MODERATOR', 'hub@lineup.example', CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO bouncecast_streamer_accounts(id, display_name, handle, email, role, status, avatar_url)
		VALUES(9010, 'Hub Test DJ', 'hub-test-dj', 'hub@lineup.example', 'streamer', 'active', '/public/profiles/hub.png')
	`); err != nil {
		t.Fatalf("insert streamer: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO bouncecast_stream_schedule(streamer_id, title, starts_at, timezone, status, visibility)
		VALUES(9010, 'Hub Headline Set', ?, 'Europe/London', 'planned', 'public')
	`, time.Now().UTC().Add(2*time.Hour)); err != nil {
		t.Fatalf("insert schedule: %v", err)
	}

	recorder := httptest.NewRecorder()
	BounceCastAccountHub(models.User{
		ID:          "hub-test-user",
		DisplayName: "Hub Test DJ",
		Email:       "hub@lineup.example",
		Scopes:      []string{models.BounceCastDJScopeKey, models.ModeratorScopeKey},
	}, recorder, httptest.NewRequest(http.MethodGet, "/api/bouncecast/account/hub", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("hub status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	var response bounceCastAccountHubResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode hub: %v", err)
	}
	if response.Stars == nil || response.Stars.Wallet.UserID != "hub-test-user" {
		t.Fatalf("expected stars wallet summary, got %+v", response.Stars)
	}
	if response.DJProfile == nil || response.DJProfile.DJ.Handle != "hub-test-dj" {
		t.Fatalf("expected linked DJ profile, got %+v", response.DJProfile)
	}
	if len(response.UpcomingSchedule) == 0 {
		t.Fatal("expected upcoming public schedule")
	}
	destinationMap := map[string]bool{}
	for _, destination := range response.Destinations {
		destinationMap[destination.Key] = destination.Available
	}
	if !destinationMap["studio"] || !destinationMap["moderation"] || destinationMap["admin"] {
		t.Fatalf("unexpected destinations: %+v", response.Destinations)
	}
}
