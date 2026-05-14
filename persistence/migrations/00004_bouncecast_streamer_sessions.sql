-- +goose Up
-- +goose StatementBegin

-- Additive BounceCast Studio sessions for DJ dashboard login. These sessions
-- do not replace Owncast admin Basic Auth.

CREATE TABLE IF NOT EXISTS bouncecast_streamer_sessions (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "streamer_id" INTEGER NOT NULL,
    "token_hash" TEXT NOT NULL UNIQUE,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "expires_at" TIMESTAMP NOT NULL,
    "last_used_at" TIMESTAMP,
    "revoked_at" TIMESTAMP,
    "user_agent" TEXT,
    "remote_addr" TEXT,
    FOREIGN KEY("streamer_id") REFERENCES bouncecast_streamer_accounts("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bouncecast_streamer_sessions_streamer_id
    ON bouncecast_streamer_sessions ("streamer_id");

CREATE INDEX IF NOT EXISTS idx_bouncecast_streamer_sessions_token_hash
    ON bouncecast_streamer_sessions ("token_hash");

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS bouncecast_streamer_sessions;

-- +goose StatementEnd
