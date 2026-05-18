package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/utils"
	webutils "github.com/owncast/owncast/webserver/utils"
)

const bounceCastStudioSessionDuration = 30 * 24 * time.Hour

var bounceCastStudioHandlePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,31}$`)

type bounceCastStudioLoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type bounceCastStudioRegisterRequest struct {
	DisplayName string `json:"displayName"`
	Handle      string `json:"handle"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type bounceCastStudioStreamer struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"displayName"`
	Handle      string `json:"handle"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	AvatarURL   string `json:"avatarUrl"`
}

type bounceCastStudioSessionResponse struct {
	Token     string                   `json:"token,omitempty"`
	ExpiresAt time.Time                `json:"expiresAt"`
	Streamer  bounceCastStudioStreamer `json:"streamer"`
}

type bounceCastStudioSession struct {
	id        int64
	expiresAt time.Time
	streamer  bounceCastStudioStreamer
}

// BounceCastStudioOptions handles CORS preflight for the additive DJ dashboard APIs.
func BounceCastStudioOptions(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w)
	w.WriteHeader(http.StatusNoContent)
}

// BounceCastStudioLogin verifies a streamer account and issues a scoped Studio bearer session.
func BounceCastStudioLogin(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w)

	var request bounceCastStudioLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	login := normalizeBounceCastStudioLogin(request.Login)
	password := strings.TrimSpace(request.Password)
	if login == "" || password == "" {
		writeBounceCastStudioUnauthorized(w)
		return
	}

	streamer, passwordHash, err := getBounceCastStudioStreamerForLogin(login)
	if err == nil && streamer.Status == "active" && passwordHash != "" {
		if compareErr := utils.CompareHash(passwordHash, password); compareErr == nil {
			allowed, allowErr := canBounceCastStudioStreamerUseStoredRole(streamer)
			if allowErr != nil {
				webutils.InternalErrorHandler(w, allowErr)
				return
			}
			if !allowed {
				writeBounceCastStudioUnauthorized(w)
				return
			}
			writeBounceCastStudioSession(w, r, streamer)
			return
		}
	}

	accountStreamer, err := getBounceCastStudioStreamerFromAccountRole(login, password)
	if err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}
	writeBounceCastStudioSession(w, r, accountStreamer)
}

