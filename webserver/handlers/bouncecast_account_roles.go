package handlers

import (
	"database/sql"
	"strings"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
)

type bounceCastAccountRoleUser struct {
	ID              string
	DisplayName     string
	Email           string
	PasswordHash    string
	ProfileImageURL string
	Scopes          []string
}

func getBounceCastAccountRoleUserForLogin(login string) (bounceCastAccountRoleUser, error) {
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" {
		return bounceCastAccountRoleUser{}, sql.ErrNoRows
	}

	var user bounceCastAccountRoleUser
	var passwordHash sql.NullString
	var profileImageURL sql.NullString
	var scopes sql.NullString
	if err := data.GetDatabase().QueryRow(`
		SELECT id, display_name, LOWER(COALESCE(email, '')), password_hash, profile_image_url, scopes
		FROM users
		WHERE LOWER(COALESCE(email, '')) = ?
			AND disabled_at IS NULL
			AND (type IS NULL OR type != 'API')
		LIMIT 1
	`, login).Scan(
		&user.ID,
		&user.DisplayName,
		&user.Email,
		&passwordHash,
		&profileImageURL,
		&scopes,
	); err != nil {
		return bounceCastAccountRoleUser{}, err
	}

	if passwordHash.Valid {
		user.PasswordHash = passwordHash.String
	}
	if profileImageURL.Valid {
		user.ProfileImageURL = profileImageURL.String
	}
	if scopes.Valid {
		user.Scopes = splitBounceCastAccountScopes(scopes.String)
	}
	return user, nil
}

func getBounceCastAccountRoleUserByEmail(email string) (bounceCastAccountRoleUser, error) {
	return getBounceCastAccountRoleUserForLogin(email)
}

func (u bounceCastAccountRoleUser) hasAnyRole(roles ...string) bool {
	scopeMap := map[string]bool{}
	for _, scope := range u.Scopes {
		scopeMap[scope] = true
	}
	for _, role := range roles {
		if scopeMap[role] {
			return true
		}
	}
	return false
}

func (u bounceCastAccountRoleUser) canUseAdmin() bool {
	return u.hasAnyRole(models.BounceCastOwnerScopeKey, models.BounceCastAdminScopeKey)
}

func (u bounceCastAccountRoleUser) canUseStudio() bool {
	return u.hasAnyRole(models.BounceCastDJScopeKey)
}

func (u bounceCastAccountRoleUser) adminRoleName() string {
	if u.hasAnyRole(models.BounceCastOwnerScopeKey) {
		return "owner"
	}
	return "admin"
}

func splitBounceCastAccountScopes(scopesString string) []string {
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
