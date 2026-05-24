-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS reward_chat_activity (
    "user_id" TEXT NOT NULL PRIMARY KEY,
    "message_count_window" INTEGER NOT NULL DEFAULT 0,
    "last_message_body_hash" TEXT,
    "last_message_at" TIMESTAMP,
    "last_awarded_at" TIMESTAMP,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS reward_task_completions (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" TEXT NOT NULL,
    "task_id" INTEGER NOT NULL,
    "ledger_id" INTEGER,
    "status" TEXT NOT NULL DEFAULT 'completed',
    "completed_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE,
    FOREIGN KEY("task_id") REFERENCES reward_tasks("id") ON DELETE CASCADE,
    FOREIGN KEY("ledger_id") REFERENCES reward_spin_ledger("id") ON DELETE SET NULL,
    UNIQUE("user_id", "task_id")
);

CREATE INDEX IF NOT EXISTS idx_reward_task_completions_user
    ON reward_task_completions ("user_id", "completed_at");

CREATE TABLE IF NOT EXISTS reward_achievement_unlocks (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" TEXT NOT NULL,
    "achievement_id" INTEGER NOT NULL,
    "condition_key" TEXT NOT NULL,
    "reference_id" TEXT,
    "ledger_id" INTEGER,
    "unlocked_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE,
    FOREIGN KEY("achievement_id") REFERENCES reward_achievements("id") ON DELETE CASCADE,
    FOREIGN KEY("ledger_id") REFERENCES reward_spin_ledger("id") ON DELETE SET NULL,
    UNIQUE("user_id", "achievement_id")
);

CREATE INDEX IF NOT EXISTS idx_reward_achievement_unlocks_user
    ON reward_achievement_unlocks ("user_id", "unlocked_at");

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS reward_achievement_unlocks;
DROP TABLE IF EXISTS reward_task_completions;
DROP TABLE IF EXISTS reward_chat_activity;

-- +goose StatementEnd
