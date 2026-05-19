-- +goose Up
-- Meta-policy-safe Facebook Messenger go-live alerts. Subscribers are only
-- users who have interacted with the Page/opted in; Page follower fan-out is
-- intentionally not modelled here.
CREATE TABLE IF NOT EXISTS bouncecast_messenger_alert_subscribers (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "psid" TEXT NOT NULL UNIQUE,
    "display_name" TEXT,
    "source" TEXT NOT NULL DEFAULT 'website_messenger_optin',
    "opted_in" INTEGER NOT NULL DEFAULT 1,
    "opt_in_at" TIMESTAMP,
    "opt_out_at" TIMESTAMP,
    "last_interaction_at" TIMESTAMP,
    "last_sent_at" TIMESTAMP,
    "send_count" INTEGER NOT NULL DEFAULT 0,
    "failure_count" INTEGER NOT NULL DEFAULT 0,
    "status" TEXT NOT NULL DEFAULT 'active',
    "metadata" TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bouncecast_messenger_alert_subscribers_status
    ON bouncecast_messenger_alert_subscribers ("status", "opted_in");

CREATE INDEX IF NOT EXISTS idx_bouncecast_messenger_alert_subscribers_last_interaction
    ON bouncecast_messenger_alert_subscribers ("last_interaction_at");

CREATE TABLE IF NOT EXISTS bouncecast_messenger_alert_campaigns (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "trigger_type" TEXT NOT NULL,
    "go_live_event_id" INTEGER,
    "schedule_id" INTEGER,
    "started_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "completed_at" TIMESTAMP,
    "attempted_count" INTEGER NOT NULL DEFAULT 0,
    "sent_count" INTEGER NOT NULL DEFAULT 0,
    "skipped_count" INTEGER NOT NULL DEFAULT 0,
    "failed_count" INTEGER NOT NULL DEFAULT 0,
    "status" TEXT NOT NULL DEFAULT 'queued',
    "error_summary" TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY("go_live_event_id") REFERENCES bouncecast_go_live_events("id") ON DELETE SET NULL,
    FOREIGN KEY("schedule_id") REFERENCES bouncecast_stream_schedule("id") ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_bouncecast_messenger_alert_campaigns_go_live_once
    ON bouncecast_messenger_alert_campaigns ("trigger_type", "go_live_event_id")
    WHERE "go_live_event_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_bouncecast_messenger_alert_campaigns_created_at
    ON bouncecast_messenger_alert_campaigns ("created_at");

CREATE TABLE IF NOT EXISTS bouncecast_messenger_alert_campaign_recipients (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "campaign_id" INTEGER NOT NULL,
    "subscriber_id" INTEGER,
    "psid" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'queued',
    "error" TEXT,
    "sent_at" TIMESTAMP,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY("campaign_id") REFERENCES bouncecast_messenger_alert_campaigns("id") ON DELETE CASCADE,
    FOREIGN KEY("subscriber_id") REFERENCES bouncecast_messenger_alert_subscribers("id") ON DELETE SET NULL,
    UNIQUE("campaign_id", "psid")
);

CREATE INDEX IF NOT EXISTS idx_bouncecast_messenger_alert_campaign_recipients_campaign
    ON bouncecast_messenger_alert_campaign_recipients ("campaign_id", "status");

ALTER TABLE bouncecast_streamer_accounts ADD COLUMN seo_title TEXT;
ALTER TABLE bouncecast_streamer_accounts ADD COLUMN share_image_url TEXT;
ALTER TABLE bouncecast_streamer_accounts ADD COLUMN featured_schedule_id INTEGER;
ALTER TABLE bouncecast_streamer_accounts ADD COLUMN featured_schedule_enabled INTEGER NOT NULL DEFAULT 0;

-- +goose Down
DROP INDEX IF EXISTS idx_bouncecast_messenger_alert_campaign_recipients_campaign;
DROP TABLE IF EXISTS bouncecast_messenger_alert_campaign_recipients;
DROP INDEX IF EXISTS idx_bouncecast_messenger_alert_campaigns_created_at;
DROP INDEX IF EXISTS idx_bouncecast_messenger_alert_campaigns_go_live_once;
DROP TABLE IF EXISTS bouncecast_messenger_alert_campaigns;
DROP INDEX IF EXISTS idx_bouncecast_messenger_alert_subscribers_last_interaction;
DROP INDEX IF EXISTS idx_bouncecast_messenger_alert_subscribers_status;
DROP TABLE IF EXISTS bouncecast_messenger_alert_subscribers;
