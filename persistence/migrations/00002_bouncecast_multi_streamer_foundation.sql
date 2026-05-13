-- +goose Up
-- +goose StatementBegin

-- Additive BounceCast tables for the multi-DJ dashboard roadmap. These do not
-- replace the existing Owncast single-channel config, chat, RTMP, or HLS data.

CREATE TABLE IF NOT EXISTS bouncecast_streamer_accounts (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "display_name" TEXT NOT NULL,
    "handle" TEXT NOT NULL UNIQUE,
    "email" TEXT UNIQUE,
    "password_hash" TEXT,
    "role" TEXT NOT NULL DEFAULT 'streamer',
    "status" TEXT NOT NULL DEFAULT 'invited',
    "avatar_url" TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "last_login_at" TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bouncecast_streamer_accounts_handle
    ON bouncecast_streamer_accounts ("handle");

CREATE INDEX IF NOT EXISTS idx_bouncecast_streamer_accounts_email
    ON bouncecast_streamer_accounts ("email");

CREATE TABLE IF NOT EXISTS bouncecast_streamer_stream_keys (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "streamer_id" INTEGER NOT NULL,
    "key_hash" TEXT NOT NULL UNIQUE,
    "label" TEXT,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "last_used_at" TIMESTAMP,
    "revoked_at" TIMESTAMP,
    FOREIGN KEY("streamer_id") REFERENCES bouncecast_streamer_accounts("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bouncecast_streamer_stream_keys_streamer_id
    ON bouncecast_streamer_stream_keys ("streamer_id");

CREATE TABLE IF NOT EXISTS bouncecast_stream_schedule (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "streamer_id" INTEGER,
    "title" TEXT NOT NULL,
    "description" TEXT,
    "starts_at" TIMESTAMP NOT NULL,
    "ends_at" TIMESTAMP,
    "timezone" TEXT NOT NULL DEFAULT 'UTC',
    "status" TEXT NOT NULL DEFAULT 'planned',
    "visibility" TEXT NOT NULL DEFAULT 'public',
    "notify_email" INTEGER NOT NULL DEFAULT 0,
    "notify_push" INTEGER NOT NULL DEFAULT 0,
    "notify_webhook" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("streamer_id") REFERENCES bouncecast_streamer_accounts("id") ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_bouncecast_stream_schedule_starts_at
    ON bouncecast_stream_schedule ("starts_at");

CREATE INDEX IF NOT EXISTS idx_bouncecast_stream_schedule_streamer_id
    ON bouncecast_stream_schedule ("streamer_id");

CREATE TABLE IF NOT EXISTS bouncecast_go_live_events (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "streamer_id" INTEGER,
    "schedule_id" INTEGER,
    "stream_key_id" INTEGER,
    "started_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "ended_at" TIMESTAMP,
    "remote_addr" TEXT,
    "status" TEXT NOT NULL DEFAULT 'live',
    "notification_state" TEXT NOT NULL DEFAULT 'pending',
    FOREIGN KEY("streamer_id") REFERENCES bouncecast_streamer_accounts("id") ON DELETE SET NULL,
    FOREIGN KEY("schedule_id") REFERENCES bouncecast_stream_schedule("id") ON DELETE SET NULL,
    FOREIGN KEY("stream_key_id") REFERENCES bouncecast_streamer_stream_keys("id") ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_bouncecast_go_live_events_started_at
    ON bouncecast_go_live_events ("started_at");

CREATE INDEX IF NOT EXISTS idx_bouncecast_go_live_events_streamer_id
    ON bouncecast_go_live_events ("streamer_id");

CREATE TABLE IF NOT EXISTS bouncecast_notification_subscribers (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "channel" TEXT NOT NULL,
    "destination" TEXT NOT NULL,
    "display_name" TEXT,
    "verified_at" TIMESTAMP,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "disabled_at" TIMESTAMP,
    UNIQUE("channel", "destination")
);

CREATE TABLE IF NOT EXISTS bouncecast_notification_deliveries (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "go_live_event_id" INTEGER,
    "subscriber_id" INTEGER,
    "channel" TEXT NOT NULL,
    "destination" TEXT,
    "status" TEXT NOT NULL DEFAULT 'queued',
    "attempt_count" INTEGER NOT NULL DEFAULT 0,
    "last_error" TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "sent_at" TIMESTAMP,
    FOREIGN KEY("go_live_event_id") REFERENCES bouncecast_go_live_events("id") ON DELETE CASCADE,
    FOREIGN KEY("subscriber_id") REFERENCES bouncecast_notification_subscribers("id") ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_bouncecast_notification_deliveries_event_id
    ON bouncecast_notification_deliveries ("go_live_event_id");

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS bouncecast_notification_deliveries;
DROP TABLE IF EXISTS bouncecast_notification_subscribers;
DROP TABLE IF EXISTS bouncecast_go_live_events;
DROP TABLE IF EXISTS bouncecast_stream_schedule;
DROP TABLE IF EXISTS bouncecast_streamer_stream_keys;
DROP TABLE IF EXISTS bouncecast_streamer_accounts;

-- +goose StatementEnd
