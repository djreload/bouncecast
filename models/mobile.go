package models

import "time"

// MobileConfigResponse is the stable public configuration contract consumed by
// official BounceCast mobile clients. It intentionally references the existing
// BounceCast stream and chat URLs instead of modelling a separate platform.
type MobileConfigResponse struct {
	Version       int                        `json:"version"`
	GeneratedAt   time.Time                  `json:"generated_at"`
	App           MobileAppSettings          `json:"app"`
	Branding      MobileBrandingSettings     `json:"branding"`
	BounceCast    MobileBounceCastConnection `json:"bouncecast"`
	Features      MobileFeatureFlags         `json:"features"`
	Ads           MobileAdSettings           `json:"ads"`
	Notifications MobileNotificationSettings `json:"notifications"`
	Navigation    []MobileNavigationItem     `json:"navigation"`
	Legal         []MobileLegalPage          `json:"legal"`
	Stream        MobileStreamStatus         `json:"stream"`
	Chat          MobileChatConfig           `json:"chat"`
}

type MobileAdminSettings struct {
	App           MobileAppSettings          `json:"app"`
	Branding      MobileBrandingSettings     `json:"branding"`
	Features      MobileFeatureFlags         `json:"features"`
	Ads           MobileAdSettings           `json:"ads"`
	Notifications MobileNotificationSettings `json:"notifications"`
	Navigation    []MobileNavigationItem     `json:"navigation"`
	Legal         []MobileLegalPage          `json:"legal"`
	DeviceCount   int64                      `json:"deviceCount"`
	LastSavedAt   *time.Time                 `json:"lastSavedAt,omitempty"`
}

type MobileAppSettings struct {
	Name                    string `json:"name"`
	PublicBaseURL           string `json:"publicBaseUrl,omitempty"`
	MaintenanceMode         bool   `json:"maintenance_mode"`
	MaintenanceMessage      string `json:"maintenance_message,omitempty"`
	MinimumSupportedVersion string `json:"minimum_supported_version"`
	RecommendedVersion      string `json:"recommended_version,omitempty"`
	ForceUpdate             bool   `json:"force_update"`
	ForceUpdateMessage      string `json:"force_update_message,omitempty"`
	HomepageMessage         string `json:"homepage_message,omitempty"`
	SupportURL              string `json:"support_url,omitempty"`
}

type MobileBrandingSettings struct {
	LogoURL         string `json:"logo_url,omitempty"`
	SplashURL       string `json:"splash_url,omitempty"`
	AppIconURL      string `json:"app_icon_url,omitempty"`
	PrimaryColor    string `json:"primary_color"`
	AccentColor     string `json:"accent_color"`
	BackgroundColor string `json:"background_color"`
	ThemeMode       string `json:"theme_mode"`
}

type MobileBounceCastConnection struct {
	BaseURL          string `json:"base_url"`
	StreamURL        string `json:"stream_url"`
	APIURL           string `json:"api_url"`
	ChatWebSocketURL string `json:"chat_websocket_url"`
	StreamStatusURL  string `json:"stream_status_url"`
	ChatHistoryURL   string `json:"chat_history_url"`
	RegisterChatURL  string `json:"register_chat_url"`
}

type MobileFeatureFlags struct {
	Chat                 bool `json:"chat"`
	GIFPicker            bool `json:"gif_picker"`
	Stickers             bool `json:"stickers"`
	Profiles             bool `json:"profiles"`
	PushNotifications    bool `json:"push_notifications"`
	Ads                  bool `json:"ads"`
	ExperimentalFeatures bool `json:"experimental_features"`
}

