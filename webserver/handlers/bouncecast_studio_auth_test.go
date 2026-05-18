package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/utils"
)

func resetBounceCastStudioAuthTestTables(t *testing.T) {
	t.Helper()

	db := data.GetDatabase()
	_, _ = db.Exec(`DELETE FROM bouncecast_notification_deliveries`)
	_, _ = db.Exec(`DELETE FROM bouncecast_notification_subscribers`)
	_, _ = db.Exec(`DELETE FROM bouncecast_go_live_events`)
	_, _ = db.Exec(`DELETE FROM bouncecast_stream_schedule`)
	_, _ = db.Exec(`DELETE FROM bouncecast_streamer_sessions`)
	_, _ = db.Exec(`DELETE FROM bouncecast_streamer_stream_keys`)
	_, _ = db.Exec(`DELETE FROM bouncecast_streamer_accounts`)
	_, _ = db.Exec(`DELETE FROM user_access_tokens WHERE user_id IN (
		SELECT id FROM users WHERE email LIKE '%@studio-account-test.example'
	)`)
	_, _ = db.Exec(`DELETE FROM users WHERE email LIKE '%@studio-account-test.example'`)
}

func insertBounceCastStudioAuthStreamer(t *testing.T, handle string, email string, status string, password string) int64 {
	t.Helper()

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	result, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_streamer_accounts(display_name, handle, email, password_hash, role, status)
		VALUES('DJ Login Test', ?, NULLIF(?, ''), ?, 'streamer', ?)
	`, handle, email, passwordHash, status)
	if err != nil {
		t.Fatalf("insert streamer: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read streamer id: %v", err)
	}
	return id
}

func insertBounceCastStudioAuthPublicUser(t *testing.T, id string, displayName string, email string, scopes string, password string) {
	t.Helper()

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("hash public user password: %v", err)
	}

	if _, err := data.GetDatabase().Exec(`
		INSERT INTO users(id, display_name, display_color, previous_names, created_at, authenticated_at, scopes, email, password_hash, profile_image_url, registered_at)
		VALUES(?, ?, 1, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, NULLIF(?, ''), ?, ?, '/img/test-dj.png', CURRENT_TIMESTAMP)
	`, id, displayName, displayName, scopes, email, passwordHash); err != nil {
		t.Fatalf("insert public role user: %v", err)
	}
}

func loginBounceCastStudioStreamer(t *testing.T, body string) (*httptest.ResponseRecorder, bounceCastStudioSessionResponse) {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/bouncecast/studio/login", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	BounceCastStudioLogin(recorder, request)

	var response bounceCastStudioSessionResponse
	if recorder.Code == http.StatusOK {
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode login response: %v", err)
		}
	}
	return recorder, response
}

func TestBounceCastStudioRegisterCreatesInactiveStreamer(t *testing.T) {
	resetBounceCastStudioAuthTestTables(t)

	request := httptest.NewRequest(http.MethodPost, "/api/bouncecast/studio/register", strings.NewReader(`{
		"displayName":"DJ Pending",
		"handle":"@dj-pending",
		"email":"DJ Pending <pending@example.com>",
		"password":"pending-password"
	}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	BounceCastStudioRegister(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("register status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	var status string
	var passwordHash string
	var email string
	if err := data.GetDatabase().QueryRow(`
		SELECT status, password_hash, email
		FROM bouncecast_streamer_accounts
		WHERE handle = 'dj-pending'
	`).Scan(&status, &passwordHash, &email); err != nil {
		t.Fatalf("read registered streamer: %v", err)
	}
	if status != "inactive" {
		t.Fatalf("status = %q, want inactive", status)
	}
	if email != "pending@example.com" {
		t.Fatalf("email = %q, want pending@example.com", email)
	}
	if err := utils.CompareHash(passwordHash, "pending-password"); err != nil {
		t.Fatalf("password hash did not match: %v", err)
	}

	loginRecorder, _ := loginBounceCastStudioStreamer(t, `{"login":"dj-pending","password":"pending-password"}`)
	if loginRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("inactive login status = %d, want 401", loginRecorder.Code)
	}
}

func TestBounceCastStudioLoginMeAndLogout(t *testing.T) {
	resetBounceCastStudioAuthTestTables(t)
	streamerID := insertBounceCastStudioAuthStreamer(t, "dj-login", "dj-login@example.com", "active", "correct-password")

	recorder, loginResponse := loginBounceCastStudioStreamer(t, `{"login":"@DJ-LOGIN","password":"correct-password"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if loginResponse.Token == "" {
		t.Fatal("expected session token")
	}
	if loginResponse.Streamer.ID != streamerID {
		t.Fatalf("streamer id = %d, want %d", loginResponse.Streamer.ID, streamerID)
	}
	if loginResponse.Streamer.Email != "dj-login@example.com" {
		t.Fatalf("email = %q, want dj-login@example.com", loginResponse.Streamer.Email)
	}

	var tokenHash string
	var lastLoginAt string
	if err := data.GetDatabase().QueryRow(`
		SELECT s.token_hash, CAST(a.last_login_at AS TEXT)
		FROM bouncecast_streamer_sessions s
		INNER JOIN bouncecast_streamer_accounts a ON a.id = s.streamer_id
		WHERE s.streamer_id = ?
	`, streamerID).Scan(&tokenHash, &lastLoginAt); err != nil {
		t.Fatalf("read session: %v", err)
	}
	if tokenHash == "" || tokenHash == loginResponse.Token {
		t.Fatal("expected stored token hash instead of raw token")
	}
	if strings.TrimSpace(lastLoginAt) == "" {
		t.Fatal("expected last_login_at to be set")
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/api/bouncecast/studio/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+loginResponse.Token)
	meRecorder := httptest.NewRecorder()

	BounceCastStudioMe(meRecorder, meRequest)
	if meRecorder.Code != http.StatusOK {
		t.Fatalf("me status = %d, want 200: %s", meRecorder.Code, meRecorder.Body.String())
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/bouncecast/studio/logout", nil)
	logoutRequest.Header.Set("Authorization", "Bearer "+loginResponse.Token)
	logoutRecorder := httptest.NewRecorder()

	BounceCastStudioLogout(logoutRecorder, logoutRequest)
	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want 200: %s", logoutRecorder.Code, logoutRecorder.Body.String())
	}

	meAfterLogout := httptest.NewRequest(http.MethodGet, "/api/bouncecast/studio/me", nil)
	meAfterLogout.Header.Set("Authorization", "Bearer "+loginResponse.Token)
	meAfterLogoutRecorder := httptest.NewRecorder()

	BounceCastStudioMe(meAfterLogoutRecorder, meAfterLogout)
	if meAfterLogoutRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d, want 401", meAfterLogoutRecorder.Code)
	}
}

func TestBounceCastStudioLoginRejectsDisabledOrInvalidStreamer(t *testing.T) {
	resetBounceCastStudioAuthTestTables(t)
	insertBounceCastStudioAuthStreamer(t, "disabled-dj", "", "disabled", "correct-password")
	insertBounceCastStudioAuthStreamer(t, "active-dj", "", "active", "correct-password")

	recorder, _ := loginBounceCastStudioStreamer(t, `{"login":"disabled-dj","password":"correct-password"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("disabled login status = %d, want 401", recorder.Code)
	}

	recorder, _ = loginBounceCastStudioStreamer(t, `{"login":"active-dj","password":"wrong-password"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status = %d, want 401", recorder.Code)
	}
}

func TestBounceCastStudioLoginAcceptsPublicDJAccountRole(t *testing.T) {
	resetBounceCastStudioAuthTestTables(t)
	insertBounceCastStudioAuthPublicUser(t, "studio-account-dj", "Studio Account DJ", "dj@studio-account-test.example", models.BounceCastDJScopeKey, "correct-password")

	recorder, loginResponse := loginBounceCastStudioStreamer(t, `{"login":"dj@studio-account-test.example","password":"correct-password"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("public dj login status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if loginResponse.Token == "" {
		t.Fatal("expected session token")
	}
	if loginResponse.Streamer.Email != "dj@studio-account-test.example" {
		t.Fatalf("streamer email = %q, want dj@studio-account-test.example", loginResponse.Streamer.Email)
	}
	if loginResponse.Streamer.Handle != "studio-account-dj" {
		t.Fatalf("streamer handle = %q, want studio-account-dj", loginResponse.Streamer.Handle)
	}
	if loginResponse.Streamer.AvatarURL != "/img/test-dj.png" {
		t.Fatalf("avatar url = %q, want /img/test-dj.png", loginResponse.Streamer.AvatarURL)
	}

	var status string
	if err := data.GetDatabase().QueryRow(`
		SELECT status
		FROM bouncecast_streamer_accounts
		WHERE email = 'dj@studio-account-test.example'
	`).Scan(&status); err != nil {
		t.Fatalf("read provisioned streamer: %v", err)
	}
	if status != "active" {
		t.Fatalf("provisioned streamer status = %q, want active", status)
	}
}

func TestBounceCastStudioLoginRejectsPublicAccountWithoutDJRole(t *testing.T) {
	resetBounceCastStudioAuthTestTables(t)
	insertBounceCastStudioAuthPublicUser(t, "studio-account-viewer", "Studio Account Viewer", "viewer@studio-account-test.example", "", "correct-password")

	recorder, _ := loginBounceCastStudioStreamer(t, `{"login":"viewer@studio-account-test.example","password":"correct-password"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("viewer public account studio login status = %d, want 401", recorder.Code)
	}
}

func TestBounceCastStudioLoginDeniesLinkedStreamerWhenDJRoleRemoved(t *testing.T) {
	resetBounceCastStudioAuthTestTables(t)
	insertBounceCastStudioAuthStreamer(t, "linked-dj", "linked@studio-account-test.example", "active", "correct-password")
	insertBounceCastStudioAuthPublicUser(t, "studio-account-linked", "Linked Viewer", "linked@studio-account-test.example", "", "correct-password")

	recorder, _ := loginBounceCastStudioStreamer(t, `{"login":"linked-dj","password":"correct-password"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("linked streamer without dj role status = %d, want 401", recorder.Code)
	}
}
