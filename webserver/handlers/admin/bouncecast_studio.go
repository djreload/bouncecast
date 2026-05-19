package admin

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/configrepository"
	"github.com/owncast/owncast/persistence/notificationsrepository"
	"github.com/owncast/owncast/utils"
	"github.com/owncast/owncast/webserver/router/middleware"
	webutils "github.com/owncast/owncast/webserver/utils"
)

type BounceCastStreamer struct {
	ID             int64                         `json:"id"`
	DisplayName    string                        `json:"displayName"`
	Handle         string                        `json:"handle"`
	Email          string                        `json:"email"`
	Role           string                        `json:"role"`
	Status         string                        `json:"status"`
	AvatarURL      string                        `json:"avatarUrl"`
	Bio            string                        `json:"bio"`
	Genres         []string                      `json:"genres"`
	SocialLinks    []models.BounceCastSocialLink `json:"socialLinks"`
	HeroImageURL   string                        `json:"heroImageUrl"`
	PasswordSet    bool                          `json:"passwordSet"`
	StreamKeyCount int64                         `json:"streamKeyCount"`
	CreatedAt      time.Time                     `json:"createdAt"`
	UpdatedAt      time.Time                     `json:"updatedAt"`
	LastLoginAt    *time.Time                    `json:"lastLoginAt,omitempty"`
}