// BounceCastStudioRegister creates an inactive DJ dashboard account for admin approval.
func BounceCastStudioRegister(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w)

	var request bounceCastStudioRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	displayName := strings.TrimSpace(request.DisplayName)
	handle := normalizeBounceCastStudioHandle(request.Handle)
	if displayName == "" || handle == "" {
		webutils.BadRequestHandler(w, errors.New("displayName and handle are required"))
		return
	}
	if !bounceCastStudioHandlePattern.MatchString(handle) {
		webutils.BadRequestHandler(w, errors.New("handle must be 1-32 letters, numbers, underscores, or hyphens"))
		return
	}

	email := strings.TrimSpace(request.Email)
	if email == "" {
		webutils.BadRequestHandler(w, errors.New("email is required"))
		return
	}
	parsedEmail, err := mail.ParseAddress(email)
	if err != nil {
		webutils.BadRequestHandler(w, errors.New("email must be a valid email address"))
		return
	}

	if err := validateBounceCastStudioPassword(request.Password); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	hashedPassword, err := utils.HashPassword(strings.TrimSpace(request.Password))
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	result, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_streamer_accounts(display_name, handle, email, password_hash, role, status)
		VALUES(?, ?, ?, ?, 'streamer', 'inactive')
	`, displayName, handle, parsedEmail.Address, hashedPassword)
	if err != nil {
		if isBounceCastStudioDuplicateAccountError(err) {
			webutils.BadRequestHandler(w, errors.New("handle or email is already registered"))
			return
		}
		webutils.InternalErrorHandler(w, err)
		return
	}

	id, _ := result.LastInsertId()
	webutils.WriteResponse(w, map[string]interface{}{
		"id":      id,
		"status":  "inactive",
		"message": "Registration received. An admin must activate this DJ account before Studio login is available.",
	})
}

// BounceCastStudioMe returns the current authenticated DJ dashboard session.
func BounceCastStudioMe(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w)

	session, err := authenticateBounceCastStudioRequest(r)
	if err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}

	webutils.WriteResponse(w, bounceCastStudioSessionResponse{
		ExpiresAt: session.expiresAt,
		Streamer:  session.streamer,
	})
}

// BounceCastStudioLogout revokes the current DJ dashboard bearer session.
func BounceCastStudioLogout(w http.ResponseWriter, r *http.Request) {
	setBounceCastStudioAPIHeaders(w)

	session, err := authenticateBounceCastStudioRequest(r)
	if err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}

	if _, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_streamer_sessions
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, session.id); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	webutils.WriteSimpleResponse(w, true, "logged out")
}

func getBounceCastStudioStreamerForLogin(login string) (bounceCastStudioStreamer, string, error) {
	var streamer bounceCastStudioStreamer
	var passwordHash sql.NullString
	if err := data.GetDatabase().QueryRow(`
		SELECT id, display_name, handle, COALESCE(email, ''), role, status, COALESCE(avatar_url, ''), password_hash
		FROM bouncecast_streamer_accounts
		WHERE LOWER(handle) = ? OR LOWER(COALESCE(email, '')) = ?
		LIMIT 1
	`, login, login).Scan(
		&streamer.ID,
		&streamer.DisplayName,
		&streamer.Handle,
		&streamer.Email,
		&streamer.Role,
		&streamer.Status,
		&streamer.AvatarURL,
		&passwordHash,
	); err != nil {
		return bounceCastStudioStreamer{}, "", err
	}

	if !passwordHash.Valid {
		return streamer, "", nil
	}
	return streamer, passwordHash.String, nil
}

func getBounceCastStudioStreamerFromAccountRole(login string, password string) (bounceCastStudioStreamer, error) {
	user, err := getBounceCastAccountRoleUserForLogin(login)
	if err != nil || user.PasswordHash == "" || !user.canUseStudio() {
		return bounceCastStudioStreamer{}, errors.New("public account is not an active DJ")
	}
	if err := utils.CompareHash(user.PasswordHash, password); err != nil {
		return bounceCastStudioStreamer{}, err
	}
	return provisionBounceCastStudioStreamerFromRoleUser(user)
}

func canBounceCastStudioStreamerUseStoredRole(streamer bounceCastStudioStreamer) (bool, error) {
	if strings.TrimSpace(streamer.Email) == "" {
		return true, nil
	}

	user, err := getBounceCastAccountRoleUserByEmail(streamer.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return user.canUseStudio(), nil
}

func provisionBounceCastStudioStreamerFromRoleUser(user bounceCastAccountRoleUser) (bounceCastStudioStreamer, error) {
	streamer, _, err := getBounceCastStudioStreamerForLogin(strings.ToLower(user.Email))
	if err == nil {
		if streamer.Status != "active" || streamer.DisplayName != user.DisplayName || streamer.AvatarURL != user.ProfileImageURL {
			if _, updateErr := data.GetDatabase().Exec(`
				UPDATE bouncecast_streamer_accounts
				SET display_name = ?, password_hash = ?, role = 'streamer', status = 'active',
					avatar_url = NULLIF(?, ''), updated_at = CURRENT_TIMESTAMP
				WHERE id = ?
			`, user.DisplayName, user.PasswordHash, user.ProfileImageURL, streamer.ID); updateErr != nil {
				return bounceCastStudioStreamer{}, updateErr
			}
			streamer, _, err = getBounceCastStudioStreamerForLogin(strings.ToLower(user.Email))
		}
		return streamer, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return bounceCastStudioStreamer{}, err
	}

	handle, err := nextBounceCastStudioAccountHandle(user.DisplayName, user.ID)
	if err != nil {
		return bounceCastStudioStreamer{}, err
	}
	result, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_streamer_accounts(display_name, handle, email, password_hash, role, status, avatar_url)
		VALUES(?, ?, ?, ?, 'streamer', 'active', NULLIF(?, ''))
	`, user.DisplayName, handle, user.Email, user.PasswordHash, user.ProfileImageURL)
	if err != nil {
		return bounceCastStudioStreamer{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return bounceCastStudioStreamer{}, err
	}
	return bounceCastStudioStreamer{
		ID:          id,
		DisplayName: user.DisplayName,
		Handle:      handle,
		Email:       user.Email,
		Role:        "streamer",
		Status:      "active",
		AvatarURL:   user.ProfileImageURL,
	}, nil
}

