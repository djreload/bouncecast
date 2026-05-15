package admin

import (
	"encoding/json"
	"net/http"

	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/starsrepository"
	webutils "github.com/owncast/owncast/webserver/utils"
)

func GetStarsAdmin(w http.ResponseWriter, r *http.Request) {
	summary, err := starsrepository.Get().GetAdminSummary()
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	// Never echo the PayPal secret back to any frontend, even the admin UI.
	summary.Settings.PayPalClientSecret = ""
	writeJSON(w, summary)
}

func SetStarsSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}

	var settings models.StarSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		webutils.WriteSimpleResponse(w, false, "invalid settings")
		return
	}
	if settings.PayPalClientSecret == "" {
		existingSettings, err := starsrepository.Get().GetSettings()
		if err != nil {
			webutils.WriteSimpleResponse(w, false, err.Error())
			return
		}
		settings.PayPalClientSecret = existingSettings.PayPalClientSecret
	}

	if err := starsrepository.Get().SetSettings(settings); err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	webutils.WriteSimpleResponse(w, true, "Stars settings updated")
}

func UpsertStarPackage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}

	var pkg models.StarPackage
	if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
		webutils.WriteSimpleResponse(w, false, "invalid package")
		return
	}
	saved, err := starsrepository.Get().UpsertPackage(pkg)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, saved)
}

func AdjustStarWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}

	var request struct {
		UserID string `json:"userId"`
		Amount int    `json:"amount"`
		Notes  string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.UserID == "" || request.Amount == 0 {
		webutils.WriteSimpleResponse(w, false, "invalid wallet adjustment")
		return
	}

	wallet, err := starsrepository.Get().AdminAdjustWallet(request.UserID, request.Amount, request.Notes)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, wallet)
}

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		webutils.InternalErrorHandler(w, err)
	}
}