type BounceCastScheduleItem struct {
	ID            int64      `json:"id"`
	StreamerID    *int64     `json:"streamerId,omitempty"`
	Streamer      string     `json:"streamer"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	StartsAt      time.Time  `json:"startsAt"`
	EndsAt        *time.Time `json:"endsAt,omitempty"`
	Timezone      string     `json:"timezone"`
	Status        string     `json:"status"`
	Visibility    string     `json:"visibility"`
	NotifyEmail   bool       `json:"notifyEmail"`
	NotifyPush    bool       `json:"notifyPush"`
	NotifyWebhook bool       `json:"notifyWebhook"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type BounceCastStreamKey struct {
	ID         int64      `json:"id"`
	StreamerID int64      `json:"streamerId"`
	Label      string     `json:"label"`
	Enabled    bool       `json:"enabled"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

type BounceCastGoLiveEvent struct {
	ID                int64      `json:"id"`
	StreamerID        *int64     `json:"streamerId,omitempty"`
	Streamer          string     `json:"streamer"`
	ScheduleID        *int64     `json:"scheduleId,omitempty"`
	ScheduleTitle     string     `json:"scheduleTitle"`
	StartedAt         time.Time  `json:"startedAt"`
	EndedAt           *time.Time `json:"endedAt,omitempty"`
	RemoteAddr        string     `json:"remoteAddr"`
	Status            string     `json:"status"`
	NotificationState string     `json:"notificationState"`
}

type BounceCastNotificationSubscriber struct {
	ID          int64      `json:"id"`
	Channel     string     `json:"channel"`
	Destination string     `json:"destination"`
	DisplayName string     `json:"displayName"`
	VerifiedAt  *time.Time `json:"verifiedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	DisabledAt  *time.Time `json:"disabledAt,omitempty"`
}

type BounceCastNotificationDelivery struct {
	ID            int64      `json:"id"`
	GoLiveEventID *int64     `json:"goLiveEventId,omitempty"`
	Streamer      string     `json:"streamer"`
	Channel       string     `json:"channel"`
	Destination   string     `json:"destination"`
	Status        string     `json:"status"`
	AttemptCount  int64      `json:"attemptCount"`
	LastError     string     `json:"lastError"`
	CreatedAt     time.Time  `json:"createdAt"`
	SentAt        *time.Time `json:"sentAt,omitempty"`
}

type BounceCastEmailSettings struct {
	Enabled     bool   `json:"enabled"`
	Provider    string `json:"provider"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	PasswordSet bool   `json:"passwordSet"`
	FromAddress string `json:"fromAddress"`
	FromName    string `json:"fromName"`
	StartTLS    bool   `json:"startTls"`
	Subject     string `json:"subject"`
}

type BounceCastPushSettings struct {
	Enabled         bool   `json:"enabled"`
	SubscriberCount int64  `json:"subscriberCount"`
	PublicKeySet    bool   `json:"publicKeySet"`
	PrivateKeySet   bool   `json:"privateKeySet"`
	GoLiveMessage   string `json:"goLiveMessage"`
}

type BounceCastMessengerSettings struct {
	Enabled            bool   `json:"enabled"`
	GraphAPIVersion    string `json:"graphApiVersion"`
	PageAccessToken    string `json:"pageAccessToken,omitempty"`
	PageAccessTokenSet bool   `json:"pageAccessTokenSet"`
	MessageTemplate    string `json:"messageTemplate"`
}

type BounceCastAdminSessionInfo struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
	Owner  bool   `json:"owner"`
	Admin  bool   `json:"admin"`
}

type BounceCastScheduleReminderAdminItem struct {
	ID                   int64      `json:"id"`
	ScheduleID           int64      `json:"scheduleId"`
	ScheduleTitle        string     `json:"scheduleTitle"`
	Streamer             string     `json:"streamer"`
	UserID               string     `json:"userId"`
	DisplayName          string     `json:"displayName"`
	Email                string     `json:"email"`
	NotifyEmail          bool       `json:"notifyEmail"`
	NotifyPush           bool       `json:"notifyPush"`
	NotifyMessenger      bool       `json:"notifyMessenger"`
	MessengerDestination string     `json:"messengerDestination"`
	BrowserPushLinked    bool       `json:"browserPushLinked"`
	LastQueuedAt         *time.Time `json:"lastQueuedAt,omitempty"`
	LastDeliveryStatus   string     `json:"lastDeliveryStatus"`
	LastDeliveryError    string     `json:"lastDeliveryError"`
	DisabledAt           *time.Time `json:"disabledAt,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

type BounceCastAuditEvent struct {
	ID          int64     `json:"id"`
	ActorUserID string    `json:"actorUserId"`
	ActorRole   string    `json:"actorRole"`
	Action      string    `json:"action"`
	TargetType  string    `json:"targetType"`
	TargetID    string    `json:"targetId"`
	Metadata    string    `json:"metadata"`
	RemoteAddr  string    `json:"remoteAddr"`
	CreatedAt   time.Time `json:"createdAt"`
}

type createStreamerRequest struct {
	DisplayName  string                        `json:"displayName"`
	Handle       string                        `json:"handle"`
	Email        string                        `json:"email"`
	Role         string                        `json:"role"`
	Status       string                        `json:"status"`
	AvatarURL    string                        `json:"avatarUrl"`
	Bio          string                        `json:"bio"`
	Genres       []string                      `json:"genres"`
	SocialLinks  []models.BounceCastSocialLink `json:"socialLinks"`
	HeroImageURL string                        `json:"heroImageUrl"`
	Password     string                        `json:"password"`
}

type updateStreamerRequest struct {
	ID           int64                         `json:"id"`
	DisplayName  string                        `json:"displayName"`
	Handle       string                        `json:"handle"`
	Email        string                        `json:"email"`
	Role         string                        `json:"role"`
	Status       string                        `json:"status"`
	AvatarURL    string                        `json:"avatarUrl"`
	Bio          string                        `json:"bio"`
	Genres       []string                      `json:"genres"`
	SocialLinks  []models.BounceCastSocialLink `json:"socialLinks"`
	HeroImageURL string                        `json:"heroImageUrl"`
}

type setStreamerPasswordRequest struct {
	ID       int64  `json:"id"`
	Password string `json:"password"`
}

type createScheduleRequest struct {
	StreamerID    *int64 `json:"streamerId"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	StartsAt      string `json:"startsAt"`
	EndsAt        string `json:"endsAt"`
	Timezone      string `json:"timezone"`
	Status        string `json:"status"`
	Visibility    string `json:"visibility"`
	NotifyEmail   bool   `json:"notifyEmail"`
	NotifyPush    bool   `json:"notifyPush"`
	NotifyWebhook bool   `json:"notifyWebhook"`
}

type createStreamKeyRequest struct {
	StreamerID int64  `json:"streamerId"`
	Label      string `json:"label"`
}

type revokeStreamKeyRequest struct {
	ID int64 `json:"id"`
}

type createNotificationSubscriberRequest struct {
	Channel     string `json:"channel"`
	Destination string `json:"destination"`
	DisplayName string `json:"displayName"`
}

type disableNotificationSubscriberRequest struct {
	ID int64 `json:"id"`
}

type disableScheduleReminderRequest struct {
	ID int64 `json:"id"`
}

const (
	bounceCastEmailProviderKey            = "email_provider"
	bounceCastEmailEnabledKey             = "email_enabled"
	bounceCastEmailHostKey                = "email_host"
	bounceCastEmailPortKey                = "email_port"
	bounceCastEmailUsernameKey            = "email_username"
	bounceCastEmailPasswordKey            = "email_password"
	bounceCastEmailFromAddressKey         = "email_from_address"
	bounceCastEmailFromNameKey            = "email_from_name"
	bounceCastEmailStartTLSKey            = "email_start_tls"
	bounceCastEmailSubjectKey             = "email_subject"
	bounceCastEmailProviderBrevo          = "brevo"
	bounceCastEmailProviderCustom         = "custom"
	bounceCastBrevoSMTPHost               = "smtp-relay.brevo.com"
	bounceCastBrevoSMTPPort               = 587
	bounceCastMessengerEnabledKey         = "messenger_enabled"
	bounceCastMessengerAPIVersionKey      = "messenger_graph_api_version"
	bounceCastMessengerPageAccessTokenKey = "messenger_page_access_token"
	bounceCastMessengerMessageTemplateKey = "messenger_message_template"
)

var bounceCastHandlePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,31}$`)

var bounceCastAllowedStreamerRoles = map[string]bool{
	"owner":     true,
	"manager":   true,
	"moderator": true,
	"streamer":  true,
}

var bounceCastAllowedStreamerStatuses = map[string]bool{
	"inactive": true,
	"active":   true,
	"invited":  true,
	"disabled": true,
}

type normalizedBounceCastStreamerProfile struct {
	avatarURL       string
	bio             string
	genresJSON      string
	socialLinksJSON string
	heroImageURL    string
}

// GetBounceCastStreamers returns BounceCast dashboard streamer accounts.
func GetBounceCastStreamers(w http.ResponseWriter, r *http.Request) {
	rows, err := data.GetDatabase().Query(`
		SELECT a.id, a.display_name, a.handle, COALESCE(a.email, ''), a.role, a.status, COALESCE(a.avatar_url, ''),
			COALESCE(a.bio, ''), COALESCE(a.genres, ''), COALESCE(a.social_links, ''), COALESCE(a.hero_image_url, ''),
			CASE WHEN COALESCE(a.password_hash, '') != '' THEN 1 ELSE 0 END,
			COUNT(k.id), a.created_at, a.updated_at, a.last_login_at
		FROM bouncecast_streamer_accounts a
		LEFT JOIN bouncecast_streamer_stream_keys k ON k.streamer_id = a.id AND k.enabled = 1 AND k.revoked_at IS NULL
		GROUP BY a.id
		ORDER BY a.display_name COLLATE NOCASE ASC
	`)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	streamers := []BounceCastStreamer{}
	for rows.Next() {
		var streamer BounceCastStreamer
		var lastLogin sql.NullTime
		var passwordSet bool
		var genresJSON string
		var socialLinksJSON string
		if err := rows.Scan(
			&streamer.ID,
			&streamer.DisplayName,
			&streamer.Handle,
			&streamer.Email,
			&streamer.Role,
			&streamer.Status,
			&streamer.AvatarURL,
			&streamer.Bio,
			&genresJSON,
			&socialLinksJSON,
			&streamer.HeroImageURL,
			&passwordSet,
			&streamer.StreamKeyCount,
			&streamer.CreatedAt,
			&streamer.UpdatedAt,
			&lastLogin,
		); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		streamer.Genres = parseBounceCastGenres(genresJSON)
		streamer.SocialLinks = parseBounceCastSocialLinks(socialLinksJSON)
		if lastLogin.Valid {
			streamer.LastLoginAt = &lastLogin.Time
		}
		streamer.PasswordSet = passwordSet
		streamers = append(streamers, streamer)
	}

	webutils.WriteResponse(w, streamers)
}

// CreateBounceCastStreamer creates a dashboard streamer account shell.
func CreateBounceCastStreamer(w http.ResponseWriter, r *http.Request) {
	var request createStreamerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	displayName := strings.TrimSpace(request.DisplayName)
	handle := strings.TrimSpace(strings.TrimPrefix(request.Handle, "@"))
	if displayName == "" || handle == "" {
		webutils.BadRequestHandler(w, errors.New("displayName and handle are required"))
		return
	}
	if !bounceCastHandlePattern.MatchString(handle) {
		webutils.BadRequestHandler(w, errors.New("handle must be 1-32 letters, numbers, underscores, or hyphens"))
		return
	}

	email := strings.TrimSpace(request.Email)
	if email != "" {
		parsedEmail, err := mail.ParseAddress(email)
		if err != nil {
			webutils.BadRequestHandler(w, errors.New("email must be a valid email address"))
			return
		}
		email = parsedEmail.Address
	}

	role, err := normalizeBounceCastStreamerRole(request.Role)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	status, err := normalizeBounceCastStreamerStatus(request.Status)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	profile, err := normalizeBounceCastStreamerProfile(request.AvatarURL, request.Bio, request.Genres, request.SocialLinks, request.HeroImageURL)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	password := strings.TrimSpace(request.Password)
	var passwordHash interface{}
	if password != "" {
		if err := validateBounceCastStreamerPassword(password); err != nil {
			webutils.BadRequestHandler(w, err)
			return
		}
		hashedPassword, err := utils.HashPassword(password)
		if err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		passwordHash = hashedPassword
	}

	result, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_streamer_accounts(display_name, handle, email, password_hash, role, status, avatar_url, bio, genres, social_links, hero_image_url, profile_updated_at)
		VALUES(?, ?, NULLIF(?, ''), ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), CURRENT_TIMESTAMP)
	`, displayName, handle, email, passwordHash, role, status, profile.avatarURL, profile.bio, profile.genresJSON, profile.socialLinksJSON, profile.heroImageURL)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	id, _ := result.LastInsertId()
	recordBounceCastAuditEvent(r, "streamer_created", "streamer", strconv.FormatInt(id, 10), map[string]interface{}{"handle": handle, "role": role, "status": status})
	webutils.WriteResponse(w, map[string]interface{}{"id": id})
}

// UpdateBounceCastStreamer updates visible account metadata and access status.
func UpdateBounceCastStreamer(w http.ResponseWriter, r *http.Request) {
	var request updateStreamerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if request.ID == 0 {
		webutils.BadRequestHandler(w, errors.New("id is required"))
		return
	}

	displayName := strings.TrimSpace(request.DisplayName)
	handle := strings.TrimSpace(strings.TrimPrefix(request.Handle, "@"))
	if displayName == "" || handle == "" {
		webutils.BadRequestHandler(w, errors.New("displayName and handle are required"))
		return
	}
	if !bounceCastHandlePattern.MatchString(handle) {
		webutils.BadRequestHandler(w, errors.New("handle must be 1-32 letters, numbers, underscores, or hyphens"))
		return
	}

	email := strings.TrimSpace(request.Email)
	if email != "" {
		parsedEmail, err := mail.ParseAddress(email)
		if err != nil {
			webutils.BadRequestHandler(w, errors.New("email must be a valid email address"))
			return
		}
		email = parsedEmail.Address
	}

	role, err := normalizeBounceCastStreamerRole(request.Role)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	status, err := normalizeBounceCastStreamerStatus(request.Status)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	profile, err := normalizeBounceCastStreamerProfile(request.AvatarURL, request.Bio, request.Genres, request.SocialLinks, request.HeroImageURL)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	result, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_streamer_accounts
		SET display_name = ?, handle = ?, email = NULLIF(?, ''), role = ?, status = ?,
			avatar_url = NULLIF(?, ''), bio = NULLIF(?, ''), genres = NULLIF(?, ''),
			social_links = NULLIF(?, ''), hero_image_url = NULLIF(?, ''),
			profile_updated_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, displayName, handle, email, role, status, profile.avatarURL, profile.bio, profile.genresJSON, profile.socialLinksJSON, profile.heroImageURL, request.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		webutils.BadRequestHandler(w, errors.New("streamer not found"))
		return
	}
	recordBounceCastAuditEvent(r, "streamer_updated", "streamer", strconv.FormatInt(request.ID, 10), map[string]interface{}{"handle": handle, "role": role, "status": status})

	webutils.WriteSimpleResponse(w, true, "updated streamer")
}

// SetBounceCastStreamerPassword sets or resets a DJ dashboard password.
func SetBounceCastStreamerPassword(w http.ResponseWriter, r *http.Request) {
	var request setStreamerPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if request.ID == 0 {
		webutils.BadRequestHandler(w, errors.New("id is required"))
		return
	}
	if err := validateBounceCastStreamerPassword(request.Password); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	hashedPassword, err := utils.HashPassword(strings.TrimSpace(request.Password))
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	result, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_streamer_accounts
		SET password_hash = ?, status = CASE WHEN status = 'invited' THEN 'active' ELSE status END, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, hashedPassword, request.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		webutils.BadRequestHandler(w, errors.New("streamer not found"))
		return
	}
	recordBounceCastAuditEvent(r, "streamer_password_updated", "streamer", strconv.FormatInt(request.ID, 10), nil)

	webutils.WriteSimpleResponse(w, true, "updated streamer password")
}

// GetBounceCastStreamKeys returns the per-streamer RTMP keys without exposing raw secrets.
func GetBounceCastStreamKeys(w http.ResponseWriter, r *http.Request) {
	rows, err := data.GetDatabase().Query(`
		SELECT id, streamer_id, COALESCE(label, ''), enabled, created_at, last_used_at, revoked_at
		FROM bouncecast_streamer_stream_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	streamKeys := []BounceCastStreamKey{}
	for rows.Next() {
		var key BounceCastStreamKey
		var lastUsedAt sql.NullTime
		var revokedAt sql.NullTime
		if err := rows.Scan(&key.ID, &key.StreamerID, &key.Label, &key.Enabled, &key.CreatedAt, &lastUsedAt, &revokedAt); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Time
		}
		if revokedAt.Valid {
			key.RevokedAt = &revokedAt.Time
		}
		streamKeys = append(streamKeys, key)
	}

	webutils.WriteResponse(w, streamKeys)
}

