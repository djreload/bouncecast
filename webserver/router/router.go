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
	r.Post("/api/admin/bouncecast/streamers", middleware.RequireAdminAuth(adminhandlers.CreateBounceCastStreamer))
	r.Options("/api/admin/bouncecast/streamkeys", middleware.RequireAdminAuth(adminhandlers.GetBounceCastStreamKeys))
	r.Get("/api/admin/bouncecast/streamkeys", middleware.RequireAdminAuth(adminhandlers.GetBounceCastStreamKeys))
	r.Post("/api/admin/bouncecast/streamkeys", middleware.RequireAdminAuth(adminhandlers.CreateBounceCastStreamKey))
	r.Post("/api/admin/bouncecast/streamkeys/revoke", middleware.RequireAdminAuth(adminhandlers.RevokeBounceCastStreamKey))
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
	r.Post("/api/admin/bouncecast/email-settings", middleware.RequireAdminAuth(adminhandlers.SetBounceCastEmailSettings))
	r.Options("/api/admin/bouncecast/push-settings", middleware.RequireAdminAuth(adminhandlers.GetBounceCastPushSettings))
	r.Get("/api/admin/bouncecast/push-settings", middleware.RequireAdminAuth(adminhandlers.GetBounceCastPushSettings))
	r.Options("/api/admin/bouncecast/schedule", middleware.RequireAdminAuth(adminhandlers.GetBounceCastSchedule))
	r.Get("/api/admin/bouncecast/schedule", middleware.RequireAdminAuth(adminhandlers.GetBounceCastSchedule))
	r.Post("/api/admin/bouncecast/schedule", middleware.RequireAdminAuth(adminhandlers.CreateBounceCastSchedule))

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
