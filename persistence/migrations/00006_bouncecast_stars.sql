-- +goose Up
-- +goose StatementBegin

-- BounceCast Stars are a site-support donation/gifting feature. They are not
-- streamer balances and have no payout or withdrawal workflow.

CREATE TABLE IF NOT EXISTS star_settings (
    "key" TEXT NOT NULL PRIMARY KEY,
    "value" TEXT,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS star_packages (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "name" TEXT NOT NULL,
    "star_amount" INTEGER NOT NULL,
    "price_cents" INTEGER NOT NULL,
    "currency" TEXT NOT NULL DEFAULT 'GBP',
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "display_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_star_packages_enabled_order
    ON star_packages ("enabled", "display_order");

CREATE TABLE IF NOT EXISTS star_wallets (
    "user_id" TEXT NOT NULL PRIMARY KEY,
    "balance" INTEGER NOT NULL DEFAULT 0 CHECK ("balance" >= 0),
    "lifetime_purchased" INTEGER NOT NULL DEFAULT 0,
    "lifetime_sent" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS star_wallet_transactions (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" TEXT NOT NULL,
    "transaction_type" TEXT NOT NULL,
    "amount" INTEGER NOT NULL,
    "balance_after" INTEGER NOT NULL CHECK ("balance_after" >= 0),
    "reference_type" TEXT,
    "reference_id" TEXT,
    "notes" TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_star_wallet_transactions_user_id
    ON star_wallet_transactions ("user_id", "created_at");

CREATE TABLE IF NOT EXISTS star_paypal_orders (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" TEXT NOT NULL,
    "package_id" INTEGER NOT NULL,
    "paypal_order_id" TEXT NOT NULL UNIQUE,
    "paypal_capture_id" TEXT UNIQUE,
    "paypal_payer_id" TEXT,
    "star_amount" INTEGER NOT NULL,
    "amount_cents" INTEGER NOT NULL,
    "currency" TEXT NOT NULL,
    "status" TEXT NOT NULL,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "completed_at" TIMESTAMP,
    "raw_status" TEXT,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE,
    FOREIGN KEY("package_id") REFERENCES star_packages("id") ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_star_paypal_orders_user_id
    ON star_paypal_orders ("user_id", "created_at");

CREATE INDEX IF NOT EXISTS idx_star_paypal_orders_status
    ON star_paypal_orders ("status", "created_at");

CREATE TABLE IF NOT EXISTS star_send_events (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" TEXT NOT NULL,
    "display_name" TEXT NOT NULL,
    "amount" INTEGER NOT NULL,
    "message" TEXT,
    "effect" TEXT NOT NULL,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_star_send_events_created_at
    ON star_send_events ("created_at");

CREATE TABLE IF NOT EXISTS star_paypal_webhook_events (
    "paypal_event_id" TEXT NOT NULL PRIMARY KEY,
    "event_type" TEXT NOT NULL,
    "resource_id" TEXT,
    "status" TEXT NOT NULL,
    "received_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "processed_at" TIMESTAMP,
    "error" TEXT
);

INSERT INTO star_settings("key", "value") VALUES
    ('enabled', 'false'),
    ('paypal_environment', 'sandbox'),
    ('paypal_client_id', ''),
    ('paypal_client_secret', ''),
    ('paypal_webhook_id', ''),
    ('currency', 'GBP'),
    ('support_message', 'Stars are a fun way to support this site and trigger live on-screen effects. Stars have no cash value and are not paid out to streamers.'),
    ('minimum_send_amount', '1'),
    ('maximum_send_amount', '7000'),
    ('send_cooldown_seconds', '10'),
    ('overlay_effects_enabled', 'true'),
    ('sound_effects_enabled', 'true'),
    ('debug_logging_enabled', 'false')
ON CONFLICT("key") DO NOTHING;

INSERT INTO star_packages("name", "star_amount", "price_cents", "currency", "enabled", "display_order") VALUES
    ('100 Stars', 100, 100, 'GBP', 1, 10),
    ('550 Stars', 550, 500, 'GBP', 1, 20),
    ('1200 Stars', 1200, 1000, 'GBP', 1, 30),
    ('2600 Stars', 2600, 2000, 'GBP', 1, 40),
    ('7000 Stars', 7000, 5000, 'GBP', 1, 50);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS star_paypal_webhook_events;
DROP TABLE IF EXISTS star_send_events;
DROP TABLE IF EXISTS star_paypal_orders;
DROP TABLE IF EXISTS star_wallet_transactions;
DROP TABLE IF EXISTS star_wallets;
DROP TABLE IF EXISTS star_packages;
DROP TABLE IF EXISTS star_settings;

-- +goose StatementEnd
