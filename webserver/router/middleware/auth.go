package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/authrepository"
	"github.com/owncast/owncast/persistence/configrepository"
	"github.com/owncast/owncast/persistence/userrepository"
	"github.com/owncast/owncast/utils"
	log "github.com/sirupsen/logrus"
)

const AdminSessionCookieName = "bouncecast_admin_session"
const AdminIdentityCookieName = "bouncecast_admin_identity"

const AdminRoleOwner = "owner"
const AdminRoleAdmin = "admin"

const adminSessionDuration = 30 * 24 * time.Hour

type adminIdentityPayload struct {
	ExpiresAt int64  `json:"exp"`
	UserID    string `json:"userId"`
	Role      string `json:"role"`
}

// AdminIdentity is the BounceCast product-level identity attached to an admin
// request. Basic Owncast admin auth is represented as owner access.
type AdminIdentity struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
}

// ExternalAccessTokenHandlerFunc is a function that is called after validing access.
type ExternalAccessTokenHandlerFunc func(models.ExternalAPIUser, http.ResponseWriter, *http.Request)

// UserAccessTokenHandlerFunc is a function that is called after validing user access.
type UserAccessTokenHandlerFunc func(models.User, http.ResponseWriter, *http.Request)

// RequireAdminAuth wraps a handler requiring HTTP basic auth for it using the given
// the stream key as the password and and a hardcoded "admin" for username.
func RequireAdminAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		realm := "Owncast Authenticated Request"

		// Alow CORS only for localhost:3000 to support Owncast development.
		validAdminHost := "http://localhost:3000"
		w.Header().Set("Access-Control-Allow-Origin", validAdminHost)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// For request needing CORS, send a 204.
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if _, ok := CurrentAdminIdentity(r); !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Debugln("Failed admin authentication")
			return
		}

		handler(w, r)
	}
}

// RequireAdminPageAuth protects the browser admin app without forcing the
// browser's Basic Auth dialog. API routes still use RequireAdminAuth so
// Owncast-compatible clients can continue using Basic auth.
func RequireAdminPageAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := CurrentAdminIdentity(r); !ok {
			next := r.URL.RequestURI()
			if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
				next = "/admin/"
			}
			http.Redirect(w, r, "/login?next="+url.QueryEscape(next), http.StatusFound)
			log.Debugln("Redirecting unauthenticated admin page request to unified login")
			return
		}

		handler(w, r)
	}
}

// RequireAdminRole wraps a handler and requires the authenticated admin session
// to carry one of the supplied BounceCast product roles. The legacy Owncast
// admin password is treated as owner-level access for compatibility.
func RequireAdminRole(handler http.HandlerFunc, roles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		realm := "Owncast Authenticated Request"

		validAdminHost := "http://localhost:3000"
		w.Header().Set("Access-Control-Allow-Origin", validAdminHost)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		identity, ok := CurrentAdminIdentity(r)
		if !ok || !adminRoleAllowed(identity.Role, roles...) {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Debugln("Failed admin role authentication")
			return
		}

		handler(w, r)
	}
}

func RequireOwner(handler http.HandlerFunc) http.HandlerFunc {
	return RequireAdminRole(handler, AdminRoleOwner)
}

func RequireOwnerOrAdmin(handler http.HandlerFunc) http.HandlerFunc {
	return RequireAdminRole(handler, AdminRoleOwner, AdminRoleAdmin)
}

func CheckAdminCredentials(username string, plainPassword string) bool {
	adminPasswordHash := configrepository.Get().GetAdminPassword()
	return subtle.ConstantTimeCompare([]byte(strings.TrimSpace(username)), []byte("admin")) == 1 &&
		utils.CompareHash(adminPasswordHash, plainPassword) == nil
}

