-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS bouncecast_notification_settings (
    "key" TEXT NOT NULL PRIMARY KEY,
    "value" TEXT,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS bouncecast_notification_settings;

-- +goose StatementEnd