// CreateBounceCastStreamKey creates a per-streamer RTMP key and returns the raw key once.
func CreateBounceCastStreamKey(w http.ResponseWriter, r *http.Request) {
	var request createStreamKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if request.StreamerID == 0 {
		webutils.BadRequestHandler(w, errors.New("streamerId is required"))
		return
	}

	rawKey, err := utils.GenerateAccessToken()
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	hashedKey, err := utils.HashPassword(rawKey)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	result, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_streamer_stream_keys(streamer_id, key_hash, label)
		VALUES(?, ?, NULLIF(?, ''))
	`, request.StreamerID, hashedKey, strings.TrimSpace(request.Label))
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	id, _ := result.LastInsertId()
	recordBounceCastAuditEvent(r, "stream_key_created", "stream_key", strconv.FormatInt(id, 10), map[string]interface{}{"streamerId": request.StreamerID})
	webutils.WriteResponse(w, map[string]interface{}{
		"id":        id,
		"streamKey": rawKey,
	})
}

// RevokeBounceCastStreamKey disables a per-streamer RTMP key.
func RevokeBounceCastStreamKey(w http.ResponseWriter, r *http.Request) {
	var request revokeStreamKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if request.ID == 0 {
		webutils.BadRequestHandler(w, errors.New("id is required"))
		return
	}

	if _, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_streamer_stream_keys
		SET enabled = 0, revoked_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, request.ID); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	recordBounceCastAuditEvent(r, "stream_key_revoked", "stream_key", strconv.FormatInt(request.ID, 10), nil)

	webutils.WriteSimpleResponse(w, true, "revoked stream key")
}

// GetBounceCastGoLiveEvents returns recent per-streamer live events.
func GetBounceCastGoLiveEvents(w http.ResponseWriter, r *http.Request) {
	rows, err := data.GetDatabase().Query(`
		SELECT e.id, e.streamer_id, COALESCE(a.display_name, ''), e.schedule_id, COALESCE(s.title, ''),
			e.started_at, e.ended_at, COALESCE(e.remote_addr, ''), e.status, e.notification_state
		FROM bouncecast_go_live_events e
		LEFT JOIN bouncecast_streamer_accounts a ON a.id = e.streamer_id
		LEFT JOIN bouncecast_stream_schedule s ON s.id = e.schedule_id
		ORDER BY e.started_at DESC
		LIMIT 25
	`)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	events := []BounceCastGoLiveEvent{}
	for rows.Next() {
		var event BounceCastGoLiveEvent
		var streamerID sql.NullInt64
		var scheduleID sql.NullInt64
		var endedAt sql.NullTime
		if err := rows.Scan(
			&event.ID,
			&streamerID,
			&event.Streamer,
			&scheduleID,
			&event.ScheduleTitle,
			&event.StartedAt,
			&endedAt,
			&event.RemoteAddr,
			&event.Status,
			&event.NotificationState,
		); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		if streamerID.Valid {
			event.StreamerID = &streamerID.Int64
		}
		if scheduleID.Valid {
			event.ScheduleID = &scheduleID.Int64
		}
		if endedAt.Valid {
			event.EndedAt = &endedAt.Time
		}
		events = append(events, event)
	}

	webutils.WriteResponse(w, events)
}

// GetBounceCastNotificationSubscribers returns notification destinations managed by BounceCast Studio.
func GetBounceCastNotificationSubscribers(w http.ResponseWriter, r *http.Request) {
	rows, err := data.GetDatabase().Query(`
		SELECT id, channel, destination, COALESCE(display_name, ''), verified_at, created_at, disabled_at
		FROM bouncecast_notification_subscribers
		ORDER BY created_at DESC
	`)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	subscribers := []BounceCastNotificationSubscriber{}
	for rows.Next() {
		var subscriber BounceCastNotificationSubscriber
		var verifiedAt sql.NullTime
		var disabledAt sql.NullTime
		if err := rows.Scan(
			&subscriber.ID,
			&subscriber.Channel,
			&subscriber.Destination,
			&subscriber.DisplayName,
			&verifiedAt,
			&subscriber.CreatedAt,
			&disabledAt,
		); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		if verifiedAt.Valid {
			subscriber.VerifiedAt = &verifiedAt.Time
		}
		if disabledAt.Valid {
			subscriber.DisabledAt = &disabledAt.Time
		}
		subscribers = append(subscribers, subscriber)
	}

	webutils.WriteResponse(w, subscribers)
}

// CreateBounceCastNotificationSubscriber adds a notification destination.
func CreateBounceCastNotificationSubscriber(w http.ResponseWriter, r *http.Request) {
	var request createNotificationSubscriberRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	channel := strings.TrimSpace(request.Channel)
	destination := strings.TrimSpace(request.Destination)
	if channel == "" || destination == "" {
		webutils.BadRequestHandler(w, errors.New("channel and destination are required"))
		return
	}
	if channel != "email" && channel != "push" && channel != "webhook" {
		webutils.BadRequestHandler(w, errors.New("channel must be email, push, or webhook"))
		return
	}
	normalizedDestination, err := normalizeBounceCastSubscriberDestination(channel, destination)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	result, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_notification_subscribers(channel, destination, display_name, verified_at)
		VALUES(?, ?, NULLIF(?, ''), CURRENT_TIMESTAMP)
		ON CONFLICT(channel, destination) DO UPDATE SET
			display_name = excluded.display_name,
			disabled_at = NULL
	`, channel, normalizedDestination, strings.TrimSpace(request.DisplayName))
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	id, _ := result.LastInsertId()
	recordBounceCastAuditEvent(r, "notification_subscriber_created", "notification_subscriber", strconv.FormatInt(id, 10), map[string]interface{}{"channel": channel})
	webutils.WriteResponse(w, map[string]interface{}{"id": id})
}

