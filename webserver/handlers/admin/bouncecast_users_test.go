package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
)

func cleanupBounceCastUserAccountTests(t *testing.T) {
	t.Helper()
	_, _ = data.GetDatabase().Exec(`DELETE FROM users WHERE id = 'account-admin-test'`)
}

func TestSetBounceCastUserPermissions(t *testing.T) {
	cleanupBounceCastUserAccountTests(t)
	t.Cleanup(func() { cleanupBounceCastUserAccountTests(t) })

	if _, err := data.GetDatabase().Exec(`
		INSERT INTO users(id, display_name, display_color, previous_names, email, registered_at, scopes)
		VALUES('account-admin-test', 'Account Admin Test', 4, 'Account Admin Test', 'admin@account-test.example', CURRENT_TIMESTAMP, 'CUSTOM_SCOPE')
	`); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/admin/bouncecast/users/permissions", strings.NewReader(`{
		"userId":"account-admin-test",
		"permissions":["visitor","moderator","dj"]
	}`))
	recorder := httptest.NewRecorder()
	SetBounceCastUserPermissions(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("set permissions status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	var scopes string
	if err := data.GetDatabase().QueryRow(`SELECT scopes FROM users WHERE id = 'account-admin-test'`).Scan(&scopes); err != nil {
		t.Fatalf("read scopes: %v", err)
	}
	scopeMap := map[string]bool{}
	for _, scope := range strings.Split(scopes, ",") {
		scopeMap[scope] = true
	}
	if !scopeMap["CUSTOM_SCOPE"] || !scopeMap[models.ModeratorScopeKey] || !scopeMap[models.BounceCastDJScopeKey] {
		t.Fatalf("scopes not preserved/added correctly: %q", scopes)
	}
	if scopeMap["visitor"] || scopeMap["VISITOR"] {
		t.Fatalf("visitor should be a derived default, not stored: %q", scopes)
	}

	listRecorder := httptest.NewRecorder()
	GetBounceCastUsers(listRecorder, httptest.NewRequest(http.MethodGet, "/api/admin/bouncecast/users", nil))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list users status = %d, want 200: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var users []BounceCastUserAccount
	if err := json.NewDecoder(listRecorder.Body).Decode(&users); err != nil {
		t.Fatalf("decode users: %v", err)
	}

	found := false
	for _, user := range users {
		if user.ID != "account-admin-test" {
			continue
		}
		found = true
		permissions := strings.Join(user.Permissions, ",")
		if !strings.Contains(permissions, "moderator") || !strings.Contains(permissions, "dj") {
			t.Fatalf("permissions = %v", user.Permissions)
		}
	}
	if !found {
		t.Fatal("expected account-admin-test in user list")
	}
}
