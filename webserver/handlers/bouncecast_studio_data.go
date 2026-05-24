package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
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

type saveBounceCastStudioScheduleRequest struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	StartsAt      string `json:"startsAt"`
	EndsAt        string `json:"endsAt"`
	Timezone      string `json:"timezone"`
	NotifyEmail   bool   `json:"notifyEmail"`
	NotifyPush    bool   `json:"notifyPush"`
	NotifyWebhook bool   `json:"notifyWebhook"`
}

type saveBounceCastStudioProfileRequest struct {
	DisplayName  string                        `json:"displayName"`
	AvatarURL    string                        `json:"avatarUrl"`
	Bio          string                        `json:"bio"`
	Genres       []string                      `json:"genres"`
	SocialLinks  []models.BounceCastSocialLink `json:"socialLinks"`
	HeroImageURL string                        `json:"heroImageUrl"`
}

type normalizedBounceCastStudioSchedule struct {
	title         string
	description   string
	startsAt      time.Time
	endsAt        interface{}
	timezone      string
	notifyEmail   bool
	notifyPush    bool
	notifyWebhook bool
}

type cancelBounceCastStudioScheduleRequest struct {
	ID int64 `json:"id"`
}

type createBounceCastStudioStreamKeyRequest struct {
	Label string `json:"label"`
}

type revokeBounceCastStudioStreamKeyRequest struct {
	ID int64 `json:"id"`
}

// BounceCastStudioSchedule returns the current DJ's own scheduled sets.
func BounceCastStudioSchedule(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w, r)

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

// BounceCastStudioCreateSchedule creates a scheduled set owned by the current DJ.
func BounceCastStudioCreateSchedule(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w, r)

	session, err := authenticateBounceCastStudioRequest(r)
	if err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}

	var request saveBounceCastStudioScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	schedule, err := normalizeBounceCastStudioScheduleRequest(request)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	result, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_stream_schedule(streamer_id, title, description, starts_at, ends_at, timezone, notify_email, notify_push, notify_webhook)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, session.streamer.ID, schedule.title, schedule.description, schedule.startsAt, schedule.endsAt, schedule.timezone, schedule.notifyEmail, schedule.notifyPush, schedule.notifyWebhook)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	id, _ := result.LastInsertId()
	webutils.WriteResponse(w, map[string]interface{}{"id": id})
}

// BounceCastStudioUpdateSchedule updates one planned scheduled set owned by the current DJ.
func BounceCastStudioUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w, r)

	session, err := authenticateBounceCastStudioRequest(r)
	if err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}

	var request saveBounceCastStudioScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if request.ID == 0 {
		webutils.BadRequestHandler(w, errors.New("id is required"))
		return
	}

	schedule, err := normalizeBounceCastStudioScheduleRequest(request)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	result, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_stream_schedule
		SET title = ?, description = ?, starts_at = ?, ends_at = ?, timezone = ?,
			notify_email = ?, notify_push = ?, notify_webhook = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND streamer_id = ? AND status = 'planned'
	`, schedule.title, schedule.description, schedule.startsAt, schedule.endsAt, schedule.timezone, schedule.notifyEmail, schedule.notifyPush, schedule.notifyWebhook, request.ID, session.streamer.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		webutils.BadRequestHandler(w, errors.New("schedule item not found or cannot be updated"))
		return
	}

	webutils.WriteSimpleResponse(w, true, "updated schedule item")
}

// BounceCastStudioCancelSchedule cancels one planned scheduled set owned by the current DJ.
func BounceCastStudioCancelSchedule(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w, r)

	session, err := authenticateBounceCastStudioRequest(r)
	if err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}

	var request cancelBounceCastStudioScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if request.ID == 0 {
		webutils.BadRequestHandler(w, errors.New("id is required"))
		return
	}

	result, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_stream_schedule
		SET status = 'cancelled', updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND streamer_id = ? AND status = 'planned'
	`, request.ID, session.streamer.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		webutils.BadRequestHandler(w, errors.New("schedule item not found or cannot be cancelled"))
		return
	}

	webutils.WriteSimpleResponse(w, true, "cancelled schedule item")
}

// BounceCastStudioStreamKeys returns masked metadata for the current DJ's stream keys.
func BounceCastStudioStreamKeys(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w, r)

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
	setBounceCastStudioAPIHeaders(w, r)

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
	setBounceCastStudioAPIHeaders(w, r)

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
	setBounceCastStudioAPIHeaders(w, r)

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

// BounceCastStudioUpdateProfile updates the current DJ's public profile fields.
func BounceCastStudioUpdateProfile(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w, r)

	session, err := authenticateBounceCastStudioRequest(r)
	if err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}

	var request saveBounceCastStudioProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	displayName := utils.MakeSafeStringOfLength(request.DisplayName, 80)
	if displayName == "" {
		webutils.BadRequestHandler(w, errors.New("displayName is required"))
		return
	}

	profile, err := normalizeBounceCastDJProfileFields(request.AvatarURL, request.Bio, request.Genres, request.SocialLinks, request.HeroImageURL)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	result, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_streamer_accounts
		SET display_name = ?, avatar_url = NULLIF(?, ''), bio = NULLIF(?, ''),
			genres = NULLIF(?, ''), social_links = NULLIF(?, ''),
			hero_image_url = NULLIF(?, ''), profile_updated_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, displayName, profile.avatarURL, profile.bio, profile.genresJSON, profile.socialLinksJSON, profile.heroImageURL, session.streamer.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		webutils.BadRequestHandler(w, errors.New("DJ profile not found"))
		return
	}

	streamer, _, err := getBounceCastStudioStreamerForLogin(session.streamer.Handle)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	webutils.WriteResponse(w, streamer)
}

func normalizeBounceCastStudioScheduleRequest(request saveBounceCastStudioScheduleRequest) (normalizedBounceCastStudioSchedule, error) {
	title := strings.TrimSpace(request.Title)
	if title == "" {
		return normalizedBounceCastStudioSchedule{}, errors.New("title is required")
	}
	if len(title) > 120 {
		return normalizedBounceCastStudioSchedule{}, errors.New("title must be 120 characters or fewer")
	}

	description := strings.TrimSpace(request.Description)
	if len(description) > 2000 {
		return normalizedBounceCastStudioSchedule{}, errors.New("description must be 2000 characters or fewer")
	}

	startsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(request.StartsAt))
	if err != nil {
		return normalizedBounceCastStudioSchedule{}, errors.New("startsAt must be RFC3339")
	}

	var endsAt interface{}
	if strings.TrimSpace(request.EndsAt) != "" {
		parsedEndsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(request.EndsAt))
		if err != nil {
			return normalizedBounceCastStudioSchedule{}, errors.New("endsAt must be RFC3339")
		}
		if !parsedEndsAt.After(startsAt) {
			return normalizedBounceCastStudioSchedule{}, errors.New("endsAt must be after startsAt")
		}
		endsAt = parsedEndsAt
	}

	timezone := strings.TrimSpace(request.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	if len(timezone) > 64 {
		return normalizedBounceCastStudioSchedule{}, errors.New("timezone must be 64 characters or fewer")
	}

	return normalizedBounceCastStudioSchedule{
		title:         title,
		description:   description,
		startsAt:      startsAt,
		endsAt:        endsAt,
		timezone:      timezone,
		notifyEmail:   request.NotifyEmail,
		notifyPush:    request.NotifyPush,
		notifyWebhook: request.NotifyWebhook,
	}, nil
}