// DisableBounceCastNotificationSubscriber disables a notification destination.
func DisableBounceCastNotificationSubscriber(w http.ResponseWriter, r *http.Request) {
	var request disableNotificationSubscriberRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if request.ID == 0 {
		webutils.BadRequestHandler(w, errors.New("id is required"))
		return
	}

	if _, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_notification_subscribers
		SET disabled_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, request.ID); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	webutils.WriteSimpleResponse(w, true, "disabled notification subscriber")
}

// GetBounceCastNotificationDeliveries returns recent notification delivery attempts.
func GetBounceCastNotificationDeliveries(w http.ResponseWriter, r *http.Request) {
	rows, err := data.GetDatabase().Query(`
		SELECT d.id, d.go_live_event_id, COALESCE(a.display_name, ''), d.channel, COALESCE(d.destination, ''),
			d.status, d.attempt_count, COALESCE(d.last_error, ''), d.created_at, d.sent_at
		FROM bouncecast_notification_deliveries d
		LEFT JOIN bouncecast_go_live_events e ON e.id = d.go_live_event_id
		LEFT JOIN bouncecast_streamer_accounts a ON a.id = e.streamer_id
		ORDER BY d.created_at DESC
		LIMIT 50
	`)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	deliveries := []BounceCastNotificationDelivery{}
	for rows.Next() {
		var delivery BounceCastNotificationDelivery
		var goLiveEventID sql.NullInt64
		var sentAt sql.NullTime
		if err := rows.Scan(
			&delivery.ID,
			&goLiveEventID,
			&delivery.Streamer,
			&delivery.Channel,
			&delivery.Destination,
			&delivery.Status,
			&delivery.AttemptCount,
			&delivery.LastError,
			&delivery.CreatedAt,
			&sentAt,
		); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		if goLiveEventID.Valid {
			delivery.GoLiveEventID = &goLiveEventID.Int64
		}
		if sentAt.Valid {
			delivery.SentAt = &sentAt.Time
		}
		deliveries = append(deliveries, delivery)
	}

	webutils.WriteResponse(w, deliveries)
}

