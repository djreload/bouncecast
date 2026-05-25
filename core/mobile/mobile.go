package mobile

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/owncast/owncast/models"
)

const PublicConfigVersion = 1

var (
	colorPattern   = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	versionPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+){0,2}([+-][0-9A-Za-z.-]+)?$`)
)

type Runtime struct {
	BaseURL                   string
	ServerName                string
	ServerSummary             string
	StreamTitle               string
	OfflineMessage            string
	LogoURL                   string
	FaviconURL                string
	SocketHostOverride        string
	ChatDisabled              bool
	ChatRequireAuthentication bool
	ChatBackgroundImageURL    string
	ChatBackgroundOpacity     float64
	ChatTenorAPIKey           string
	AppearanceVariables       map[string]string
	MaxSocketPayloadSize      int
	HideViewerCount           bool
	StarsEnabled              bool
	StarsOverlayEnabled       bool
	ChatReactionsEnabled      bool
	RewardWheelEnabled        bool
	RewardOverlayEnabled      bool
	Status                    models.Status
}

func GetAdminSettings(db *sql.DB) (models.MobileAdminSettings, error) {
	if db == nil {
		return models.MobileAdminSettings{}, errors.New("database is not available")
	}

	settings := defaultAdminSettings()
	if err := scanAppSettings(db, &settings.App); err != nil {
		return settings, err
	}
	if err := scanBrandingSettings(db, &settings.Branding); err != nil {
		return settings, err
	}
	if err := scanFeatureFlags(db, &settings.Features); err != nil {
		return settings, err
	}
	if err := scanAdSettings(db, &settings.Ads); err != nil {
		return settings, err
	}
	if err := scanNotificationSettings(db, &settings.Notifications); err != nil {
		return settings, err
	}

	nav, err := listNavigationItems(db, false)
	if err != nil {
		return settings, err
	}
	settings.Navigation = nav

	legal, err := listLegalPages(db, false)
	if err != nil {
		return settings, err
	}
	settings.Legal = legal

	_ = db.QueryRow(`SELECT COUNT(*) FROM mobile_device_tokens WHERE enabled = 1`).Scan(&settings.DeviceCount)
	applyEnvironmentDefaults(&settings)
	return settings, nil
}

func SaveAdminSettings(db *sql.DB, settings models.MobileAdminSettings) (models.MobileAdminSettings, error) {
	if db == nil {
		return models.MobileAdminSettings{}, errors.New("database is not available")
	}
	normalized, err := normalizeAdminSettings(settings)
	if err != nil {
		return models.MobileAdminSettings{}, err
	}

	tx, err := db.Begin()
	if err != nil {
		return models.MobileAdminSettings{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(`
		UPDATE mobile_app_settings SET
			app_name = ?,
			public_base_url = ?,
			maintenance_mode = ?,
			maintenance_message = ?,
			minimum_supported_version = ?,
			recommended_version = ?,
			force_update = ?,
			force_update_message = ?,
			homepage_message = ?,
			support_url = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, normalized.App.Name, emptyToNull(normalized.App.PublicBaseURL), boolInt(normalized.App.MaintenanceMode),
		emptyToNull(normalized.App.MaintenanceMessage), normalized.App.MinimumSupportedVersion,
		emptyToNull(normalized.App.RecommendedVersion), boolInt(normalized.App.ForceUpdate),
		emptyToNull(normalized.App.ForceUpdateMessage), emptyToNull(normalized.App.HomepageMessage),
		emptyToNull(normalized.App.SupportURL)); err != nil {
		return models.MobileAdminSettings{}, err
	}

	if _, err := tx.Exec(`
		UPDATE mobile_branding_settings SET
			logo_url = ?,
			splash_url = ?,
			app_icon_url = ?,
			app_background_url = ?,
			primary_color = ?,
			accent_color = ?,
			background_color = ?,
			theme_mode = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, emptyToNull(normalized.Branding.LogoURL), emptyToNull(normalized.Branding.SplashURL),
		emptyToNull(normalized.Branding.AppIconURL), emptyToNull(normalized.Branding.AppBackgroundURL),
		normalized.Branding.PrimaryColor,
		normalized.Branding.AccentColor, normalized.Branding.BackgroundColor,
		normalized.Branding.ThemeMode); err != nil {
		return models.MobileAdminSettings{}, err
	}

	if _, err := tx.Exec(`
		UPDATE mobile_feature_flags SET
			chat_enabled = ?,
			gif_picker_enabled = ?,
			stickers_enabled = ?,
			profiles_enabled = ?,
			stars_enabled = ?,
			chat_reactions_enabled = ?,
			stars_overlay_enabled = ?,
			reward_wheel_enabled = ?,
			reward_overlay_enabled = ?,
			push_notifications_enabled = ?,
			ads_enabled = ?,
			experimental_features_enabled = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, boolInt(normalized.Features.Chat), boolInt(normalized.Features.GIFPicker),
		boolInt(normalized.Features.Stickers), boolInt(normalized.Features.Profiles),
		boolInt(normalized.Features.Stars), boolInt(normalized.Features.ChatReactions),
		boolInt(normalized.Features.StarsOverlay), boolInt(normalized.Features.RewardWheel),
		boolInt(normalized.Features.RewardOverlay),
		boolInt(normalized.Features.PushNotifications), boolInt(normalized.Features.Ads),
		boolInt(normalized.Features.ExperimentalFeatures)); err != nil {
		return models.MobileAdminSettings{}, err
	}

	priorityJSON, _ := json.Marshal(normalized.Ads.ProviderPriority)
	if _, err := tx.Exec(`
		UPDATE mobile_ad_settings SET
			enabled = ?,
			test_mode = ?,
			fallback_enabled = ?,
			provider_priority = ?,
			google_enabled = ?,
			unity_enabled = ?,
			google_app_id = ?,
			google_banner_ad_unit_id = ?,
			google_app_open_ad_unit_id = ?,
			unity_game_id = ?,
			unity_banner_placement_id = ?,
			unity_app_open_placement_id = ?,
			banner_enabled = ?,
			banner_position = ?,
			app_open_enabled = ?,
			app_open_cooldown_minutes = ?,
			app_open_show_on_first_launch = ?,
			timeout_ms = ?,
			retry_limit = ?,
			consent_required = ?,
			personalized_ads_allowed = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, boolInt(normalized.Ads.Enabled), boolInt(normalized.Ads.TestMode), boolInt(normalized.Ads.FallbackEnabled),
		string(priorityJSON), boolInt(normalized.Ads.GoogleEnabled), boolInt(normalized.Ads.UnityEnabled),
		emptyToNull(normalized.Ads.GoogleAppID), emptyToNull(normalized.Ads.GoogleBannerAdUnitID),
		emptyToNull(normalized.Ads.GoogleAppOpenAdUnitID), emptyToNull(normalized.Ads.UnityGameID),
		emptyToNull(normalized.Ads.UnityBannerPlacementID), emptyToNull(normalized.Ads.UnityAppOpenPlacementID),
		boolInt(normalized.Ads.BannerEnabled), normalized.Ads.BannerPosition,
		boolInt(normalized.Ads.AppOpenEnabled), normalized.Ads.AppOpenCooldownMinutes,
		boolInt(normalized.Ads.AppOpenShowOnFirstLaunch), normalized.Ads.TimeoutMS,
		normalized.Ads.RetryLimit, boolInt(normalized.Ads.ConsentRequired),
		boolInt(normalized.Ads.PersonalizedAdsAllowed)); err != nil {
		return models.MobileAdminSettings{}, err
	}

	if _, err := tx.Exec(`
		UPDATE mobile_notification_settings SET
			go_live_enabled = ?,
			title_template = ?,
			body_template = ?,
			image_url = ?,
			icon_url = ?,
			topic = ?,
			fcm_project_id = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, boolInt(normalized.Notifications.GoLiveEnabled), normalized.Notifications.TitleTemplate,
		normalized.Notifications.BodyTemplate, emptyToNull(normalized.Notifications.ImageURL),
		emptyToNull(normalized.Notifications.IconURL), normalized.Notifications.Topic,
		emptyToNull(normalized.Notifications.FCMProjectID)); err != nil {
		return models.MobileAdminSettings{}, err
	}

	if _, err := tx.Exec(`DELETE FROM mobile_navigation_items`); err != nil {
		return models.MobileAdminSettings{}, err
	}
	for _, item := range normalized.Navigation {
		if _, err := tx.Exec(`
			INSERT INTO mobile_navigation_items(label, url, item_type, display_order, enabled)
			VALUES(?, ?, ?, ?, ?)
		`, item.Label, item.URL, item.ItemType, item.DisplayOrder, boolInt(item.Enabled)); err != nil {
			return models.MobileAdminSettings{}, err
		}
	}

	if _, err := tx.Exec(`DELETE FROM mobile_legal_pages`); err != nil {
		return models.MobileAdminSettings{}, err
	}
	for _, page := range normalized.Legal {
		if _, err := tx.Exec(`
			INSERT INTO mobile_legal_pages(slug, title, content, url, display_order, enabled)
			VALUES(?, ?, ?, ?, ?, ?)
		`, page.Slug, page.Title, emptyToNull(page.Content), emptyToNull(page.URL),
			page.DisplayOrder, boolInt(page.Enabled)); err != nil {
			return models.MobileAdminSettings{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return models.MobileAdminSettings{}, err
	}
	return GetAdminSettings(db)
}

func GetPublicConfig(db *sql.DB, runtime Runtime) (models.MobileConfigResponse, error) {
	settings, err := GetAdminSettings(db)
	if err != nil {
		return models.MobileConfigResponse{}, err
	}
	return BuildPublicConfig(settings, runtime), nil
}

func BuildPublicConfig(settings models.MobileAdminSettings, runtime Runtime) models.MobileConfigResponse {
	applyRuntimeDefaults(&settings, runtime)
	baseURL := resolveBaseURL(settings.App.PublicBaseURL, runtime.BaseURL)
	settings.Branding.LogoURL = absoluteOrEmpty(baseURL, settings.Branding.LogoURL)
	settings.Branding.SplashURL = absoluteOrEmpty(baseURL, settings.Branding.SplashURL)
	settings.Branding.AppIconURL = absoluteOrEmpty(baseURL, settings.Branding.AppIconURL)
	settings.Branding.AppBackgroundURL = absoluteOrEmpty(baseURL, settings.Branding.AppBackgroundURL)
	settings.Notifications.ImageURL = absoluteOrEmpty(baseURL, settings.Notifications.ImageURL)
	settings.Notifications.IconURL = absoluteOrEmpty(baseURL, settings.Notifications.IconURL)
	status := runtime.Status

	streamStatus := models.MobileStreamStatus{
		Online:      status.Online,
		StreamTitle: coalesce(status.StreamTitle, runtime.StreamTitle),
		ViewerCount: status.ViewerCount,
		ServerTime:  time.Now().UTC(),
	}
	if runtime.HideViewerCount {
		streamStatus.ViewerCount = 0
	}
	if status.LastConnectTime != nil && status.LastConnectTime.Valid {
		t := status.LastConnectTime.Time
		streamStatus.LastLiveAt = &t
	}
	if status.LastDisconnectTime != nil && status.LastDisconnectTime.Valid {
		t := status.LastDisconnectTime.Time
		streamStatus.LastOfflineAt = &t
	}

	if runtime.ChatDisabled {
		settings.Features.Chat = false
	}
	settings.Features.ChatReactions = settings.Features.ChatReactions && runtime.ChatReactionsEnabled
	settings.Features.Stars = settings.Features.Stars && runtime.StarsEnabled
	settings.Features.StarsOverlay = settings.Features.Stars && settings.Features.StarsOverlay && runtime.StarsOverlayEnabled
	settings.Features.RewardWheel = settings.Features.RewardWheel && runtime.RewardWheelEnabled
	settings.Features.RewardOverlay = settings.Features.RewardWheel && settings.Features.RewardOverlay && runtime.RewardOverlayEnabled
	if !settings.Features.Ads {
		settings.Ads.Enabled = false
	}
	if !settings.Features.PushNotifications {
		settings.Notifications.GoLiveEnabled = false
	}

	settings.Navigation = enabledNavigation(settings.Navigation)
	settings.Legal = enabledLegal(settings.Legal)

	return models.MobileConfigResponse{
		Version:     PublicConfigVersion,
		GeneratedAt: time.Now().UTC(),
		App:         settings.App,
		Branding:    settings.Branding,
		BounceCast: models.MobileBounceCastConnection{
			BaseURL:          baseURL,
			StreamURL:        joinURL(baseURL, "/hls/stream.m3u8"),
			APIURL:           joinURL(baseURL, "/api"),
			ChatWebSocketURL: chatWebSocketURL(baseURL, runtime.SocketHostOverride),
			StreamStatusURL:  joinURL(baseURL, "/api/mobile/v1/stream-status"),
			ChatHistoryURL:   joinURL(baseURL, "/api/chat"),
			RegisterChatURL:  joinURL(baseURL, "/api/chat/register"),
		},
		Features:      settings.Features,
		Ads:           settings.Ads,
		Notifications: settings.Notifications,
		Navigation:    settings.Navigation,
		Legal:         settings.Legal,
		Stream:        streamStatus,
		Chat: models.MobileChatConfig{
			Enabled:               settings.Features.Chat && !runtime.ChatDisabled,
			RequireAuthentication: runtime.ChatRequireAuthentication,
			MaxSocketPayloadSize:  runtime.MaxSocketPayloadSize,
			BackgroundImageURL:    absoluteOrEmpty(baseURL, runtime.ChatBackgroundImageURL),
			BackgroundOpacity:     runtime.ChatBackgroundOpacity,
			TenorEnabled:          strings.TrimSpace(runtime.ChatTenorAPIKey) != "",
			TenorAPIKey:           runtime.ChatTenorAPIKey,
			AppearanceVariables:   runtime.AppearanceVariables,
		},
	}
}

func StreamStatusFromConfig(settings models.MobileAdminSettings, runtime Runtime) models.MobileStreamStatus {
	return BuildPublicConfig(settings, runtime).Stream
}

func RegisterDevice(db *sql.DB, request models.MobileDeviceRegistrationRequest, userID string) error {
	if db == nil {
		return errors.New("database is not available")
	}
	token := strings.TrimSpace(request.Token)
	if len(token) < 20 || len(token) > 4096 {
		return errors.New("device token is invalid")
	}
	platform := strings.ToLower(strings.TrimSpace(request.Platform))
	if platform == "" {
		platform = "android"
	}
	if platform != "android" {
		return errors.New("only android device tokens are supported in this mobile foundation")
	}
	hash := tokenHash(token)
	preview := tokenPreview(token)
	protected := protectSecret(token)
	enabled := request.NotificationsEnabled

	_, err := db.Exec(`
		INSERT INTO mobile_device_tokens(
			user_id, platform, device_id, token_protected, token_hash, token_preview,
			enabled, notifications_enabled, app_version, locale, last_seen_at, updated_at
		)
		VALUES(?, ?, ?, ?, ?, ?, 1, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(token_hash) DO UPDATE SET
			user_id = excluded.user_id,
			platform = excluded.platform,
			device_id = excluded.device_id,
			token_protected = excluded.token_protected,
			token_preview = excluded.token_preview,
			enabled = 1,
			notifications_enabled = excluded.notifications_enabled,
			app_version = excluded.app_version,
			locale = excluded.locale,
			last_seen_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
	`, emptyToNull(userID), platform, emptyToNull(request.DeviceID), protected, hash, preview,
		boolInt(enabled), emptyToNull(request.AppVersion), emptyToNull(request.Locale))
	return err
}

func UnregisterDevice(db *sql.DB, token string) error {
	if db == nil {
		return errors.New("database is not available")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("device token is required")
	}
	_, err := db.Exec(`
		UPDATE mobile_device_tokens SET
			enabled = 0,
			notifications_enabled = 0,
			updated_at = CURRENT_TIMESTAMP
		WHERE token_hash = ?
	`, tokenHash(token))
	return err
}

func UpdateDevicePreferences(db *sql.DB, request models.MobileDevicePreferencesRequest) error {
	if db == nil {
		return errors.New("database is not available")
	}
	token := strings.TrimSpace(request.Token)
	if token == "" {
		return errors.New("device token is required")
	}
	_, err := db.Exec(`
		UPDATE mobile_device_tokens SET
			notifications_enabled = ?,
			enabled = CASE WHEN ? = 1 THEN 1 ELSE enabled END,
			updated_at = CURRENT_TIMESTAMP
		WHERE token_hash = ?
	`, boolInt(request.NotificationsEnabled), boolInt(request.NotificationsEnabled), tokenHash(token))
	return err
}

func scanAppSettings(db *sql.DB, app *models.MobileAppSettings) error {
	var baseURL, maintenance, recommended, forceMessage, homepage, support sql.NullString
	var maintenanceMode, forceUpdate int
	err := db.QueryRow(`
		SELECT app_name, public_base_url, maintenance_mode, maintenance_message,
			minimum_supported_version, recommended_version, force_update,
			force_update_message, homepage_message, support_url
		FROM mobile_app_settings WHERE id = 1
	`).Scan(&app.Name, &baseURL, &maintenanceMode, &maintenance,
		&app.MinimumSupportedVersion, &recommended, &forceUpdate, &forceMessage, &homepage, &support)
	if err != nil {
		return err
	}
	app.PublicBaseURL = nullString(baseURL)
	app.MaintenanceMode = intBool(maintenanceMode)
	app.MaintenanceMessage = nullString(maintenance)
	app.RecommendedVersion = nullString(recommended)
	app.ForceUpdate = intBool(forceUpdate)
	app.ForceUpdateMessage = nullString(forceMessage)
	app.HomepageMessage = nullString(homepage)
	app.SupportURL = nullString(support)
	return nil
}

func scanBrandingSettings(db *sql.DB, branding *models.MobileBrandingSettings) error {
	var logo, splash, appIcon, appBackground sql.NullString
	err := db.QueryRow(`
		SELECT logo_url, splash_url, app_icon_url, app_background_url, primary_color, accent_color,
			background_color, theme_mode
		FROM mobile_branding_settings WHERE id = 1
	`).Scan(&logo, &splash, &appIcon, &appBackground, &branding.PrimaryColor,
		&branding.AccentColor, &branding.BackgroundColor, &branding.ThemeMode)
	branding.LogoURL = nullString(logo)
	branding.SplashURL = nullString(splash)
	branding.AppIconURL = nullString(appIcon)
	branding.AppBackgroundURL = nullString(appBackground)
	return err
}

func scanFeatureFlags(db *sql.DB, features *models.MobileFeatureFlags) error {
	var chat, gifs, stickers, profiles, stars, reactions, starsOverlay, rewardWheel, rewardOverlay, push, ads, experimental int
	err := db.QueryRow(`
		SELECT chat_enabled, gif_picker_enabled, stickers_enabled, profiles_enabled,
			stars_enabled, chat_reactions_enabled, stars_overlay_enabled, reward_wheel_enabled,
			reward_overlay_enabled, push_notifications_enabled, ads_enabled, experimental_features_enabled
		FROM mobile_feature_flags WHERE id = 1
	`).Scan(&chat, &gifs, &stickers, &profiles, &stars, &reactions, &starsOverlay,
		&rewardWheel, &rewardOverlay, &push, &ads, &experimental)
	features.Chat = intBool(chat)
	features.GIFPicker = intBool(gifs)
	features.Stickers = intBool(stickers)
	features.Profiles = intBool(profiles)
	features.Stars = intBool(stars)
	features.ChatReactions = intBool(reactions)
	features.StarsOverlay = intBool(starsOverlay)
	features.RewardWheel = intBool(rewardWheel)
	features.RewardOverlay = intBool(rewardOverlay)
	features.PushNotifications = intBool(push)
	features.Ads = intBool(ads)
	features.ExperimentalFeatures = intBool(experimental)
	return err
}

func scanAdSettings(db *sql.DB, ads *models.MobileAdSettings) error {
	var enabled, testMode, fallback, google, unity, banner, appOpen, firstLaunch, consent, personalized int
	var priority string
	var googleApp, googleBanner, googleAppOpen, unityGame, unityBanner, unityOpen sql.NullString
	err := db.QueryRow(`
		SELECT enabled, test_mode, fallback_enabled, provider_priority, google_enabled,
			unity_enabled, google_app_id, google_banner_ad_unit_id,
			google_app_open_ad_unit_id, unity_game_id, unity_banner_placement_id,
			unity_app_open_placement_id, banner_enabled, banner_position,
			app_open_enabled, app_open_cooldown_minutes, app_open_show_on_first_launch,
			timeout_ms, retry_limit, consent_required, personalized_ads_allowed
		FROM mobile_ad_settings WHERE id = 1
	`).Scan(&enabled, &testMode, &fallback, &priority, &google, &unity,
		&googleApp, &googleBanner, &googleAppOpen, &unityGame, &unityBanner, &unityOpen,
		&banner, &ads.BannerPosition, &appOpen, &ads.AppOpenCooldownMinutes,
		&firstLaunch, &ads.TimeoutMS, &ads.RetryLimit, &consent, &personalized)
	if err != nil {
		return err
	}
	ads.Enabled = intBool(enabled)
	ads.TestMode = intBool(testMode)
	ads.FallbackEnabled = intBool(fallback)
	ads.ProviderPriority = parseProviderPriority(priority)
	ads.GoogleEnabled = intBool(google)
	ads.UnityEnabled = intBool(unity)
	ads.GoogleAppID = nullString(googleApp)
	ads.GoogleBannerAdUnitID = nullString(googleBanner)
	ads.GoogleAppOpenAdUnitID = nullString(googleAppOpen)
	ads.UnityGameID = nullString(unityGame)
	ads.UnityBannerPlacementID = nullString(unityBanner)
	ads.UnityAppOpenPlacementID = nullString(unityOpen)
	ads.BannerEnabled = intBool(banner)
	ads.AppOpenEnabled = intBool(appOpen)
	ads.AppOpenShowOnFirstLaunch = intBool(firstLaunch)
	ads.ConsentRequired = intBool(consent)
	ads.PersonalizedAdsAllowed = intBool(personalized)
	return nil
}

func scanNotificationSettings(db *sql.DB, notifications *models.MobileNotificationSettings) error {
	var enabled int
	var image, icon, projectID, lastError sql.NullString
	var lastSuccess sql.NullTime
	err := db.QueryRow(`
		SELECT go_live_enabled, title_template, body_template, image_url, icon_url,
			topic, fcm_project_id, last_successful_send_at, last_error
		FROM mobile_notification_settings WHERE id = 1
	`).Scan(&enabled, &notifications.TitleTemplate, &notifications.BodyTemplate,
		&image, &icon, &notifications.Topic, &projectID, &lastSuccess, &lastError)
	if err != nil {
		return err
	}
	notifications.GoLiveEnabled = intBool(enabled)
	notifications.ImageURL = nullString(image)
	notifications.IconURL = nullString(icon)
	notifications.FCMProjectID = nullString(projectID)
	if lastSuccess.Valid {
		notifications.LastSuccessfulSendAt = &lastSuccess.Time
	}
	notifications.LastError = nullString(lastError)
	return nil
}

func listNavigationItems(db *sql.DB, enabledOnly bool) ([]models.MobileNavigationItem, error) {
	query := `SELECT id, label, url, item_type, display_order, enabled FROM mobile_navigation_items`
	if enabledOnly {
		query += ` WHERE enabled = 1`
	}
	query += ` ORDER BY display_order ASC, id ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.MobileNavigationItem{}
	for rows.Next() {
		var item models.MobileNavigationItem
		var enabled int
		if err := rows.Scan(&item.ID, &item.Label, &item.URL, &item.ItemType, &item.DisplayOrder, &enabled); err != nil {
			return nil, err
		}
		item.Enabled = intBool(enabled)
		items = append(items, item)
	}
	return items, rows.Err()
}

func listLegalPages(db *sql.DB, enabledOnly bool) ([]models.MobileLegalPage, error) {
	query := `SELECT id, slug, title, content, url, display_order, enabled FROM mobile_legal_pages`
	if enabledOnly {
		query += ` WHERE enabled = 1`
	}
	query += ` ORDER BY display_order ASC, id ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pages := []models.MobileLegalPage{}
	for rows.Next() {
		var page models.MobileLegalPage
		var content, pageURL sql.NullString
		var enabled int
		if err := rows.Scan(&page.ID, &page.Slug, &page.Title, &content, &pageURL, &page.DisplayOrder, &enabled); err != nil {
			return nil, err
		}
		page.Content = nullString(content)
		page.URL = nullString(pageURL)
		page.Enabled = intBool(enabled)
		pages = append(pages, page)
	}
	return pages, rows.Err()
}

func defaultAdminSettings() models.MobileAdminSettings {
	return models.MobileAdminSettings{
		App: models.MobileAppSettings{
			Name:                    "BounceCast",
			MinimumSupportedVersion: "1.0.0",
		},
		Branding: models.MobileBrandingSettings{
			PrimaryColor:    "#080711",
			AccentColor:     "#ff2a8a",
			BackgroundColor: "#05050f",
			ThemeMode:       "system",
		},
		Features: models.MobileFeatureFlags{
			Chat:          true,
			GIFPicker:     true,
			Stickers:      true,
			Profiles:      true,
			Stars:         true,
			ChatReactions: true,
			StarsOverlay:  true,
			RewardWheel:   true,
			RewardOverlay: true,
		},
		Ads: models.MobileAdSettings{
			TestMode:                 true,
			FallbackEnabled:          true,
			ProviderPriority:         []string{"google", "unity"},
			BannerPosition:           "bottom",
			AppOpenCooldownMinutes:   30,
			AppOpenShowOnFirstLaunch: true,
			TimeoutMS:                5000,
			RetryLimit:               1,
			ConsentRequired:          true,
		},
		Notifications: models.MobileNotificationSettings{
			TitleTemplate: "BounceCast is live",
			BodyTemplate:  "{stream_title} is live now. Tap to watch.",
			Topic:         "go-live",
		},
	}
}

func normalizeAdminSettings(settings models.MobileAdminSettings) (models.MobileAdminSettings, error) {
	if strings.TrimSpace(settings.App.Name) == "" {
		settings.App.Name = "BounceCast"
	}
	settings.App.Name = truncate(strings.TrimSpace(settings.App.Name), 80)
	settings.App.PublicBaseURL = trimURL(settings.App.PublicBaseURL)
	settings.App.MaintenanceMessage = truncate(strings.TrimSpace(settings.App.MaintenanceMessage), 280)
	settings.App.MinimumSupportedVersion = strings.TrimSpace(settings.App.MinimumSupportedVersion)
	if settings.App.MinimumSupportedVersion == "" {
		settings.App.MinimumSupportedVersion = "1.0.0"
	}
	if !versionPattern.MatchString(settings.App.MinimumSupportedVersion) {
		return settings, errors.New("minimum supported version must look like a semantic version")
	}
	settings.App.RecommendedVersion = strings.TrimSpace(settings.App.RecommendedVersion)
	settings.App.ForceUpdateMessage = truncate(strings.TrimSpace(settings.App.ForceUpdateMessage), 280)
	settings.App.HomepageMessage = truncate(strings.TrimSpace(settings.App.HomepageMessage), 500)
	settings.App.SupportURL = trimURL(settings.App.SupportURL)

	settings.Branding.LogoURL = trimURL(settings.Branding.LogoURL)
	settings.Branding.SplashURL = trimURL(settings.Branding.SplashURL)
	settings.Branding.AppIconURL = trimURL(settings.Branding.AppIconURL)
	settings.Branding.AppBackgroundURL = trimURL(settings.Branding.AppBackgroundURL)
	settings.Branding.PrimaryColor = normalizeColor(settings.Branding.PrimaryColor, "#080711")
	settings.Branding.AccentColor = normalizeColor(settings.Branding.AccentColor, "#ff2a8a")
	settings.Branding.BackgroundColor = normalizeColor(settings.Branding.BackgroundColor, "#05050f")
	settings.Branding.ThemeMode = normalizeChoice(settings.Branding.ThemeMode, "system", "system", "dark", "light")

	for _, rawURL := range []string{
		settings.App.PublicBaseURL,
		settings.App.SupportURL,
		settings.Branding.LogoURL,
		settings.Branding.SplashURL,
		settings.Branding.AppIconURL,
		settings.Branding.AppBackgroundURL,
		settings.Notifications.ImageURL,
		settings.Notifications.IconURL,
	} {
		if err := validateOptionalHTTPURL(rawURL); err != nil {
			return settings, err
		}
	}

	if settings.Ads.ProviderPriority == nil {
		settings.Ads.ProviderPriority = []string{"google", "unity"}
	}
	settings.Ads.ProviderPriority = normalizeProviderPriority(settings.Ads.ProviderPriority)
	settings.Ads.BannerPosition = normalizeChoice(settings.Ads.BannerPosition, "bottom", "bottom", "top")
	if settings.Ads.AppOpenCooldownMinutes < 1 {
		settings.Ads.AppOpenCooldownMinutes = 30
	}
	if settings.Ads.TimeoutMS < 1000 {
		settings.Ads.TimeoutMS = 5000
	}
	if settings.Ads.TimeoutMS > 30000 {
		settings.Ads.TimeoutMS = 30000
	}
	if settings.Ads.RetryLimit < 0 {
		settings.Ads.RetryLimit = 0
	}
	if settings.Ads.RetryLimit > 5 {
		settings.Ads.RetryLimit = 5
	}

	settings.Notifications.TitleTemplate = truncate(strings.TrimSpace(settings.Notifications.TitleTemplate), 120)
	if settings.Notifications.TitleTemplate == "" {
		settings.Notifications.TitleTemplate = "BounceCast is live"
	}
	settings.Notifications.BodyTemplate = truncate(strings.TrimSpace(settings.Notifications.BodyTemplate), 500)
	if settings.Notifications.BodyTemplate == "" {
		settings.Notifications.BodyTemplate = "{stream_title} is live now. Tap to watch."
	}
	settings.Notifications.Topic = strings.TrimSpace(settings.Notifications.Topic)
	if settings.Notifications.Topic == "" {
		settings.Notifications.Topic = "go-live"
	}
	settings.Notifications.FCMProjectID = truncate(strings.TrimSpace(settings.Notifications.FCMProjectID), 120)

	settings.Navigation = normalizeNavigation(settings.Navigation)
	settings.Legal = normalizeLegal(settings.Legal)
	return settings, nil
}

func applyEnvironmentDefaults(settings *models.MobileAdminSettings) {
	if strings.TrimSpace(settings.App.PublicBaseURL) == "" {
		settings.App.PublicBaseURL = strings.TrimSpace(os.Getenv("BOUNCECAST_MOBILE_BASE_URL"))
	}
	if strings.TrimSpace(settings.Notifications.FCMProjectID) == "" {
		settings.Notifications.FCMProjectID = strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID"))
	}
}

func applyRuntimeDefaults(settings *models.MobileAdminSettings, runtime Runtime) {
	if strings.TrimSpace(settings.App.Name) == "" {
		settings.App.Name = coalesce(runtime.ServerName, "BounceCast")
	}
	if strings.TrimSpace(settings.App.HomepageMessage) == "" {
		settings.App.HomepageMessage = runtime.ServerSummary
	}
	if strings.TrimSpace(settings.Branding.LogoURL) == "" {
		settings.Branding.LogoURL = absoluteOrEmpty(resolveBaseURL(settings.App.PublicBaseURL, runtime.BaseURL), coalesce(runtime.LogoURL, "/logo"))
	}
	if strings.TrimSpace(settings.Branding.AppIconURL) == "" {
		settings.Branding.AppIconURL = absoluteOrEmpty(resolveBaseURL(settings.App.PublicBaseURL, runtime.BaseURL), coalesce(runtime.FaviconURL, "/favicon.ico"))
	}
	if settings.App.MinimumSupportedVersion == "" {
		settings.App.MinimumSupportedVersion = "1.0.0"
	}
	if settings.Branding.PrimaryColor == "" {
		settings.Branding.PrimaryColor = "#080711"
	}
	if settings.Branding.AccentColor == "" {
		settings.Branding.AccentColor = "#ff2a8a"
	}
	if settings.Branding.BackgroundColor == "" {
		settings.Branding.BackgroundColor = "#05050f"
	}
	if settings.Branding.ThemeMode == "" {
		settings.Branding.ThemeMode = "system"
	}
	if len(settings.Navigation) == 0 {
		settings.Navigation = []models.MobileNavigationItem{
			{Label: "Live", URL: "/", ItemType: "internal", DisplayOrder: 10, Enabled: true},
			{Label: "Schedule", URL: "/schedule", ItemType: "internal", DisplayOrder: 20, Enabled: true},
			{Label: "DJs", URL: "/djs", ItemType: "internal", DisplayOrder: 30, Enabled: true},
		}
	}
}

func normalizeNavigation(items []models.MobileNavigationItem) []models.MobileNavigationItem {
	if len(items) == 0 {
		return []models.MobileNavigationItem{
			{Label: "Live", URL: "/", ItemType: "internal", DisplayOrder: 10, Enabled: true},
			{Label: "Schedule", URL: "/schedule", ItemType: "internal", DisplayOrder: 20, Enabled: true},
			{Label: "DJs", URL: "/djs", ItemType: "internal", DisplayOrder: 30, Enabled: true},
		}
	}
	result := make([]models.MobileNavigationItem, 0, len(items))
	for i, item := range items {
		item.Label = truncate(strings.TrimSpace(item.Label), 40)
		item.URL = strings.TrimSpace(item.URL)
		item.ItemType = normalizeChoice(item.ItemType, "internal", "internal", "external")
		if item.Label == "" || item.URL == "" {
			continue
		}
		if item.DisplayOrder == 0 {
			item.DisplayOrder = (i + 1) * 10
		}
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].DisplayOrder < result[j].DisplayOrder
	})
	return result
}

