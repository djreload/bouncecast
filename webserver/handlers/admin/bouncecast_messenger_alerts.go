package admin

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/core/facebookmessenger"
	"github.com/owncast/owncast/core/rtmp"
	webutils "github.com/owncast/owncast/webserver/utils"
)

type bounceCastFacebookMessengerAlertsResponse struct {
	Settings    facebookmessenger.AdminSettings       `json:"settings"`
	Stats       facebookmessenger.SubscriberStats     `json:"stats"`
	Subscribers []facebookmessenger.Subscriber        `json:"subscribers"`
	Campaigns   []facebookmessenger.Campaign          `json:"campaigns"`
	Recipients  []facebookmessenger.CampaignRecipient `json:"recipients"`
	Public      facebookmessenger.PublicConfig        `json:"public"`
}

type facebookMessengerAlertsSettingsRequest struct {
	Enabled                 bool   `json:"enabled"`
	AppID                   string `json:"appId"`
	AppSecret               string `json:"appSecret"`
	PageID                  string `json:"pageId"`
	PageAccessToken         string `json:"pageAccessToken"`
	WebhookVerifyToken      string `json:"webhookVerifyToken"`
	ValidateAppSecret       bool   `json:"validateAppSecret"`
	GraphAPIVersion         string `json:"graphApiVersion"`
	MessageTemplate         string `json:"messageTemplate"`
	LiveURLOverride         string `json:"liveUrlOverride"`
	ButtonLabel             string `json:"buttonLabel"`
	SendDelaySeconds        int    `json:"sendDelaySeconds"`
	CooldownSeconds         int    `json:"cooldownSeconds"`
	TestRecipientPSID       string `json:"testRecipientPsid"`
	PagePostFallbackEnabled bool   `json:"pagePostFallbackEnabled"`
	PagePostTemplate        string `json:"pagePostTemplate"`
}

type facebookMessengerTestRequest struct {
	PSID        string `json:"psid"`
	GoLiveStyle bool   `json:"goLiveStyle"`
}

type retryNotificationDeliveriesRequest struct {
	GoLiveEventID int64  `json:"goLiveEventId"`
	Channel       string `json:"channel"`
}

// GetBounceCastFacebookMessengerAlerts returns the full admin view model for
// Meta Messenger go-live alert setup without exposing stored secrets.
func GetBounceCastFacebookMessengerAlerts(w http.ResponseWriter, r *http.Request) {
	db := data.GetDatabase()
	settings := facebookmessenger.ReadSettings(db)
	stats, err := facebookmessenger.GetSubscriberStats(db)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	subscribers, err := facebookmessenger.ListSubscribers(db, 100)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	campaigns, err := facebookmessenger.ListCampaigns(db, 50)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	recipients, err := facebookmessenger.ListCampaignRecipients(db, 0, 100)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	webutils.WriteResponse(w, bounceCastFacebookMessengerAlertsResponse{
		Settings:    settings.AdminResponse(),
		Stats:       stats,
		Subscribers: subscribers,
		Campaigns:   campaigns,
		Recipients:  recipients,
		Public:      facebookmessenger.GetPublicConfig(db),
	})
}

// SetBounceCastFacebookMessengerAlertsSettings saves the Messenger integration
// settings. Blank secret fields preserve previously saved/env values.
func SetBounceCastFacebookMessengerAlertsSettings(w http.ResponseWriter, r *http.Request) {
	var request facebookMessengerAlertsSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	db := data.GetDatabase()
	settings := facebookmessenger.ReadSettings(db)
	settings.Enabled = request.Enabled
	settings.AppID = strings.TrimSpace(request.AppID)
	settings.PageID = strings.TrimSpace(request.PageID)
	settings.ValidateAppSecret = request.ValidateAppSecret
	settings.GraphAPIVersion = strings.TrimSpace(request.GraphAPIVersion)
	settings.MessageTemplate = strings.TrimSpace(request.MessageTemplate)
	settings.LiveURLOverride = strings.TrimSpace(request.LiveURLOverride)
	settings.ButtonLabel = strings.TrimSpace(request.ButtonLabel)
	settings.SendDelaySeconds = request.SendDelaySeconds
	settings.CooldownSeconds = request.CooldownSeconds
	settings.TestRecipientPSID = strings.TrimSpace(request.TestRecipientPSID)
	settings.PagePostFallbackEnabled = request.PagePostFallbackEnabled
	settings.PagePostTemplate = strings.TrimSpace(request.PagePostTemplate)
	if strings.TrimSpace(request.AppSecret) != "" {
		settings.AppSecret = strings.TrimSpace(request.AppSecret)
	}
	if strings.TrimSpace(request.PageAccessToken) != "" {
		settings.PageAccessToken = strings.TrimSpace(request.PageAccessToken)
	}
	if strings.TrimSpace(request.WebhookVerifyToken) != "" {
		settings.WebhookVerifyToken = strings.TrimSpace(request.WebhookVerifyToken)
	}

	if err := facebookmessenger.SaveSettings(db, settings, true); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	recordBounceCastAuditEvent(r, "facebook_messenger_alerts_settings_updated", "integration", "facebook_messenger_alerts", map[string]interface{}{"enabled": settings.Enabled})
	webutils.WriteResponse(w, facebookmessenger.ReadSettings(db).AdminResponse())
}