// GetBounceCastEmailSettings returns SMTP delivery settings without exposing the saved password.
func GetBounceCastEmailSettings(w http.ResponseWriter, r *http.Request) {
	settings := readBounceCastEmailSettings()
	settings.Password = ""
	webutils.WriteResponse(w, settings)
}

// SetBounceCastEmailSettings saves SMTP delivery settings.
func SetBounceCastEmailSettings(w http.ResponseWriter, r *http.Request) {
	var request BounceCastEmailSettings
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	provider, err := normalizeBounceCastEmailProvider(request.Provider)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if provider == bounceCastEmailProviderBrevo {
		request.Host = bounceCastBrevoSMTPHost
		if request.Port == 0 {
			request.Port = bounceCastBrevoSMTPPort
		}
		request.StartTLS = true
	}

	if request.Enabled {
		if strings.TrimSpace(request.Host) == "" {
			webutils.BadRequestHandler(w, errors.New("host is required when email is enabled"))
			return
		}
		if strings.TrimSpace(request.FromAddress) == "" {
			webutils.BadRequestHandler(w, errors.New("fromAddress is required when email is enabled"))
			return
		}
		if provider == bounceCastEmailProviderBrevo {
			if strings.TrimSpace(request.Username) == "" {
				webutils.BadRequestHandler(w, errors.New("Brevo SMTP login email is required when Brevo email is enabled"))
				return
			}
			savedPassword := getBounceCastNotificationSetting(bounceCastEmailPasswordKey)
			if strings.TrimSpace(request.Password) == "" && savedPassword == "" {
				webutils.BadRequestHandler(w, errors.New("Brevo SMTP key is required when Brevo email is enabled"))
				return
			}
		}
	}
	if request.Port == 0 {
		request.Port = 587
	}
	if request.Port < 1 || request.Port > 65535 {
		webutils.BadRequestHandler(w, errors.New("port must be between 1 and 65535"))
		return
	}
	if strings.TrimSpace(request.FromName) == "" {
		request.FromName = "BounceCast"
	}
	if strings.TrimSpace(request.Subject) == "" {
		request.Subject = "{{streamer}} is live on BounceCast"
	}
	if err := validateBounceCastEmailHeader(request.FromName, "fromName"); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if err := validateBounceCastEmailHeader(request.Subject, "subject"); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	fromAddress := strings.TrimSpace(request.FromAddress)
	if fromAddress != "" {
		parsedFromAddress, err := mail.ParseAddress(fromAddress)
		if err != nil {
			webutils.BadRequestHandler(w, errors.New("fromAddress must be a valid email address"))
			return
		}
		fromAddress = parsedFromAddress.Address
	}

	settings := map[string]string{
		bounceCastEmailProviderKey:    provider,
		bounceCastEmailEnabledKey:     strconv.FormatBool(request.Enabled),
		bounceCastEmailHostKey:        strings.TrimSpace(request.Host),
		bounceCastEmailPortKey:        strconv.Itoa(request.Port),
		bounceCastEmailUsernameKey:    strings.TrimSpace(request.Username),
		bounceCastEmailFromAddressKey: fromAddress,
		bounceCastEmailFromNameKey:    strings.TrimSpace(request.FromName),
		bounceCastEmailStartTLSKey:    strconv.FormatBool(request.StartTLS),
		bounceCastEmailSubjectKey:     strings.TrimSpace(request.Subject),
	}
	if strings.TrimSpace(request.Password) != "" {
		settings[bounceCastEmailPasswordKey] = request.Password
	}

	for key, value := range settings {
		if err := setBounceCastNotificationSetting(key, value); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
	}
	recordBounceCastAuditEvent(r, "email_settings_updated", "notification_settings", "email", map[string]interface{}{"enabled": request.Enabled, "provider": provider})

	savedSettings := readBounceCastEmailSettings()
	savedSettings.Password = ""
	webutils.WriteResponse(w, savedSettings)
}

func normalizeBounceCastSubscriberDestination(channel string, destination string) (string, error) {
	switch channel {
	case "email":
		parsedEmail, err := mail.ParseAddress(destination)
		if err != nil {
			return "", errors.New("email destination must be a valid email address")
		}
		return parsedEmail.Address, nil
	case "webhook":
		parsedURL, err := url.Parse(destination)
		if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return "", errors.New("webhook destination must be a valid http or https URL")
		}
		return parsedURL.String(), nil
	case "push":
		var subscription struct {
			Endpoint string `json:"endpoint"`
			Keys     struct {
				P256DH string `json:"p256dh"`
				Auth   string `json:"auth"`
			} `json:"keys"`
		}
		if err := json.Unmarshal([]byte(destination), &subscription); err != nil {
			return "", errors.New("push destination must be a browser push subscription JSON payload")
		}
		endpoint := strings.TrimSpace(subscription.Endpoint)
		parsedEndpoint, err := url.Parse(endpoint)
		if err != nil || parsedEndpoint.Scheme != "https" || parsedEndpoint.Host == "" {
			return "", errors.New("push destination endpoint must be a valid https URL")
		}
		if strings.TrimSpace(subscription.Keys.P256DH) == "" || strings.TrimSpace(subscription.Keys.Auth) == "" {
			return "", errors.New("push destination must include browser push keys")
		}
		return destination, nil
	default:
		return "", fmt.Errorf("unsupported notification channel: %s", channel)
	}
}

func validateBounceCastEmailHeader(value string, field string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%s cannot contain line breaks", field)
	}
	return nil
}

func normalizeBounceCastStreamerRole(role string) (string, error) {
	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	if normalizedRole == "" {
		normalizedRole = "streamer"
	}
	if !bounceCastAllowedStreamerRoles[normalizedRole] {
		return "", errors.New("role must be owner, manager, moderator, or streamer")
	}
	return normalizedRole, nil
}

func normalizeBounceCastStreamerStatus(status string) (string, error) {
	normalizedStatus := strings.ToLower(strings.TrimSpace(status))
	if normalizedStatus == "" {
		normalizedStatus = "active"
	}
	if !bounceCastAllowedStreamerStatuses[normalizedStatus] {
		return "", errors.New("status must be inactive, active, invited, or disabled")
	}
	return normalizedStatus, nil
}

