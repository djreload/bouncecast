-- +goose Up
-- +goose StatementBegin

-- BounceCast Rewards Wheel is an internal/manual prize system. It does not
-- integrate shops, checkout, paid spins, streamer payouts, or external reward
-- balances.

CREATE TABLE IF NOT EXISTS reward_config (
    "key" TEXT NOT NULL PRIMARY KEY,
    "value" TEXT,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS reward_prizes (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "image" TEXT,
    "prize_type" TEXT NOT NULL DEFAULT 'sorry',
    "odds_weight" INTEGER NOT NULL DEFAULT 1,
    "stock_quantity" INTEGER,
    "active" INTEGER NOT NULL DEFAULT 1,
    "display_order" INTEGER NOT NULL DEFAULT 0,
    "claim_required" INTEGER NOT NULL DEFAULT 0,
    "marketing_consent_required" INTEGER NOT NULL DEFAULT 0,
    "terms" TEXT,
    "fulfilment_notes" TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_reward_prizes_active_weight
    ON reward_prizes ("active", "odds_weight");

CREATE TABLE IF NOT EXISTS reward_spin_balances (
    "user_id" TEXT NOT NULL PRIMARY KEY,
    "balance" INTEGER NOT NULL DEFAULT 0 CHECK ("balance" >= 0),
    "lifetime_earned" INTEGER NOT NULL DEFAULT 0,
    "lifetime_spent" INTEGER NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS reward_spin_ledger (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" TEXT NOT NULL,
    "amount" INTEGER NOT NULL,
    "balance_after" INTEGER NOT NULL CHECK ("balance_after" >= 0),
    "source" TEXT NOT NULL,
    "reference_id" TEXT,
    "note" TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_reward_spin_ledger_user
    ON reward_spin_ledger ("user_id", "created_at");

CREATE UNIQUE INDEX IF NOT EXISTS idx_reward_spin_ledger_reference
    ON reward_spin_ledger ("source", "reference_id")
    WHERE "reference_id" IS NOT NULL AND "reference_id" != '';

CREATE TABLE IF NOT EXISTS reward_spins (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" TEXT NOT NULL,
    "prize_id" INTEGER,
    "prize_snapshot" TEXT NOT NULL,
    "image_snapshot" TEXT,
    "type_snapshot" TEXT NOT NULL,
    "odds_snapshot" INTEGER NOT NULL,
    "ledger_id" INTEGER NOT NULL,
    "result_type" TEXT NOT NULL,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE,
    FOREIGN KEY("prize_id") REFERENCES reward_prizes("id") ON DELETE SET NULL,
    FOREIGN KEY("ledger_id") REFERENCES reward_spin_ledger("id") ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_reward_spins_user
    ON reward_spins ("user_id", "created_at");

CREATE TABLE IF NOT EXISTS reward_claims (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" TEXT NOT NULL,
    "spin_id" INTEGER NOT NULL,
    "prize_id" INTEGER,
    "status" TEXT NOT NULL DEFAULT 'pending_details',
    "full_name" TEXT,
    "address_line_1" TEXT,
    "address_line_2" TEXT,
    "town_city" TEXT,
    "county_state" TEXT,
    "postcode" TEXT,
    "country" TEXT,
    "email" TEXT,
    "phone" TEXT,
    "delivery_notes" TEXT,
    "marketing_consent" INTEGER NOT NULL DEFAULT 0,
    "consent_at" TIMESTAMP,
    "consent_text" TEXT,
    "ip_address" TEXT,
    "user_agent" TEXT,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE,
    FOREIGN KEY("spin_id") REFERENCES reward_spins("id") ON DELETE CASCADE,
    FOREIGN KEY("prize_id") REFERENCES reward_prizes("id") ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_reward_claims_user
    ON reward_claims ("user_id", "created_at");

CREATE TABLE IF NOT EXISTS reward_orders (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "winner_user_id" TEXT NOT NULL,
    "username_snapshot" TEXT NOT NULL,
    "prize_id" INTEGER,
    "prize_snapshot" TEXT NOT NULL,
    "image_snapshot" TEXT,
    "spin_id" INTEGER NOT NULL,
    "claim_id" INTEGER,
    "address_snapshot" TEXT,
    "order_status" TEXT NOT NULL,
    "dispatch_status" TEXT NOT NULL DEFAULT 'not_dispatched',
    "admin_notes" TEXT,
    "courier" TEXT,
    "tracking_reference" TEXT,
    "tracking_url" TEXT,
    "dispatch_note" TEXT,
    "dispatch_date" TIMESTAMP,
    "delivered_date" TIMESTAMP,
    "dispatched_by_admin_id" TEXT,
    "dispatched_at" TIMESTAMP,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("winner_user_id") REFERENCES users("id") ON DELETE CASCADE,
    FOREIGN KEY("prize_id") REFERENCES reward_prizes("id") ON DELETE SET NULL,
    FOREIGN KEY("spin_id") REFERENCES reward_spins("id") ON DELETE CASCADE,
    FOREIGN KEY("claim_id") REFERENCES reward_claims("id") ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_reward_orders_status
    ON reward_orders ("order_status", "created_at");

CREATE TABLE IF NOT EXISTS reward_winners (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" TEXT NOT NULL,
    "username_snapshot" TEXT NOT NULL,
    "email" TEXT,
    "prize_id" INTEGER,
    "prize_snapshot" TEXT NOT NULL,
    "image_snapshot" TEXT,
    "type_snapshot" TEXT NOT NULL,
    "spin_id" INTEGER NOT NULL,
    "claim_id" INTEGER,
    "order_id" INTEGER,
    "notification_status" TEXT NOT NULL DEFAULT 'pending',
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE,
    FOREIGN KEY("prize_id") REFERENCES reward_prizes("id") ON DELETE SET NULL,
    FOREIGN KEY("spin_id") REFERENCES reward_spins("id") ON DELETE CASCADE,
    FOREIGN KEY("claim_id") REFERENCES reward_claims("id") ON DELETE SET NULL,
    FOREIGN KEY("order_id") REFERENCES reward_orders("id") ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_reward_winners_created_at
    ON reward_winners ("created_at");

CREATE TABLE IF NOT EXISTS reward_admin_messages (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "type" TEXT NOT NULL,
    "severity" TEXT NOT NULL DEFAULT 'info',
    "title" TEXT NOT NULL,
    "body" TEXT,
    "user_id" TEXT,
    "prize_id" INTEGER,
    "spin_id" INTEGER,
    "claim_id" INTEGER,
    "order_id" INTEGER,
    "read_at" TIMESTAMP,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE SET NULL,
    FOREIGN KEY("prize_id") REFERENCES reward_prizes("id") ON DELETE SET NULL,
    FOREIGN KEY("spin_id") REFERENCES reward_spins("id") ON DELETE SET NULL,
    FOREIGN KEY("claim_id") REFERENCES reward_claims("id") ON DELETE SET NULL,
    FOREIGN KEY("order_id") REFERENCES reward_orders("id") ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_reward_admin_messages_read
    ON reward_admin_messages ("read_at", "created_at");

CREATE TABLE IF NOT EXISTS reward_user_notifications (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "order_id" INTEGER,
    "prize_id" INTEGER,
    "title" TEXT NOT NULL,
    "message" TEXT,
    "email_sent" INTEGER NOT NULL DEFAULT 0,
    "email_error" TEXT,
    "read_at" TIMESTAMP,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE,
    FOREIGN KEY("order_id") REFERENCES reward_orders("id") ON DELETE SET NULL,
    FOREIGN KEY("prize_id") REFERENCES reward_prizes("id") ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_reward_user_notifications_user
    ON reward_user_notifications ("user_id", "read_at", "created_at");

CREATE TABLE IF NOT EXISTS reward_tasks (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "title" TEXT NOT NULL,
    "description" TEXT,
    "credit_reward" INTEGER NOT NULL DEFAULT 1,
    "active" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS reward_achievements (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "name" TEXT NOT NULL,
    "condition_key" TEXT NOT NULL,
    "reward_amount" INTEGER NOT NULL DEFAULT 1,
    "active" INTEGER NOT NULL DEFAULT 1,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

INSERT INTO reward_config("key", "value") VALUES
    ('enabled', 'false'),
    ('spin_cost', '1'),
    ('chat_rewards_enabled', 'false'),
    ('chat_valid_message_count', '10'),
    ('chat_cooldown_seconds', '300'),
    ('chat_credit_reward', '1'),
    ('top_supporter_first_credits', '25'),
    ('top_supporter_second_credits', '15'),
    ('top_supporter_third_credits', '10'),
    ('overlay_enabled', 'true'),
    ('overlay_duration_seconds', '5'),
    ('overlay_sound_enabled', 'true'),
    ('overlay_show_image', 'true'),
    ('overlay_template', '{viewer} just won {prize} on the Rewards Wheel!'),
    ('debug_logging_enabled', 'false')
ON CONFLICT("key") DO NOTHING;

INSERT INTO reward_prizes("name", "description", "prize_type", "odds_weight", "stock_quantity", "active", "claim_required", "display_order")
SELECT 'Try again', 'No win this time, but the next spin could hit.', 'sorry', 100, NULL, 1, 0, 100
WHERE NOT EXISTS (SELECT 1 FROM reward_prizes);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS reward_achievements;
DROP TABLE IF EXISTS reward_tasks;
DROP TABLE IF EXISTS reward_user_notifications;
DROP TABLE IF EXISTS reward_admin_messages;
DROP TABLE IF EXISTS reward_winners;
DROP TABLE IF EXISTS reward_orders;
DROP TABLE IF EXISTS reward_claims;
DROP TABLE IF EXISTS reward_spins;
DROP TABLE IF EXISTS reward_spin_ledger;
DROP TABLE IF EXISTS reward_spin_balances;
DROP TABLE IF EXISTS reward_prizes;
DROP TABLE IF EXISTS reward_config;

-- +goose StatementEnd