type MobileAdSettings struct {
	Enabled                  bool     `json:"enabled"`
	TestMode                 bool     `json:"test_mode"`
	FallbackEnabled          bool     `json:"fallback_enabled"`
	ProviderPriority         []string `json:"priority"`
	GoogleEnabled            bool     `json:"google_enabled"`
	UnityEnabled             bool     `json:"unity_enabled"`
	GoogleAppID              string   `json:"google_app_id,omitempty"`
	GoogleBannerAdUnitID     string   `json:"google_banner_ad_unit_id,omitempty"`
	GoogleAppOpenAdUnitID    string   `json:"google_app_open_ad_unit_id,omitempty"`
	UnityGameID              string   `json:"unity_game_id,omitempty"`
	UnityBannerPlacementID   string   `json:"unity_banner_placement_id,omitempty"`
	UnityAppOpenPlacementID  string   `json:"unity_app_open_placement_id,omitempty"`
	BannerEnabled            bool     `json:"banner_enabled"`
	BannerPosition           string   `json:"banner_position"`
	AppOpenEnabled           bool     `json:"app_open_enabled"`
	AppOpenCooldownMinutes   int      `json:"app_open_cooldown_minutes"`
	AppOpenShowOnFirstLaunch bool     `json:"app_open_show_on_first_launch"`
	TimeoutMS                int      `json:"timeout_ms"`
	RetryLimit               int      `json:"retry_limit"`
	ConsentRequired          bool     `json:"consent_required"`
	PersonalizedAdsAllowed   bool     `json:"personalized_ads_allowed"`
}

type MobileNotificationSettings struct {
	GoLiveEnabled        bool       `json:"go_live_enabled"`
	TitleTemplate        string     `json:"title_template"`
	BodyTemplate         string     `json:"body_template"`
	ImageURL             string     `json:"image_url,omitempty"`
	IconURL              string     `json:"icon_url,omitempty"`
	Topic                string     `json:"topic"`
	FCMProjectID         string     `json:"fcm_project_id,omitempty"`
	LastSuccessfulSendAt *time.Time `json:"last_successful_send_at,omitempty"`
	LastError            string     `json:"last_error,omitempty"`
}

type MobileNavigationItem struct {
	ID           int64  `json:"id,omitempty"`
	Label        string `json:"label"`
	URL          string `json:"url"`
	ItemType     string `json:"item_type"`
	DisplayOrder int    `json:"display_order"`
	Enabled      bool   `json:"enabled"`
}

type MobileLegalPage struct {
	ID           int64  `json:"id,omitempty"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	Content      string `json:"content,omitempty"`
	URL          string `json:"url,omitempty"`
	DisplayOrder int    `json:"display_order"`
	Enabled      bool   `json:"enabled"`
}

type MobileStreamStatus struct {
	Online        bool       `json:"online"`
	StreamTitle   string     `json:"stream_title"`
	ViewerCount   int        `json:"viewer_count,omitempty"`
	ServerTime    time.Time  `json:"server_time"`
	LastLiveAt    *time.Time `json:"last_live_at,omitempty"`
	LastOfflineAt *time.Time `json:"last_offline_at,omitempty"`
}

type MobileChatConfig struct {
	Enabled               bool              `json:"enabled"`
	RequireAuthentication bool              `json:"require_authentication"`
	MaxSocketPayloadSize  int               `json:"max_socket_payload_size"`
	BackgroundImageURL    string            `json:"background_image_url,omitempty"`
	BackgroundOpacity     float64           `json:"background_opacity"`
	TenorEnabled          bool              `json:"tenor_enabled"`
	TenorAPIKey           string            `json:"tenor_api_key,omitempty"`
	AppearanceVariables   map[string]string `json:"appearance_variables,omitempty"`
}

type MobileDeviceRegistrationRequest struct {
	Token                string `json:"token"`
	Platform             string `json:"platform"`
	DeviceID             string `json:"deviceId"`
	AppVersion           string `json:"appVersion"`
	Locale               string `json:"locale"`
	NotificationsEnabled bool   `json:"notificationsEnabled"`
}

type MobileDevicePreferencesRequest struct {
	Token                string `json:"token"`
	NotificationsEnabled bool   `json:"notificationsEnabled"`
}
