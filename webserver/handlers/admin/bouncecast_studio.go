package admin

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/utils"
	webutils "github.com/owncast/owncast/webserver/utils"
)

type BounceCastStreamer struct {
	ID             int64      `json:"id"`
	DisplayName    string     `json:"displayName"`
	Handle         string     `json:"handle"`
	Email          string     `json:"email"`
	Role           string     `json:"role"`
	Status         string     `json:"status"`
	AvatarURL      string     `json:"avatarUrl"`
	StreamKeyCount int64      `json:"streamKeyCount"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	LastLoginAt    *time.Time `json:"lastLoginAt,omitempty"`
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

type createStreamerRequest struct {
	DisplayName string `json:"displayName"`
	Handle      string `json:"handle"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}

type createScheduleRequest struct {
	StreamerID    *int64 `json:"streamerId"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	StartsAt      string `json:"startsAt"`
	EndsAt        string `json:"endsAt"`
	Timezone      string `json:"timezone"`
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

// GetBounceCastStreamers returns BounceCast dashboard streamer accounts.
func GetBounceCastStreamers(w http.ResponseWriter, r *http.Request) {
	rows, err := data.GetDatabase().Query(`
		SELECT a.id, a.display_name, a.handle, COALESCE(a.email, ''), a.role, a.status, COALESCE(a.avatar_url, ''),
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
		if err := rows.Scan(
			&streamer.ID,
			&streamer.DisplayName,
			&streamer.Handle,
			&streamer.Email,
			&streamer.Role,
			&streamer.Status,
			&streamer.AvatarURL,
			&streamer.StreamKeyCount,
			&streamer.CreatedAt,
			&streamer.UpdatedAt,
			&lastLogin,
		); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		if lastLogin.Valid {
			streamer.LastLoginAt = &lastLogin.Time
		}
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

	role := strings.TrimSpace(request.Role)
	if role == "" {
		role = "streamer"
	}

	result, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_streamer_accounts(display_name, handle, email, role, status)
		VALUES(?, ?, NULLIF(?, ''), ?, 'active')
	`, displayName, handle, strings.TrimSpace(request.Email), role)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	id, _ := result.LastInsertId()
	webutils.WriteResponse(w, map[string]interface{}{"id": id})
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
		endsAt = parsedEndsAt
	}

	timezone := strings.TrimSpace(request.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}

	result, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_stream_schedule(streamer_id, title, description, starts_at, ends_at, timezone, notify_email, notify_push, notify_webhook)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, request.StreamerID, title, strings.TrimSpace(request.Description), startsAt, endsAt, timezone, request.NotifyEmail, request.NotifyPush, request.NotifyWebhook)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	id, _ := result.LastInsertId()
	webutils.WriteResponse(w, map[string]interface{}{"id": id})
}
