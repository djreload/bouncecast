-- +goose Up
-- +goose StatementBegin

-- Additive account/profile fields for BounceCast public chat users. Existing
-- anonymous Owncast chat identities and access tokens continue to work.

ALTER TABLE users ADD COLUMN email TEXT;
ALTER TABLE users ADD COLUMN password_hash TEXT;
ALTER TABLE users ADD COLUMN profile_image_url TEXT;
ALTER TABLE users ADD COLUMN registered_at TIMESTAMP;
ALTER TABLE users ADD COLUMN last_login_at TIMESTAMP;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email
    ON users ("email")
    WHERE email IS NOT NULL AND email != '';

CREATE INDEX IF NOT EXISTS idx_users_registered_at
    ON users ("registered_at");

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_users_registered_at;
DROP INDEX IF EXISTS idx_users_email;

-- SQLite column removal is intentionally not attempted here for compatibility
-- with older SQLite runtimes.

-- +goose StatementEnd
