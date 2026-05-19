-- +goose Up
-- Account-bound browser push subscriptions let schedule reminders target the
-- logged-in viewer that asked for the reminder instead of only the global
-- Owncast browser notification list.
CREATE TABLE IF NOT EXISTS bouncecast_account_push_subscriptions (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "user_id" TEXT NOT NULL,
    "endpoint_hash" TEXT NOT NULL,
    "subscription_json" TEXT NOT NULL,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "last_seen_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "disabled_at" TIMESTAMP,
    UNIQUE("user_id", "endpoint_hash"),
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bouncecast_account_push_subscriptions_user_id
    ON bouncecast_account_push_subscriptions ("user_id");

ALTER TABLE bouncecast_schedule_reminders ADD COLUMN browser_push_endpoint TEXT;
ALTER TABLE bouncecast_schedule_reminders ADD COLUMN last_queued_at TIMESTAMP;
ALTER TABLE bouncecast_schedule_reminders ADD COLUMN last_delivery_status TEXT;
ALTER TABLE bouncecast_schedule_reminders ADD COLUMN last_delivery_error TEXT;

ALTER TABLE bouncecast_notification_deliveries ADD COLUMN reminder_id INTEGER;

CREATE INDEX IF NOT EXISTS idx_bouncecast_notification_deliveries_reminder_id
    ON bouncecast_notification_deliveries ("reminder_id");

CREATE TABLE IF NOT EXISTS bouncecast_audit_events (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "actor_user_id" TEXT,
    "actor_role" TEXT,
    "action" TEXT NOT NULL,
    "target_type" TEXT NOT NULL,
    "target_id" TEXT,
    "metadata" TEXT,
    "remote_addr" TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bouncecast_audit_events_created_at
    ON bouncecast_audit_events ("created_at");

CREATE INDEX IF NOT EXISTS idx_bouncecast_audit_events_action
    ON bouncecast_audit_events ("action");

-- +goose Down
DROP INDEX IF EXISTS idx_bouncecast_audit_events_action;
DROP INDEX IF EXISTS idx_bouncecast_audit_events_created_at;
DROP TABLE IF EXISTS bouncecast_audit_events;
DROP INDEX IF EXISTS idx_bouncecast_notification_deliveries_reminder_id;
DROP INDEX IF EXISTS idx_bouncecast_account_push_subscriptions_user_id;
DROP TABLE IF EXISTS bouncecast_account_push_subscriptions;
