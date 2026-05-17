package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/persistence/userrepository"
)

func cleanupBounceCastAccountTestUsers(t *testing.T) {
	t.Helper()
	db := data.GetDatabase()
	_, _ = db.Exec(`DELETE FROM user_access_tokens WHERE user_id IN (
		SELECT id FROM users WHERE email LIKE '%@account-test.example' OR display_name LIKE 'Account Test%'
	)`)
	_, _ = db.Exec(`DELETE FROM users WHERE email LIKE '%@account-test.example' OR display_name LIKE 'Account Test%'`)
}

func TestBounceCastAccountRegisterLoginAndProfile(t *testing.T) {
	cleanupBounceCastAccountTestUsers(t)
	t.Cleanup(func() { cleanupBounceCastAccountTestUsers(t) })

	anonymousUser, accessToken, err := userrepository.Get().CreateAnonymousUser("Account Test Visitor")
	if err != nil {
		t.Fatalf("create anonymous user: %v", err)
	}

	registerRequest := httptest.NewRequest(http.MethodPost, "/api/bouncecast/account/register?accessToken="+accessToken, strings.NewReader(`{
		"displayName":"Account Test DJ",
		"email":"Account Test <dj@account-test.example>",
		"password":"correct-password",
		"profileImageUrl":"https://example.com/avatar.png"
	}`))
	registerRecorder := httptest.NewRecorder()
	BounceCastAccountRegister(registerRecorder, registerRequest)
	if registerRecorder.Code != http.StatusOK {
		t.Fatalf("register status = %d, want 200: %s", registerRecorder.Code, registerRecorder.Body.String())
	}

	var registerResponse bounceCastAccountResponse
	if err := json.NewDecoder(registerRecorder.Body).Decode(&registerResponse); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if registerResponse.AccessToken != accessToken {
		t.Fatalf("register access token changed during anonymous upgrade")
	}
	if registerResponse.User.ID != anonymousUser.ID {
		t.Fatalf("registered user id = %q, want %q", registerResponse.User.ID, anonymousUser.ID)
	}
	if registerResponse.User.Email != "dj@account-test.example" {
		t.Fatalf("email = %q, want normalized email", registerResponse.User.Email)
	}
	if registerResponse.User.ProfileImageURL != "https://example.com/avatar.png" {
		t.Fatalf("profile image url = %q", registerResponse.User.ProfileImageURL)
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/bouncecast/account/login", strings.NewReader(`{
		"email":"dj@account-test.example",
		"password":"correct-password"
	}`))
	loginRecorder := httptest.NewRecorder()
	BounceCastAccountLogin(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200: %s", loginRecorder.Code, loginRecorder.Body.String())
	}

	var loginResponse bounceCastAccountResponse
	if err := json.NewDecoder(loginRecorder.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginResponse.AccessToken == "" || loginResponse.AccessToken == accessToken {
		t.Fatal("expected a new login access token")
	}
	if loginResponse.User.ID != anonymousUser.ID {
		t.Fatalf("login user id = %q, want %q", loginResponse.User.ID, anonymousUser.ID)
	}

	loggedInUser := userrepository.Get().GetUserByToken(loginResponse.AccessToken)
	profileRequest := httptest.NewRequest(http.MethodPost, "/api/bouncecast/account/profile?accessToken="+loginResponse.AccessToken, strings.NewReader(`{
		"displayName":"Account Test Headliner",
		"profileImageUrl":"/public/avatar.png"
	}`))
	profileRecorder := httptest.NewRecorder()
	BounceCastAccountUpdateProfile(*loggedInUser, profileRecorder, profileRequest)
	if profileRecorder.Code != http.StatusOK {
		t.Fatalf("profile status = %d, want 200: %s", profileRecorder.Code, profileRecorder.Body.String())
	}

	updatedUser := userrepository.Get().GetUserByToken(loginResponse.AccessToken)
	if updatedUser.DisplayName != "Account Test Headliner" {
		t.Fatalf("display name = %q", updatedUser.DisplayName)
	}
	if updatedUser.ProfileImageURL != "/public/avatar.png" {
		t.Fatalf("profile image url = %q", updatedUser.ProfileImageURL)
	}
}