// CurrentAdminIdentity returns the product role for the current admin request,
// if the request has valid Owncast basic auth or a signed BounceCast admin
// session/identity cookie.
func CurrentAdminIdentity(r *http.Request) (AdminIdentity, bool) {
	adminPasswordHash := configrepository.Get().GetAdminPassword()
	if isValidAdminBasicAuth(r, "admin", adminPasswordHash) {
		return AdminIdentity{UserID: "owncast-admin", Role: AdminRoleOwner}, true
	}
	if !isValidAdminSessionCookie(r, adminPasswordHash) {
		return AdminIdentity{}, false
	}
	identity, ok := getValidAdminIdentity(r, adminPasswordHash)
	if !ok {
		return AdminIdentity{}, false
	}
	if !storedAdminIdentityStillAllowed(identity) {
		return AdminIdentity{}, false
	}
	return AdminIdentity{UserID: identity.UserID, Role: identity.Role}, true
}

// RequestAdminHasRole checks the current admin request against allowed
// BounceCast product roles.
func RequestAdminHasRole(r *http.Request, roles ...string) bool {
	identity, ok := CurrentAdminIdentity(r)
	if !ok {
		return false
	}
	return adminRoleAllowed(identity.Role, roles...)
}

func SetAdminSessionCookie(w http.ResponseWriter, r *http.Request) {
	setAdminSessionCookie(w, r, "owncast-admin", AdminRoleOwner)
}

func SetAdminRoleSessionCookie(w http.ResponseWriter, r *http.Request, userID string, role string) {
	setAdminSessionCookie(w, r, userID, role)
}

func ClearAdminSessionCookies(w http.ResponseWriter, r *http.Request) {
	for _, name := range []string{AdminSessionCookieName, AdminIdentityCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   r.TLS != nil,
		})
	}
}

func setAdminSessionCookie(w http.ResponseWriter, r *http.Request, userID string, role string) {
	adminPasswordHash := configrepository.Get().GetAdminPassword()
	expiresAt := time.Now().Add(adminSessionDuration)
	expiresUnix := expiresAt.Unix()
	payload := strconv.FormatInt(expiresUnix, 10)
	signature := signAdminSession(payload, adminPasswordHash)

	http.SetCookie(w, &http.Cookie{
		Name:     AdminSessionCookieName,
		Value:    payload + "." + signature,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(adminSessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})

	setAdminIdentityCookie(w, r, adminIdentityPayload{
		ExpiresAt: expiresUnix,
		UserID:    strings.TrimSpace(userID),
		Role:      strings.ToLower(strings.TrimSpace(role)),
	}, adminPasswordHash)
}

func isValidAdminBasicAuth(r *http.Request, username string, passwordHash string) bool {
	user, pass, ok := r.BasicAuth()
	return ok &&
		subtle.ConstantTimeCompare([]byte(user), []byte(username)) == 1 &&
		utils.CompareHash(passwordHash, pass) == nil
}

func isValidAdminSessionCookie(r *http.Request, adminPasswordHash string) bool {
	cookie, err := r.Cookie(AdminSessionCookieName)
	if err != nil {
		return false
	}

	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return false
	}

	expiresUnix, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > expiresUnix {
		return false
	}

	expectedSignature := signAdminSession(parts[0], adminPasswordHash)
	return hmac.Equal([]byte(parts[1]), []byte(expectedSignature))
}

