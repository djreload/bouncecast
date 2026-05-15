-- +goose Up
-- +goose StatementBegin

-- Additive BounceCast chat reactions. This stores emoji reactions separately
-- from the existing chat message rows so message history and moderation stay
-- compatible with upstream Owncast behavior.

CREATE TABLE IF NOT EXISTS chat_message_reactions (
    "message_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "reaction" TEXT NOT NULL,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY ("message_id", "user_id", "reaction"),
    FOREIGN KEY("message_id") REFERENCES messages("id") ON DELETE CASCADE,
    FOREIGN KEY("user_id") REFERENCES users("id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chat_message_reactions_message_id
    ON chat_message_reactions ("message_id");

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS chat_message_reactions;

-- +goose StatementEnd
