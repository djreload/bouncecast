-- +goose Up
-- +goose StatementBegin

-- Store public viewer notification preferences on the existing chat user row.
-- Delivery destinations still use the existing notifications and
-- bouncecast_notification_subscribers tables so go-live delivery behavior stays
-- aligned with the rest of BounceCast.

ALTER TABLE users ADD COLUMN notification_email_opt_in INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN notification_browser_push_opt_in INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN notification_messenger_opt_in INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN notification_messenger_destination TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite column removal is intentionally not attempted here for compatibility
-- with older SQLite runtimes.

-- +goose StatementEnd