func setAdminIdentityCookie(w http.ResponseWriter, r *http.Request, identity adminIdentityPayload, adminPasswordHash string) {
	body, err := json.Marshal(identity)
	if err != nil {
		return
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	signature := signAdminSession("identity:"+payload, adminPasswordHash)
	http.SetCookie(w, &http.Cookie{
		Name:     AdminIdentityCookieName,
		Value:    payload + "." + signature,
		Path:     "/",
		Expires:  time.Unix(identity.ExpiresAt, 0),
		MaxAge:   int(adminSessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
}

func getValidAdminIdentity(r *http.Request, adminPasswordHash string) (adminIdentityPayload, bool) {
	cookie, err := r.Cookie(AdminIdentityCookieName)
	if err != nil {
		return adminIdentityPayload{}, false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return adminIdentityPayload{}, false
	}
	expectedSignature := signAdminSession("identity:"+parts[0], adminPasswordHash)
	if !hmac.Equal([]byte(parts[1]), []byte(expectedSignature)) {
		return adminIdentityPayload{}, false
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return adminIdentityPayload{}, false
	}
	var identity adminIdentityPayload
	if err := json.Unmarshal(body, &identity); err != nil {
		return adminIdentityPayload{}, false
	}
	if time.Now().Unix() > identity.ExpiresAt {
		return adminIdentityPayload{}, false
	}
	identity.Role = strings.ToLower(strings.TrimSpace(identity.Role))
	return identity, identity.Role != ""
}

func adminRoleAllowed(role string, allowedRoles ...string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	for _, allowedRole := range allowedRoles {
		if role == strings.ToLower(strings.TrimSpace(allowedRole)) {
			return true
		}
	}
	return false
}

func storedAdminIdentityStillAllowed(identity adminIdentityPayload) bool {
	if identity.UserID == "" || identity.UserID == "owncast-admin" {
		return true
	}

	user := userrepository.Get().GetUserByID(identity.UserID)
	if user == nil || !user.IsEnabled() {
		return false
	}

	switch identity.Role {
	case AdminRoleOwner:
		return user.IsOwner()
	case AdminRoleAdmin:
		return user.IsOwner() || user.IsAdmin()
	default:
		return false
	}
}

func signAdminSession(payload string, adminPasswordHash string) string {
	mac := hmac.New(sha256.New, []byte(adminPasswordHash))
	_, _ = mac.Write([]byte(fmt.Sprintf("bouncecast-admin-session:%s", payload)))
	return hex.EncodeToString(mac.Sum(nil))
}

func accessDenied(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized) //nolint
	w.Write([]byte("unauthorized"))        //nolint
}

// RequireExternalAPIAccessToken will validate a 3rd party access token.
func RequireExternalAPIAccessToken(scope string, handler ExternalAccessTokenHandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// We should accept 3rd party preflight OPTIONS requests.
		if r.Method == "OPTIONS" {
			// All OPTIONS requests should have a wildcard CORS header.
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		authHeader := r.Header.Get("Authorization")
		token := ""
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			token = authHeader[len("bearer "):]
		}

		if token == "" {
			log.Warnln("invalid access token")
			accessDenied(w)
			return
		}

		userRepository := userrepository.Get()

		integration, err := userRepository.GetExternalAPIUserForAccessTokenAndScope(token, scope)
		if integration == nil || err != nil {
			accessDenied(w)
			return
		}

		// All auth'ed 3rd party requests should have a wildcard CORS header.
		w.Header().Set("Access-Control-Allow-Origin", "*")

		handler(*integration, w, r)

		if err := userRepository.SetExternalAPIUserAccessTokenAsUsed(token); err != nil {
			log.Debugln("token not found when updating last_used timestamp")
		}
	})
}

// RequireUserAccessToken will validate a provided user's access token and make sure the associated user is enabled.
// Not to be used for validating 3rd party access.
func RequireUserAccessToken(handler UserAccessTokenHandlerFunc) http.HandlerFunc {
	authRepository := authrepository.Get()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.URL.Query().Get("accessToken")
		if accessToken == "" {
			accessDenied(w)
			return
		}

		ipAddress := utils.GetIPAddressFromRequest(r)
		// Check if this client's IP address is banned.
		if blocked, err := authRepository.IsIPAddressBanned(ipAddress); blocked {
			log.Debugln("Client ip address has been blocked. Rejecting.")
			accessDenied(w)
			return
		} else if err != nil {
			log.Errorln("error determining if IP address is blocked: ", err)
		}

		userRepository := userrepository.Get()

		// A user is required to use the websocket
		user := userRepository.GetUserByToken(accessToken)
		if user == nil || !user.IsEnabled() {
			accessDenied(w)
			return
		}

		handler(*user, w, r)
	})
}

// RequireUserModerationScopeAccesstoken will validate a provided user's access token and make sure the associated user is enabled
// and has "MODERATOR" scope assigned to the user.
func RequireUserModerationScopeAccesstoken(handler http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.URL.Query().Get("accessToken")
		if accessToken == "" {
			accessDenied(w)
			return
		}

		userRepository := userrepository.Get()

		// A user is required to use the websocket
		user := userRepository.GetUserByToken(accessToken)
		if user == nil || !user.IsEnabled() || !user.IsModerator() {
			accessDenied(w)
			return
		}

		handler(w, r)
	})
}
