package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/owncast/owncast/webserver/router/middleware"
	webutils "github.com/owncast/owncast/webserver/utils"
)

type bounceCastAdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// BounceCastAdminLogin creates a browser admin session cookie so admins can use
// a normal login form instead of only relying on the HTTP Basic Auth prompt.
func BounceCastAdminLogin(w http.ResponseWriter, r *http.Request) {
	setBounceCastAdminSessionHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var request bounceCastAdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	if !middleware.CheckAdminCredentials(strings.TrimSpace(request.Username), request.Password) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(webutils.J{"error": "invalid admin credentials"})
		return
	}

	middleware.SetAdminSessionCookie(w, r)
	webutils.WriteResponse(w, webutils.J{
		"role":        "admin",
		"destination": "/admin/",
		"message":     "Admin login successful.",
	})
}

func setBounceCastAdminSessionHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
}
