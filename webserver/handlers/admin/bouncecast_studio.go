package admin

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/owncast/owncast/core/data"
	webutils "github.com/owncast/owncast/webserver/utils"
)

type BounceCastStreamer struct {
	ID          int64      `json:"id"`
	DisplayName string     `json:"displayName"`
	Handle      string     `json:"handle"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	AvatarURL   string     `json:"avatarUrl"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
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

// GetBounceCastStreamers returns BounceCast dashboard streamer accounts.
func GetBounceCastStreamers(w http.ResponseWriter, r *http.Request) {
	rows, err := data.GetDatabase().Query(`
		SELECT id, display_name, handle, COALESCE(email, ''), role, status, COALESCE(avatar_url, ''), created_at, updated_at, last_login_at
		FROM bouncecast_streamer_accounts
		ORDER BY display_name COLLATE NOCASE ASC
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
