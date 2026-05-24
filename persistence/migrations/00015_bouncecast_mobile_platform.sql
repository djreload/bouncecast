-- +goose Up
-- +goose StatementBegin

-- BounceCast mobile platform foundation. These tables configure official
-- mobile clients only; livestream playback and chat remain owned by the
-- existing BounceCast/Owncast stream, websocket, and user tables.

CREATE TABLE IF NOT EXISTS mobile_app_settings (
    "id" INTEGER NOT NULL PRIMARY KEY CHECK ("id" = 1),
    "app_name" TEXT NOT NULL DEFAULT 'BounceCast',
    "public_base_url" TEXT,
    "maintenance_mode" INTEGER NOT NULL DEFAULT 0,
    "maintenance_message" TEXT,
    "minimum_supported_version" TEXT NOT NULL DEFAULT '1.0.0',
    "recommended_version" TEXT,
    "force_update" INTEGER NOT NULL DEFAULT 0,
    "force_update_message" TEXT,
    "homepage_message" TEXT,
    "support_url" TEXT,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

INSERT INTO mobile_app_settings ("id", "app_name", "minimum_supported_version", "homepage_message")
VALUES (1, 'BounceCast', '1.0.0', 'Welcome to BounceCast.')
ON CONFLICT("id") DO NOTHING;

CREATE TABLE IF NOT EXISTS mobile_branding_settings (
    "id" INTEGER NOT NULL PRIMARY KEY CHECK ("id" = 1),
    "logo_url" TEXT,
    "splash_url" TEXT,
    "app_icon_url" TEXT,
    "primary_color" TEXT NOT NULL DEFAULT '#080711',
    "accent_color" TEXT NOT NULL DEFAULT '#ff2a8a',
    "background_color" TEXT NOT NULL DEFAULT '#05050f',
    "theme_mode" TEXT NOT NULL DEFAULT 'system',
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

INSERT INTO mobile_branding_settings ("id")
VALUES (1)
ON CONFLICT("id") DO NOTHING;

CREATE TABLE IF NOT EXISTS mobile_feature_flags (
    "id" INTEGER NOT NULL PRIMARY KEY CHECK ("id" = 1),
    "chat_enabled" INTEGER NOT NULL DEFAULT 1,
    "gif_picker_enabled" INTEGER NOT NULL DEFAULT 1,
    "stickers_enabled" INTEGER NOT NULL DEFAULT 1,
    "profiles_enabled" INTEGER NOT NULL DEFAULT 1,
    "push_notifications_enabled" INTEGER NOT NULL DEFAULT 0,
    "ads_enabled" INTEGER NOT NULL DEFAULT 0,
    "experimental_features_enabled" INTEGER NOT NULL DEFAULT 0,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

INSERT INTO mobile_feature_flags ("id")
VALUES (1)
ON CONFLICT("id") DO NOTHING;

CREATE TABLE IF NOT EXISTS mobile_ad_settings (
    "id" INTEGER NOT NULL PRIMARY KEY CHECK ("id" = 1),
    "enabled" INTEGER NOT NULL DEFAULT 0,
    "test_mode" INTEGER NOT NULL DEFAULT 1,
    "fallback_enabled" INTEGER NOT NULL DEFAULT 1,
    "provider_priority" TEXT NOT NULL DEFAULT '["google","unity"]',
    "google_enabled" INTEGER NOT NULL DEFAULT 0,
    "unity_enabled" INTEGER NOT NULL DEFAULT 0,
    "google_app_id" TEXT,
    "google_banner_ad_unit_id" TEXT,
    "google_app_open_ad_unit_id" TEXT,
    "unity_game_id" TEXT,
    "unity_banner_placement_id" TEXT,
    "unity_app_open_placement_id" TEXT,
    "banner_enabled" INTEGER NOT NULL DEFAULT 0,
    "banner_position" TEXT NOT NULL DEFAULT 'bottom',
    "app_open_enabled" INTEGER NOT NULL DEFAULT 0,
    "app_open_cooldown_minutes" INTEGER NOT NULL DEFAULT 30,
    "app_open_show_on_first_launch" INTEGER NOT NULL DEFAULT 1,
    "timeout_ms" INTEGER NOT NULL DEFAULT 5000,
    "retry_limit" INTEGER NOT NULL DEFAULT 1,
    "consent_required" INTEGER NOT NULL DEFAULT 1,
    "personalized_ads_allowed" INTEGER NOT NULL DEFAULT 0,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

INSERT INTO mobile_ad_settings ("id")
VALUES (1)
ON CONFLICT("id") DO NOTHING;

CREATE TABLE IF NOT EXISTS mobile_notification_settings (
    "id" INTEGER NOT NULL PRIMARY KEY CHECK ("id" = 1),
    "go_live_enabled" INTEGER NOT NULL DEFAULT 0,
    "title_template" TEXT NOT NULL DEFAULT 'BounceCast is live',
    "body_template" TEXT NOT NULL DEFAULT '{stream_title} is live now. Tap to watch.',
    "image_url" TEXT,
    "icon_url" TEXT,
    "topic" TEXT NOT NULL DEFAULT 'go-live',
    "fcm_project_id" TEXT,
    "last_successful_send_at" TIMESTAMP,
    "last_error" TEXT,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

INSERT INTO mobile_notification_settings ("id")
VALUES (1)
ON CONFLICT("id") DO NOTHING;

CREATE TABLE IF NOT EXISTS mobile_navigation_items (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "label" TEXT NOT NULL,
    "url" TEXT NOT NULL,
    "item_type" TEXT NOT NULL DEFAULT 'internal',
    "display_order" INTEGER NOT NULL DEFAULT 0,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

INSERT INTO mobile_navigation_items ("label", "url", "item_type", "display_order", "enabled")
SELECT 'Live', '/', 'internal', 10, 1
WHERE NOT EXISTS (SELECT 1 FROM mobile_navigation_items);

INSERT INTO mobile_navigation_items ("label", "url", "item_type", "display_order", "enabled")
SELECT 'Schedule', '/schedule', 'internal', 20, 1
WHERE NOT EXISTS (SELECT 1 FROM mobile_navigation_items WHERE "url" = '/schedule');

INSERT INTO mobile_navigation_items ("label", "url", "item_type", "display_order", "enabled")
SELECT 'DJs', '/djs', 'internal', 30, 1
WHERE NOT EXISTS (SELECT 1 FROM mobile_navigation_items WHERE "url" = '/djs');

CREATE TABLE IF NOT EXISTS mobile_legal_pages (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "slug" TEXT NOT NULL UNIQUE,
    "title" TEXT NOT NULL,
    "content" TEXT,
    "url" TEXT,
    "display_order" INTEGER NOT NULL DEFAULT 0,
    "enabled" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

INSERT INTO mobile_legal_pages ("slug", "title", "display_order", "enabled")
VALUES ('privacy', 'Privacy Policy', 10, 0)
ON CONFLICT("slug") DO NOTHING;

INSERT INTO mobile_legal_pages ("slug", "title", "display_order", "enabled")
VALUES ('terms', 'Terms and Conditions', 20, 0)
ON CONFLICT("slug") DO NOTHING;

CREATE TABLE IF NOT EXISTS mobile_asset_uploads (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "asset_type" TEXT NOT NULL,
    "url" TEXT NOT NULL,
    "content_type" TEXT,
    "size_bytes" INTEGER,
    "created_by" TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_mobile_asset_uploads_type
    ON mobile_asset_uploads ("asset_type", "created_at");

CREATE TABLE IF NOT EXISTS mobile_device_tokens (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "user_id" TEXT,
    "platform" TEXT NOT NULL DEFAULT 'android',
    "device_id" TEXT,
    "token_protected" TEXT NOT NULL,
    "token_hash" TEXT NOT NULL UNIQUE,
    "token_preview" TEXT NOT NULL,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "notifications_enabled" INTEGER NOT NULL DEFAULT 1,
    "app_version" TEXT,
    "locale" TEXT,
    "last_seen_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_mobile_device_tokens_user
    ON mobile_device_tokens ("user_id", "enabled");

CREATE INDEX IF NOT EXISTS idx_mobile_device_tokens_last_seen
    ON mobile_device_tokens ("last_seen_at");

CREATE TABLE IF NOT EXISTS mobile_notification_logs (
    "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    "trigger_type" TEXT NOT NULL,
    "target" TEXT NOT NULL,
    "title" TEXT,
    "body" TEXT,
    "status" TEXT NOT NULL DEFAULT 'queued',
    "attempt_count" INTEGER NOT NULL DEFAULT 0,
    "success_count" INTEGER NOT NULL DEFAULT 0,
    "failure_count" INTEGER NOT NULL DEFAULT 0,
    "error_summary" TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "sent_at" TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mobile_notification_logs_created
    ON mobile_notification_logs ("trigger_type", "created_at");

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_mobile_notification_logs_created;
DROP TABLE IF EXISTS mobile_notification_logs;
DROP INDEX IF EXISTS idx_mobile_device_tokens_last_seen;
DROP INDEX IF EXISTS idx_mobile_device_tokens_user;
DROP TABLE IF EXISTS mobile_device_tokens;
DROP INDEX IF EXISTS idx_mobile_asset_uploads_type;
DROP TABLE IF EXISTS mobile_asset_uploads;
DROP TABLE IF EXISTS mobile_legal_pages;
DROP TABLE IF EXISTS mobile_navigation_items;
DROP TABLE IF EXISTS mobile_notification_settings;
DROP TABLE IF EXISTS mobile_ad_settings;
DROP TABLE IF EXISTS mobile_feature_flags;
DROP TABLE IF EXISTS mobile_branding_settings;
DROP TABLE IF EXISTS mobile_app_settings;

-- +goose StatementEnd
