package facebookmessenger

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/persistence/configrepository"
	log "github.com/sirupsen/logrus"
)

const (
	SettingEnabled                 = "facebook_messenger_enabled"
	SettingAppID                   = "facebook_messenger_app_id"
	SettingAppSecret               = "facebook_messenger_app_secret"
	SettingPageID                  = "facebook_messenger_page_id"
	SettingPageAccessToken         = "facebook_messenger_page_access_token"
	SettingWebhookVerifyToken      = "facebook_messenger_webhook_verify_token"
	SettingValidateAppSecret       = "facebook_messenger_validate_app_secret"
	SettingGraphAPIVersion         = "facebook_messenger_graph_api_version"
	SettingMessageTemplate         = "facebook_messenger_message_template"
	SettingLiveURLOverride         = "facebook_messenger_live_url_override"
	SettingButtonLabel             = "facebook_messenger_button_label"
	SettingSendDelaySeconds        = "facebook_messenger_send_delay_seconds"
	SettingCooldownSeconds         = "facebook_messenger_cooldown_seconds"
	SettingTestRecipientPSID       = "facebook_messenger_test_recipient_psid"
	SettingLastSuccessfulSendAt    = "facebook_messenger_last_successful_send_at"
	SettingLastError               = "facebook_messenger_last_error"
	SettingPagePostFallbackEnabled = "facebook_messenger_page_post_fallback_enabled"
	SettingPagePostTemplate        = "facebook_messenger_page_post_template"

	DefaultGraphAPIVersion  = "v25.0"
	DefaultButtonLabel      = "Watch Live"
	DefaultSendDelaySeconds = 30
	DefaultCooldownSeconds  = 21600
	DefaultMessageTemplate  = "\U0001F6A8 We're live now!\n{stream_title}\n\nWatch here:\n{stream_url}"
	DefaultPagePostTemplate = "We're live now! {stream_title}\n\nWatch here: {stream_url}"

	StatusActive   = "active"
	StatusPaused   = "paused"
	StatusBlocked  = "blocked"
	StatusFailed   = "failed"
	StatusOptedOut = "opted_out"
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

var (
	GraphBaseURL          = "https://graph.facebook.com"
	HTTPClient   HTTPDoer = &http.Client{Timeout: 10 * time.Second}
	Now                   = func() time.Time { return time.Now().UTC() }

	graphVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+$`)
)

type Settings struct {
	Enabled                 bool
	AppID                   string
	AppSecret               string
	PageID                  string
	PageAccessToken         string
	WebhookVerifyToken      string
	ValidateAppSecret       bool
	GraphAPIVersion         string
	MessageTemplate         string
	LiveURLOverride         string
	ButtonLabel             string
	SendDelaySeconds        int
	CooldownSeconds         int
	TestRecipientPSID       string
	LastSuccessfulSendAt    string
	LastError               string
	PagePostFallbackEnabled bool
	PagePostTemplate        string
}

type AdminSettings struct {
	Enabled                 bool   `json:"enabled"`
	AppID                   string `json:"appId"`
	AppSecretSet            bool   `json:"appSecretSet"`
	PageID                  string `json:"pageId"`
	PageAccessTokenSet      bool   `json:"pageAccessTokenSet"`
	WebhookVerifyTokenSet   bool   `json:"webhookVerifyTokenSet"`
	ValidateAppSecret       bool   `json:"validateAppSecret"`
	GraphAPIVersion         string `json:"graphApiVersion"`
	MessageTemplate         string `json:"messageTemplate"`
	LiveURLOverride         string `json:"liveUrlOverride"`
	ButtonLabel             string `json:"buttonLabel"`
	SendDelaySeconds        int    `json:"sendDelaySeconds"`
	CooldownSeconds         int    `json:"cooldownSeconds"`
	TestRecipientPSID       string `json:"testRecipientPsid"`
	LastSuccessfulSendAt    string `json:"lastSuccessfulSendAt"`
	LastError               string `json:"lastError"`
	PagePostFallbackEnabled bool   `json:"pagePostFallbackEnabled"`
	PagePostTemplate        string `json:"pagePostTemplate"`
	WebhookCallbackPath     string `json:"webhookCallbackPath"`
	OptInKeyword            string `json:"optInKeyword"`
}

type PublicConfig struct {
	Enabled      bool   `json:"enabled"`
	PageID       string `json:"pageId,omitempty"`
	PageURL      string `json:"pageUrl,omitempty"`
	OptInKeyword string `json:"optInKeyword"`
	Message      string `json:"message"`
}

type Subscriber struct {
	ID                int64      `json:"id"`
	PSID              string     `json:"psid"`
	DisplayName       string     `json:"displayName"`
	Source            string     `json:"source"`
	OptedIn           bool       `json:"optedIn"`
	OptInAt           *time.Time `json:"optInAt,omitempty"`
	OptOutAt          *time.Time `json:"optOutAt,omitempty"`
	LastInteractionAt *time.Time `json:"lastInteractionAt,omitempty"`
	LastSentAt        *time.Time `json:"lastSentAt,omitempty"`
	SendCount         int64      `json:"sendCount"`
	FailureCount      int64      `json:"failureCount"`
	Status            string     `json:"status"`
	Metadata          string     `json:"metadata"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type SubscriberStats struct {
	Total         int64 `json:"total"`
	Active        int64 `json:"active"`
	OptedOut      int64 `json:"optedOut"`
	FailedBlocked int64 `json:"failedBlocked"`
}

type Campaign struct {
	ID             int64      `json:"id"`
	TriggerType    string     `json:"triggerType"`
	GoLiveEventID  *int64     `json:"goLiveEventId,omitempty"`
	ScheduleID     *int64     `json:"scheduleId,omitempty"`
	StartedAt      time.Time  `json:"startedAt"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	AttemptedCount int64      `json:"attemptedCount"`
	SentCount      int64      `json:"sentCount"`
	SkippedCount   int64      `json:"skippedCount"`
	FailedCount    int64      `json:"failedCount"`
	Status         string     `json:"status"`
	ErrorSummary   string     `json:"errorSummary"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type CampaignRecipient struct {
	ID           int64      `json:"id"`
	CampaignID   int64      `json:"campaignId"`
	SubscriberID *int64     `json:"subscriberId,omitempty"`
	PSID         string     `json:"psid"`
	Status       string     `json:"status"`
	Error        string     `json:"error"`
	SentAt       *time.Time `json:"sentAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type PageInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TemplateContext struct {
	SiteName         string
	StreamTitle      string
	StreamURL        string
	ChannelName      string
	StartedAt        string
	FacebookPageName string
}

type Service struct {
	Settings Settings
	Client   HTTPDoer
}

type GraphAPIError struct {
	StatusCode int
	Message    string
	Type       string
	Code       int
	Subcode    int
}

func (e GraphAPIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("Graph API error %d: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("Graph API HTTP %d: %s", e.StatusCode, e.Message)
}

func ReadSettings(db *sql.DB) Settings {
	settings := Settings{
		GraphAPIVersion:  DefaultGraphAPIVersion,
		MessageTemplate:  DefaultMessageTemplate,
		ButtonLabel:      DefaultButtonLabel,
		SendDelaySeconds: DefaultSendDelaySeconds,
		CooldownSeconds:  DefaultCooldownSeconds,
		PagePostTemplate: DefaultPagePostTemplate,
	}
	if db == nil {
		return settings
	}

	settings.Enabled = boolSetting(db, SettingEnabled, "FACEBOOK_MESSENGER_ENABLED", false)
	settings.AppID = stringSetting(db, SettingAppID, "FACEBOOK_APP_ID", "")
	settings.AppSecret = secretSetting(db, SettingAppSecret, "FACEBOOK_APP_SECRET")
	settings.PageID = stringSetting(db, SettingPageID, "FACEBOOK_PAGE_ID", "")
	settings.PageAccessToken = secretSetting(db, SettingPageAccessToken, "FACEBOOK_PAGE_ACCESS_TOKEN")
	settings.WebhookVerifyToken = secretSetting(db, SettingWebhookVerifyToken, "FACEBOOK_WEBHOOK_VERIFY_TOKEN")
	settings.ValidateAppSecret = boolSetting(db, SettingValidateAppSecret, "", true)
	settings.GraphAPIVersion = stringSetting(db, SettingGraphAPIVersion, "FACEBOOK_GRAPH_API_VERSION", DefaultGraphAPIVersion)
	settings.MessageTemplate = stringSetting(db, SettingMessageTemplate, "", DefaultMessageTemplate)
	settings.LiveURLOverride = stringSetting(db, SettingLiveURLOverride, "", "")
	settings.ButtonLabel = stringSetting(db, SettingButtonLabel, "", DefaultButtonLabel)
	settings.SendDelaySeconds = intSetting(db, SettingSendDelaySeconds, DefaultSendDelaySeconds)
	settings.CooldownSeconds = intSetting(db, SettingCooldownSeconds, DefaultCooldownSeconds)
	settings.TestRecipientPSID = stringSetting(db, SettingTestRecipientPSID, "", "")
	settings.LastSuccessfulSendAt = stringSetting(db, SettingLastSuccessfulSendAt, "", "")
	settings.LastError = stringSetting(db, SettingLastError, "", "")
	settings.PagePostFallbackEnabled = boolSetting(db, SettingPagePostFallbackEnabled, "", false)
	settings.PagePostTemplate = stringSetting(db, SettingPagePostTemplate, "", DefaultPagePostTemplate)
	return settings.Normalized()
}

func (s Settings) Normalized() Settings {
	s.AppID = strings.TrimSpace(s.AppID)
	s.AppSecret = strings.TrimSpace(s.AppSecret)
	s.PageID = strings.TrimSpace(s.PageID)
	s.PageAccessToken = strings.TrimSpace(s.PageAccessToken)
	s.WebhookVerifyToken = strings.TrimSpace(s.WebhookVerifyToken)
	s.GraphAPIVersion = strings.TrimSpace(s.GraphAPIVersion)
	if s.GraphAPIVersion == "" {
		s.GraphAPIVersion = DefaultGraphAPIVersion
	}
	s.MessageTemplate = strings.TrimSpace(s.MessageTemplate)
	if s.MessageTemplate == "" {
		s.MessageTemplate = DefaultMessageTemplate
	}
	s.ButtonLabel = strings.TrimSpace(s.ButtonLabel)
	if s.ButtonLabel == "" {
		s.ButtonLabel = DefaultButtonLabel
	}
	if s.SendDelaySeconds < 0 {
		s.SendDelaySeconds = 0
	}
	if s.SendDelaySeconds > 900 {
		s.SendDelaySeconds = 900
	}
	if s.CooldownSeconds < 0 {
		s.CooldownSeconds = 0
	}
	if s.CooldownSeconds == 0 {
		s.CooldownSeconds = DefaultCooldownSeconds
	}
	if s.CooldownSeconds > 86400 {
		s.CooldownSeconds = 86400
	}
	s.TestRecipientPSID = strings.TrimSpace(s.TestRecipientPSID)
	s.LiveURLOverride = strings.TrimSpace(s.LiveURLOverride)
	s.PagePostTemplate = strings.TrimSpace(s.PagePostTemplate)
	if s.PagePostTemplate == "" {
		s.PagePostTemplate = DefaultPagePostTemplate
	}
	return s
}

func (s Settings) Validate() error {
	s = s.Normalized()
	if !graphVersionPattern.MatchString(s.GraphAPIVersion) {
		return fmt.Errorf("Graph API version must look like v25.0")
	}
	if s.SendDelaySeconds < 0 || s.SendDelaySeconds > 900 {
		return fmt.Errorf("send delay must be between 0 and 900 seconds")
	}
	if s.CooldownSeconds < 0 || s.CooldownSeconds > 86400 {
		return fmt.Errorf("cooldown must be between 0 and 86400 seconds")
	}
	if s.LiveURLOverride != "" {
		if err := validatePublicHTTPURL(s.LiveURLOverride); err != nil {
			return fmt.Errorf("live URL override: %w", err)
		}
	}
	if s.Enabled {
		if s.PageID == "" {
			return fmt.Errorf("Facebook Page ID is required when alerts are enabled")
		}
		if s.PageAccessToken == "" {
			return fmt.Errorf("Facebook Page Access Token is required when alerts are enabled")
		}
	}
	return nil
}

func (s Settings) AdminResponse() AdminSettings {
	return AdminSettings{
		Enabled:                 s.Enabled,
		AppID:                   s.AppID,
		AppSecretSet:            s.AppSecret != "",
		PageID:                  s.PageID,
		PageAccessTokenSet:      s.PageAccessToken != "",
		WebhookVerifyTokenSet:   s.WebhookVerifyToken != "",
		ValidateAppSecret:       s.ValidateAppSecret,
		GraphAPIVersion:         s.GraphAPIVersion,
		MessageTemplate:         s.MessageTemplate,
		LiveURLOverride:         s.LiveURLOverride,
		ButtonLabel:             s.ButtonLabel,
		SendDelaySeconds:        s.SendDelaySeconds,
		CooldownSeconds:         s.CooldownSeconds,
		TestRecipientPSID:       s.TestRecipientPSID,
		LastSuccessfulSendAt:    s.LastSuccessfulSendAt,
		LastError:               s.LastError,
		PagePostFallbackEnabled: s.PagePostFallbackEnabled,
		PagePostTemplate:        s.PagePostTemplate,
		WebhookCallbackPath:     "/integrations/facebook/messenger/webhook",
		OptInKeyword:            "LIVE",
	}
}

func SaveSettings(db *sql.DB, settings Settings, preserveBlankSecrets bool) error {
	if db == nil {
		return fmt.Errorf("database unavailable")
	}
	settings = settings.Normalized()
	if err := settings.Validate(); err != nil {
		return err
	}

	values := map[string]string{
		SettingEnabled:                 strconv.FormatBool(settings.Enabled),
		SettingAppID:                   settings.AppID,
		SettingPageID:                  settings.PageID,
		SettingValidateAppSecret:       strconv.FormatBool(settings.ValidateAppSecret),
		SettingGraphAPIVersion:         settings.GraphAPIVersion,
		SettingMessageTemplate:         settings.MessageTemplate,
		SettingLiveURLOverride:         settings.LiveURLOverride,
		SettingButtonLabel:             settings.ButtonLabel,
		SettingSendDelaySeconds:        strconv.Itoa(settings.SendDelaySeconds),
		SettingCooldownSeconds:         strconv.Itoa(settings.CooldownSeconds),
		SettingTestRecipientPSID:       settings.TestRecipientPSID,
		SettingPagePostFallbackEnabled: strconv.FormatBool(settings.PagePostFallbackEnabled),
		SettingPagePostTemplate:        settings.PagePostTemplate,
	}
	for key, value := range values {
		if err := setSetting(db, key, value); err != nil {
			return err
		}
	}

	secrets := map[string]string{
		SettingAppSecret:          settings.AppSecret,
		SettingPageAccessToken:    settings.PageAccessToken,
		SettingWebhookVerifyToken: settings.WebhookVerifyToken,
	}
	for key, value := range secrets {
		if preserveBlankSecrets && strings.TrimSpace(value) == "" {
			continue
		}
		if err := setSetting(db, key, protectSecret(value)); err != nil {
			return err
		}
	}
	return nil
}

func GetPublicConfig(db *sql.DB) PublicConfig {
	settings := ReadSettings(db)
	config := PublicConfig{
		Enabled:      settings.Enabled && settings.PageID != "",
		PageID:       settings.PageID,
		OptInKeyword: "LIVE",
		Message:      "Get a Messenger notification when we go live. You can opt out anytime.",
	}
	if settings.PageID != "" {
		config.PageURL = "https://m.me/" + url.PathEscape(settings.PageID) + "?ref=LIVE"
	}
	return config
}

func VerifySignature(appSecret string, body []byte, signatureHeader string) bool {
	appSecret = strings.TrimSpace(appSecret)
	signatureHeader = strings.TrimSpace(signatureHeader)
	if appSecret == "" || signatureHeader == "" {
		return false
	}
	const prefix = "sha256="
	if !strings.HasPrefix(signatureHeader, prefix) {
		return false
	}
	expectedMAC := hmac.New(sha256.New, []byte(appSecret))
	_, _ = expectedMAC.Write(body)
	expected := expectedMAC.Sum(nil)
	got, err := hex.DecodeString(strings.TrimPrefix(signatureHeader, prefix))
	if err != nil {
		return false
	}
	return hmac.Equal(got, expected)
}

func HandleIncomingMessage(db *sql.DB, psid string, messageText string, source string) (string, string, error) {
	psid = strings.TrimSpace(psid)
	if psid == "" {
		return "", "", fmt.Errorf("sender PSID is required")
	}
	keyword := strings.ToUpper(strings.TrimSpace(messageText))
	keyword = strings.Fields(keyword + " ")[0]
	if source == "" {
		source = "page_message"
	}

	switch keyword {
	case "LIVE", "START", "SUBSCRIBE":
		if err := upsertSubscriberOptIn(db, psid, source); err != nil {
			return "", "", err
		}
		return "opt_in", "You're subscribed to BounceCast Messenger live alerts. Reply STOP anytime to opt out.", nil
	case "STOP", "UNSUBSCRIBE", "CANCEL":
		if err := optOutSubscriber(db, psid); err != nil {
			return "", "", err
		}
		return "opt_out", "You're unsubscribed from BounceCast Messenger live alerts. Reply LIVE to subscribe again.", nil
	case "HELP":
		if err := touchSubscriberInteraction(db, psid, source); err != nil {
			return "", "", err
		}
		return "help", "Reply LIVE to get BounceCast go-live alerts. Reply STOP to opt out.", nil
	default:
		if err := touchSubscriberInteraction(db, psid, source); err != nil {
			return "", "", err
		}
		return "message", "Reply LIVE to get BounceCast go-live alerts, or STOP to opt out.", nil
	}
}

func QueueGoLiveAlert(goLiveEventID int64) {
	db := data.GetDatabase()
	if db == nil || goLiveEventID == 0 {
		return
	}
	settings := ReadSettings(db)
	if !settings.Enabled {
		return
	}
	delay := time.Duration(settings.SendDelaySeconds) * time.Second
	go func() {
		if delay > 0 {
			time.Sleep(delay)
		}
		if _, err := SendGoLiveCampaign(db, goLiveEventID); err != nil {
			log.Debugln("unable to send BounceCast Facebook Messenger go-live campaign", err)
			_ = setSetting(db, SettingLastError, safeError(err))
		}
	}()
}

func SendGoLiveCampaign(db *sql.DB, goLiveEventID int64) (Campaign, error) {
	settings := ReadSettings(db)
	if !settings.Enabled {
		return Campaign{}, fmt.Errorf("Facebook Messenger alerts are disabled")
	}
	if err := settings.Validate(); err != nil {
		return Campaign{}, err
	}

	context, scheduleID, err := buildTemplateContext(db, settings, goLiveEventID)
	if err != nil {
		return Campaign{}, err
	}

	campaignID, created, err := createCampaign(db, "go_live", goLiveEventID, scheduleID)
	if err != nil {
		return Campaign{}, err
	}
	if !created {
		return readCampaign(db, campaignID)
	}

	if cooldownActive(settings) {
		err := updateCampaignComplete(db, campaignID, "skipped", 0, 0, 1, 0, "cooldown active")
		if err != nil {
			return Campaign{}, err
		}
		return readCampaign(db, campaignID)
	}

	if settings.PagePostFallbackEnabled {
		go func() {
			if err := NewService(settings).CreatePagePost(RenderTemplate(settings.PagePostTemplate, context)); err != nil {
				_ = setSetting(db, SettingLastError, "Page post fallback failed: "+safeError(err))
			}
		}()
	}

	result, sendErr := sendCampaignToSubscribers(db, campaignID, settings, context)
	if sendErr != nil {
		_ = setSetting(db, SettingLastError, safeError(sendErr))
	}
	if result.SentCount > 0 {
		_ = setSetting(db, SettingLastSuccessfulSendAt, Now().Format(time.RFC3339))
		_ = setSetting(db, SettingLastError, "")
	}
	return readCampaign(db, campaignID)
}

func SendManualTest(db *sql.DB, psid string, goLiveStyle bool) (Campaign, error) {
	settings := ReadSettings(db)
	if err := settings.Validate(); err != nil {
		return Campaign{}, err
	}
	psid = strings.TrimSpace(psid)
	if psid == "" {
		psid = settings.TestRecipientPSID
	}
	if psid == "" {
		return Campaign{}, fmt.Errorf("test recipient PSID is required")
	}

	context := fallbackTemplateContext(settings)
	campaignID, _, err := createCampaign(db, "manual_test", 0, sql.NullInt64{})
	if err != nil {
		return Campaign{}, err
	}

	message := "BounceCast Messenger test message."
	if goLiveStyle {
		message = RenderTemplate(settings.MessageTemplate, context)
	}
	service := NewService(settings)
	err = service.SendButtonMessage(psid, message, settings.ButtonLabel, context.StreamURL)
	if err != nil {
		err = service.SendText(psid, message)
	}
	if err != nil {
		_ = insertCampaignRecipient(db, campaignID, 0, psid, "failed", safeError(err))
		_ = updateCampaignComplete(db, campaignID, "failed", 1, 0, 0, 1, safeError(err))
		_ = setSetting(db, SettingLastError, safeError(err))
		return readCampaign(db, campaignID)
	}
	_ = insertCampaignRecipient(db, campaignID, 0, psid, "sent", "")
	_ = updateCampaignComplete(db, campaignID, "sent", 1, 1, 0, 0, "")
	_ = setSetting(db, SettingLastSuccessfulSendAt, Now().Format(time.RFC3339))
	_ = setSetting(db, SettingLastError, "")
	return readCampaign(db, campaignID)
}

func NewService(settings Settings) Service {
	client := HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return Service{Settings: settings.Normalized(), Client: client}
}

func (s Service) SendText(psid string, message string) error {
	body := map[string]interface{}{
		"recipient": map[string]string{"id": strings.TrimSpace(psid)},
		"message":   map[string]string{"text": truncateMessengerText(message)},
	}
	return s.postGraph("/me/messages", body)
}

func (s Service) SendButtonMessage(psid string, text string, buttonTitle string, rawURL string) error {
	if err := validatePublicHTTPURL(rawURL); err != nil {
		return err
	}
	buttonTitle = strings.TrimSpace(buttonTitle)
	if buttonTitle == "" {
		buttonTitle = DefaultButtonLabel
	}
	if len(buttonTitle) > 20 {
		buttonTitle = buttonTitle[:20]
	}
	body := map[string]interface{}{
		"recipient": map[string]string{"id": strings.TrimSpace(psid)},
		"message": map[string]interface{}{
			"attachment": map[string]interface{}{
				"type": "template",
				"payload": map[string]interface{}{
					"template_type": "button",
					"text":          truncateMessengerText(text),
					"buttons": []map[string]string{
						{"type": "web_url", "url": rawURL, "title": buttonTitle},
					},
				},
			},
		},
	}
	return s.postGraph("/me/messages", body)
}

func (s Service) ValidateToken() (PageInfo, error) {
	pageID := strings.TrimSpace(s.Settings.PageID)
	if pageID == "" {
		pageID = "me"
	}
	endpoint, err := s.graphURL("/"+url.PathEscape(pageID), url.Values{"fields": []string{"id,name"}})
	if err != nil {
		return PageInfo{}, err
	}
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return PageInfo{}, err
	}
	response, err := s.Client.Do(request)
	if err != nil {
		return PageInfo{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return PageInfo{}, parseGraphAPIError(response)
	}
	var info PageInfo
	if err := json.NewDecoder(response.Body).Decode(&info); err != nil {
		return PageInfo{}, err
	}
	return info, nil
}

func (s Service) GetPageInfo() (PageInfo, error) {
	return s.ValidateToken()
}

func (s Service) FetchUserProfile(psid string) (map[string]interface{}, error) {
	endpoint, err := s.graphURL("/"+url.PathEscape(strings.TrimSpace(psid)), url.Values{"fields": []string{"first_name,last_name,profile_pic"}})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	response, err := s.Client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, parseGraphAPIError(response)
	}
	var profile map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func (s Service) CreatePagePost(message string) error {
	if strings.TrimSpace(s.Settings.PageID) == "" {
		return fmt.Errorf("Facebook Page ID is required")
	}
	body := map[string]string{"message": truncateMessengerText(message)}
	return s.postGraph("/"+url.PathEscape(s.Settings.PageID)+"/feed", body)
}

func (s Service) HandleAPIError(err error) string {
	if err == nil {
		return ""
	}
	return safeError(err)
}

func (s Service) postGraph(path string, body interface{}) error {
	if strings.TrimSpace(s.Settings.PageAccessToken) == "" {
		return fmt.Errorf("Facebook Page Access Token is not configured")
	}
	endpoint, err := s.graphURL(path, nil)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "BounceCast")
	response, err := s.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return parseGraphAPIError(response)
	}
	return nil
}

func (s Service) graphURL(path string, values url.Values) (string, error) {
	version := s.Settings.GraphAPIVersion
	if version == "" {
		version = DefaultGraphAPIVersion
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	base, err := url.Parse(strings.TrimRight(GraphBaseURL, "/") + "/" + version + path)
	if err != nil {
		return "", err
	}
	query := base.Query()
	for key, entries := range values {
		for _, entry := range entries {
			query.Add(key, entry)
		}
	}
	query.Set("access_token", s.Settings.PageAccessToken)
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func RenderTemplate(template string, context TemplateContext) string {
	if strings.TrimSpace(template) == "" {
		template = DefaultMessageTemplate
	}
	replacements := map[string]string{
		"{site_name}":          context.SiteName,
		"{stream_title}":       context.StreamTitle,
		"{stream_url}":         context.StreamURL,
		"{channel_name}":       context.ChannelName,
		"{started_at}":         context.StartedAt,
		"{facebook_page_name}": context.FacebookPageName,
		"{{site_name}}":        context.SiteName,
		"{{stream_title}}":     context.StreamTitle,
		"{{stream_url}}":       context.StreamURL,
		"{{channel_name}}":     context.ChannelName,
		"{{started_at}}":       context.StartedAt,
	}
	message := template
	for token, value := range replacements {
		message = strings.ReplaceAll(message, token, value)
	}
	return truncateMessengerText(strings.TrimSpace(message))
}

func ListSubscribers(db *sql.DB, limit int) ([]Subscriber, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := db.Query(`
		SELECT id, psid, COALESCE(display_name, ''), source, opted_in, opt_in_at, opt_out_at,
			last_interaction_at, last_sent_at, send_count, failure_count, status,
			COALESCE(metadata, ''), created_at, updated_at
		FROM bouncecast_messenger_alert_subscribers
		ORDER BY updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subscribers := []Subscriber{}
	for rows.Next() {
		subscriber, err := scanSubscriber(rows)
		if err != nil {
			return nil, err
		}
		subscribers = append(subscribers, subscriber)
	}
	return subscribers, rows.Err()
}

func GetSubscriberStats(db *sql.DB) (SubscriberStats, error) {
	var stats SubscriberStats
	if err := db.QueryRow(`SELECT COUNT(*) FROM bouncecast_messenger_alert_subscribers`).Scan(&stats.Total); err != nil {
		return stats, err
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM bouncecast_messenger_alert_subscribers WHERE opted_in = 1 AND status = 'active'`).Scan(&stats.Active); err != nil {
		return stats, err
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM bouncecast_messenger_alert_subscribers WHERE opted_in = 0 OR status = 'opted_out'`).Scan(&stats.OptedOut); err != nil {
		return stats, err
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM bouncecast_messenger_alert_subscribers WHERE status IN ('blocked', 'failed')`).Scan(&stats.FailedBlocked); err != nil {
		return stats, err
	}
	return stats, nil
}

func ListCampaigns(db *sql.DB, limit int) ([]Campaign, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.Query(`
		SELECT id, trigger_type, go_live_event_id, schedule_id, started_at, completed_at,
			attempted_count, sent_count, skipped_count, failed_count, status,
			COALESCE(error_summary, ''), created_at, updated_at
		FROM bouncecast_messenger_alert_campaigns
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	campaigns := []Campaign{}
	for rows.Next() {
		campaign, err := scanCampaign(rows)
		if err != nil {
			return nil, err
		}
		campaigns = append(campaigns, campaign)
	}
	return campaigns, rows.Err()
}

func ListCampaignRecipients(db *sql.DB, campaignID int64, limit int) ([]CampaignRecipient, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := db.Query(`
		SELECT id, campaign_id, subscriber_id, psid, status, COALESCE(error, ''), sent_at, created_at, updated_at
		FROM bouncecast_messenger_alert_campaign_recipients
		WHERE (? = 0 OR campaign_id = ?)
		ORDER BY created_at DESC
		LIMIT ?
	`, campaignID, campaignID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	recipients := []CampaignRecipient{}
	for rows.Next() {
		recipient, err := scanCampaignRecipient(rows)
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, recipient)
	}
	return recipients, rows.Err()
}

func buildTemplateContext(db *sql.DB, settings Settings, goLiveEventID int64) (TemplateContext, sql.NullInt64, error) {
	var streamer string
	var scheduleTitle sql.NullString
	var scheduleID sql.NullInt64
	var startedAt time.Time
	err := db.QueryRow(`
		SELECT COALESCE(a.display_name, ''), e.schedule_id, s.title, e.started_at
		FROM bouncecast_go_live_events e
		LEFT JOIN bouncecast_streamer_accounts a ON a.id = e.streamer_id
		LEFT JOIN bouncecast_stream_schedule s ON s.id = e.schedule_id
		WHERE e.id = ?
	`, goLiveEventID).Scan(&streamer, &scheduleID, &scheduleTitle, &startedAt)
	if err != nil {
		return TemplateContext{}, scheduleID, err
	}
	context := fallbackTemplateContext(settings)
	context.ChannelName = streamer
	if scheduleTitle.Valid && strings.TrimSpace(scheduleTitle.String) != "" {
		context.StreamTitle = strings.TrimSpace(scheduleTitle.String)
	}
	context.StartedAt = startedAt.UTC().Format(time.RFC3339)
	return context, scheduleID, nil
}

func fallbackTemplateContext(settings Settings) TemplateContext {
	siteName := "BounceCast"
	streamURL := strings.TrimSpace(settings.LiveURLOverride)
	if data.GetStore() != nil {
		configRepository := configrepository.Get()
		if name := strings.TrimSpace(configRepository.GetServerName()); name != "" {
			siteName = name
		}
		if streamURL == "" {
			streamURL = strings.TrimSpace(configRepository.GetServerURL())
		}
	}
	if streamURL == "" {
		streamURL = "/"
	}
	return TemplateContext{
		SiteName:         siteName,
		StreamTitle:      "Live DJ stream",
		StreamURL:        streamURL,
		ChannelName:      siteName,
		StartedAt:        Now().Format(time.RFC3339),
		FacebookPageName: siteName,
	}
}

func createCampaign(db *sql.DB, triggerType string, goLiveEventID int64, scheduleID sql.NullInt64) (int64, bool, error) {
	var goLiveArg interface{}
	if goLiveEventID > 0 {
		goLiveArg = goLiveEventID
	}
	result, err := db.Exec(`
		INSERT OR IGNORE INTO bouncecast_messenger_alert_campaigns(trigger_type, go_live_event_id, schedule_id)
		VALUES(?, ?, ?)
	`, triggerType, goLiveArg, scheduleID)
	if err != nil {
		return 0, false, err
	}
	if rows, _ := result.RowsAffected(); rows > 0 {
		id, _ := result.LastInsertId()
		return id, true, nil
	}
	var id int64
	err = db.QueryRow(`
		SELECT id
		FROM bouncecast_messenger_alert_campaigns
		WHERE trigger_type = ? AND go_live_event_id = ?
		LIMIT 1
	`, triggerType, goLiveEventID).Scan(&id)
	return id, false, err
}

func sendCampaignToSubscribers(db *sql.DB, campaignID int64, settings Settings, context TemplateContext) (Campaign, error) {
	rows, err := db.Query(`
		SELECT id, psid, COALESCE(metadata, ''), last_interaction_at
		FROM bouncecast_messenger_alert_subscribers
		WHERE opted_in = 1
			AND status = 'active'
		ORDER BY opt_in_at ASC
	`)
	if err != nil {
		return Campaign{}, err
	}

	type target struct {
		id              int64
		psid            string
		metadata        string
		lastInteraction sql.NullTime
	}
	targets := []target{}

	for rows.Next() {
		var target target
		if err := rows.Scan(&target.id, &target.psid, &target.metadata, &target.lastInteraction); err != nil {
			_ = rows.Close()
			return Campaign{}, err
		}
		targets = append(targets, target)
	}
	if err := rows.Close(); err != nil {
		return Campaign{}, err
	}
	if err := rows.Err(); err != nil {
		return Campaign{}, err
	}

	service := NewService(settings)
	message := RenderTemplate(settings.MessageTemplate, context)
	var attempted, sent, skipped, failed int64
	var campaignErr error

	for _, target := range targets {
		if !messengerPolicyEligible(target.lastInteraction, target.metadata) {
			skipped++
			_ = insertCampaignRecipient(db, campaignID, target.id, target.psid, "skipped", "outside Messenger 24-hour/allowed opt-in window")
			continue
		}

		attempted++
		sendErr := service.SendButtonMessage(target.psid, message, settings.ButtonLabel, context.StreamURL)
		if sendErr != nil {
			sendErr = service.SendText(target.psid, message)
		}
		if sendErr != nil {
			failed++
			summary := safeError(sendErr)
			_ = insertCampaignRecipient(db, campaignID, target.id, target.psid, "failed", summary)
			_ = markSubscriberSendFailure(db, target.id, sendErr)
			if isTokenOrPermissionError(sendErr) {
				campaignErr = sendErr
				break
			}
			continue
		}
		sent++
		_ = insertCampaignRecipient(db, campaignID, target.id, target.psid, "sent", "")
		_ = markSubscriberSent(db, target.id)
	}

	status := "sent"
	errorSummary := ""
	if campaignErr != nil {
		status = "failed"
		errorSummary = safeError(campaignErr)
	} else if sent == 0 && attempted == 0 {
		status = "skipped"
		errorSummary = "no eligible Messenger subscribers"
	} else if failed > 0 {
		status = "partial"
	}
	if err := updateCampaignComplete(db, campaignID, status, attempted, sent, skipped, failed, errorSummary); err != nil {
		return Campaign{}, err
	}
	campaign, err := readCampaign(db, campaignID)
	if err != nil {
		return Campaign{}, err
	}
	return campaign, campaignErr
}

func cooldownActive(settings Settings) bool {
	if settings.CooldownSeconds <= 0 || strings.TrimSpace(settings.LastSuccessfulSendAt) == "" {
		return false
	}
	lastSent, err := time.Parse(time.RFC3339, settings.LastSuccessfulSendAt)
	if err != nil {
		return false
	}
	return Now().Sub(lastSent) < time.Duration(settings.CooldownSeconds)*time.Second
}

func messengerPolicyEligible(lastInteraction sql.NullTime, metadata string) bool {
	if metadataAllowsOutOfWindow(metadata) {
		return true
	}
	if !lastInteraction.Valid {
		return false
	}
	return Now().Sub(lastInteraction.Time.UTC()) <= 24*time.Hour
}

func metadataAllowsOutOfWindow(metadata string) bool {
	var decoded map[string]interface{}
	if json.Unmarshal([]byte(metadata), &decoded) != nil {
		return false
	}
	for _, key := range []string{"messenger_notification_optin", "allowed_notification", "one_time_notification"} {
		if value, ok := decoded[key].(bool); ok && value {
			return true
		}
	}
	return false
}

func insertCampaignRecipient(db *sql.DB, campaignID int64, subscriberID int64, psid string, status string, errText string) error {
	var subscriberArg interface{}
	if subscriberID > 0 {
		subscriberArg = subscriberID
	}
	var sentArg interface{}
	if status == "sent" {
		sentArg = Now()
	}
	_, err := db.Exec(`
		INSERT INTO bouncecast_messenger_alert_campaign_recipients(campaign_id, subscriber_id, psid, status, error, sent_at)
		VALUES(?, ?, ?, ?, NULLIF(?, ''), ?)
		ON CONFLICT(campaign_id, psid) DO UPDATE SET
			status = excluded.status,
			error = excluded.error,
			sent_at = excluded.sent_at,
			updated_at = CURRENT_TIMESTAMP
	`, campaignID, subscriberArg, psid, status, safeErrorString(errText), sentArg)
	return err
}

func updateCampaignComplete(db *sql.DB, campaignID int64, status string, attempted int64, sent int64, skipped int64, failed int64, errorSummary string) error {
	_, err := db.Exec(`
		UPDATE bouncecast_messenger_alert_campaigns
		SET status = ?,
			attempted_count = ?,
			sent_count = ?,
			skipped_count = ?,
			failed_count = ?,
			error_summary = NULLIF(?, ''),
			completed_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, attempted, sent, skipped, failed, safeErrorString(errorSummary), campaignID)
	return err
}

func readCampaign(db *sql.DB, campaignID int64) (Campaign, error) {
	row := db.QueryRow(`
		SELECT id, trigger_type, go_live_event_id, schedule_id, started_at, completed_at,
			attempted_count, sent_count, skipped_count, failed_count, status,
			COALESCE(error_summary, ''), created_at, updated_at
		FROM bouncecast_messenger_alert_campaigns
		WHERE id = ?
	`, campaignID)
	return scanCampaign(row)
}

func upsertSubscriberOptIn(db *sql.DB, psid string, source string) error {
	_, err := db.Exec(`
		INSERT INTO bouncecast_messenger_alert_subscribers(psid, source, opted_in, opt_in_at, last_interaction_at, status)
		VALUES(?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'active')
		ON CONFLICT(psid) DO UPDATE SET
			source = excluded.source,
			opted_in = 1,
			opt_in_at = COALESCE(bouncecast_messenger_alert_subscribers.opt_in_at, CURRENT_TIMESTAMP),
			opt_out_at = NULL,
			last_interaction_at = CURRENT_TIMESTAMP,
			status = 'active',
			updated_at = CURRENT_TIMESTAMP
	`, psid, source)
	return err
}

func optOutSubscriber(db *sql.DB, psid string) error {
	_, err := db.Exec(`
		INSERT INTO bouncecast_messenger_alert_subscribers(psid, source, opted_in, opt_out_at, last_interaction_at, status)
		VALUES(?, 'page_message', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'opted_out')
		ON CONFLICT(psid) DO UPDATE SET
			opted_in = 0,
			opt_out_at = CURRENT_TIMESTAMP,
			last_interaction_at = CURRENT_TIMESTAMP,
			status = 'opted_out',
			updated_at = CURRENT_TIMESTAMP
	`, psid)
	return err
}

func touchSubscriberInteraction(db *sql.DB, psid string, source string) error {
	_, err := db.Exec(`
		INSERT INTO bouncecast_messenger_alert_subscribers(psid, source, opted_in, last_interaction_at, status)
		VALUES(?, ?, 0, CURRENT_TIMESTAMP, 'paused')
		ON CONFLICT(psid) DO UPDATE SET
			last_interaction_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
	`, psid, source)
	return err
}

func markSubscriberSent(db *sql.DB, subscriberID int64) error {
	_, err := db.Exec(`
		UPDATE bouncecast_messenger_alert_subscribers
		SET last_sent_at = CURRENT_TIMESTAMP,
			send_count = send_count + 1,
			failure_count = 0,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, subscriberID)
	return err
}

func markSubscriberSendFailure(db *sql.DB, subscriberID int64, sendErr error) error {
	status := StatusFailed
	if isBlockedError(sendErr) {
		status = StatusBlocked
	}
	_, err := db.Exec(`
		UPDATE bouncecast_messenger_alert_subscribers
		SET failure_count = failure_count + 1,
			status = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, subscriberID)
	return err
}

func parseGraphAPIError(response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	var decoded struct {
		Error struct {
			Message      string `json:"message"`
			Type         string `json:"type"`
			Code         int    `json:"code"`
			ErrorSubcode int    `json:"error_subcode"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &decoded)
	message := strings.TrimSpace(decoded.Error.Message)
	if message == "" {
		message = http.StatusText(response.StatusCode)
	}
	return GraphAPIError{
		StatusCode: response.StatusCode,
		Message:    safeErrorString(message),
		Type:       decoded.Error.Type,
		Code:       decoded.Error.Code,
		Subcode:    decoded.Error.ErrorSubcode,
	}
}

func isTokenOrPermissionError(err error) bool {
	var graphErr GraphAPIError
	if errors.As(err, &graphErr) {
		return graphErr.Code == 190 || graphErr.Code == 10 || graphErr.Code == 200 || graphErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

func isBlockedError(err error) bool {
	var graphErr GraphAPIError
	if errors.As(err, &graphErr) {
		return graphErr.Code == 551 || strings.Contains(strings.ToLower(graphErr.Message), "blocked")
	}
	return false
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanCampaign(row scanner) (Campaign, error) {
	var campaign Campaign
	var goLiveID sql.NullInt64
	var scheduleID sql.NullInt64
	var completedAt sql.NullTime
	if err := row.Scan(
		&campaign.ID,
		&campaign.TriggerType,
		&goLiveID,
		&scheduleID,
		&campaign.StartedAt,
		&completedAt,
		&campaign.AttemptedCount,
		&campaign.SentCount,
		&campaign.SkippedCount,
		&campaign.FailedCount,
		&campaign.Status,
		&campaign.ErrorSummary,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
	); err != nil {
		return Campaign{}, err
	}
	if goLiveID.Valid {
		campaign.GoLiveEventID = &goLiveID.Int64
	}
	if scheduleID.Valid {
		campaign.ScheduleID = &scheduleID.Int64
	}
	if completedAt.Valid {
		campaign.CompletedAt = &completedAt.Time
	}
	return campaign, nil
}

func scanCampaignRecipient(row scanner) (CampaignRecipient, error) {
	var recipient CampaignRecipient
	var subscriberID sql.NullInt64
	var sentAt sql.NullTime
	if err := row.Scan(&recipient.ID, &recipient.CampaignID, &subscriberID, &recipient.PSID, &recipient.Status, &recipient.Error, &sentAt, &recipient.CreatedAt, &recipient.UpdatedAt); err != nil {
		return CampaignRecipient{}, err
	}
	if subscriberID.Valid {
		recipient.SubscriberID = &subscriberID.Int64
	}
	if sentAt.Valid {
		recipient.SentAt = &sentAt.Time
	}
	return recipient, nil
}

func scanSubscriber(row scanner) (Subscriber, error) {
	var subscriber Subscriber
	var optInAt sql.NullTime
	var optOutAt sql.NullTime
	var lastInteractionAt sql.NullTime
	var lastSentAt sql.NullTime
	if err := row.Scan(
		&subscriber.ID,
		&subscriber.PSID,
		&subscriber.DisplayName,
		&subscriber.Source,
		&subscriber.OptedIn,
		&optInAt,
		&optOutAt,
		&lastInteractionAt,
		&lastSentAt,
		&subscriber.SendCount,
		&subscriber.FailureCount,
		&subscriber.Status,
		&subscriber.Metadata,
		&subscriber.CreatedAt,
		&subscriber.UpdatedAt,
	); err != nil {
		return Subscriber{}, err
	}
	if optInAt.Valid {
		subscriber.OptInAt = &optInAt.Time
	}
	if optOutAt.Valid {
		subscriber.OptOutAt = &optOutAt.Time
	}
	if lastInteractionAt.Valid {
		subscriber.LastInteractionAt = &lastInteractionAt.Time
	}
	if lastSentAt.Valid {
		subscriber.LastSentAt = &lastSentAt.Time
	}
	return subscriber, nil
}

func stringSetting(db *sql.DB, key string, envKey string, defaultValue string) string {
	if value, ok := readSetting(db, key); ok {
		return unprotectSecret(value)
	}
	if envKey != "" {
		if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
			return value
		}
	}
	return defaultValue
}

func secretSetting(db *sql.DB, key string, envKey string) string {
	return stringSetting(db, key, envKey, "")
}

func boolSetting(db *sql.DB, key string, envKey string, defaultValue bool) bool {
	raw, ok := readSetting(db, key)
	if !ok && envKey != "" {
		raw = os.Getenv(envKey)
		ok = raw != ""
	}
	if !ok {
		return defaultValue
	}
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return defaultValue
	}
	return value
}

func intSetting(db *sql.DB, key string, defaultValue int) int {
	raw, ok := readSetting(db, key)
	if !ok {
		return defaultValue
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return defaultValue
	}
	return value
}

func readSetting(db *sql.DB, key string) (string, bool) {
	var value sql.NullString
	if err := db.QueryRow(`SELECT value FROM bouncecast_notification_settings WHERE key = ?`, key).Scan(&value); err != nil {
		return "", false
	}
	if !value.Valid {
		return "", true
	}
	return value.String, true
}

func setSetting(db *sql.DB, key string, value string) error {
	_, err := db.Exec(`
		INSERT INTO bouncecast_notification_settings("key", "value", updated_at)
		VALUES(?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT("key") DO UPDATE SET
			"value" = excluded."value",
			updated_at = CURRENT_TIMESTAMP
	`, key, value)
	return err
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
	encoded := base64.StdEncoding.EncodeToString(append(nonce, ciphertext...))
	return "enc:v1:" + encoded
}

func unprotectSecret(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "enc:v1:") {
		return value
	}
	key := encryptionKey()
	if len(key) == 0 {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "enc:v1:"))
	if err != nil {
		return ""
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(raw) < gcm.NonceSize() {
		return ""
	}
	nonce := raw[:gcm.NonceSize()]
	ciphertext := raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ""
	}
	return string(plain)
}

func encryptionKey() []byte {
	secret := strings.TrimSpace(os.Getenv("BOUNCECAST_SECRET_KEY"))
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("FACEBOOK_MESSENGER_SECRET_KEY"))
	}
	if secret == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	return safeErrorString(err.Error())
}

func safeErrorString(value string) string {
	value = strings.TrimSpace(value)
	for _, secret := range []string{
		os.Getenv("FACEBOOK_PAGE_ACCESS_TOKEN"),
		os.Getenv("FACEBOOK_APP_SECRET"),
		os.Getenv("FACEBOOK_WEBHOOK_VERIFY_TOKEN"),
	} {
		if strings.TrimSpace(secret) != "" {
			value = strings.ReplaceAll(value, secret, "[secret]")
		}
	}
	if len(value) > 500 {
		return value[:500]
	}
	return value
}

func validatePublicHTTPURL(rawURL string) error {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return err
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http or https")
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("URL host is required")
	}
	return nil
}

func truncateMessengerText(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 640 {
		return value[:640]
	}
	return value
}
