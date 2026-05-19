package router

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/CAFxX/httpcompression"
	"github.com/go-chi/chi/v5"
	chiMW "github.com/go-chi/chi/v5/middleware"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/owncast/owncast/activitypub"
	aphandlers "github.com/owncast/owncast/activitypub/controllers"
	"github.com/owncast/owncast/config"
	"github.com/owncast/owncast/core/chat"
	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/webserver/handlers"
	adminhandlers "github.com/owncast/owncast/webserver/handlers/admin"
	"github.com/owncast/owncast/webserver/router/middleware"
)

// Start starts the router for the http, ws, and rtmp.
func Start(enableVerboseLogging bool) error {
	// @behlers New Router
	r := chi.NewRouter()

	// Middlewares
	if enableVerboseLogging {
		r.Use(chiMW.RequestLogger(&chiMW.DefaultLogFormatter{Logger: log.StandardLogger(), NoColor: true}))
	}
	r.Use(chiMW.Recoverer)

	addStaticFileEndpoints(r)

	// websocket
	r.HandleFunc("/ws", chat.HandleClientConnection)

	// serve files
	fs := http.FileServer(http.Dir(config.PublicFilesPath))
	r.Handle("/public/*", http.StripPrefix("/public/", fs))

	// Return HLS video
	r.HandleFunc("/hls/*", handlers.HandleHLSRequest)

	// The admin web app.
	r.HandleFunc("/admin/*", middleware.RequireAdminAuth(handlers.IndexHandler))

	// BounceCast Studio admin APIs. These are kept separate from the generated
	// Owncast-compatible API while the multi-streamer surface is still growing.
	r.Options("/api/admin/bouncecast/streamers", middleware.RequireAdminAuth(adminhandlers.GetBounceCastStreamers))
	r.Get("/api/admin/bouncecast/streamers", middleware.RequireAdminAuth(adminhandlers.GetBounceCastStreamers))
	r.Post("/api/admin/bouncecast/streamers", middleware.RequireAdminRole(adminhandlers.CreateBounceCastStreamer, "owner", "admin"))
	r.Post("/api/admin/bouncecast/streamers/update", middleware.RequireAdminRole(adminhandlers.UpdateBounceCastStreamer, "owner", "admin"))
	r.Post("/api/admin/bouncecast/streamers/password", middleware.RequireAdminRole(adminhandlers.SetBounceCastStreamerPassword, "owner", "admin"))
	r.Options("/api/admin/bouncecast/streamkeys", middleware.RequireAdminAuth(adminhandlers.GetBounceCastStreamKeys))
	r.Get("/api/admin/bouncecast/streamkeys", middleware.RequireAdminAuth(adminhandlers.GetBounceCastStreamKeys))
	r.Post("/api/admin/bouncecast/streamkeys", middleware.RequireAdminRole(adminhandlers.CreateBounceCastStreamKey, "owner", "admin"))
	r.Post("/api/admin/bouncecast/streamkeys/revoke", middleware.RequireAdminRole(adminhandlers.RevokeBounceCastStreamKey, "owner", "admin"))
	r.Options("/api/admin/bouncecast/live-events", middleware.RequireAdminAuth(adminhandlers.GetBounceCastGoLiveEvents))
	r.Get("/api/admin/bouncecast/live-events", middleware.RequireAdminAuth(adminhandlers.GetBounceCastGoLiveEvents))
	r.Options("/api/admin/bouncecast/notification-subscribers", middleware.RequireAdminAuth(adminhandlers.GetBounceCastNotificationSubscribers))
	r.Get("/api/admin/bouncecast/notification-subscribers", middleware.RequireAdminAuth(adminhandlers.GetBounceCastNotificationSubscribers))
	r.Post("/api/admin/bouncecast/notification-subscribers", middleware.RequireAdminAuth(adminhandlers.CreateBounceCastNotificationSubscriber))
	r.Post("/api/admin/bouncecast/notification-subscribers/disable", middleware.RequireAdminAuth(adminhandlers.DisableBounceCastNotificationSubscriber))
	r.Options("/api/admin/bouncecast/notification-deliveries", middleware.RequireAdminAuth(adminhandlers.GetBounceCastNotificationDeliveries))
	r.Get("/api/admin/bouncecast/notification-deliveries", middleware.RequireAdminAuth(adminhandlers.GetBounceCastNotificationDeliveries))
	r.Options("/api/admin/bouncecast/email-settings", middleware.RequireAdminAuth(adminhandlers.GetBounceCastEmailSettings))
	r.Get("/api/admin/bouncecast/email-settings", middleware.RequireAdminAuth(adminhandlers.GetBounceCastEmailSettings))
	r.Post("/api/admin/bouncecast/email-settings", middleware.RequireAdminRole(adminhandlers.SetBounceCastEmailSettings, "owner"))
	r.Options("/api/admin/bouncecast/messenger-settings", middleware.RequireAdminAuth(adminhandlers.GetBounceCastMessengerSettings))
	r.Get("/api/admin/bouncecast/messenger-settings", middleware.RequireAdminAuth(adminhandlers.GetBounceCastMessengerSettings))
	r.Post("/api/admin/bouncecast/messenger-settings", middleware.RequireAdminRole(adminhandlers.SetBounceCastMessengerSettings, "owner"))
	r.Options("/api/admin/bouncecast/push-settings", middleware.RequireAdminAuth(adminhandlers.GetBounceCastPushSettings))
	r.Get("/api/admin/bouncecast/push-settings", middleware.RequireAdminAuth(adminhandlers.GetBounceCastPushSettings))
	r.Options("/api/admin/bouncecast/session", middleware.RequireAdminAuth(adminhandlers.GetBounceCastAdminSession))
	r.Get("/api/admin/bouncecast/session", middleware.RequireAdminAuth(adminhandlers.GetBounceCastAdminSession))
	r.Options("/api/admin/bouncecast/schedule", middleware.RequireAdminAuth(adminhandlers.GetBounceCastSchedule))
	r.Get("/api/admin/bouncecast/schedule", middleware.RequireAdminAuth(adminhandlers.GetBounceCastSchedule))
	r.Post("/api/admin/bouncecast/schedule", middleware.RequireAdminRole(adminhandlers.CreateBounceCastSchedule, "owner", "admin"))
	r.Options("/api/admin/bouncecast/schedule/reminders", middleware.RequireAdminAuth(adminhandlers.GetBounceCastScheduleReminders))
	r.Get("/api/admin/bouncecast/schedule/reminders", middleware.RequireAdminAuth(adminhandlers.GetBounceCastScheduleReminders))
	r.Post("/api/admin/bouncecast/schedule/reminders/disable", middleware.RequireAdminRole(adminhandlers.DisableBounceCastScheduleReminder, "owner", "admin"))
	r.Options("/api/admin/bouncecast/audit-events", middleware.RequireAdminRole(adminhandlers.GetBounceCastAuditEvents, "owner"))
	r.Get("/api/admin/bouncecast/audit-events", middleware.RequireAdminRole(adminhandlers.GetBounceCastAuditEvents, "owner"))

	r.Options("/api/admin/bouncecast/stars", middleware.RequireAdminAuth(adminhandlers.GetStarsAdmin))
	r.Get("/api/admin/bouncecast/stars", middleware.RequireAdminAuth(adminhandlers.GetStarsAdmin))
	r.Post("/api/admin/bouncecast/stars/settings", middleware.RequireAdminRole(adminhandlers.SetStarsSettings, "owner"))
	r.Post("/api/admin/bouncecast/stars/packages", middleware.RequireAdminRole(adminhandlers.UpsertStarPackage, "owner"))
	r.Post("/api/admin/bouncecast/stars/wallets/adjust", middleware.RequireAdminRole(adminhandlers.AdjustStarWallet, "owner"))
	r.Post("/api/admin/bouncecast/stars/test-overlay", middleware.RequireAdminAuth(adminhandlers.TestStarsOverlay))
	r.Options("/api/admin/bouncecast/users", middleware.RequireAdminAuth(adminhandlers.GetBounceCastUsers))
	r.Get("/api/admin/bouncecast/users", middleware.RequireAdminAuth(adminhandlers.GetBounceCastUsers))
	r.Post("/api/admin/bouncecast/users/permissions", middleware.RequireAdminRole(adminhandlers.SetBounceCastUserPermissions, "owner"))
	r.Options("/api/admin/bouncecast/command-center", middleware.RequireAdminAuth(adminhandlers.GetBounceCastCommandCenter))
	r.Get("/api/admin/bouncecast/command-center", middleware.RequireAdminAuth(adminhandlers.GetBounceCastCommandCenter))

	r.Options("/api/admin/config/chat/backgroundimage", middleware.RequireAdminAuth(adminhandlers.SetChatBackgroundImageURL))
	r.Post("/api/admin/config/chat/backgroundimage", middleware.RequireAdminAuth(adminhandlers.SetChatBackgroundImageURL))
	r.Options("/api/admin/config/chat/backgroundopacity", middleware.RequireAdminAuth(adminhandlers.SetChatBackgroundOpacity))
	r.Post("/api/admin/config/chat/backgroundopacity", middleware.RequireAdminAuth(adminhandlers.SetChatBackgroundOpacity))
	r.Options("/api/admin/config/chat/tenorapikey", middleware.RequireAdminAuth(adminhandlers.SetChatTenorAPIKey))
	r.Post("/api/admin/config/chat/tenorapikey", middleware.RequireAdminAuth(adminhandlers.SetChatTenorAPIKey))

	// BounceCast Studio DJ dashboard auth. This is additive and does not replace
	// the existing Owncast admin authentication path.
	r.Options("/api/bouncecast/studio/login", handlers.BounceCastStudioOptions)
	r.Post("/api/bouncecast/studio/login", handlers.BounceCastStudioLogin)
	r.Options("/api/bouncecast/studio/register", handlers.BounceCastStudioOptions)
	r.Post("/api/bouncecast/studio/register", handlers.BounceCastStudioRegister)
	r.Options("/api/bouncecast/studio/me", handlers.BounceCastStudioOptions)
	r.Get("/api/bouncecast/studio/me", handlers.BounceCastStudioMe)
	r.Options("/api/bouncecast/studio/logout", handlers.BounceCastStudioOptions)
	r.Post("/api/bouncecast/studio/logout", handlers.BounceCastStudioLogout)
	r.Options("/api/bouncecast/studio/schedule", handlers.BounceCastStudioOptions)
	r.Get("/api/bouncecast/studio/schedule", handlers.BounceCastStudioSchedule)
	r.Post("/api/bouncecast/studio/schedule", handlers.BounceCastStudioCreateSchedule)
	r.Options("/api/bouncecast/studio/schedule/update", handlers.BounceCastStudioOptions)
	r.Post("/api/bouncecast/studio/schedule/update", handlers.BounceCastStudioUpdateSchedule)
	r.Options("/api/bouncecast/studio/schedule/cancel", handlers.BounceCastStudioOptions)
	r.Post("/api/bouncecast/studio/schedule/cancel", handlers.BounceCastStudioCancelSchedule)
	r.Options("/api/bouncecast/studio/streamkeys", handlers.BounceCastStudioOptions)
	r.Get("/api/bouncecast/studio/streamkeys", handlers.BounceCastStudioStreamKeys)
	r.Post("/api/bouncecast/studio/streamkeys", handlers.BounceCastStudioCreateStreamKey)
	r.Options("/api/bouncecast/studio/streamkeys/revoke", handlers.BounceCastStudioOptions)
	r.Post("/api/bouncecast/studio/streamkeys/revoke", handlers.BounceCastStudioRevokeStreamKey)
	r.Options("/api/bouncecast/studio/live-events", handlers.BounceCastStudioOptions)
	r.Get("/api/bouncecast/studio/live-events", handlers.BounceCastStudioLiveEvents)
	r.Options("/api/bouncecast/studio/profile", handlers.BounceCastStudioOptions)
	r.Post("/api/bouncecast/studio/profile", handlers.BounceCastStudioUpdateProfile)

	r.Options("/api/bouncecast/admin/login", handlers.BounceCastAdminLogin)
	r.Post("/api/bouncecast/admin/login", handlers.BounceCastAdminLogin)

	r.Options("/api/bouncecast/account/register", handlers.BounceCastAccountOptions)
	r.Post("/api/bouncecast/account/register", handlers.BounceCastAccountRegister)
	r.Options("/api/bouncecast/account/login", handlers.BounceCastAccountOptions)
	r.Post("/api/bouncecast/account/login", handlers.BounceCastAccountLogin)
	r.Options("/api/bouncecast/account/me", handlers.BounceCastAccountOptions)
	r.Get("/api/bouncecast/account/me", middleware.RequireUserAccessToken(handlers.BounceCastAccountMe))
	r.Options("/api/bouncecast/account/profile", handlers.BounceCastAccountOptions)
	r.Post("/api/bouncecast/account/profile", middleware.RequireUserAccessToken(handlers.BounceCastAccountUpdateProfile))
	r.Options("/api/bouncecast/account/profile-image", handlers.BounceCastAccountOptions)
	r.Post("/api/bouncecast/account/profile-image", middleware.RequireUserAccessToken(handlers.BounceCastAccountUploadProfileImage))
	r.Options("/api/bouncecast/account/notifications", handlers.BounceCastAccountOptions)
	r.Post("/api/bouncecast/account/notifications", middleware.RequireUserAccessToken(handlers.BounceCastAccountUpdateNotifications))
	r.Options("/api/bouncecast/account/hub", handlers.BounceCastAccountOptions)
	r.Get("/api/bouncecast/account/hub", middleware.RequireUserAccessToken(handlers.BounceCastAccountHub))

	r.Options("/api/bouncecast/djs", handlers.BounceCastPublicOptions)
	r.Get("/api/bouncecast/djs", handlers.GetBounceCastPublicDJs)
	r.Options("/api/bouncecast/djs/{handle}", handlers.BounceCastPublicOptions)
	r.Get("/api/bouncecast/djs/{handle}", handlers.GetBounceCastPublicDJProfile)
	r.Options("/api/bouncecast/schedule", handlers.BounceCastPublicOptions)
	r.Get("/api/bouncecast/schedule", handlers.GetBounceCastPublicSchedule)
	r.Options("/api/bouncecast/schedule/reminders", handlers.BounceCastAccountOptions)
	r.Post("/api/bouncecast/schedule/reminders", middleware.RequireUserAccessToken(handlers.SetBounceCastScheduleReminder))

	r.Options("/api/stars/config", handlers.GetStarsConfig)
	r.Get("/api/stars/config", handlers.GetStarsConfig)
	r.Options("/api/stars/leaderboard", handlers.GetStarsLeaderboard)
	r.Get("/api/stars/leaderboard", handlers.GetStarsLeaderboard)
	r.Get("/api/stars/wallet", middleware.RequireUserAccessToken(handlers.GetStarsWallet))
	r.Post("/api/stars/paypal/order", middleware.RequireUserAccessToken(handlers.CreateStarsPayPalOrder))
	r.Post("/api/stars/paypal/capture", middleware.RequireUserAccessToken(handlers.CaptureStarsPayPalOrder))
	r.Post("/api/stars/send", middleware.RequireUserAccessToken(handlers.SendStars))
	r.Post("/api/stars/paypal/webhook", handlers.PayPalStarsWebhook)

	r.Get("/djs/{handle}", handlers.BounceCastDJProfilePageHandler)

	// Single ActivityPub Actor
	r.HandleFunc("/federation/user/*", middleware.RequireActivityPubOrRedirect(aphandlers.ActorHandler))

	// Single AP object
	r.HandleFunc("/federation/*", middleware.RequireActivityPubOrRedirect(aphandlers.ObjectHandler))

	// The primary web app.
	r.HandleFunc("/*", handlers.IndexHandler)

	// mount the api
	r.Mount("/api/", handlers.New().Handler())

	// ActivityPub has its own router
	activitypub.Start(data.GetDatastore())

	// Create a custom mux handler to intercept the /debug/vars endpoint.
	// This is a hack because Prometheus enables this endpoint by default
	// due to its use of expvar and we do not want this exposed.
	h2s := &http2.Server{}
	http2Handler := h2c.NewHandler(r, h2s)
	m := http.NewServeMux()

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/debug/vars":
			w.WriteHeader(http.StatusNotFound)
			return
		case "/embed/chat/", "/embed/chat":
			// Redirect /embed/chat
			http.Redirect(w, r, "/embed/chat/readonly", http.StatusTemporaryRedirect)
		default:
			http2Handler.ServeHTTP(w, r)
		}
	})

	port := config.WebServerPort
	ip := config.WebServerIP

	compress, _ := httpcompression.DefaultAdapter() // Use the default configuration
	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", ip, port),
		ReadHeaderTimeout: 4 * time.Second,
		Handler:           compress(m),
	}

	if ip != "0.0.0.0" {
		log.Infof("Web server is listening at %s:%d.", ip, port)
	} else {
		log.Infof("Web server is listening on port %d.", port)
	}
	log.Infoln("Configure this server by visiting /admin.")

	return server.ListenAndServe()
}

