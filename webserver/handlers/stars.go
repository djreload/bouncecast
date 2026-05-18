package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/owncast/owncast/core/stars"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/starsrepository"
	"github.com/owncast/owncast/webserver/router/middleware"
	webutils "github.com/owncast/owncast/webserver/utils"
	log "github.com/sirupsen/logrus"
)

func GetStarsConfig(w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	config, err := stars.GetService().GetPublicConfig()
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, config)
}

func GetStarsLeaderboard(w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	repository := starsrepository.Get()
	settings, err := repository.GetSettings()
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if !settings.Enabled {
		writeJSON(w, []models.StarLeaderboardEntry{})
		return
	}

	leaderboard, err := repository.GetLeaderboard(10)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, leaderboard)
}

func GetStarsWallet(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	summary, err := starsrepository.Get().GetWalletSummary(user.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, summary)
}

func CreateStarsPayPalOrder(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}

	var request struct {
		PackageID int64 `json:"packageId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.WriteSimpleResponse(w, false, "invalid request")
		return
	}

	result, err := stars.GetService().CreatePayPalOrder(r.Context(), user.ID, request.PackageID)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, result)
}

func CaptureStarsPayPalOrder(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}

	var request struct {
		PayPalOrderID string `json:"paypalOrderId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.PayPalOrderID == "" {
		webutils.WriteSimpleResponse(w, false, "invalid request")
		return
	}

	result, err := stars.GetService().CapturePayPalOrder(r.Context(), user.ID, request.PayPalOrderID)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, result)
}

func SendStars(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}

	var request struct {
		Amount  int    `json:"amount"`
		Message string `json:"message"`
		Effect  string `json:"effect"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.WriteSimpleResponse(w, false, "invalid request")
		return
	}

	event, err := stars.GetService().SendStars(user, request.Amount, request.Message, request.Effect)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, event)
}

func PayPalStarsWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	result, err := stars.GetService().ProcessPayPalWebhook(r.Context(), r.Header, body)
	if err != nil {
		log.Warnln("PayPal Stars webhook rejected:", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(webutils.J{"success": false, "message": err.Error()})
		return
	}
	writeJSON(w, result)
}

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		webutils.InternalErrorHandler(w, err)
	}
}
