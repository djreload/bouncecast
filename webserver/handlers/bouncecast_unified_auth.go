package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/owncast/owncast/config"
	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/userrepository"
	"github.com/owncast/owncast/utils"
	"github.com/owncast/owncast/webserver/router/middleware"
	webutils "github.com/owncast/owncast/webserver/utils"
	"github.com/teris-io/shortid"
)

const bounceCastLegacyAdminUserID = "owncast-admin"

type bounceCastUnifiedLoginRequest struct {
	Login    string `json:"login"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Next     string `json:"next"`
}

type bounceCastUnifiedLoginResponse struct {
	User            *models.User              `json:"user,omitempty"`
	Role            string                    `json:"role"`
	Permissions     []string                  `json:"permissions"`
	Destination     string                    `json:"destination"`
	Message         string                    `json:"message"`
	AccessToken     string                    `json:"accessToken,omitempty"`
	StudioToken     string                    `json:"studioToken,omitempty"`
	StudioExpiresAt *time.Time                `json:"studioExpiresAt,omitempty"`
	Streamer        *bounceCastStudioStreamer `json:"streamer,omitempty"`
}

type bounceCastUnifiedSession struct {
	user            *models.User
	streamer        *bounceCastStudioStreamer
	studioExpiresAt *time.Time
	role            string
	permissions     []string
	accessToken     string
	studioToken     string
	canAdmin        bool
	canStudio       bool
}

// BounceCastUnifiedAuthOptions handles CORS preflight for the single
// user-facing login/logout flow.
func BounceCastUnifiedAuthOptions(w http.ResponseWriter, r *http.Request) {
	setBounceCastUnifiedAuthHeaders(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// BounceCastUnifiedLogin is the single browser login entry point. It preserves
// the original Owncast admin password, BounceCast account roles, and existing
// Studio streamer accounts, then returns every session token the browser needs.
func BounceCastUnifiedLogin(w http.ResponseWriter, r *http.Request) {
	setBounceCastUnifiedAuthHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !enforceBounceCastRateLimit(w, r, bounceCastUnifiedLoginIPRateLimit, bounceCastRateLimitIPSubject(r)) {
		return
	}

	var request bounceCastUnifiedLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	login := normalizeBounceCastUnifiedLogin(request)
	password := strings.TrimSpace(request.Password)
	if login == "" || password == "" {
		writeBounceCastUnifiedUnauthorized(w)
		return
	}
	if !enforceBounceCastRateLimit(w, r, bounceCastUnifiedLoginIdentityRateLimit, bounceCastRateLimitEmailSubject(login)) {
		return
	}

	session, err := authenticateBounceCastUnifiedLogin(w, r, login, password)
	if err != nil {
		writeBounceCastUnifiedUnauthorized(w)
		return
	}

	webutils.WriteResponse(w, bounceCastUnifiedLoginResponse{
		User:            session.user,
		Role:            session.role,
		Permissions:     session.permissions,
		Destination:     resolveBounceCastUnifiedDestination(request.Next, session),
		Message:         "Logged in.",
		AccessToken:     session.accessToken,
		StudioToken:     session.studioToken,
		StudioExpiresAt: session.studioExpiresAt,
		Streamer:        session.streamer,
	})
}

// BounceCastUnifiedLogout clears the browser admin cookies and revokes optional
// bearer/access tokens supplied by the frontend. Legacy Basic auth is not a
// server-side session, so it cannot be cleared here.
func BounceCastUnifiedLogout(w http.ResponseWriter, r *http.Request) {
	setBounceCastUnifiedAuthHeaders(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	middleware.ClearAdminSessionCookies(w, r)

	var request struct {
		AccessToken string `json:"accessToken"`
		StudioToken string `json:"studioToken"`
	}
	_ = json.NewDecoder(r.Body).Decode(&request)

	if token := strings.TrimSpace(request.AccessToken); token != "" {
		_, _ = data.GetDatabase().Exec(`DELETE FROM user_access_tokens WHERE token = ?`, token)
	}
	if token := strings.TrimSpace(request.StudioToken); token != "" {
		_, _ = data.GetDatabase().Exec(`
			UPDATE bouncecast_streamer_sessions
			SET revoked_at = CURRENT_TIMESTAMP
			WHERE token_hash = ?
		`, hashBounceCastStudioToken(token))
	}

	webutils.WriteSimpleResponse(w, true, "logged out")
}

func BounceCastUnifiedLoginRedirect(w http.ResponseWriter, r *http.Request) {
	next := strings.TrimSpace(r.URL.Query().Get("next"))
	if !isSafeBounceCastRedirect(next) {
		next = "/"
		if strings.HasPrefix(r.URL.Path, "/admin") {
			next = "/admin/"
		}
		if strings.HasPrefix(r.URL.Path, "/studio") {
			next = "/studio"
		}
	}
	http.Redirect(w, r, "/login?next="+url.QueryEscape(next), http.StatusFound)
}

func authenticateBounceCastUnifiedLogin(w http.ResponseWriter, r *http.Request, login string, password string) (bounceCastUnifiedSession, error) {
	if middleware.CheckAdminCredentials(login, password) {
		middleware.SetAdminSessionCookie(w, r)
		user, accessToken, err := ensureBounceCastLegacyAdminAccountToken()
		if err != nil {
			return bounceCastUnifiedSession{}, err
		}
		session := bounceCastUnifiedSession{
			user:        user,
			role:        "owner",
			permissions: bounceCastPermissionsFromScopes(user.Scopes),
			accessToken: accessToken,
			canAdmin:    true,
		}
		session.attachStudioForRoleUser(r, bounceCastAccountRoleUser{
			ID:              user.ID,
			DisplayName:     user.DisplayName,
			Email:           user.Email,
			ProfileImageURL: user.ProfileImageURL,
			Scopes:          user.Scopes,
		})
		return session, nil
	}

	if session, err := authenticateBounceCastUnifiedAccount(w, r, login, password); err == nil {
		return session, nil
	}

	return authenticateBounceCastUnifiedStudio(r, login, password)
}

func authenticateBounceCastUnifiedAccount(w http.ResponseWriter, r *http.Request, login string, password string) (bounceCastUnifiedSession, error) {
	email, err := normalizeBounceCastAccountEmail(login)
	if err != nil {
		return bounceCastUnifiedSession{}, err
	}

	roleUser, err := getBounceCastAccountRoleUserForLogin(email)
	if err != nil || roleUser.PasswordHash == "" {
		return bounceCastUnifiedSession{}, errors.New("account not found")
	}
	if err := utils.CompareHash(roleUser.PasswordHash, password); err != nil {
		return bounceCastUnifiedSession{}, err
	}

	accessToken, err := issueBounceCastAccountAccessToken(roleUser.ID)
	if err != nil {
		return bounceCastUnifiedSession{}, err
	}
	user := userrepository.Get().GetUserByID(roleUser.ID)
	if user == nil {
		return bounceCastUnifiedSession{}, errors.New("account user not found")
	}

	if roleUser.canUseAdmin() {
		middleware.SetAdminRoleSessionCookie(w, r, roleUser.ID, roleUser.adminRoleName())
	}

	session := bounceCastUnifiedSession{
		user:        user,
		role:        bounceCastPrimaryRoleFromScopes(user.Scopes),
		permissions: bounceCastPermissionsFromScopes(user.Scopes),
		accessToken: accessToken,
		canAdmin:    roleUser.canUseAdmin(),
	}
	session.attachStudioForRoleUser(r, roleUser)
	return session, nil
}

func authenticateBounceCastUnifiedStudio(r *http.Request, login string, password string) (bounceCastUnifiedSession, error) {
	studioLogin := normalizeBounceCastStudioLogin(login)
	streamer, passwordHash, err := getBounceCastStudioStreamerForLogin(studioLogin)
	if err != nil || streamer.Status != "active" || passwordHash == "" {
		return bounceCastUnifiedSession{}, errors.New("studio account not found")
	}
	if err := utils.CompareHash(passwordHash, password); err != nil {
		return bounceCastUnifiedSession{}, err
	}
	allowed, err := canBounceCastStudioStreamerUseStoredRole(streamer)
	if err != nil || !allowed {
		if err != nil {
			return bounceCastUnifiedSession{}, err
		}
		return bounceCastUnifiedSession{}, errors.New("studio account role is inactive")
	}

	studioToken, expiresAt, err := createBounceCastStudioSession(r, streamer)
	if err != nil {
		return bounceCastUnifiedSession{}, err
	}

	user, accessToken, err := ensureBounceCastAccountForStudioStreamer(streamer, passwordHash)
	if err != nil {
		return bounceCastUnifiedSession{}, err
	}

	session := bounceCastUnifiedSession{
		user:            user,
		role:            "dj",
		permissions:     []string{"dj"},
		accessToken:     accessToken,
		studioToken:     studioToken,
		studioExpiresAt: &expiresAt,
		streamer:        &streamer,
		canStudio:       true,
	}
	if user != nil {
		session.permissions = bounceCastPermissionsFromScopes(user.Scopes)
		session.role = bounceCastPrimaryRoleFromScopes(user.Scopes)
	}
	return session, nil
}

func (s *bounceCastUnifiedSession) attachStudioForRoleUser(r *http.Request, roleUser bounceCastAccountRoleUser) {
	if !roleUser.canUseStudio() {
		return
	}
	streamer, err := ensureBounceCastUnifiedStudioStreamer(roleUser)
	if err != nil {
		return
	}
	token, expiresAt, err := createBounceCastStudioSession(r, streamer)
	if err != nil {
		return
	}
	s.studioToken = token
	s.studioExpiresAt = &expiresAt
	s.streamer = &streamer
	s.canStudio = true
}

func ensureBounceCastUnifiedStudioStreamer(roleUser bounceCastAccountRoleUser) (bounceCastStudioStreamer, error) {
	if strings.TrimSpace(roleUser.Email) != "" {
		return provisionBounceCastStudioStreamerFromRoleUser(roleUser)
	}

	handle := "bouncecast-owner"
	streamer, _, err := getBounceCastStudioStreamerForLogin(handle)
	if err == nil {
		if streamer.Status != "active" || streamer.DisplayName != roleUser.DisplayName || streamer.AvatarURL != roleUser.ProfileImageURL {
			if _, updateErr := data.GetDatabase().Exec(`
				UPDATE bouncecast_streamer_accounts
				SET display_name = ?, password_hash = NULL, role = 'streamer', status = 'active',
					avatar_url = NULLIF(?, ''), updated_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, roleUser.DisplayName, roleUser.ProfileImageURL, streamer.ID); updateErr != nil {
				return bounceCastStudioStreamer{}, updateErr
			}
			streamer, _, err = getBounceCastStudioStreamerForLogin(handle)
		}
		return streamer, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return bounceCastStudioStreamer{}, err
	}

	result, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_streamer_accounts(display_name, handle, email, password_hash, role, status, avatar_url)
		VALUES(?, ?, NULL, NULL, 'streamer', 'active', NULLIF(?, ''))
	`, roleUser.DisplayName, handle, roleUser.ProfileImageURL)
	if err != nil {
		return bounceCastStudioStreamer{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return bounceCastStudioStreamer{}, err
	}
	return bounceCastStudioStreamer{
		ID:          id,
		DisplayName: roleUser.DisplayName,
		Handle:      handle,
		Role:        "streamer",
		Status:      "active",
		AvatarURL:   roleUser.ProfileImageURL,
	}, nil
}

func ensureBounceCastLegacyAdminAccountToken() (*models.User, string, error) {
	scopes := strings.Join([]string{
		models.BounceCastOwnerScopeKey,
		models.BounceCastAdminScopeKey,
		models.ModeratorScopeKey,
		models.BounceCastDJScopeKey,
	}, ",")

	db := data.GetDatabase()
	if _, err := db.Exec(`
		INSERT INTO users(id, display_name, display_color, previous_names, created_at, authenticated_at, scopes, type, registered_at, last_login_at)
		VALUES(?, 'BounceCast Owner', ?, 'BounceCast Owner', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, 'STANDARD', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			display_name = COALESCE(NULLIF(users.display_name, ''), excluded.display_name),
			previous_names = COALESCE(NULLIF(users.previous_names, ''), excluded.previous_names),
			authenticated_at = COALESCE(users.authenticated_at, CURRENT_TIMESTAMP),
			scopes = ?,
			type = 'STANDARD',
			disabled_at = NULL,
			registered_at = COALESCE(users.registered_at, CURRENT_TIMESTAMP),
			last_login_at = CURRENT_TIMESTAMP
	`, bounceCastLegacyAdminUserID, utils.GenerateRandomDisplayColor(config.MaxUserColor), scopes, scopes); err != nil {
		return nil, "", err
	}

	token, err := issueBounceCastAccountAccessToken(bounceCastLegacyAdminUserID)
	if err != nil {
		return nil, "", err
	}
	user := userrepository.Get().GetUserByID(bounceCastLegacyAdminUserID)
	if user == nil {
		return nil, "", errors.New("legacy admin account could not be loaded")
	}
	return user, token, nil
}

func ensureBounceCastAccountForStudioStreamer(streamer bounceCastStudioStreamer, passwordHash string) (*models.User, string, error) {
	email := strings.ToLower(strings.TrimSpace(streamer.Email))
	if email == "" {
		return nil, "", nil
	}

	roleUser, err := getBounceCastAccountRoleUserForLogin(email)
	if err == nil {
		token, tokenErr := issueBounceCastAccountAccessToken(roleUser.ID)
		if tokenErr != nil {
			return nil, "", tokenErr
		}
		return userrepository.Get().GetUserByID(roleUser.ID), token, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, "", err
	}

	userID := shortid.MustGenerate()
	displayName := strings.TrimSpace(streamer.DisplayName)
	if displayName == "" {
		displayName = streamer.Handle
	}
	now := time.Now().UTC()
	if _, err := data.GetDatabase().Exec(`
		INSERT INTO users(id, display_name, display_color, previous_names, created_at, authenticated_at, scopes, email, password_hash, profile_image_url, registered_at, last_login_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?)
	`, userID, displayName, utils.GenerateRandomDisplayColor(config.MaxUserColor), displayName, now, now, models.BounceCastDJScopeKey, email, passwordHash, streamer.AvatarURL, now, now); err != nil {
		return nil, "", err
	}

	token, err := issueBounceCastAccountAccessToken(userID)
	if err != nil {
		return nil, "", err
	}
	return userrepository.Get().GetUserByID(userID), token, nil
}

func normalizeBounceCastUnifiedLogin(request bounceCastUnifiedLoginRequest) string {
	for _, value := range []string{request.Login, request.Username, request.Email} {
		value = strings.TrimSpace(strings.TrimPrefix(value, "@"))
		if value != "" {
			return strings.ToLower(value)
		}
	}
	return ""
}

func bounceCastPermissionsFromScopes(scopes []string) []string {
	scopeMap := map[string]bool{}
	for _, scope := range scopes {
		scopeMap[scope] = true
	}

	permissions := []string{}
	if scopeMap[models.BounceCastOwnerScopeKey] {
		permissions = append(permissions, "owner", "admin")
	} else if scopeMap[models.BounceCastAdminScopeKey] {
		permissions = append(permissions, "admin")
	}
	if scopeMap[models.ModeratorScopeKey] {
		permissions = append(permissions, "moderator")
	}
	if scopeMap[models.BounceCastDJScopeKey] {
		permissions = append(permissions, "dj")
	}
	if len(permissions) == 0 {
		return []string{"user"}
	}
	return uniqueBounceCastPermissions(permissions)
}

func uniqueBounceCastPermissions(values []string) []string {
	seen := map[string]bool{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

func bounceCastPrimaryRoleFromScopes(scopes []string) string {
	scopeMap := map[string]bool{}
	for _, scope := range scopes {
		scopeMap[scope] = true
	}
	if scopeMap[models.BounceCastOwnerScopeKey] {
		return "owner"
	}
	if scopeMap[models.BounceCastAdminScopeKey] {
		return "admin"
	}
	if scopeMap[models.BounceCastDJScopeKey] {
		return "dj"
	}
	if scopeMap[models.ModeratorScopeKey] {
		return "moderator"
	}
	return "user"
}

func resolveBounceCastUnifiedDestination(next string, session bounceCastUnifiedSession) string {
	if isSafeBounceCastRedirect(next) {
		if strings.HasPrefix(next, "/admin") && !session.canAdmin {
			return defaultBounceCastUnifiedDestination(session)
		}
		if strings.HasPrefix(next, "/studio") && !session.canStudio {
			return defaultBounceCastUnifiedDestination(session)
		}
		return next
	}
	return defaultBounceCastUnifiedDestination(session)
}

func defaultBounceCastUnifiedDestination(session bounceCastUnifiedSession) string {
	if session.canAdmin {
		return "/admin/"
	}
	if session.canStudio {
		return "/studio"
	}
	return "/account"
}

func isSafeBounceCastRedirect(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") && !strings.HasPrefix(strings.ToLower(value), "/\\")
}

func setBounceCastUnifiedAuthHeaders(w http.ResponseWriter, r *http.Request) {
	middleware.SetBounceCastCORSHeaders(w, r, "GET, POST, OPTIONS")
}

func writeBounceCastUnifiedUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(webutils.J{"error": "invalid login credentials"})
}