func normalizeLegal(pages []models.MobileLegalPage) []models.MobileLegalPage {
	result := make([]models.MobileLegalPage, 0, len(pages))
	seen := map[string]bool{}
	for i, page := range pages {
		page.Slug = slugify(page.Slug)
		page.Title = truncate(strings.TrimSpace(page.Title), 80)
		page.Content = strings.TrimSpace(page.Content)
		page.URL = trimURL(page.URL)
		if page.Slug == "" || page.Title == "" || seen[page.Slug] {
			continue
		}
		if page.DisplayOrder == 0 {
			page.DisplayOrder = (i + 1) * 10
		}
		seen[page.Slug] = true
		result = append(result, page)
	}
	return result
}

func normalizeProviderPriority(priority []string) []string {
	result := []string{}
	seen := map[string]bool{}
	for _, provider := range priority {
		provider = strings.ToLower(strings.TrimSpace(provider))
		if provider != "google" && provider != "unity" {
			continue
		}
		if seen[provider] {
			continue
		}
		result = append(result, provider)
		seen[provider] = true
	}
	if len(result) == 0 {
		return []string{"google", "unity"}
	}
	return result
}

func parseProviderPriority(raw string) []string {
	var priority []string
	if err := json.Unmarshal([]byte(raw), &priority); err != nil {
		priority = strings.Split(raw, ",")
	}
	return normalizeProviderPriority(priority)
}