func validateBounceCastStreamerPassword(password string) error {
	if strings.ContainsAny(password, "\r\n") {
		return errors.New("password cannot contain line breaks")
	}
	trimmedPassword := strings.TrimSpace(password)
	if len(trimmedPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func normalizeBounceCastStreamerProfile(avatarURL string, bio string, genres []string, socialLinks []models.BounceCastSocialLink, heroImageURL string) (normalizedBounceCastStreamerProfile, error) {
	normalizedAvatarURL, err := normalizeBounceCastProfileImageURL(avatarURL)
	if err != nil {
		return normalizedBounceCastStreamerProfile{}, err
	}
	normalizedHeroImageURL, err := normalizeBounceCastProfileImageURL(heroImageURL)
	if err != nil {
		return normalizedBounceCastStreamerProfile{}, err
	}
	normalizedGenres, err := normalizeBounceCastGenres(genres)
	if err != nil {
		return normalizedBounceCastStreamerProfile{}, err
	}
	normalizedLinks, err := normalizeBounceCastSocialLinks(socialLinks)
	if err != nil {
		return normalizedBounceCastStreamerProfile{}, err
	}

	genresJSON, err := marshalBounceCastProfileJSON(normalizedGenres)
	if err != nil {
		return normalizedBounceCastStreamerProfile{}, err
	}
	socialLinksJSON, err := marshalBounceCastProfileJSON(normalizedLinks)
	if err != nil {
		return normalizedBounceCastStreamerProfile{}, err
	}

	return normalizedBounceCastStreamerProfile{
		avatarURL:       normalizedAvatarURL,
		bio:             utils.MakeSafeStringOfLength(bio, 600),
		genresJSON:      genresJSON,
		socialLinksJSON: socialLinksJSON,
		heroImageURL:    normalizedHeroImageURL,
	}, nil
}

func normalizeBounceCastProfileImageURL(value string) (string, error) {
	imageURL := utils.MakeSafeStringOfLength(value, 500)
	if imageURL == "" {
		return "", nil
	}
	if strings.HasPrefix(imageURL, "/") && !strings.HasPrefix(imageURL, "//") {
		return imageURL, nil
	}
	parsed, err := url.ParseRequestURI(imageURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("profile images must be http(s) URLs or local paths")
	}
	return imageURL, nil
}

func normalizeBounceCastGenres(values []string) ([]string, error) {
	genres := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		genre := utils.MakeSafeStringOfLength(value, 32)
		if genre == "" {
			continue
		}
		key := strings.ToLower(genre)
		if seen[key] {
			continue
		}
		seen[key] = true
		genres = append(genres, genre)
		if len(genres) > 8 {
			return nil, errors.New("genres must contain 8 items or fewer")
		}
	}
	return genres, nil
}

func normalizeBounceCastSocialLinks(values []models.BounceCastSocialLink) ([]models.BounceCastSocialLink, error) {
	links := []models.BounceCastSocialLink{}
	for _, value := range values {
		label := utils.MakeSafeStringOfLength(value.Label, 40)
		linkURL := strings.TrimSpace(value.URL)
		if label == "" && linkURL == "" {
			continue
		}
		if label == "" || linkURL == "" {
			return nil, errors.New("social links require both label and URL")
		}
		if len(links) >= 6 {
			return nil, errors.New("social links must contain 6 items or fewer")
		}
		parsedURL, err := url.Parse(linkURL)
		if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return nil, errors.New("social link URLs must be valid http(s) URLs")
		}
		links = append(links, models.BounceCastSocialLink{
			Label: label,
			URL:   parsedURL.String(),
		})
	}
	return links, nil
}

func marshalBounceCastProfileJSON(value interface{}) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	if string(body) == "[]" || string(body) == "null" {
		return "", nil
	}
	return string(body), nil
}

func parseBounceCastGenres(value string) []string {
	var genres []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &genres); err == nil {
		normalized, _ := normalizeBounceCastGenres(genres)
		return normalized
	}

	parts := strings.Split(value, ",")
	normalized, _ := normalizeBounceCastGenres(parts)
	return normalized
}

func parseBounceCastSocialLinks(value string) []models.BounceCastSocialLink {
	var links []models.BounceCastSocialLink
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &links); err != nil {
		return []models.BounceCastSocialLink{}
	}
	normalized, err := normalizeBounceCastSocialLinks(links)
	if err != nil {
		return []models.BounceCastSocialLink{}
	}
	return normalized
}

// GetBounceCastPushSettings returns the current browser push status for BounceCast go-live alerts.
func GetBounceCastPushSettings(w http.ResponseWriter, r *http.Request) {
	configRepository := configrepository.Get()
	browserConfig := configRepository.GetBrowserPushConfig()
	publicKey, _ := configRepository.GetBrowserPushPublicKey()
	privateKey, _ := configRepository.GetBrowserPushPrivateKey()

	var subscriberCount int64
	if err := data.GetDatabase().QueryRow(`
		SELECT COUNT(*)
		FROM notifications
		WHERE channel = ?
	`, notificationsrepository.BrowserPushNotification).Scan(&subscriberCount); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	webutils.WriteResponse(w, BounceCastPushSettings{
		Enabled:         browserConfig.Enabled,
		SubscriberCount: subscriberCount,
		PublicKeySet:    publicKey != "",
		PrivateKeySet:   privateKey != "",
		GoLiveMessage:   browserConfig.GoLiveMessage,
	})
}

// GetBounceCastAdminSession returns the BounceCast product role attached to the
// current admin session so the admin UI can hide owner-only actions.
func GetBounceCastAdminSession(w http.ResponseWriter, r *http.Request) {
	identity, ok := middleware.CurrentAdminIdentity(r)
	if !ok {
		webutils.WriteResponse(w, BounceCastAdminSessionInfo{})
		return
	}
	webutils.WriteResponse(w, BounceCastAdminSessionInfo{
		UserID: identity.UserID,
		Role:   identity.Role,
		Owner:  identity.Role == "owner",
		Admin:  identity.Role == "admin" || identity.Role == "owner",
	})
}

// GetBounceCastMessengerSettings returns Messenger transport configuration
// without exposing the saved Page Access Token.
func GetBounceCastMessengerSettings(w http.ResponseWriter, r *http.Request) {
	settings := readBounceCastMessengerSettings()
	settings.PageAccessToken = ""
	webutils.WriteResponse(w, settings)
}

