package mobile

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/migrations"
)

func openMobileTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migrations.Run(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestBuildPublicConfigUsesExistingBounceCastEndpoints(t *testing.T) {
	settings := defaultAdminSettings()
	settings.App.PublicBaseURL = "https://k-nrg.co.uk"
	settings.Features.Ads = true
	settings.Ads.Enabled = true
	settings.Navigation = []models.MobileNavigationItem{
		{Label: "Live", URL: "/", ItemType: "internal", DisplayOrder: 10, Enabled: true},
		{Label: "Hidden", URL: "/hidden", ItemType: "internal", DisplayOrder: 20, Enabled: false},
	}

	config := BuildPublicConfig(settings, Runtime{
		BaseURL:                   "https://fallback.example",
		ServerName:                "BounceCast",
		StreamTitle:               "Sunday Sessions",
		LogoURL:                   "/logo",
		FaviconURL:                "/favicon.ico",
		ChatTenorAPIKey:           "tenor-public-key",
		MaxSocketPayloadSize:      4096,
		ChatRequireAuthentication: true,
		Status: models.Status{
			Online:      true,
			StreamTitle: "Sunday Sessions",
			ViewerCount: 42,
		},
	})

	if config.BounceCast.StreamURL != "https://k-nrg.co.uk/hls/stream.m3u8" {
		t.Fatalf("stream URL = %q", config.BounceCast.StreamURL)
	}
	if config.BounceCast.ChatWebSocketURL != "wss://k-nrg.co.uk/ws" {
		t.Fatalf("chat websocket URL = %q", config.BounceCast.ChatWebSocketURL)
	}
	if config.BounceCast.ChatHistoryURL != "https://k-nrg.co.uk/api/chat" {
		t.Fatalf("chat history URL = %q", config.BounceCast.ChatHistoryURL)
	}
	if !config.Chat.Enabled || !config.Chat.RequireAuthentication || !config.Chat.TenorEnabled {
		t.Fatalf("chat config did not inherit BounceCast chat settings: %+v", config.Chat)
	}
	if len(config.Navigation) != 1 || config.Navigation[0].Label != "Live" {
		t.Fatalf("disabled navigation items leaked into public config: %+v", config.Navigation)
	}
	if config.Stream.ViewerCount != 42 {
		t.Fatalf("viewer count = %d", config.Stream.ViewerCount)
	}
}

func TestSaveAdminSettingsValidatesAndPersists(t *testing.T) {
	db := openMobileTestDB(t)

	settings, err := GetAdminSettings(db)
	if err != nil {
		t.Fatal(err)
	}
	settings.App.PublicBaseURL = "ftp://example.com"
	if _, err := SaveAdminSettings(db, settings); err == nil {
		t.Fatal("expected invalid public base URL to fail")
	}

	settings.App.PublicBaseURL = "https://app.example.com"
	settings.Branding.PrimaryColor = "#111111"
	settings.Features.PushNotifications = true
	settings.Ads.ProviderPriority = []string{"unity", "google", "bad-provider"}
	settings.Notifications.GoLiveEnabled = true
	settings.Navigation = []models.MobileNavigationItem{
		{Label: "Live", URL: "/", ItemType: "internal", DisplayOrder: 10, Enabled: true},
	}

	saved, err := SaveAdminSettings(db, settings)
	if err != nil {
		t.Fatal(err)
	}
	if saved.App.PublicBaseURL != "https://app.example.com" {
		t.Fatalf("public base URL = %q", saved.App.PublicBaseURL)
	}
	if saved.Branding.PrimaryColor != "#111111" {
		t.Fatalf("primary color = %q", saved.Branding.PrimaryColor)
	}
	if got := strings.Join(saved.Ads.ProviderPriority, ","); got != "unity,google" {
		t.Fatalf("provider priority = %q", got)
	}
	if !saved.Notifications.GoLiveEnabled {
		t.Fatal("go-live notification setting was not saved")
	}
}

func TestRegisterDeviceStoresProtectedTokenAndPreferences(t *testing.T) {
	db := openMobileTestDB(t)
	t.Setenv("BOUNCECAST_SECRET_KEY", "test-secret")

	token := "android-device-token-that-is-long-enough"
	err := RegisterDevice(db, models.MobileDeviceRegistrationRequest{
		Token:                token,
		Platform:             "android",
		DeviceID:             "pixel-test",
		AppVersion:           "1.0.0",
		Locale:               "en-GB",
		NotificationsEnabled: true,
	}, "user-1")
	if err != nil {
		t.Fatal(err)
	}

	var protected string
	var enabled int
	if err := db.QueryRow(`SELECT token_protected, notifications_enabled FROM mobile_device_tokens WHERE token_preview = ?`, tokenPreview(token)).Scan(&protected, &enabled); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(protected, "enc:v1:") || protected == token {
		t.Fatalf("device token was not protected: %q", protected)
	}
	if enabled != 1 {
		t.Fatalf("notifications enabled = %d", enabled)
	}

	if err := UpdateDevicePreferences(db, models.MobileDevicePreferencesRequest{Token: token, NotificationsEnabled: false}); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT notifications_enabled FROM mobile_device_tokens WHERE token_preview = ?`, tokenPreview(token)).Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 0 {
		t.Fatalf("notifications enabled after update = %d", enabled)
	}

	if err := UnregisterDevice(db, token); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT enabled FROM mobile_device_tokens WHERE token_preview = ?`, tokenPreview(token)).Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 0 {
		t.Fatalf("enabled after unregister = %d", enabled)
	}
}
