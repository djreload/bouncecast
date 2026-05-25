package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/owncast/owncast/config"
	"github.com/owncast/owncast/core"
	"github.com/owncast/owncast/core/data"
	mobilecore "github.com/owncast/owncast/core/mobile"
	"github.com/owncast/owncast/core/stars"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/configrepository"
	"github.com/owncast/owncast/persistence/rewardsrepository"
	"github.com/owncast/owncast/persistence/userrepository"
	"github.com/owncast/owncast/webserver/router/middleware"
	webutils "github.com/owncast/owncast/webserver/utils"
)

func MobileOptions(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "GET, POST, PUT, OPTIONS")
	w.WriteHeader(http.StatusNoContent)
}

func GetMobileConfig(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "GET, OPTIONS")
	config, err := loadMobilePublicConfig(r)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, config)
}

func GetMobileTheme(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "GET, OPTIONS")
	config, err := loadMobilePublicConfig(r)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, config.Branding)
}

func GetMobileNavigation(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "GET, OPTIONS")
	config, err := loadMobilePublicConfig(r)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, config.Navigation)
}

func GetMobileFeatures(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "GET, OPTIONS")
	config, err := loadMobilePublicConfig(r)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, config.Features)
}

func GetMobileAssets(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "GET, OPTIONS")
	config, err := loadMobilePublicConfig(r)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, webutils.J{
		"logo_url":           config.Branding.LogoURL,
		"splash_url":         config.Branding.SplashURL,
		"app_icon_url":       config.Branding.AppIconURL,
		"app_background_url": config.Branding.AppBackgroundURL,
	})
}

func GetMobileLegal(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "GET, OPTIONS")
	config, err := loadMobilePublicConfig(r)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, config.Legal)
}

func GetMobileStreamStatus(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "GET, OPTIONS")
	config, err := loadMobilePublicConfig(r)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, config.Stream)
}

func GetMobileChatConfig(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "GET, OPTIONS")
	config, err := loadMobilePublicConfig(r)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, config.Chat)
}

func GetMobileAds(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "GET, OPTIONS")
	config, err := loadMobilePublicConfig(r)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, config.Ads)
}

func GetMobileNotificationsConfig(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "GET, OPTIONS")
	config, err := loadMobilePublicConfig(r)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, config.Notifications)
}

func RegisterMobileDevice(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !enforceBounceCastRateLimit(w, r, bounceCastMobileDeviceRateLimit, bounceCastRateLimitIPSubject(r)) {
		return
	}

	var request models.MobileDeviceRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if err := mobilecore.RegisterDevice(data.GetDatabase(), request, mobileUserIDFromRequest(r)); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	writeJSON(w, webutils.J{"success": true})
}

func UnregisterMobileDevice(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !enforceBounceCastRateLimit(w, r, bounceCastMobileDeviceRateLimit, bounceCastRateLimitIPSubject(r)) {
		return
	}

	var request models.MobileDevicePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if err := mobilecore.UnregisterDevice(data.GetDatabase(), request.Token); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	writeJSON(w, webutils.J{"success": true})
}

