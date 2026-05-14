package handlers

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

type bounceCastStudioScheduleItem struct {
	ID            int64      `json:"id"`
	StreamerID    int64      `json:"streamerId"`
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
}

type bounceCastStudioStreamKey struct {
	ID         int64      `json:"id"`
	Label      string     `json:"label"`
	Enabled    bool       `json:"enabled"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

type bounceCastStudioGoLiveEvent struct {
	ID                int64      `json:"id"`
	ScheduleID        *int64     `json:"scheduleId,omitempty"`
	ScheduleTitle     string     `json:"scheduleTitle"`
	StartedAt         time.Time  `json:"startedAt"`
	EndedAt           *time.Time `json:"endedAt,omitempty"`
	Status            string     `json:"status"`
	NotificationState string     `json:"notificationState"`
}

type createBounceCastStudioStreamKeyRequest struct {
	Label string `json:"label"`
}

type revokeBounceCastStudioStreamKeyRequest struct {
	ID int64 `json:"id"`
}

// BounceCastStudioSchedule returns the current DJ's own scheduled sets.
func BounceCastStudioSchedule(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w)

	session, err := authenticateBounceCastStudioRequest(r)
	if err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}

	rows, err := data.GetDatabase().Query(`
		SELECT id, title, COALESCE(description, ''), starts_at, ends_at, timezone, status, visibility,
			notify_email, notify_push, notify_webhook
		FROM bouncecast_stream_schedule
		WHERE streamer_id = ?
		ORDER BY starts_at ASC
		LIMIT 50
	`, session.streamer.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	schedule := []bounceCastStudioScheduleItem{}
	for rows.Next() {
		var item bounceCastStudioScheduleItem
		var endsAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
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
		); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		item.StreamerID = session.streamer.ID
		item.Streamer = session.streamer.DisplayName
		if endsAt.Valid {
			item.EndsAt = &endsAt.Time
		}
		schedule = append(schedule, item)
	}

	webutils.WriteResponse(w, schedule)
}

// BounceCastStudioStreamKeys returns masked metadata for the current DJ's stream keys.
func BounceCastStudioStreamKeys(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w)

	session, err := authenticateBounceCastStudioRequest(r)
	if err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}

	rows, err := data.GetDatabase().Query(`
		SELECT id, COALESCE(label, ''), enabled, created_at, last_used_at, revoked_at
		FROM bouncecast_streamer_stream_keys
		WHERE streamer_id = ?
		ORDER BY created_at DESC
	`, session.streamer.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	streamKeys := []bounceCastStudioStreamKey{}
	for rows.Next() {
		var key bounceCastStudioStreamKey
		var lastUsedAt sql.NullTime
		var revokedAt sql.NullTime
		if err := rows.Scan(&key.ID, &key.Label, &key.Enabled, &key.CreatedAt, &lastUsedAt, &revokedAt); err != nil {
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

// BounceCastStudioCreateStreamKey creates a stream key owned by the current DJ and returns it once.
func BounceCastStudioCreateStreamKey(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w)

	session, err := authenticateBounceCastStudioRequest(r)
	if err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}

	var request createBounceCastStudioStreamKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	label := strings.TrimSpace(request.Label)
	if len(label) > 80 {
		webutils.BadRequestHandler(w, errors.New("label must be 80 characters or fewer"))
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
	`, session.streamer.ID, hashedKey, label)
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

// BounceCastStudioRevokeStreamKey revokes one of the current DJ's stream keys.
func BounceCastStudioRevokeStreamKey(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w)

	session, err := authenticateBounceCastStudioRequest(r)
	if err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}

	var request revokeBounceCastStudioStreamKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if request.ID == 0 {
		webutils.BadRequestHandler(w, errors.New("id is required"))
		return
	}

	result, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_streamer_stream_keys
		SET enabled = 0, revoked_at = CURRENT_TIMESTAMP
		WHERE id = ? AND streamer_id = ?
	`, request.ID, session.streamer.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		webutils.BadRequestHandler(w, errors.New("stream key not found"))
		return
	}

	webutils.WriteSimpleResponse(w, true, "revoked stream key")
}

// BounceCastStudioLiveEvents returns recent go-live events for the current DJ.
func BounceCastStudioLiveEvents(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w)

	session, err := authenticateBounceCastStudioRequest(r)
	if err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}

	rows, err := data.GetDatabase().Query(`
		SELECT e.id, e.schedule_id, COALESCE(s.title, ''), e.started_at, e.ended_at, e.status, e.notification_state
		FROM bouncecast_go_live_events e
		LEFT JOIN bouncecast_stream_schedule s ON s.id = e.schedule_id
		WHERE e.streamer_id = ?
		ORDER BY e.started_at DESC
		LIMIT 25
	`, session.streamer.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	events := []bounceCastStudioGoLiveEvent{}
	for rows.Next() {
		var event bounceCastStudioGoLiveEvent
		var scheduleID sql.NullInt64
		var endedAt sql.NullTime
		if err := rows.Scan(
			&event.ID,
			&scheduleID,
			&event.ScheduleTitle,
			&event.StartedAt,
			&endedAt,
			&event.Status,
			&event.NotificationState,
		); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
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
