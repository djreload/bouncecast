package rtmp

import (
	"database/sql"
	"net"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/utils"
	log "github.com/sirupsen/logrus"
)

type bounceCastStreamerKeyMatch struct {
	streamerID    int64
	streamKeyID   int64
	displayName   string
	goLiveEventID int64
}

func validateBounceCastStreamerKey(path string) *bounceCastStreamerKeyMatch {
	streamingKey, ok := getStreamKeyFromPath(path)
	if !ok {
		return nil
	}

	db := data.GetDatabase()
	if db == nil {
		return nil
	}

	rows, err := db.Query(`
		SELECT k.id, k.streamer_id, k.key_hash, a.display_name
		FROM bouncecast_streamer_stream_keys k
		INNER JOIN bouncecast_streamer_accounts a ON a.id = k.streamer_id
		WHERE k.enabled = 1 AND k.revoked_at IS NULL AND a.status = 'active'
	`)
	if err != nil {
		log.Debugln("unable to query BounceCast streamer stream keys", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var keyID int64
		var streamerID int64
		var keyHash string
		var displayName sql.NullString
		if err := rows.Scan(&keyID, &streamerID, &keyHash, &displayName); err != nil {
			log.Debugln("unable to scan BounceCast streamer stream key", err)
			continue
		}

		if utils.CompareHash(keyHash, streamingKey) == nil {
			if _, err := db.Exec(`UPDATE bouncecast_streamer_stream_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?`, keyID); err != nil {
				log.Debugln("unable to update BounceCast streamer stream key last_used_at", err)
			}
			if displayName.Valid {
				log.Infoln("Accepted BounceCast streamer key for", displayName.String)
			}
			return &bounceCastStreamerKeyMatch{
				streamerID:  streamerID,
				streamKeyID: keyID,
				displayName: displayName.String,
			}
		}
	}

	return nil
}

func beginBounceCastGoLiveEvent(match *bounceCastStreamerKeyMatch, remoteAddr net.Addr) {
	if match == nil {
		return
	}

	db := data.GetDatabase()
	if db == nil {
		return
	}

	var scheduleID sql.NullInt64
	err := db.QueryRow(`
		SELECT id
		FROM bouncecast_stream_schedule
		WHERE streamer_id = ?
			AND status IN ('planned', 'live')
			AND starts_at <= datetime('now', '+30 minutes')
			AND (ends_at IS NULL OR ends_at >= datetime('now', '-30 minutes'))
		ORDER BY starts_at DESC
		LIMIT 1
	`, match.streamerID).Scan(&scheduleID)
	if err != nil && err != sql.ErrNoRows {
		log.Debugln("unable to find BounceCast schedule for go-live event", err)
	}

	var remoteAddrString string
	if remoteAddr != nil {
		remoteAddrString = remoteAddr.String()
	}

	result, err := db.Exec(`
		INSERT INTO bouncecast_go_live_events(streamer_id, schedule_id, stream_key_id, remote_addr)
		VALUES(?, ?, ?, NULLIF(?, ''))
	`, match.streamerID, scheduleID, match.streamKeyID, remoteAddrString)
	if err != nil {
		log.Debugln("unable to create BounceCast go-live event", err)
		return
	}

	eventID, err := result.LastInsertId()
	if err == nil {
		match.goLiveEventID = eventID
	}

	if scheduleID.Valid {
		if _, err := db.Exec(`UPDATE bouncecast_stream_schedule SET status = 'live', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, scheduleID.Int64); err != nil {
			log.Debugln("unable to mark BounceCast schedule live", err)
		}
	}

	queueBounceCastGoLiveNotifications(eventID, scheduleID)
}

func endBounceCastGoLiveEvent(match *bounceCastStreamerKeyMatch) {
	if match == nil || match.goLiveEventID == 0 {
		return
	}

	db := data.GetDatabase()
	if db == nil {
		return
	}

	var scheduleID sql.NullInt64
	if err := db.QueryRow(`SELECT schedule_id FROM bouncecast_go_live_events WHERE id = ?`, match.goLiveEventID).Scan(&scheduleID); err != nil {
		log.Debugln("unable to find BounceCast go-live event schedule", err)
	}

	if _, err := db.Exec(`
		UPDATE bouncecast_go_live_events
		SET ended_at = CURRENT_TIMESTAMP, status = 'ended'
		WHERE id = ?
	`, match.goLiveEventID); err != nil {
		log.Debugln("unable to end BounceCast go-live event", err)
	}

	if scheduleID.Valid {
		if _, err := db.Exec(`UPDATE bouncecast_stream_schedule SET status = 'completed', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, scheduleID.Int64); err != nil {
			log.Debugln("unable to complete BounceCast schedule", err)
		}
	}
}

func queueBounceCastGoLiveNotifications(goLiveEventID int64, scheduleID sql.NullInt64) {
	if goLiveEventID == 0 {
		return
	}

	db := data.GetDatabase()
	if db == nil {
		return
	}

	enabledChannels := map[string]bool{
		"email":   true,
		"push":    true,
		"webhook": true,
	}

	if scheduleID.Valid {
		var notifyEmail bool
		var notifyPush bool
		var notifyWebhook bool
		if err := db.QueryRow(`
			SELECT notify_email, notify_push, notify_webhook
			FROM bouncecast_stream_schedule
			WHERE id = ?
		`, scheduleID.Int64).Scan(&notifyEmail, &notifyPush, &notifyWebhook); err != nil {
			log.Debugln("unable to read BounceCast schedule notification settings", err)
		} else {
			enabledChannels["email"] = notifyEmail
			enabledChannels["push"] = notifyPush
			enabledChannels["webhook"] = notifyWebhook
		}
	}

	rows, err := db.Query(`
		SELECT id, channel, destination
		FROM bouncecast_notification_subscribers
		WHERE disabled_at IS NULL
	`)
	if err != nil {
		log.Debugln("unable to query BounceCast notification subscribers", err)
		return
	}
	defer rows.Close()

	queuedCount := 0
	for rows.Next() {
		var subscriberID int64
		var channel string
		var destination string
		if err := rows.Scan(&subscriberID, &channel, &destination); err != nil {
			log.Debugln("unable to scan BounceCast notification subscriber", err)
			continue
		}
		if !enabledChannels[channel] {
			continue
		}
		if _, err := db.Exec(`
			INSERT INTO bouncecast_notification_deliveries(go_live_event_id, subscriber_id, channel, destination)
			VALUES(?, ?, ?, ?)
		`, goLiveEventID, subscriberID, channel, destination); err != nil {
			log.Debugln("unable to queue BounceCast notification delivery", err)
			continue
		}
		queuedCount++
	}

	state := "none"
	if queuedCount > 0 {
		state = "queued"
	}
	if _, err := db.Exec(`UPDATE bouncecast_go_live_events SET notification_state = ? WHERE id = ?`, state, goLiveEventID); err != nil {
		log.Debugln("unable to update BounceCast notification state", err)
	}
}
