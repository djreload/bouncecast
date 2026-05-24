package admin

import (
	"encoding/json"
	"net/http"

	"github.com/owncast/owncast/core/data"
	mobilecore "github.com/owncast/owncast/core/mobile"
	"github.com/owncast/owncast/models"
	webutils "github.com/owncast/owncast/webserver/utils"
)

func GetMobileAdmin(w http.ResponseWriter, r *http.Request) {
	settings, err := mobilecore.GetAdminSettings(data.GetDatabase())
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, settings)
}

func SetMobileAdminSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	var request models.MobileAdminSettings
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	settings, err := mobilecore.SaveAdminSettings(data.GetDatabase(), request)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	recordBounceCastAuditEvent(r, "mobile_settings_updated", "mobile", "settings", map[string]interface{}{
		"app":                     settings.App.Name,
		"adsEnabled":              settings.Ads.Enabled,
		"pushNotifications":       settings.Features.PushNotifications,
		"maintenanceMode":         settings.App.MaintenanceMode,
		"minimumSupportedVersion": settings.App.MinimumSupportedVersion,
	})
	writeJSON(w, settings)
}
