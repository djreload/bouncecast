-- +goose Up
-- +goose StatementBegin

-- Enforce one selected reaction per user per chat message. Existing installs
-- may already have multiple reactions for the same user/message pair, so keep
-- the oldest row before adding the unique index.

DELETE FROM chat_message_reactions
WHERE rowid NOT IN (
    SELECT MIN(rowid)
    FROM chat_message_reactions
    GROUP BY message_id, user_id
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_message_reactions_message_user
    ON chat_message_reactions ("message_id", "user_id");

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_chat_message_reactions_message_user;

-- +goose StatementEnd