// SetBounceCastMessengerSettings saves Facebook Messenger Graph API delivery
// settings for schedule reminder notifications.
func SetBounceCastMessengerSettings(w http.ResponseWriter, r *http.Request) {
	var request BounceCastMessengerSettings
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	version := strings.TrimSpace(request.GraphAPIVersion)
	if version == "" {
		version = "v20.0"
	}
	if !regexp.MustCompile(`^v[0-9]+(\.[0-9]+)?$`).MatchString(version) {
		webutils.BadRequestHandler(w, errors.New("graphApiVersion must look like v20.0"))
		return
	}

	template := strings.TrimSpace(request.MessageTemplate)
	if template == "" {
		template = "{{streamer}} is live on BounceCast: {{schedule}}"
	}
	if strings.ContainsAny(template, "\r\n") {
		webutils.BadRequestHandler(w, errors.New("messageTemplate cannot contain line breaks"))
		return
	}

	token := strings.TrimSpace(request.PageAccessToken)
	if request.Enabled && token == "" && getBounceCastNotificationSetting(bounceCastMessengerPageAccessTokenKey) == "" {
		webutils.BadRequestHandler(w, errors.New("pageAccessToken is required when Messenger delivery is enabled"))
		return
	}

	values := map[string]string{
		bounceCastMessengerEnabledKey:         strconv.FormatBool(request.Enabled),
		bounceCastMessengerAPIVersionKey:      version,
		bounceCastMessengerMessageTemplateKey: template,
	}
	if token != "" {
		values[bounceCastMessengerPageAccessTokenKey] = token
	}
	for key, value := range values {
		if err := setBounceCastNotificationSetting(key, value); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
	}
	recordBounceCastAuditEvent(r, "messenger_settings_updated", "notification_settings", "messenger", map[string]interface{}{"enabled": request.Enabled, "graphApiVersion": version})

	response := readBounceCastMessengerSettings()
	response.PageAccessToken = ""
	webutils.WriteResponse(w, response)
}

// GetBounceCastScheduleReminders returns public account schedule reminder
// registrations with delivery status for admin review.
func GetBounceCastScheduleReminders(w http.ResponseWriter, r *http.Request) {
	rows, err := data.GetDatabase().Query(`
		SELECT r.id, r.schedule_id, s.title, COALESCE(a.display_name, ''), r.user_id,
			COALESCE(u.display_name, ''), COALESCE(r.email, ''),
			r.notify_email, r.notify_push, r.notify_messenger, COALESCE(r.messenger_destination, ''),
			CASE WHEN COALESCE(r.browser_push_endpoint, '') != '' OR EXISTS (
				SELECT 1 FROM bouncecast_account_push_subscriptions p
				WHERE p.user_id = r.user_id AND p.enabled = 1 AND p.disabled_at IS NULL
			) THEN 1 ELSE 0 END AS browser_push_linked,
			r.last_queued_at, COALESCE(r.last_delivery_status, ''), COALESCE(r.last_delivery_error, ''),
			r.disabled_at, r.created_at, r.updated_at
		FROM bouncecast_schedule_reminders r
		INNER JOIN bouncecast_stream_schedule s ON s.id = r.schedule_id
		LEFT JOIN bouncecast_streamer_accounts a ON a.id = s.streamer_id
		LEFT JOIN users u ON u.id = r.user_id
		ORDER BY r.updated_at DESC
		LIMIT 100
	`)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	reminders := []BounceCastScheduleReminderAdminItem{}
	for rows.Next() {
		var reminder BounceCastScheduleReminderAdminItem
		var lastQueuedAt sql.NullTime
		var disabledAt sql.NullTime
		if err := rows.Scan(
			&reminder.ID,
			&reminder.ScheduleID,
			&reminder.ScheduleTitle,
			&reminder.Streamer,
			&reminder.UserID,
			&reminder.DisplayName,
			&reminder.Email,
			&reminder.NotifyEmail,
			&reminder.NotifyPush,
			&reminder.NotifyMessenger,
			&reminder.MessengerDestination,
			&reminder.BrowserPushLinked,
			&lastQueuedAt,
			&reminder.LastDeliveryStatus,
			&reminder.LastDeliveryError,
			&disabledAt,
			&reminder.CreatedAt,
			&reminder.UpdatedAt,
		); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		if lastQueuedAt.Valid {
			reminder.LastQueuedAt = &lastQueuedAt.Time
		}
		if disabledAt.Valid {
			reminder.DisabledAt = &disabledAt.Time
		}
		reminders = append(reminders, reminder)
	}
	if err := rows.Err(); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	webutils.WriteResponse(w, reminders)
}

// DisableBounceCastScheduleReminder disables a viewer's reminder without
// deleting the audit trail around when it was created or last queued.
func DisableBounceCastScheduleReminder(w http.ResponseWriter, r *http.Request) {
	var request disableScheduleReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if request.ID == 0 {
		webutils.BadRequestHandler(w, errors.New("id is required"))
		return
	}

	result, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_schedule_reminders
		SET disabled_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, request.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		webutils.BadRequestHandler(w, errors.New("reminder not found"))
		return
	}
	recordBounceCastAuditEvent(r, "schedule_reminder_disabled", "schedule_reminder", strconv.FormatInt(request.ID, 10), nil)

	webutils.WriteSimpleResponse(w, true, "disabled schedule reminder")
}

// GetBounceCastAuditEvents returns recent product-level admin actions.
func GetBounceCastAuditEvents(w http.ResponseWriter, r *http.Request) {
	rows, err := data.GetDatabase().Query(`
		SELECT id, COALESCE(actor_user_id, ''), COALESCE(actor_role, ''), action, target_type,
			COALESCE(target_id, ''), COALESCE(metadata, ''), COALESCE(remote_addr, ''), created_at
		FROM bouncecast_audit_events
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	events := []BounceCastAuditEvent{}
	for rows.Next() {
		var event BounceCastAuditEvent
		if err := rows.Scan(&event.ID, &event.ActorUserID, &event.ActorRole, &event.Action, &event.TargetType, &event.TargetID, &event.Metadata, &event.RemoteAddr, &event.CreatedAt); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	webutils.WriteResponse(w, events)
}

func readBounceCastEmailSettings() BounceCastEmailSettings {
	provider, _ := normalizeBounceCastEmailProvider(getBounceCastNotificationSetting(bounceCastEmailProviderKey))
	port, err := strconv.Atoi(getBounceCastNotificationSetting(bounceCastEmailPortKey))
	if err != nil || port == 0 {
		port = 587
	}
	host := getBounceCastNotificationSetting(bounceCastEmailHostKey)
	startTLS := getBounceCastNotificationSetting(bounceCastEmailStartTLSKey) == "true"
	if provider == bounceCastEmailProviderBrevo {
		host = bounceCastBrevoSMTPHost
		if port == 0 {
			port = bounceCastBrevoSMTPPort
		}
		startTLS = true
	}

	fromName := getBounceCastNotificationSetting(bounceCastEmailFromNameKey)
	if fromName == "" {
		fromName = "BounceCast"
	}
	subject := getBounceCastNotificationSetting(bounceCastEmailSubjectKey)
	if subject == "" {
		subject = "{{streamer}} is live on BounceCast"
	}

	password := getBounceCastNotificationSetting(bounceCastEmailPasswordKey)
	return BounceCastEmailSettings{
		Enabled:     getBounceCastNotificationSetting(bounceCastEmailEnabledKey) == "true",
		Provider:    provider,
		Host:        host,
		Port:        port,
		Username:    getBounceCastNotificationSetting(bounceCastEmailUsernameKey),
		PasswordSet: password != "",
		FromAddress: getBounceCastNotificationSetting(bounceCastEmailFromAddressKey),
		FromName:    fromName,
		StartTLS:    startTLS,
		Subject:     subject,
	}
}

func readBounceCastMessengerSettings() BounceCastMessengerSettings {
	graphVersion := getBounceCastNotificationSetting(bounceCastMessengerAPIVersionKey)
	if graphVersion == "" {
		graphVersion = "v20.0"
	}
	template := getBounceCastNotificationSetting(bounceCastMessengerMessageTemplateKey)
	if template == "" {
		template = "{{streamer}} is live on BounceCast: {{schedule}}"
	}
	token := getBounceCastNotificationSetting(bounceCastMessengerPageAccessTokenKey)
	return BounceCastMessengerSettings{
		Enabled:            getBounceCastNotificationSetting(bounceCastMessengerEnabledKey) == "true",
		GraphAPIVersion:    graphVersion,
		PageAccessTokenSet: token != "",
		MessageTemplate:    template,
	}
}

func recordBounceCastAuditEvent(r *http.Request, action string, targetType string, targetID string, metadata map[string]interface{}) {
	action = strings.TrimSpace(action)
	targetType = strings.TrimSpace(targetType)
	if action == "" || targetType == "" {
		return
	}
	identity, _ := middleware.CurrentAdminIdentity(r)
	metadataJSON := ""
	if len(metadata) > 0 {
		if encoded, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(encoded)
		}
	}
	remoteAddr := ""
	if r != nil {
		remoteAddr = r.RemoteAddr
	}
	if _, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_audit_events(actor_user_id, actor_role, action, target_type, target_id, metadata, remote_addr)
		VALUES(NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''))
	`, identity.UserID, identity.Role, action, targetType, strings.TrimSpace(targetID), metadataJSON, remoteAddr); err != nil {
		// Audit logging should never block the product action.
		return
	}
}

