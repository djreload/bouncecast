package rtmp

import (
	"database/sql"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/utils"
	log "github.com/sirupsen/logrus"
)

func validateBounceCastStreamerKey(path string) bool {
	streamingKey, ok := getStreamKeyFromPath(path)
	if !ok {
		return false
	}

	db := data.GetDatabase()
	if db == nil {
		return false
	}

	rows, err := db.Query(`
		SELECT k.id, k.key_hash, a.display_name
		FROM bouncecast_streamer_stream_keys k
		INNER JOIN bouncecast_streamer_accounts a ON a.id = k.streamer_id
		WHERE k.enabled = 1 AND k.revoked_at IS NULL AND a.status = 'active'
	`)
	if err != nil {
		log.Debugln("unable to query BounceCast streamer stream keys", err)
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var keyID int64
		var keyHash string
		var displayName sql.NullString
		if err := rows.Scan(&keyID, &keyHash, &displayName); err != nil {
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
			return true
		}
	}

	return false
}
