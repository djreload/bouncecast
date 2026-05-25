-- +goose Up
-- +goose StatementBegin

ALTER TABLE mobile_branding_settings
    ADD COLUMN "app_background_url" TEXT;

ALTER TABLE mobile_feature_flags
    ADD COLUMN "stars_enabled" INTEGER NOT NULL DEFAULT 1;

ALTER TABLE mobile_feature_flags
    ADD COLUMN "chat_reactions_enabled" INTEGER NOT NULL DEFAULT 1;

ALTER TABLE mobile_feature_flags
    ADD COLUMN "stars_overlay_enabled" INTEGER NOT NULL DEFAULT 1;

ALTER TABLE mobile_feature_flags
    ADD COLUMN "reward_wheel_enabled" INTEGER NOT NULL DEFAULT 1;

ALTER TABLE mobile_feature_flags
    ADD COLUMN "reward_overlay_enabled" INTEGER NOT NULL DEFAULT 1;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite cannot drop columns without rebuilding the table; keep this migration
-- forward-only to preserve existing mobile settings data.

-- +goose StatementEnd
