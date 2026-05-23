package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/configrepository"
	"github.com/owncast/owncast/persistence/userrepository"
	"github.com/owncast/owncast/utils"
	"github.com/owncast/owncast/webserver/router/middleware"
)

func cleanupBounceCastUnifiedAuthTest(t *testing.T) {
	t.Helper()
	db := data.GetDatabase()
	_, _ = db.Exec(`DELETE FROM bouncecast_streamer_sessions`)
	_, _ = db.Exec(`DELETE FROM bouncecast_streamer_accounts WHERE email LIKE '%@unified-auth-test.example' OR handle LIKE 'unified-%' OR handle LIKE 'bouncecast-owner%'`)
	_, _ = db.Exec(`DELETE FROM user_access_tokens WHERE user_id IN (
		SELECT id FROM users WHERE id = 'owncast-admin' OR email LIKE '%@unified-auth-test.example'
	)`)
	_, _ = db.Exec(`DELETE FROM users WHERE id = 'owncast-admin' OR email LIKE '%@unified-auth-test.example'`)
}

func withBounceCastUnifiedAdminPassword(t *testing.T, password string) {
	t.Helper()
	oldHash := configrepository.Get().GetAdminPassword()
	if err := configrepository.Get().SetAdminPassword(password); err != nil {
		t.Fatalf("set admin password: %v", err)
	}
	t.Cleanup(func() {
		_, _ = data.GetDatabase().Exec(`
			INSERT INTO datastore(key, value) VALUES('admin_password_key', ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value
		`, oldHash)
	})
}

func insertBounceCastUnifiedUser(t *testing.T, id string, displayName string, email string, scopes string, password string) {
	t.Helper()
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := data.GetDatabase().Exec(`
		INSERT INTO users(id, display_name, display_color, previous_names, created_at, authenticated_at, scopes, email, password_hash, profile_image_url, registered_at)
		VALUES(?, ?, 1, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, NULLIF(?, ''), ?, ?, '/img/unified.png', CURRENT_TIMESTAMP)
	`, id, displayName, displayName, scopes, email, passwordHash); err != nil {
		t.Fatalf("insert unified user: %v", err)
	}
}

func unifiedLogin(t *testing.T, body string) (*httptest.ResponseRecorder, bounceCastUnifiedLoginResponse) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/bouncecast/auth/login", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	BounceCastUnifiedLogin(recorder, request)

	var response bounceCastUnifiedLoginResponse
	if recorder.Code == http.StatusOK {
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode unified login response: %v", err)
		}
	}
	return recorder, response
}

func TestBounceCastUnifiedLoginAcceptsLegacyAdminAndIssuesUnifiedSessions(t *testing.T) {
	resetBounceCastRateLimitersForTesting()
	cleanupBounceCastUnifiedAuthTest(t)
	t.Cleanup(func() { cleanupBounceCastUnifiedAuthTest(t) })
	withBounceCastUnifiedAdminPassword(t, "correct-admin-password")

	recorder, response := unifiedLogin(t, `{"login":"admin","password":"correct-admin-password","next":"/admin/viewer-info/"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unified admin login status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if response.Role != "owner" {
		t.Fatalf("role = %q, want owner", response.Role)
	}
	if response.Destination != "/admin/viewer-info/" {
		t.Fatalf("destination = %q, want requested admin page", response.Destination)
	}
	if response.AccessToken == "" {
		t.Fatal("expected normal account access token for legacy admin login")
	}
	if response.StudioToken == "" {
		t.Fatal("expected Studio token for owner-level legacy admin login")
	}
	if !hasBounceCastAdminSessionCookie(recorder) || !hasBounceCastAdminIdentityCookie(recorder) {
		t.Fatal("expected admin session and identity cookies")
	}

	user := userrepository.Get().GetUserByToken(response.AccessToken)
	if user == nil || !user.IsOwner() || !user.IsAdmin() || !user.IsDJ() || !user.IsModerator() {
		t.Fatalf("legacy admin account did not receive owner/admin/dj/moderator scopes: %+v", user)
	}
}

func TestBounceCastUnifiedLoginRoutesAccountsByRole(t *testing.T) {
	resetBounceCastRateLimitersForTesting()
	cleanupBounceCastUnifiedAuthTest(t)
	t.Cleanup(func() { cleanupBounceCastUnifiedAuthTest(t) })

	insertBounceCastUnifiedUser(t, "unified-admin", "Unified Admin", "admin@unified-auth-test.example", models.BounceCastAdminScopeKey, "correct-password")
	insertBounceCastUnifiedUser(t, "unified-dj", "Unified DJ", "dj@unified-auth-test.example", models.BounceCastDJScopeKey, "correct-password")

	adminRecorder, adminResponse := unifiedLogin(t, `{"login":"admin@unified-auth-test.example","password":"correct-password","next":"/studio"}`)
	if adminRecorder.Code != http.StatusOK {
		t.Fatalf("admin role login status = %d, want 200: %s", adminRecorder.Code, adminRecorder.Body.String())
	}
	if adminResponse.Destination != "/admin/" {
		t.Fatalf("admin next=/studio destination = %q, want /admin/", adminResponse.Destination)
	}
	if adminResponse.StudioToken != "" {
		t.Fatal("admin account without DJ scope should not get a Studio token")
	}
	if !hasBounceCastAdminSessionCookie(adminRecorder) {
		t.Fatal("admin role login did not set admin cookie")
	}

	djRecorder, djResponse := unifiedLogin(t, `{"login":"dj@unified-auth-test.example","password":"correct-password","next":"/studio"}`)
	if djRecorder.Code != http.StatusOK {
		t.Fatalf("dj role login status = %d, want 200: %s", djRecorder.Code, djRecorder.Body.String())
	}
	if djResponse.Destination != "/studio" {
		t.Fatalf("dj destination = %q, want /studio", djResponse.Destination)
	}
	if djResponse.StudioToken == "" {
		t.Fatal("dj account should receive Studio token")
	}
	if hasBounceCastAdminSessionCookie(djRecorder) {
		t.Fatal("dj-only login should not set admin cookie")
	}
}

func TestBounceCastAdminPageAuthRedirectsToUnifiedLogin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/admin/viewer-info/?check=1", nil)
	recorder := httptest.NewRecorder()

	middleware.RequireAdminPageAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("admin page unauth status = %d, want 302", recorder.Code)
	}
	location := recorder.Header().Get("Location")
	if !strings.HasPrefix(location, "/login?next=") || !strings.Contains(location, "%2Fadmin%2Fviewer-info%2F%3Fcheck%3D1") {
		t.Fatalf("unexpected redirect location: %q", location)
	}
}

func TestBounceCastUnifiedLoginRateLimitsAttempts(t *testing.T) {
	resetBounceCastRateLimitersForTesting()
	cleanupBounceCastUnifiedAuthTest(t)
	t.Cleanup(func() { cleanupBounceCastUnifiedAuthTest(t) })

	var recorder *httptest.ResponseRecorder
	for i := 0; i < 21; i++ {
		recorder, _ = unifiedLogin(t, `{"login":"missing@unified-auth-test.example","password":"wrong-password"}`)
	}
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("final repeated login status = %d, want 429", recorder.Code)
	}
}
