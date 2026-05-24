package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/starsrepository"
	"github.com/owncast/owncast/utils"
	"github.com/owncast/owncast/webserver/router/middleware"
	webutils "github.com/owncast/owncast/webserver/utils"
)

const (
	bounceCastPublicDefaultLimit = 12
	bounceCastPublicMaxLimit     = 50
)

type bounceCastPublicDJ struct {
	DisplayName        string                        `json:"displayName"`
	Handle             string                        `json:"handle"`
	AvatarURL          string                        `json:"avatarUrl,omitempty"`
	Bio                string                        `json:"bio,omitempty"`
	Genres             []string                      `json:"genres"`
	SocialLinks        []models.BounceCastSocialLink `json:"socialLinks"`
	HeroImageURL       string                        `json:"heroImageUrl,omitempty"`
	SEOTitle           string                        `json:"seoTitle,omitempty"`
	ShareImageURL      string                        `json:"shareImageUrl,omitempty"`
	FeaturedScheduleID *int64                        `json:"featuredScheduleId,omitempty"`
	FeaturedSchedule   *bounceCastPublicScheduleItem `json:"featuredSchedule,omitempty"`
	Role               string                        `json:"role"`
	UpcomingSet        string                        `json:"upcomingSet,omitempty"`
	UpcomingStarts     *time.Time                    `json:"upcomingStarts,omitempty"`
	UpcomingCount      int                           `json:"upcomingCount"`
	TotalLiveEvents    int                           `json:"totalLiveEvents"`
	LastLiveAt         *time.Time                    `json:"lastLiveAt,omitempty"`
}

type bounceCastPublicScheduleItem struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	StartsAt    time.Time  `json:"startsAt"`
	EndsAt      *time.Time `json:"endsAt,omitempty"`
	Timezone    string     `json:"timezone"`
	Status      string     `json:"status"`
	Streamer    string     `json:"streamer,omitempty"`
	Handle      string     `json:"handle,omitempty"`
	AvatarURL   string     `json:"avatarUrl,omitempty"`
}

type bounceCastPublicDJProfile struct {
	DJ       bounceCastPublicDJ             `json:"dj"`
	Schedule []bounceCastPublicScheduleItem `json:"schedule"`
}

type bounceCastPublicScheduleFilter struct {
	Limit       int
	Handle      string
	Status      string
	Query       string
	From        *time.Time
	To          *time.Time
	IncludePast bool
}

type bounceCastAccountDestination struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	URL       string `json:"url"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

type bounceCastAccountHubResponse struct {
	User                    models.User                              `json:"user"`
	Permissions             []string                                 `json:"permissions"`
	NotificationPreferences bounceCastAccountNotificationPreferences `json:"notificationPreferences"`
	Destinations            []bounceCastAccountDestination           `json:"destinations"`
	Stars                   *models.StarWalletSummary                `json:"stars,omitempty"`
	Leaderboard             []models.StarLeaderboardEntry            `json:"leaderboard"`
	DJProfile               *bounceCastPublicDJProfile               `json:"djProfile,omitempty"`
	UpcomingSchedule        []bounceCastPublicScheduleItem           `json:"upcomingSchedule"`
	Reminders               []bounceCastAccountReminderStatus        `json:"reminders"`
}

type bounceCastScheduleReminderRequest struct {
	ScheduleID          int64  `json:"scheduleId"`
	Email               bool   `json:"email"`
	BrowserPush         bool   `json:"browserPush"`
	BrowserPushEndpoint string `json:"browserPushEndpoint"`
	Messenger           bool   `json:"messenger"`
}

type bounceCastScheduleReminderResponse struct {
	ScheduleID  int64  `json:"scheduleId"`
	Email       bool   `json:"email"`
	BrowserPush bool   `json:"browserPush"`
	Messenger   bool   `json:"messenger"`
	Message     string `json:"message"`
}

type bounceCastAccountReminderStatus struct {
	ID                 int64      `json:"id"`
	ScheduleID         int64      `json:"scheduleId"`
	Title              string     `json:"title"`
	Streamer           string     `json:"streamer,omitempty"`
	StartsAt           time.Time  `json:"startsAt"`
	Email              bool       `json:"email"`
	BrowserPush        bool       `json:"browserPush"`
	Messenger          bool       `json:"messenger"`
	LastQueuedAt       *time.Time `json:"lastQueuedAt,omitempty"`
	LastDeliveryStatus string     `json:"lastDeliveryStatus,omitempty"`
	LastDeliveryError  string     `json:"lastDeliveryError,omitempty"`
	DisabledAt         *time.Time `json:"disabledAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// BounceCastPublicOptions handles preflight for public BounceCast discovery APIs.