func normalizeBounceCastEmailProvider(provider string) (string, error) {
	normalizedProvider := strings.ToLower(strings.TrimSpace(provider))
	if normalizedProvider == "" {
		normalizedProvider = bounceCastEmailProviderCustom
	}
	switch normalizedProvider {
	case bounceCastEmailProviderCustom, bounceCastEmailProviderBrevo:
		return normalizedProvider, nil
	default:
		return "", errors.New("email provider must be custom or brevo")
	}
}

func getBounceCastNotificationSetting(key string) string {
	var value sql.NullString
	if err := data.GetDatabase().QueryRow(`SELECT value FROM bouncecast_notification_settings WHERE key = ?`, key).Scan(&value); err != nil {
		return ""
	}
	if !value.Valid {
		return ""
	}
	return value.String
}

func setBounceCastNotificationSetting(key string, value string) error {
	_, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_notification_settings("key", "value", updated_at)
		VALUES(?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT("key") DO UPDATE SET
			"value" = excluded."value",
			updated_at = CURRENT_TIMESTAMP
	`, key, value)
	return err
}

// GetBounceCastSchedule returns upcoming BounceCast schedule rows.
func GetBounceCastSchedule(w http.ResponseWriter, r *http.Request) {
	rows, err := data.GetDatabase().Query(`
		SELECT s.id, s.streamer_id, COALESCE(a.display_name, ''), s.title, COALESCE(s.description, ''),
			s.starts_at, s.ends_at, s.timezone, s.status, s.visibility,
			s.notify_email, s.notify_push, s.notify_webhook, s.created_at, s.updated_at
		FROM bouncecast_stream_schedule s
		LEFT JOIN bouncecast_streamer_accounts a ON a.id = s.streamer_id
		ORDER BY s.starts_at ASC
	`)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	schedule := []BounceCastScheduleItem{}
	for rows.Next() {
		var item BounceCastScheduleItem
		var streamerID sql.NullInt64
		var endsAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&streamerID,
			&item.Streamer,
			&item.Title,
			&item.Description,
			&item.StartsAt,
			&endsAt,
			&item.Timezone,
			&item.Status,
			&item.Visibility,
			&item.NotifyEmail,
			&item.NotifyPush,
			&item.NotifyWebhook,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		if streamerID.Valid {
			item.StreamerID = &streamerID.Int64
		}
		if endsAt.Valid {
			item.EndsAt = &endsAt.Time
		}
		schedule = append(schedule, item)
	}

	webutils.WriteResponse(w, schedule)
}

// CreateBounceCastSchedule creates a scheduled DJ set.
func CreateBounceCastSchedule(w http.ResponseWriter, r *http.Request) {
	var request createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	title := strings.TrimSpace(request.Title)
	if title == "" {
		webutils.BadRequestHandler(w, errors.New("title is required"))
		return
	}

	startsAt, err := time.Parse(time.RFC3339, request.StartsAt)
	if err != nil {
		webutils.BadRequestHandler(w, errors.New("startsAt must be RFC3339"))
		return
	}

	var endsAt interface{}
	if strings.TrimSpace(request.EndsAt) != "" {
		parsedEndsAt, err := time.Parse(time.RFC3339, request.EndsAt)
		if err != nil {
			webutils.BadRequestHandler(w, errors.New("endsAt must be RFC3339"))
			return
		}
		if !parsedEndsAt.After(startsAt) {
			webutils.BadRequestHandler(w, errors.New("endsAt must be after startsAt"))
			return
		}
		endsAt = parsedEndsAt
	}

	timezone := strings.TrimSpace(request.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	status := strings.ToLower(strings.TrimSpace(request.Status))
	if status == "" {
		status = "planned"
	}
	if status != "planned" && status != "live" {
		webutils.BadRequestHandler(w, errors.New("status must be planned or live"))
		return
	}
	visibility := strings.ToLower(strings.TrimSpace(request.Visibility))
	if visibility == "" {
		visibility = "public"
	}
	if visibility != "public" && visibility != "private" {
		webutils.BadRequestHandler(w, errors.New("visibility must be public or private"))
		return
	}

	result, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_stream_schedule(streamer_id, title, description, starts_at, ends_at, timezone, status, visibility, notify_email, notify_push, notify_webhook)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, request.StreamerID, title, strings.TrimSpace(request.Description), startsAt, endsAt, timezone, status, visibility, request.NotifyEmail, request.NotifyPush, request.NotifyWebhook)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	id, _ := result.LastInsertId()
	recordBounceCastAuditEvent(r, "schedule_created", "schedule", strconv.FormatInt(id, 10), map[string]interface{}{"status": status, "visibility": visibility})
	webutils.WriteResponse(w, map[string]interface{}{"id": id})
}
