package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/core/facebookmessenger"
)

func resetMessengerWebhookTestData(t *testing.T) {
	t.Helper()
	resetBounceCastRateLimitersForTesting()
	db := data.GetDatabase()
	_, _ = db.Exec(`DELETE FROM bouncecast_messenger_alert_campaign_recipients`)
	_, _ = db.Exec(`DELETE FROM bouncecast_messenger_alert_campaigns`)
	_, _ = db.Exec(`DELETE FROM bouncecast_messenger_alert_subscribers`)
	_, _ = db.Exec(`DELETE FROM bouncecast_notification_settings WHERE key LIKE 'facebook_messenger_%'`)
}

func TestFacebookMessengerWebhookVerification(t *testing.T) {
	resetMessengerWebhookTestData(t)
	if err := facebookmessenger.SaveSettings(data.GetDatabase(), facebookmessenger.Settings{
		WebhookVerifyToken: "verify-me",
		PageID:             "page-1",
		PageAccessToken:    "token",
	}, false); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/integrations/facebook/messenger/webhook?hub.mode=subscribe&hub.verify_token=verify-me&hub.challenge=hello", nil)
	FacebookMessengerWebhookVerify(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "hello" {
		t.Fatalf("verification = %d %q, want challenge", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/integrations/facebook/messenger/webhook?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=hello", nil)
	FacebookMessengerWebhookVerify(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("bad verification status = %d, want 403", recorder.Code)
	}
}

func TestFacebookMessengerWebhookSignatureAndOptIn(t *testing.T) {
	resetMessengerWebhookTestData(t)
	if err := facebookmessenger.SaveSettings(data.GetDatabase(), facebookmessenger.Settings{
		Enabled:           true,
		AppSecret:         "app-secret",
		PageID:            "page-1",
		PageAccessToken:   "token",
		ValidateAppSecret: true,
	}, false); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	body := `{"object":"page","entry":[{"messaging":[{"sender":{"id":"psid-webhook"},"message":{"text":"LIVE"}}]}]}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/integrations/facebook/messenger/webhook", strings.NewReader(body))
	request.Header.Set("X-Hub-Signature-256", messengerTestSignature("app-secret", body))
	FacebookMessengerWebhook(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("webhook status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	var optedIn bool
	var status string
	if err := data.GetDatabase().QueryRow(`SELECT opted_in, status FROM bouncecast_messenger_alert_subscribers WHERE psid = 'psid-webhook'`).Scan(&optedIn, &status); err != nil {
		t.Fatalf("read subscriber: %v", err)
	}
	if !optedIn || status != facebookmessenger.StatusActive {
		t.Fatalf("subscriber = optedIn:%t status:%q", optedIn, status)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/integrations/facebook/messenger/webhook", strings.NewReader(body))
	request.Header.Set("X-Hub-Signature-256", messengerTestSignature("wrong-secret", body))
	FacebookMessengerWebhook(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("bad signature status = %d, want 403", recorder.Code)
	}
}

func messengerTestSignature(secret string, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