func BounceCastPublicOptions(w http.ResponseWriter, r *http.Request) {
	setBounceCastPublicHeaders(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// GetBounceCastPublicDJs returns active DJ profiles without exposing emails,
// stream keys, or dashboard-only account state.
func GetBounceCastPublicDJs(w http.ResponseWriter, r *http.Request) {
	setBounceCastPublicHeaders(w, r)

	djs, err := queryBounceCastPublicDJs("")
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	webutils.WriteResponse(w, djs)
}

// GetBounceCastPublicDJProfile returns one active DJ profile by public handle.
func GetBounceCastPublicDJProfile(w http.ResponseWriter, r *http.Request) {
	setBounceCastPublicHeaders(w, r)

	handle := normalizeBounceCastPublicHandle(chi.URLParam(r, "handle"))
	if handle == "" {
		webutils.BadRequestHandler(w, errors.New("handle is required"))
		return
	}

	djs, err := queryBounceCastPublicDJs(handle)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if len(djs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(webutils.J{"error": "DJ profile not found"})
		return
	}

	schedule, err := queryBounceCastPublicSchedule(bounceCastPublicScheduleFilter{Limit: bounceCastPublicMaxLimit, Handle: handle})
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	dj := djs[0]
	if dj.FeaturedScheduleID != nil {
		for _, item := range schedule {
			if item.ID == *dj.FeaturedScheduleID {
				featured := item
				dj.FeaturedSchedule = &featured
				break
			}
		}
	}

	webutils.WriteResponse(w, bounceCastPublicDJProfile{
		DJ:       dj,
		Schedule: schedule,
	})
}

// GetBounceCastPublicSchedule returns the public lineup for the homepage and DJ pages.
func GetBounceCastPublicSchedule(w http.ResponseWriter, r *http.Request) {
	setBounceCastPublicHeaders(w, r)

	filter, err := parseBounceCastPublicScheduleFilter(r)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	schedule, err := queryBounceCastPublicSchedule(filter)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	webutils.WriteResponse(w, schedule)
}

// BounceCastAccountHub combines the current public account, role destinations,
// Stars summary, and public DJ/schedule context for the unified Account Hub.
func BounceCastAccountHub(user models.User, w http.ResponseWriter, r *http.Request) {
	setBounceCastAccountHeaders(w, r)

	notificationPreferences, err := getBounceCastAccountNotificationPreferences(user.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	permissions := bounceCastPermissionsFromUserScopes(user.Scopes)
	destinations := bounceCastAccountDestinations(user)

	var walletSummary *models.StarWalletSummary
	leaderboard := []models.StarLeaderboardEntry{}
	starsRepository := starsrepository.Get()
	if summary, walletErr := starsRepository.GetWalletSummary(user.ID); walletErr == nil {
		walletSummary = &summary
	}
	if board, boardErr := starsRepository.GetLeaderboard(3); boardErr == nil {
		leaderboard = board
	}

	upcomingSchedule, err := queryBounceCastPublicSchedule(bounceCastPublicScheduleFilter{Limit: 6})
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	reminders, err := queryBounceCastAccountReminderStatuses(user.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	var djProfile *bounceCastPublicDJProfile
	if user.Email != "" && user.IsDJ() {
		profile, profileErr := queryBounceCastDJProfileForAccountEmail(user.Email)
		if profileErr != nil && !errors.Is(profileErr, sql.ErrNoRows) {
			webutils.InternalErrorHandler(w, profileErr)
			return
		}
		if profileErr == nil {
			djProfile = &profile
		}
	}

	webutils.WriteResponse(w, bounceCastAccountHubResponse{
		User:                    user,
		Permissions:             permissions,
		NotificationPreferences: notificationPreferences,
		Destinations:            destinations,
		Stars:                   walletSummary,
		Leaderboard:             leaderboard,
		DJProfile:               djProfile,
		UpcomingSchedule:        upcomingSchedule,
		Reminders:               reminders,
	})
}

// SetBounceCastScheduleReminder stores a logged-in viewer's reminder choices
// for a public scheduled set.
func SetBounceCastScheduleReminder(user models.User, w http.ResponseWriter, r *http.Request) {
	setBounceCastAccountHeaders(w, r)
	if !enforceBounceCastRateLimit(w, r, bounceCastAccountProfileRateLimit, bounceCastRateLimitUserSubject(user.ID), bounceCastRateLimitIPSubject(r)) {
		return
	}

	var request bounceCastScheduleReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if request.ScheduleID == 0 {
		webutils.BadRequestHandler(w, errors.New("scheduleId is required"))
		return
	}
	if !request.Email && !request.BrowserPush && !request.Messenger {
		webutils.BadRequestHandler(w, errors.New("choose at least one reminder channel"))
		return
	}

	notificationPreferences, err := getBounceCastAccountNotificationPreferences(user.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if request.Email && strings.TrimSpace(user.Email) == "" {
		webutils.BadRequestHandler(w, errors.New("email reminders require a registered email"))
		return
	}
	if request.Email && !notificationPreferences.Email {
		webutils.BadRequestHandler(w, errors.New("enable email notifications in Account Hub first"))
		return
	}
	if request.BrowserPush && !notificationPreferences.BrowserPush {
		webutils.BadRequestHandler(w, errors.New("enable browser push notifications in Account Hub first"))
		return
	}
	if request.Messenger && (!notificationPreferences.Messenger || strings.TrimSpace(notificationPreferences.MessengerDestination) == "") {
		webutils.BadRequestHandler(w, errors.New("enable Messenger notifications in Account Hub first"))
		return
	}

	var exists int
	if err := data.GetDatabase().QueryRow(`
		SELECT 1
		FROM bouncecast_stream_schedule s
		LEFT JOIN bouncecast_streamer_accounts a ON a.id = s.streamer_id
		WHERE s.id = ?
			AND s.visibility = 'public'
			AND s.status IN ('planned', 'live')
			AND (s.ends_at IS NULL OR s.ends_at >= datetime('now', '-30 minutes'))
			AND (a.id IS NULL OR a.status = 'active')
		LIMIT 1
	`, request.ScheduleID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			webutils.BadRequestHandler(w, errors.New("public schedule item not found"))
			return
		}
		webutils.InternalErrorHandler(w, err)
		return
	}

	browserPushEndpoint := utils.MakeSafeStringOfLength(request.BrowserPushEndpoint, 2000)
	if request.BrowserPush && browserPushEndpoint != "" {
		if err := saveBounceCastBrowserPushDestination(user.ID, browserPushEndpoint, true); err != nil {
			webutils.BadRequestHandler(w, err)
			return
		}
	}

	if _, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_schedule_reminders(schedule_id, user_id, email, notify_email, notify_push, notify_messenger, messenger_destination, browser_push_endpoint)
		VALUES(?, ?, NULLIF(?, ''), ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))
		ON CONFLICT(schedule_id, user_id) DO UPDATE SET
			email = excluded.email,
			notify_email = excluded.notify_email,
			notify_push = excluded.notify_push,
			notify_messenger = excluded.notify_messenger,
			messenger_destination = excluded.messenger_destination,
			browser_push_endpoint = excluded.browser_push_endpoint,
			disabled_at = NULL,
			updated_at = CURRENT_TIMESTAMP
	`, request.ScheduleID, user.ID, user.Email, bounceCastBoolToInt(request.Email), bounceCastBoolToInt(request.BrowserPush), bounceCastBoolToInt(request.Messenger), notificationPreferences.MessengerDestination, browserPushEndpoint); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	webutils.WriteResponse(w, bounceCastScheduleReminderResponse{
		ScheduleID:  request.ScheduleID,
		Email:       request.Email,
		BrowserPush: request.BrowserPush,
		Messenger:   request.Messenger,
		Message:     "Reminder saved.",
	})
}

func queryBounceCastPublicDJs(handle string) ([]bounceCastPublicDJ, error) {
	query := `
		SELECT a.display_name, a.handle, COALESCE(a.avatar_url, ''), COALESCE(a.bio, ''),
			COALESCE(a.genres, ''), COALESCE(a.social_links, ''), COALESCE(a.hero_image_url, ''), a.role,
			COALESCE(a.seo_title, ''), COALESCE(a.share_image_url, ''), a.featured_schedule_id, a.featured_schedule_enabled,
			COALESCE((
				SELECT s.title
				FROM bouncecast_stream_schedule s
				WHERE s.streamer_id = a.id
					AND s.visibility = 'public'
					AND s.status IN ('planned', 'live')
					AND (s.ends_at IS NULL OR s.ends_at >= datetime('now', '-30 minutes'))
				ORDER BY s.starts_at ASC
				LIMIT 1
			), '') AS upcoming_title,
			(
				SELECT s.starts_at
				FROM bouncecast_stream_schedule s
				WHERE s.streamer_id = a.id
					AND s.visibility = 'public'
					AND s.status IN ('planned', 'live')
					AND (s.ends_at IS NULL OR s.ends_at >= datetime('now', '-30 minutes'))
				ORDER BY s.starts_at ASC
				LIMIT 1
			) AS upcoming_starts_at,
			(
				SELECT COUNT(*)
				FROM bouncecast_stream_schedule s
				WHERE s.streamer_id = a.id
					AND s.visibility = 'public'
					AND s.status IN ('planned', 'live')
					AND (s.ends_at IS NULL OR s.ends_at >= datetime('now', '-30 minutes'))
			) AS upcoming_count,
			(
				SELECT COUNT(*)
				FROM bouncecast_go_live_events e
				WHERE e.streamer_id = a.id
			) AS total_live_events,
			(
				SELECT MAX(e.started_at)
				FROM bouncecast_go_live_events e
				WHERE e.streamer_id = a.id
			) AS last_live_at
		FROM bouncecast_streamer_accounts a
		WHERE a.status = 'active'
	`
	args := []interface{}{}
	if handle != "" {
		query += " AND LOWER(a.handle) = ?"
		args = append(args, handle)
	}
	query += " ORDER BY a.display_name ASC"

	rows, err := data.GetDatabase().Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	djs := []bounceCastPublicDJ{}
	for rows.Next() {
		var dj bounceCastPublicDJ
		var upcomingStarts sql.NullString
		var lastLiveAt sql.NullString
		var featuredScheduleID sql.NullInt64
		var featuredScheduleEnabled bool
		var genresJSON string
		var socialLinksJSON string
		if err := rows.Scan(
			&dj.DisplayName,
			&dj.Handle,
			&dj.AvatarURL,
			&dj.Bio,
			&genresJSON,
			&socialLinksJSON,
			&dj.HeroImageURL,
			&dj.Role,
			&dj.SEOTitle,
			&dj.ShareImageURL,
			&featuredScheduleID,
			&featuredScheduleEnabled,
			&dj.UpcomingSet,
			&upcomingStarts,
			&dj.UpcomingCount,
			&dj.TotalLiveEvents,
			&lastLiveAt,
		); err != nil {
			return nil, err
		}
		dj.Genres = parseBounceCastGenres(genresJSON)
		dj.SocialLinks = parseBounceCastSocialLinks(socialLinksJSON)
		dj.Role = publicBounceCastDJRole(dj.Role)
		if featuredScheduleEnabled && featuredScheduleID.Valid {
			dj.FeaturedScheduleID = &featuredScheduleID.Int64
		}
		if upcomingStarts.Valid {
			if parsed, err := parseBounceCastSQLiteTime(upcomingStarts.String); err == nil {
				dj.UpcomingStarts = &parsed
			}
		}
		if lastLiveAt.Valid {
			if parsed, err := parseBounceCastSQLiteTime(lastLiveAt.String); err == nil {
				dj.LastLiveAt = &parsed
			}
		}
		djs = append(djs, dj)
	}
	return djs, rows.Err()
}

func queryBounceCastPublicSchedule(filter bounceCastPublicScheduleFilter) ([]bounceCastPublicScheduleItem, error) {
	if filter.Limit == 0 {
		filter.Limit = bounceCastPublicDefaultLimit
	}
	query := `
		SELECT s.id, s.title, COALESCE(s.description, ''), s.starts_at, s.ends_at,
			s.timezone, s.status, COALESCE(a.display_name, ''), COALESCE(a.handle, ''),
			COALESCE(a.avatar_url, '')
		FROM bouncecast_stream_schedule s
		LEFT JOIN bouncecast_streamer_accounts a ON a.id = s.streamer_id
		WHERE s.visibility = 'public'
			AND (a.id IS NULL OR a.status = 'active')
	`
	args := []interface{}{}
	switch filter.Status {
	case "planned", "live":
		query += " AND s.status = ?"
		args = append(args, filter.Status)
	case "all":
		query += " AND s.status IN ('planned', 'live')"
	default:
		query += " AND s.status IN ('planned', 'live')"
	}
	if !filter.IncludePast && filter.From == nil {
		query += " AND (s.ends_at IS NULL OR s.ends_at >= datetime('now', '-30 minutes'))"
	}
	if filter.From != nil {
		query += " AND (s.ends_at IS NULL OR s.ends_at >= ?)"
		args = append(args, filter.From.UTC())
	}
	if filter.To != nil {
		query += " AND s.starts_at <= ?"
		args = append(args, filter.To.UTC())
	}
	if filter.Handle != "" {
		query += " AND LOWER(COALESCE(a.handle, '')) = ?"
		args = append(args, filter.Handle)
	}
	if filter.Query != "" {
		query += " AND (LOWER(s.title) LIKE ? OR LOWER(COALESCE(s.description, '')) LIKE ? OR LOWER(COALESCE(a.display_name, '')) LIKE ? OR LOWER(COALESCE(a.genres, '')) LIKE ?)"
		search := "%" + filter.Query + "%"
		args = append(args, search, search, search, search)
	}
	query += " ORDER BY s.starts_at ASC LIMIT ?"
	args = append(args, filter.Limit)

	rows, err := data.GetDatabase().Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schedule := []bounceCastPublicScheduleItem{}
	for rows.Next() {
		var item bounceCastPublicScheduleItem
		var endsAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.Description,
			&item.StartsAt,
			&endsAt,
			&item.Timezone,
			&item.Status,
			&item.Streamer,
			&item.Handle,
			&item.AvatarURL,
		); err != nil {
			return nil, err
		}
		if endsAt.Valid {
			item.EndsAt = &endsAt.Time
		}
		schedule = append(schedule, item)
	}
	return schedule, rows.Err()
}

func queryBounceCastDJProfileForAccountEmail(email string) (bounceCastPublicDJProfile, error) {
	var handle string
	if err := data.GetDatabase().QueryRow(`
		SELECT handle
		FROM bouncecast_streamer_accounts
		WHERE LOWER(COALESCE(email, '')) = LOWER(?)
			AND status = 'active'
		LIMIT 1
	`, strings.TrimSpace(email)).Scan(&handle); err != nil {
		return bounceCastPublicDJProfile{}, err
	}

	djs, err := queryBounceCastPublicDJs(normalizeBounceCastPublicHandle(handle))
	if err != nil {
		return bounceCastPublicDJProfile{}, err
	}
	if len(djs) == 0 {
		return bounceCastPublicDJProfile{}, sql.ErrNoRows
	}
	schedule, err := queryBounceCastPublicSchedule(bounceCastPublicScheduleFilter{Limit: bounceCastPublicMaxLimit, Handle: normalizeBounceCastPublicHandle(handle)})
	if err != nil {
		return bounceCastPublicDJProfile{}, err
	}
	dj := djs[0]
	if dj.FeaturedScheduleID != nil {
		for _, item := range schedule {
			if item.ID == *dj.FeaturedScheduleID {
				featured := item
				dj.FeaturedSchedule = &featured
				break
			}
		}
	}
	return bounceCastPublicDJProfile{DJ: dj, Schedule: schedule}, nil
}

func queryBounceCastAccountReminderStatuses(userID string) ([]bounceCastAccountReminderStatus, error) {
	rows, err := data.GetDatabase().Query(`
		SELECT r.id, r.schedule_id, s.title, COALESCE(a.display_name, ''), s.starts_at,
			r.notify_email, r.notify_push, r.notify_messenger,
			r.last_queued_at, COALESCE(r.last_delivery_status, ''), COALESCE(r.last_delivery_error, ''),
			r.disabled_at, r.created_at, r.updated_at
		FROM bouncecast_schedule_reminders r
		INNER JOIN bouncecast_stream_schedule s ON s.id = r.schedule_id
		LEFT JOIN bouncecast_streamer_accounts a ON a.id = s.streamer_id
		WHERE r.user_id = ?
		ORDER BY s.starts_at ASC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reminders := []bounceCastAccountReminderStatus{}
	for rows.Next() {
		var reminder bounceCastAccountReminderStatus
		var lastQueuedAt sql.NullTime
		var disabledAt sql.NullTime
		if err := rows.Scan(
			&reminder.ID,
			&reminder.ScheduleID,
			&reminder.Title,
			&reminder.Streamer,
			&reminder.StartsAt,
			&reminder.Email,
			&reminder.BrowserPush,
			&reminder.Messenger,
			&lastQueuedAt,
			&reminder.LastDeliveryStatus,
			&reminder.LastDeliveryError,
			&disabledAt,
			&reminder.CreatedAt,
			&reminder.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if lastQueuedAt.Valid {
			reminder.LastQueuedAt = &lastQueuedAt.Time
		}
		if disabledAt.Valid {
			reminder.DisabledAt = &disabledAt.Time
		}
		reminders = append(reminders, reminder)
	}
	return reminders, rows.Err()
}

func bounceCastPermissionsFromUserScopes(scopes []string) []string {
	return bounceCastPermissionsFromScopes(scopes)
}

func bounceCastAccountDestinations(user models.User) []bounceCastAccountDestination {
	return []bounceCastAccountDestination{
		{
			Key:       "watch",
			Label:     "Watch live stream",
			URL:       "/",
			Available: true,
		},
		{
			Key:       "admin",
			Label:     "Admin control panel",
			URL:       "/admin/",
			Available: user.IsOwner() || user.IsAdmin(),
			Reason:    "Requires owner or admin permission.",
		},
		{
			Key:       "studio",
			Label:     "DJ Studio dashboard",
			URL:       "/studio",
			Available: user.IsDJ(),
			Reason:    "Requires DJ permission and an active Studio account.",
		},
		{
			Key:       "moderation",
			Label:     "Chat moderation tools",
			URL:       "/",
			Available: user.IsModerator(),
			Reason:    "Requires moderator permission.",
		},
	}
}

func normalizeBounceCastPublicHandle(handle string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(handle, "@")))
}

func parseBounceCastPublicLimit(value string) int {
	limit, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || limit <= 0 {
		return bounceCastPublicDefaultLimit
	}
	if limit > bounceCastPublicMaxLimit {
		return bounceCastPublicMaxLimit
	}
	return limit
}

func parseBounceCastPublicScheduleFilter(r *http.Request) (bounceCastPublicScheduleFilter, error) {
	query := r.URL.Query()
	filter := bounceCastPublicScheduleFilter{
		Limit:       parseBounceCastPublicLimit(query.Get("limit")),
		Handle:      normalizeBounceCastPublicHandle(query.Get("handle")),
		Status:      strings.ToLower(strings.TrimSpace(query.Get("status"))),
		Query:       strings.ToLower(strings.TrimSpace(query.Get("q"))),
		IncludePast: query.Get("includePast") == "true" || query.Get("past") == "true",
	}
	if filter.Status == "" {
		filter.Status = "all"
	}
	if filter.Status != "all" && filter.Status != "planned" && filter.Status != "live" {
		return bounceCastPublicScheduleFilter{}, errors.New("status must be all, planned, or live")
	}
	if len(filter.Query) > 80 {
		filter.Query = filter.Query[:80]
	}

	if from := strings.TrimSpace(query.Get("from")); from != "" {
		parsed, err := parseBounceCastPublicDateFilter(from)
		if err != nil {
			return bounceCastPublicScheduleFilter{}, errors.New("from must be RFC3339 or YYYY-MM-DD")
		}
		filter.From = &parsed
	}
	if to := strings.TrimSpace(query.Get("to")); to != "" {
		parsed, err := parseBounceCastPublicDateFilter(to)
		if err != nil {
			return bounceCastPublicScheduleFilter{}, errors.New("to must be RFC3339 or YYYY-MM-DD")
		}
		filter.To = &parsed
	}
	if filter.From != nil && filter.To != nil && filter.To.Before(*filter.From) {
		return bounceCastPublicScheduleFilter{}, errors.New("to must be after from")
	}
	return filter, nil
}

func parseBounceCastPublicDateFilter(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, errors.New("invalid date")
}

func publicBounceCastDJRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "owner":
		return "resident"
	case "manager", "moderator":
		return "host"
	default:
		return "dj"
	}
}

func parseBounceCastSQLiteTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, errors.New("unsupported time format")
}

func setBounceCastPublicHeaders(w http.ResponseWriter, r *http.Request) {
	middleware.SetBounceCastCORSHeaders(w, r, "GET, OPTIONS")
}