func addStaticFileEndpoints(r chi.Router) {
	// Images
	r.HandleFunc("/thumbnail.jpg", handlers.GetThumbnail)
	r.HandleFunc("/preview.gif", handlers.GetPreview)
	r.HandleFunc("/logo", handlers.GetLogo)
	r.HandleFunc("/favicon.ico", handlers.GetFavicon)
	// return a logo that's compatible with external social networks
	r.HandleFunc("/logo/external", handlers.GetCompatibleLogo)

	// Custom Javascript
	r.HandleFunc("/customjavascript", handlers.ServeCustomJavascript)

	// robots.txt
	r.HandleFunc("/robots.txt", handlers.GetRobotsDotTxt)

	// Return a single emoji image.
	emojiDir := config.EmojiDir
	if !strings.HasSuffix(emojiDir, "*") {
		emojiDir += "*"
	}
	r.HandleFunc(emojiDir, handlers.GetCustomEmojiImage)

	// WebFinger
	r.HandleFunc("/.well-known/webfinger", aphandlers.WebfingerHandler)

	// Host Metadata
	r.HandleFunc("/.well-known/host-meta", aphandlers.HostMetaController)

	// Nodeinfo v1
	r.HandleFunc("/.well-known/nodeinfo", aphandlers.NodeInfoController)

	// x-nodeinfo v2
	r.HandleFunc("/.well-known/x-nodeinfo2", aphandlers.XNodeInfo2Controller)

	// Nodeinfo v2
	r.HandleFunc("/nodeinfo/2.0", aphandlers.NodeInfoV2Controller)

	// Instance details
	r.HandleFunc("/api/v1/instance", aphandlers.InstanceV1Controller)
}