func enabledNavigation(items []models.MobileNavigationItem) []models.MobileNavigationItem {
	result := []models.MobileNavigationItem{}
	for _, item := range items {
		if item.Enabled {
			result = append(result, item)
		}
	}
	return result
}

func enabledLegal(pages []models.MobileLegalPage) []models.MobileLegalPage {
	result := []models.MobileLegalPage{}
	for _, page := range pages {
		if page.Enabled {
			result = append(result, page)
		}
	}
	return result
}

func validateOptionalHTTPURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || strings.HasPrefix(rawURL, "/") {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL %q must use http or https", rawURL)
	}
	if parsed.Host == "" {
		return fmt.Errorf("URL %q requires a host", rawURL)
	}
	return nil
}

func resolveBaseURL(setting string, runtimeBase string) string {
	for _, candidate := range []string{setting, runtimeBase, os.Getenv("BOUNCECAST_MOBILE_BASE_URL")} {
		candidate = strings.TrimRight(strings.TrimSpace(candidate), "/")
		if candidate == "" {
			continue
		}
		if parsed, err := url.Parse(candidate); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return candidate
		}
	}
	return "http://localhost:8080"
}

func joinURL(base string, path string) string {
	base = strings.TrimRight(base, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func chatWebSocketURL(baseURL string, override string) string {
	if override = strings.TrimSpace(override); override != "" {
		if strings.HasPrefix(override, "ws://") || strings.HasPrefix(override, "wss://") {
			return strings.TrimRight(override, "/") + "/ws"
		}
		if strings.HasPrefix(override, "http://") || strings.HasPrefix(override, "https://") {
			return strings.Replace(strings.TrimRight(override, "/"), "http", "ws", 1) + "/ws"
		}
		return "wss://" + strings.TrimRight(override, "/") + "/ws"
	}
	return strings.Replace(strings.TrimRight(baseURL, "/"), "http", "ws", 1) + "/ws"
}

func absoluteOrEmpty(baseURL string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return joinURL(baseURL, value)
}

func normalizeColor(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	if !colorPattern.MatchString(value) {
		return fallback
	}
	return strings.ToLower(value)
}

func normalizeChoice(value string, fallback string, allowed ...string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	return fallback
}

func trimURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func tokenPreview(token string) string {
	token = strings.TrimSpace(token)
	if len(token) <= 12 {
		return token
	}
	return token[:4] + "..." + token[len(token)-6:]
}

func protectSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	key := encryptionKey()
	if len(key) == 0 {
		return value
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return value
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return value
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return value
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(value), nil)
	return "enc:v1:" + base64.StdEncoding.EncodeToString(append(nonce, ciphertext...))
}

func encryptionKey() []byte {
	secret := strings.TrimSpace(os.Getenv("BOUNCECAST_SECRET_KEY"))
	if secret == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func nullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func emptyToNull(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func intBool(value int) bool {
	return value != 0
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func coalesce(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
