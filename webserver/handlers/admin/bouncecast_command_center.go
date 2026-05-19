package admin

import (
	"database/sql"
	"net/http"

	"github.com/owncast/owncast/core/data"
	webutils "github.com/owncast/owncast/webserver/utils"
)

type BounceCastCommandCenterSummary struct {
	Accounts  BounceCastCommandCenterAccounts  `json:"accounts"`
	Streamers BounceCastCommandCenterStreamers `json:"streamers"`
	Schedule  BounceCastCommandCenterSchedule  `json:"schedule"`
	Stars     BounceCastCommandCenterStars     `json:"stars"`
}

type BounceCastCommandCenterAccounts struct {
	Total      int `json:"total"`
	Registered int `json:"registered"`
	Owners     int `json:"owners"`
	Admins     int `json:"admins"`
	Moderators int `json:"moderators"`
	DJs        int `json:"djs"`
	Disabled   int `json:"disabled"`
}

type BounceCastCommandCenterStreamers struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Inactive int `json:"inactive"`
	Disabled int `json:"disabled"`
}

type BounceCastCommandCenterSchedule struct {
	Upcoming  int `json:"upcoming"`
	Live      int `json:"live"`
	Reminders int `json:"reminders"`
}

type BounceCastCommandCenterStars struct {
	Enabled         bool `json:"enabled"`
	Wallets         int  `json:"wallets"`
	PendingOrders   int  `json:"pendingOrders"`
	CompletedOrders int  `json:"completedOrders"`
	SendEvents      int  `json:"sendEvents"`
}

// GetBounceCastCommandCenter returns high-level admin counts for the BounceCast
// command-center dashboard without exposing secrets or stream keys.
func GetBounceCastCommandCenter(w http.ResponseWriter, r *http.Request) {
	db := data.GetDatabase()
	summary := BounceCastCommandCenterSummary{
		Accounts: BounceCastCommandCenterAccounts{
			Total:      queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM users WHERE type IS NULL OR type != 'API'`),
			Registered: queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM users WHERE (type IS NULL OR type != 'API') AND email IS NOT NULL AND email != '' AND registered_at IS NOT NULL`),
			Owners:     queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM users WHERE (type IS NULL OR type != 'API') AND (',' || COALESCE(scopes, '') || ',') LIKE '%,OWNER,%'`),
			Admins:     queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM users WHERE (type IS NULL OR type != 'API') AND (',' || COALESCE(scopes, '') || ',') LIKE '%,ADMIN,%'`),
			Moderators: queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM users WHERE (type IS NULL OR type != 'API') AND (',' || COALESCE(scopes, '') || ',') LIKE '%,MODERATOR,%'`),
			DJs:        queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM users WHERE (type IS NULL OR type != 'API') AND (',' || COALESCE(scopes, '') || ',') LIKE '%,DJ,%'`),
			Disabled:   queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM users WHERE (type IS NULL OR type != 'API') AND disabled_at IS NOT NULL`),
		},
		Streamers: BounceCastCommandCenterStreamers{
			Total:    queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM bouncecast_streamer_accounts`),
			Active:   queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM bouncecast_streamer_accounts WHERE status = 'active'`),
			Inactive: queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM bouncecast_streamer_accounts WHERE status IN ('inactive', 'invited')`),
			Disabled: queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM bouncecast_streamer_accounts WHERE status = 'disabled'`),
		},
		Schedule: BounceCastCommandCenterSchedule{
			Upcoming:  queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM bouncecast_stream_schedule WHERE visibility = 'public' AND status = 'planned' AND starts_at >= datetime('now', '-30 minutes')`),
			Live:      queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM bouncecast_go_live_events WHERE status = 'live' AND ended_at IS NULL`),
			Reminders: queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM bouncecast_schedule_reminders WHERE disabled_at IS NULL`),
		},
		Stars: BounceCastCommandCenterStars{
			Enabled:         queryBounceCastStarsEnabled(db),
			Wallets:         queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM star_wallets`),
			PendingOrders:   queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM star_paypal_orders WHERE status = 'pending'`),
			CompletedOrders: queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM star_paypal_orders WHERE status = 'completed'`),
			SendEvents:      queryBounceCastCommandCount(db, `SELECT COUNT(*) FROM star_send_events`),
		},
	}

	webutils.WriteResponse(w, summary)
}

func queryBounceCastCommandCount(db *sql.DB, query string) int {
	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		return 0
	}
	return count
}

func queryBounceCastStarsEnabled(db *sql.DB) bool {
	var value string
	if err := db.QueryRow(`SELECT value FROM star_settings WHERE key = 'enabled'`).Scan(&value); err != nil {
		return false
	}
	return value == "true" || value == "1"
}
