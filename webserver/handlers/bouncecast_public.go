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
	webutils "github.com/owncast/owncast/webserver/utils"
)

const (
	bounceCastPublicDefaultLimit = 12
	bounceCastPublicMaxLimit     = 50
)

type bounceCastPublicDJ struct {
	DisplayName     string     `json:"displayName"`
	Handle          string     `json:"handle"`
	AvatarURL       string     `json:"avatarUrl,omitempty"`
	Role            string     `json:"role"`
	UpcomingSet     string     `json:"upcomingSet,omitempty"`
	UpcomingStarts  *time.Time `json:"upcomingStarts,omitempty"`
	UpcomingCount   int        `json:"upcomingCount"`
	TotalLiveEvents int        `json:"totalLiveEvents"`
	LastLiveAt      *time.Time `json:"lastLiveAt,omitempty"`
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
}

// BounceCastPublicOptions handles preflight for public BounceCast discovery APIs.
func BounceCastPublicOptions(w http.ResponseWriter, r *http.Request) {
	setBounceCastPublicHeaders(w)
	w.WriteHeader(http.StatusNoContent)
}

// GetBounceCastPublicDJs returns active DJ profiles without exposing emails,
// stream keys, or dashboard-only account state.
func GetBounceCastPublicDJs(w http.ResponseWriter, r *http.Request) {
	setBounceCastPublicHeaders(w)

	djs, err := queryBounceCastPublicDJs("")
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	webutils.WriteResponse(w, djs)
}

// GetBounceCastPublicDJProfile returns one active DJ profile by public handle.
func GetBounceCastPublicDJProfile(w http.ResponseWriter, r *http.Request) {
	setBounceCastPublicHeaders(w)

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

	schedule, err := queryBounceCastPublicSchedule(bounceCastPublicMaxLimit, handle)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	webutils.WriteResponse(w, bounceCastPublicDJProfile{
		DJ:       djs[0],
		Schedule: schedule,
	})
}

// GetBounceCastPublicSchedule returns the public lineup for the homepage and DJ pages.
func GetBounceCastPublicSchedule(w http.ResponseWriter, r *http.Request) {
	setBounceCastPublicHeaders(w)

	limit := parseBounceCastPublicLimit(r.URL.Query().Get("limit"))
	handle := normalizeBounceCastPublicHandle(r.URL.Query().Get("handle"))
	schedule, err := queryBounceCastPublicSchedule(limit, handle)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	webutils.WriteResponse(w, schedule)
}

// BounceCastAccountHub combines the current public account, role destinations,
// Stars summary, and public DJ/schedule context for the unified Account Hub.
func BounceCastAccountHub(user models.User, w http.ResponseWriter, r *http.Request) {
	setBounceCastAccountHeaders(w)

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

	upcomingSchedule, err := queryBounceCastPublicSchedule(6, "")
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
	})
}

func queryBounceCastPublicDJs(handle string) ([]bounceCastPublicDJ, error) {
	query := `
		SELECT a.display_name, a.handle, COALESCE(a.avatar_url, ''), a.role,
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
		if err := rows.Scan(
			&dj.DisplayName,
			&dj.Handle,
			&dj.AvatarURL,
			&dj.Role,
			&dj.UpcomingSet,
			&upcomingStarts,
			&dj.UpcomingCount,
			&dj.TotalLiveEvents,
			&lastLiveAt,
		); err != nil {
			return nil, err
		}
		dj.Role = publicBounceCastDJRole(dj.Role)
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

func queryBounceCastPublicSchedule(limit int, handle string) ([]bounceCastPublicScheduleItem, error) {
	query := `
		SELECT s.id, s.title, COALESCE(s.description, ''), s.starts_at, s.ends_at,
			s.timezone, s.status, COALESCE(a.display_name, ''), COALESCE(a.handle, ''),
			COALESCE(a.avatar_url, '')
		FROM bouncecast_stream_schedule s
		LEFT JOIN bouncecast_streamer_accounts a ON a.id = s.streamer_id
		WHERE s.visibility = 'public'
			AND s.status IN ('planned', 'live')
			AND (s.ends_at IS NULL OR s.ends_at >= datetime('now', '-30 minutes'))
			AND (a.id IS NULL OR a.status = 'active')
	`
	args := []interface{}{}
	if handle != "" {
		query += " AND LOWER(COALESCE(a.handle, '')) = ?"
		args = append(args, handle)
	}
	query += " ORDER BY s.starts_at ASC LIMIT ?"
	args = append(args, limit)

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
	schedule, err := queryBounceCastPublicSchedule(bounceCastPublicMaxLimit, normalizeBounceCastPublicHandle(handle))
	if err != nil {
		return bounceCastPublicDJProfile{}, err
	}
	return bounceCastPublicDJProfile{DJ: djs[0], Schedule: schedule}, nil
}

func bounceCastPermissionsFromUserScopes(scopes []string) []string {
	scopeMap := map[string]bool{}
	for _, scope := range scopes {
		scopeMap[scope] = true
	}

	permissions := []string{}
	if scopeMap[models.BounceCastOwnerScopeKey] {
		permissions = append(permissions, "owner")
	}
	if scopeMap[models.BounceCastAdminScopeKey] {
		permissions = append(permissions, "admin")
	}
	if scopeMap[models.ModeratorScopeKey] {
		permissions = append(permissions, "moderator")
	}
	if scopeMap[models.BounceCastDJScopeKey] {
		permissions = append(permissions, "dj")
	}
	if len(permissions) == 0 {
		return []string{"visitor"}
	}
	return permissions
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
			URL:       "/login",
			Available: user.IsOwner() || user.IsAdmin(),
			Reason:    "Requires owner or admin permission.",
		},
		{
			Key:       "studio",
			Label:     "DJ Studio dashboard",
			URL:       "/login",
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

func setBounceCastPublicHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
}
