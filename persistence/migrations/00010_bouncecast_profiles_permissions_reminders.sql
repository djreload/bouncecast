-- +goose Up
-- +goose StatementBegin

-- Rich public DJ profile metadata. These are additive and do not change the
-- existing RTMP/login/storage compatibility contract.
ALTER TABLE bouncecast_streamer_accounts ADD COLUMN bio TEXT;
ALTER TABLE bouncecast_streamer_accounts ADD COLUMN genres TEXT;
ALTER TABLE bouncecast_streamer_accounts ADD COLUMN social_links TEXT;
ALTER TABLE bouncecast_streamer_accounts ADD COLUMN hero_image_url TEXT;
ALTER TABLE bouncecast_streamer_accounts ADD COLUMN profile_updated_at TIMESTAMP;

-- Per-account reminders for public scheduled sets. Global subscribers continue
-- to work; this table lets logged-in viewers explicitly request alerts for a
-- set they care about.
CREATE TABLE IF NOT EXISTS bouncecast_schedule_reminders (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "schedule_id" INTEGER NOT NULL,
    "user_id" TEXT NOT NULL,
    "email" TEXT,
    "notify_email" INTEGER NOT NULL DEFAULT 0,
    "notify_push" INTEGER NOT NULL DEFAULT 0,
    "notify_messenger" INTEGER NOT NULL DEFAULT 0,
    "messenger_destination" TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "disabled_at" TIMESTAMP,
    UNIQUE("schedule_id", "user_id"),
    FOREIGN KEY("schedule_id") REFERENCES bouncecast_stream_schedule("id") ON DELETE CASCADE,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bouncecast_schedule_reminders_schedule_id
    ON bouncecast_schedule_reminders ("schedule_id");

CREATE INDEX IF NOT EXISTS idx_bouncecast_schedule_reminders_user_id
    ON bouncecast_schedule_reminders ("user_id");

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_bouncecast_schedule_reminders_user_id;
DROP INDEX IF EXISTS idx_bouncecast_schedule_reminders_schedule_id;
DROP TABLE IF EXISTS bouncecast_schedule_reminders;

-- SQLite column removal is intentionally not attempted here for compatibility
-- with older SQLite runtimes.

-- +goose StatementEnd
