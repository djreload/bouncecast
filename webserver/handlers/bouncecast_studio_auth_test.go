package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/utils"
)

func resetBounceCastStudioAuthTestTables(t *testing.T) {
	t.Helper()

	db := data.GetDatabase()
	_, _ = db.Exec(`DELETE FROM bouncecast_streamer_sessions`)
	_, _ = db.Exec(`DELETE FROM bouncecast_streamer_stream_keys`)
	_, _ = db.Exec(`DELETE FROM bouncecast_streamer_accounts`)
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
