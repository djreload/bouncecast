-- +goose Up
ALTER TABLE bouncecast_streamer_stream_keys ADD COLUMN raw_key TEXT;

-- +goose Down
-- SQLite cannot drop columns without rebuilding the table. Keeping this
-- additive column is safer than rewriting stored stream key state on rollback.
