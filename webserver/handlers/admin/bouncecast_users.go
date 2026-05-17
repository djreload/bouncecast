package admin

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/owncast/owncast/core/chat"
	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/utils"
	webutils "github.com/owncast/owncast/webserver/utils"
)

type BounceCastUserAccount struct {
	ID              string     `json:"id"`
	DisplayName     string     `json:"displayName"`
	Email           string     `json:"email"`
	ProfileImageURL string     `json:"profileImageUrl"`
	Permissions     []string   `json:"permissions"`
	Scopes          []string   `json:"scopes"`
	DisplayColor    int        `json:"displayColor"`
	Registered      bool       `json:"registered"`
	Authenticated   bool       `json:"authenticated"`
	MessageCount    int64      `json:"messageCount"`
	CreatedAt       time.Time  `json:"createdAt"`
	DisabledAt      *time.Time `json:"disabledAt,omitempty"`
	RegisteredAt    *time.Time `json:"registeredAt,omitempty"`
	LastLoginAt     *time.Time `json:"lastLoginAt,omitempty"`
}

type setBounceCastUserPermissionsRequest struct {
	UserID      string   `json:"userId"`
	Permissions []string `json:"permissions"`
}

var bounceCastUserPermissionToScope = map[string]string{
	"owner":     models.BounceCastOwnerScopeKey,
	"admin":     models.BounceCastAdminScopeKey,
	"moderator": models.ModeratorScopeKey,
	"dj":        models.BounceCastDJScopeKey,
}

var bounceCastUserScopeToPermission = map[string]string{
	models.BounceCastOwnerScopeKey: "owner",
	models.BounceCastAdminScopeKey: "admin",
	models.ModeratorScopeKey:       "moderator",
	models.BounceCastDJScopeKey:    "dj",
}

// GetBounceCastUsers returns public chat users and their BounceCast role flags.
func GetBounceCastUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := data.GetDatabase().Query(`
		SELECT u.id, u.display_name, COALESCE(u.email, ''), COALESCE(u.profile_image_url, ''),
			COALESCE(u.scopes, ''), u.display_color, u.created_at, u.disabled_at, u.authenticated_at,
			u.registered_at, u.last_login_at, COUNT(m.id)
		FROM users u
		LEFT JOIN messages m ON m.user_id = u.id
		WHERE u.type IS NULL OR u.type != 'API'
		GROUP BY u.id
		ORDER BY COALESCE(u.last_login_at, u.registered_at, u.created_at) DESC
	`)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	defer rows.Close()

	users := []BounceCastUserAccount{}
	for rows.Next() {
		var user BounceCastUserAccount
		var scopes string
		var disabledAt sql.NullTime
		var authenticatedAt sql.NullTime
		var registeredAt sql.NullTime
		var lastLoginAt sql.NullTime
		if err := rows.Scan(
			&user.ID,
			&user.DisplayName,
			&user.Email,
			&user.ProfileImageURL,
			&scopes,
			&user.DisplayColor,
			&user.CreatedAt,
			&disabledAt,
			&authenticatedAt,
			&registeredAt,
			&lastLoginAt,
			&user.MessageCount,
		); err != nil {
			webutils.InternalErrorHandler(w, err)
			return
		}
		user.Scopes = splitBounceCastScopes(scopes)
		user.Permissions = bounceCastPermissionsFromScopes(user.Scopes)
		user.Registered = strings.TrimSpace(user.Email) != "" && registeredAt.Valid
		user.Authenticated = authenticatedAt.Valid
		if disabledAt.Valid {
			user.DisabledAt = &disabledAt.Time
		}
		if registeredAt.Valid {
			user.RegisteredAt = &registeredAt.Time
		}
		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	webutils.WriteResponse(w, users)
}

// SetBounceCastUserPermissions replaces only the BounceCast account role flags
// while preserving unrelated internal/access-token scopes.
func SetBounceCastUserPermissions(w http.ResponseWriter, r *http.Request) {
	var request setBounceCastUserPermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	request.UserID = strings.TrimSpace(request.UserID)
	if request.UserID == "" {
		webutils.BadRequestHandler(w, errors.New("userId is required"))
		return
	}

	nextScopes, err := normalizeBounceCastPermissionScopes(request.Permissions)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	db := data.GetDatabase()
	var scopesString sql.NullString
	if err := db.QueryRow("SELECT scopes FROM users WHERE id = ? AND (type IS NULL OR type != 'API')", request.UserID).Scan(&scopesString); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			webutils.BadRequestHandler(w, errors.New("user not found"))
			return
		}
		webutils.InternalErrorHandler(w, err)
		return
	}

	preservedScopes := []string{}
	for _, scope := range splitBounceCastScopes(scopesString.String) {
		if _, isBounceCastPermission := bounceCastUserScopeToPermission[scope]; !isBounceCastPermission {
			preservedScopes = append(preservedScopes, scope)
		}
	}

	finalScopes := uniqueSortedScopes(append(preservedScopes, nextScopes...))
	finalScopeString := strings.Join(finalScopes, ",")

	datastore := data.GetDatastore()
	if err := func() error {
		datastore.DbLock.Lock()
		defer datastore.DbLock.Unlock()

		var value interface{}
		if finalScopeString != "" {
			value = finalScopeString
		}
		result, err := datastore.DB.Exec("UPDATE users SET scopes = ? WHERE id = ?", value, request.UserID)
		if err != nil {
			return err
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			return errors.New("user not found")
		}
		return nil
	}(); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	_ = chat.SendConnectedClientInfoToUser(request.UserID)
	webutils.WriteSimpleResponse(w, true, "updated user permissions")
}

func normalizeBounceCastPermissionScopes(permissions []string) ([]string, error) {
	scopes := []string{}
	seen := map[string]bool{}
	for _, permission := range permissions {
		normalized := strings.ToLower(strings.TrimSpace(permission))
		if normalized == "" || normalized == "visitor" {
			continue
		}
		scope, ok := bounceCastUserPermissionToScope[normalized]
		if !ok {
			return nil, errors.New("permissions must be visitor, owner, admin, moderator, or dj")
		}
		if !seen[scope] {
			seen[scope] = true
			scopes = append(scopes, scope)
		}
	}
	return scopes, nil
}

func bounceCastPermissionsFromScopes(scopes []string) []string {
	permissions := []string{}
	for _, scope := range scopes {
		if permission, ok := bounceCastUserScopeToPermission[scope]; ok {
			permissions = append(permissions, permission)
		}
	}
	sort.Strings(permissions)
	if len(permissions) == 0 {
		return []string{"visitor"}
	}
	return permissions
}

func splitBounceCastScopes(scopesString string) []string {
	parts := strings.Split(scopesString, ",")
	scopes := make([]string, 0, len(parts))
	for _, scope := range parts {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			scopes = append(scopes, scope)
		}
	}
	return scopes
}

func uniqueSortedScopes(scopes []string) []string {
	scopeMap := utils.StringSliceToMap(scopes)
	finalScopes := utils.StringMapKeys(scopeMap)
	sort.Strings(finalScopes)
	return finalScopes
}
