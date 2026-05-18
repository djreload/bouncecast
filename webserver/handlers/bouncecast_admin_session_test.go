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
	"github.com/owncast/owncast/webserver/router/middleware"
)

func cleanupBounceCastAdminSessionRoleUsers(t *testing.T) {
	t.Helper()
	db := data.GetDatabase()
	_, _ = db.Exec(`DELETE FROM user_access_tokens WHERE user_id IN (
		SELECT id FROM users WHERE email LIKE '%@admin-session-test.example'
	)`)
	_, _ = db.Exec(`DELETE FROM users WHERE email LIKE '%@admin-session-test.example'`)
}

func insertBounceCastAdminSessionRoleUser(t *testing.T, id string, email string, scopes string, password string) {
	t.Helper()

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := data.GetDatabase().Exec(`
		INSERT INTO users(id, display_name, display_color, previous_names, created_at, authenticated_at, scopes, email, password_hash, registered_at)
		VALUES(?, 'Admin Session Test', 1, 'Admin Session Test', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?, ?, CURRENT_TIMESTAMP)
	`, id, scopes, email, passwordHash); err != nil {
		t.Fatalf("insert role user: %v", err)
	}
}

func loginBounceCastAdminSession(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/bouncecast/admin/login", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	BounceCastAdminLogin(recorder, request)
	return recorder
}

func TestBounceCastAdminLoginAcceptsOwnerAndAdminAccountRoles(t *testing.T) {
	cleanupBounceCastAdminSessionRoleUsers(t)
	t.Cleanup(func() { cleanupBounceCastAdminSessionRoleUsers(t) })

	insertBounceCastAdminSessionRoleUser(t, "admin-session-owner", "owner@admin-session-test.example", models.BounceCastOwnerScopeKey, "correct-password")
	insertBounceCastAdminSessionRoleUser(t, "admin-session-admin", "admin@admin-session-test.example", models.BounceCastAdminScopeKey, "correct-password")

	for _, testCase := range []struct {
		email string
		role  string
	}{
		{email: "owner@admin-session-test.example", role: "owner"},
		{email: "admin@admin-session-test.example", role: "admin"},
	} {
		recorder := loginBounceCastAdminSession(t, `{"username":"`+testCase.email+`","password":"correct-password"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s login status = %d, want 200: %s", testCase.role, recorder.Code, recorder.Body.String())
		}
		if !hasBounceCastAdminSessionCookie(recorder) {
			t.Fatalf("%s login did not set admin session cookie", testCase.role)
		}

		var response map[string]string
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode %s response: %v", testCase.role, err)
		}
		if response["role"] != testCase.role {
			t.Fatalf("role = %q, want %q", response["role"], testCase.role)
		}
		if response["destination"] != "/admin/" {
			t.Fatalf("destination = %q, want /admin/", response["destination"])
		}
	}
}

func TestBounceCastAdminLoginRejectsNonAdminAccountRoles(t *testing.T) {
	cleanupBounceCastAdminSessionRoleUsers(t)
	t.Cleanup(func() { cleanupBounceCastAdminSessionRoleUsers(t) })

	insertBounceCastAdminSessionRoleUser(t, "admin-session-dj", "dj@admin-session-test.example", models.BounceCastDJScopeKey, "correct-password")

	recorder := loginBounceCastAdminSession(t, `{"username":"dj@admin-session-test.example","password":"correct-password"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("dj role admin login status = %d, want 401", recorder.Code)
	}
}

func hasBounceCastAdminSessionCookie(recorder *httptest.ResponseRecorder) bool {
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == middleware.AdminSessionCookieName {
			return true
		}
	}
	return false
}