func UpdateMobileDevicePreferences(w http.ResponseWriter, r *http.Request) {
	setMobileHeaders(w, r, "PUT, POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !enforceBounceCastRateLimit(w, r, bounceCastMobileDeviceRateLimit, bounceCastRateLimitIPSubject(r)) {
		return
	}

	var request models.MobileDevicePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if err := mobilecore.UpdateDevicePreferences(data.GetDatabase(), request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	writeJSON(w, webutils.J{"success": true})
}

func loadMobilePublicConfig(r *http.Request) (models.MobileConfigResponse, error) {
	return mobilecore.GetPublicConfig(data.GetDatabase(), mobileRuntimeFromRequest(r))
}

func mobileRuntimeFromRequest(r *http.Request) mobilecore.Runtime {
	configRepository := configrepository.Get()
	status := core.GetStatus()
	featureRuntime := mobileFeatureRuntime()

	runtime := mobilecore.Runtime{
		BaseURL:                   inferMobileBaseURL(r, configRepository.GetServerURL()),
		ServerName:                configRepository.GetServerName(),
		ServerSummary:             configRepository.GetServerSummary(),
		StreamTitle:               configRepository.GetStreamTitle(),
		OfflineMessage:            configRepository.GetCustomOfflineMessage(),
		LogoURL:                   "/logo",
		FaviconURL:                "/favicon.ico",
		SocketHostOverride:        configRepository.GetWebsocketOverrideHost(),
		ChatDisabled:              configRepository.GetChatDisabled(),
		ChatRequireAuthentication: configRepository.GetChatRequireAuthentication(),
		ChatBackgroundImageURL:    configRepository.GetChatBackgroundImageURL(),
		ChatBackgroundOpacity:     configRepository.GetChatBackgroundOpacity(),
		ChatTenorAPIKey:           configRepository.GetChatTenorAPIKey(),
		AppearanceVariables:       configRepository.GetCustomColorVariableValues(),
		MaxSocketPayloadSize:      config.MaxSocketPayloadSize,
		HideViewerCount:           configRepository.GetHideViewerCount(),
		Status:                    status,
	}
	runtime.StarsEnabled = featureRuntime.StarsEnabled
	runtime.StarsOverlayEnabled = featureRuntime.StarsOverlayEnabled
	runtime.ChatReactionsEnabled = featureRuntime.ChatReactionsEnabled
	runtime.RewardWheelEnabled = featureRuntime.RewardWheelEnabled
	runtime.RewardOverlayEnabled = featureRuntime.RewardOverlayEnabled
	return runtime
}

func mobileFeatureRuntime() mobilecore.Runtime {
	runtime := mobilecore.Runtime{ChatReactionsEnabled: true}

	if starConfig, err := stars.GetService().GetPublicConfig(); err == nil {
		runtime.StarsEnabled = starConfig.Enabled
		runtime.StarsOverlayEnabled = starConfig.OverlayEffectsEnabled
	}

	if rewardSettings, err := rewardsrepository.Get().GetSettings(); err == nil {
		runtime.RewardWheelEnabled = rewardSettings.Enabled
		runtime.RewardOverlayEnabled = rewardSettings.OverlayEnabled
	}

	return runtime
}

func inferMobileBaseURL(r *http.Request, configuredURL string) string {
	configuredURL = strings.TrimRight(strings.TrimSpace(configuredURL), "/")
	scheme := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if host == "" {
		host = "localhost:8080"
	}
	if configuredURL != "" && shouldUseConfiguredMobileURL(configuredURL, host) {
		return configuredURL
	}
	return strings.TrimRight(scheme+"://"+host, "/")
}

func shouldUseConfiguredMobileURL(configuredURL string, requestHost string) bool {
	parsedConfigured, err := url.Parse(configuredURL)
	if err != nil || parsedConfigured.Hostname() == "" {
		return false
	}
	parsedRequest, err := url.Parse("//" + requestHost)
	if err != nil || parsedRequest.Hostname() == "" {
		return true
	}
	return !isMobileLocalHost(parsedConfigured.Hostname()) || isMobileLocalHost(parsedRequest.Hostname())
}

func isMobileLocalHost(host string) bool {
	host = strings.ToLower(strings.Trim(host, "[] "))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func mobileUserIDFromRequest(r *http.Request) string {
	token := strings.TrimSpace(r.URL.Query().Get("accessToken"))
	if token == "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			token = strings.TrimSpace(auth[7:])
		}
	}
	if token == "" {
		return ""
	}
	user := userrepository.Get().GetUserByToken(token)
	if user == nil {
		return ""
	}
	return user.ID
}

func setMobileHeaders(w http.ResponseWriter, r *http.Request, methods string) {
	middleware.SetBounceCastCORSHeaders(w, r, methods)
	middleware.DisableCache(w)
}