// ValidateBounceCastFacebookMessengerAlertsConfig checks the Page token by
// calling the Graph API Page metadata endpoint.
func ValidateBounceCastFacebookMessengerAlertsConfig(w http.ResponseWriter, r *http.Request) {
	settings := facebookmessenger.ReadSettings(data.GetDatabase())
	info, err := facebookmessenger.NewService(settings).ValidateToken()
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	recordBounceCastAuditEvent(r, "facebook_messenger_alerts_validated", "integration", "facebook_messenger_alerts", map[string]interface{}{"pageId": info.ID})
	webutils.WriteResponse(w, webutils.J{"success": true, "page": info})
}

// SendBounceCastFacebookMessengerTestMessage sends a manual test to one PSID
// and records it as a manual_test campaign.
func SendBounceCastFacebookMessengerTestMessage(w http.ResponseWriter, r *http.Request) {
	var request facebookMessengerTestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	campaign, err := facebookmessenger.SendManualTest(data.GetDatabase(), request.PSID, request.GoLiveStyle)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	recordBounceCastAuditEvent(r, "facebook_messenger_alerts_test_sent", "integration", "facebook_messenger_alerts", map[string]interface{}{"campaignId": campaign.ID})
	webutils.WriteResponse(w, campaign)
}

func PreviewBounceCastFacebookMessengerAlert(w http.ResponseWriter, r *http.Request) {
	settings := facebookmessenger.ReadSettings(data.GetDatabase())
	preview := facebookmessenger.RenderTemplate(settings.MessageTemplate, facebookmessenger.TemplateContext{
		SiteName:         "BounceCast",
		StreamTitle:      "Friday Night Bounce",
		StreamURL:        "https://example.com/",
		ChannelName:      "BounceCast",
		StartedAt:        facebookmessenger.Now().Format("2006-01-02 15:04 MST"),
		FacebookPageName: "BounceCast",
	})
	webutils.WriteResponse(w, webutils.J{"message": preview})
}

// RetryBounceCastNotificationDeliveries requeues failed go-live notification
// deliveries, including reminder-linked rows, and redispatches the affected
// go-live events.
func RetryBounceCastNotificationDeliveries(w http.ResponseWriter, r *http.Request) {
	var request retryNotificationDeliveriesRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&request)
	}
	count, err := rtmp.RetryBounceCastFailedNotificationDeliveries(request.GoLiveEventID, request.Channel)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	recordBounceCastAuditEvent(r, "notification_deliveries_retry_failed", "notification_delivery", "", map[string]interface{}{"count": count, "channel": request.Channel})
	webutils.WriteResponse(w, webutils.J{"success": true, "requeued": count})
}

// ExportBounceCastNotificationDeliveries writes a small CSV export of recent
// notification deliveries for failed-send support/debugging.
func ExportBounceCastNotificationDeliveries(w http.ResponseWriter, r *http.Request) {
	db := data.GetDatabase()
	rows, err := db.Query(`
		SELECT d.id, COALESCE(d.go_live_event_id, 0), COALESCE(d.reminder_id, 0),
			COALESCE(a.display_name, ''), d.channel, COALESCE(d.destination, ''),
			d.status, d.attempt_count, COALESCE(d.last_error, ''), d.created_at, d.sent_at
		FROM bouncecast_notification_deliveries d
		LEFT JOIN bouncecast_go_live_events e ON e.id = d.go_live_event_id
		LEFT JOIN bouncecast_streamer_accounts a ON a.id = e.streamer_id
		ORDER BY d.created_at DESC
		LIMIT 1000
	`)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="bouncecast-notification-deliveries.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"id", "go_live_event_id", "reminder_id", "streamer", "channel", "destination", "status", "attempt_count", "last_error", "created_at", "sent_at"})
	for rows.Next() {
		var id int64
		var eventID int64
		var reminderID int64
		var streamer string
		var channel string
		var destination string
		var status string
		var attemptCount int64
		var lastError string
		var createdAt string
		var sentAt sql.NullString
		if err := rows.Scan(&id, &eventID, &reminderID, &streamer, &channel, &destination, &status, &attemptCount, &lastError, &createdAt, &sentAt); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		_ = writer.Write([]string{
			strconv.FormatInt(id, 10),
			strconv.FormatInt(eventID, 10),
			strconv.FormatInt(reminderID, 10),
			streamer,
			channel,
			destination,
			status,
			strconv.FormatInt(attemptCount, 10),
			lastError,
			createdAt,
			sentAt.String,
		})
	}
	writer.Flush()
}

func GetBounceCastFacebookMessengerAlertRecipients(w http.ResponseWriter, r *http.Request) {
	campaignID, _ := strconv.ParseInt(r.URL.Query().Get("campaignId"), 10, 64)
	if campaignID == 0 {
		webutils.BadRequestHandler(w, errors.New("campaignId is required"))
		return
	}
	recipients, err := facebookmessenger.ListCampaignRecipients(data.GetDatabase(), campaignID, 500)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	webutils.WriteResponse(w, recipients)
}