func nextBounceCastStudioAccountHandle(displayName string, userID string) (string, error) {
	base := makeBounceCastStudioHandleCandidate(displayName)
	if base == "" {
		base = makeBounceCastStudioHandleCandidate(userID)
	}
	if base == "" {
		base = "dj"
	}

	candidates := []string{base}
	suffixSource := makeBounceCastStudioHandleCandidate(userID)
	if suffixSource != "" {
		if len(suffixSource) > 8 {
			suffixSource = suffixSource[:8]
		}
		candidates = append(candidates, trimBounceCastStudioHandle(base, 23)+"-"+suffixSource)
	}
	for index := 2; index <= 50; index++ {
		candidates = append(candidates, fmt.Sprintf("%s-%d", trimBounceCastStudioHandle(base, 28), index))
	}

	for _, candidate := range candidates {
		var existingID int64
		err := data.GetDatabase().QueryRow(`SELECT id FROM bouncecast_streamer_accounts WHERE LOWER(handle) = LOWER(?) LIMIT 1`, candidate).Scan(&existingID)
		if errors.Is(err, sql.ErrNoRows) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", errors.New("could not create a unique Studio handle")
}

func makeBounceCastStudioHandleCandidate(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	builder := strings.Builder{}
	lastDash := false
	for _, char := range value {
		isAllowed := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
		if isAllowed {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if (char == '-' || char == '_' || char == ' ') && builder.Len() > 0 && !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return trimBounceCastStudioHandle(strings.Trim(builder.String(), "-_"), 32)
}

func trimBounceCastStudioHandle(handle string, maxLength int) string {
	handle = strings.Trim(handle, "-_")
	if len(handle) > maxLength {
		handle = strings.Trim(handle[:maxLength], "-_")
	}
	if handle == "" {
		return ""
	}
	if !bounceCastStudioHandlePattern.MatchString(handle) {
		return ""
	}
	return handle
}

func writeBounceCastStudioSession(w http.ResponseWriter, r *http.Request, streamer bounceCastStudioStreamer) {
	token, err := utils.GenerateAccessToken()
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	expiresAt := time.Now().UTC().Add(bounceCastStudioSessionDuration)
	if _, err := data.GetDatabase().Exec(`
		INSERT INTO bouncecast_streamer_sessions(streamer_id, token_hash, expires_at, user_agent, remote_addr)
		VALUES(?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))
	`, streamer.ID, hashBounceCastStudioToken(token), expiresAt, r.UserAgent(), utils.GetIPAddressFromRequest(r)); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if _, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_streamer_accounts
		SET last_login_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, streamer.ID); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	webutils.WriteResponse(w, bounceCastStudioSessionResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		Streamer:  streamer,
	})
}

func authenticateBounceCastStudioRequest(r *http.Request) (bounceCastStudioSession, error) {
	token := getBounceCastStudioBearerToken(r)
	if token == "" {
		return bounceCastStudioSession{}, errors.New("missing bearer token")
	}

	var session bounceCastStudioSession
	if err := data.GetDatabase().QueryRow(`
		SELECT s.id, s.expires_at, a.id, a.display_name, a.handle, COALESCE(a.email, ''), a.role, a.status, COALESCE(a.avatar_url, '')
		FROM bouncecast_streamer_sessions s
		INNER JOIN bouncecast_streamer_accounts a ON a.id = s.streamer_id
		WHERE s.token_hash = ?
			AND s.revoked_at IS NULL
			AND s.expires_at > CURRENT_TIMESTAMP
			AND a.status = 'active'
		LIMIT 1
	`, hashBounceCastStudioToken(token)).Scan(
		&session.id,
		&session.expiresAt,
		&session.streamer.ID,
		&session.streamer.DisplayName,
		&session.streamer.Handle,
		&session.streamer.Email,
		&session.streamer.Role,
		&session.streamer.Status,
		&session.streamer.AvatarURL,
	); err != nil {
		return bounceCastStudioSession{}, err
	}

	if _, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_streamer_sessions
		SET last_used_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, session.id); err != nil {
		return bounceCastStudioSession{}, err
	}

	return session, nil
}

func getBounceCastStudioBearerToken(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return ""
	}
	return strings.TrimSpace(authHeader[len("bearer "):])
}

func hashBounceCastStudioToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeBounceCastStudioLogin(login string) string {
	login = strings.TrimSpace(strings.TrimPrefix(login, "@"))
	return strings.ToLower(login)
}

func normalizeBounceCastStudioHandle(handle string) string {
	return strings.TrimSpace(strings.TrimPrefix(handle, "@"))
}

func validateBounceCastStudioPassword(password string) error {
	if strings.ContainsAny(password, "\r\n") {
		return errors.New("password cannot contain line breaks")
	}
	if len(strings.TrimSpace(password)) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func isBounceCastStudioDuplicateAccountError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(fmt.Sprint(err)), "unique constraint failed")
}

func setBounceCastStudioAPIHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

func writeBounceCastStudioUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(webutils.J{"error": "invalid BounceCast Studio credentials"})
}
