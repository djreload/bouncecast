package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/utils"
	webutils "github.com/owncast/owncast/webserver/utils"
)

const bounceCastStudioSessionDuration = 30 * 24 * time.Hour

type bounceCastStudioLoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
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
	if err != nil || streamer.Status != "active" || passwordHash == "" {
		writeBounceCastStudioUnauthorized(w)
		return
	}
	if err := utils.CompareHash(passwordHash, password); err != nil {
		writeBounceCastStudioUnauthorized(w)
		return
	}

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
