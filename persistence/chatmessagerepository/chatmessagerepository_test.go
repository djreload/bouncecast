package chatmessagerepository

import (
	"database/sql"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/owncast/owncast/core/chat/events"
	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/persistence/migrations"
)

func newChatMessageTestRepository(t *testing.T) (*SqlChatMessageRepository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migrations.Run(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO users(id, display_name, display_color) VALUES('user-1', 'Kevin', 1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO messages(id, user_id, body, eventType, timestamp) VALUES('message-1', 'user-1', 'Hello', ?, CURRENT_TIMESTAMP)", events.MessageSent); err != nil {
		t.Fatal(err)
	}
	return New(&data.Datastore{DB: db, DbLock: &sync.Mutex{}}).(*SqlChatMessageRepository), db
}

func TestToggleMessageReactionAllowsOneReactionPerUserPerMessage(t *testing.T) {
	repository, db := newChatMessageTestRepository(t)

	counts, err := repository.ToggleMessageReaction("message-1", "user-1", "🔥")
	if err != nil {
		t.Fatal(err)
	}
	if counts["🔥"] != 1 {
		t.Fatalf("expected fire reaction count, got %+v", counts)
	}

	counts, err = repository.ToggleMessageReaction("message-1", "user-1", "❤️")
	if err != nil {
		t.Fatal(err)
	}
	if counts["🔥"] != 0 || counts["❤️"] != 1 {
		t.Fatalf("expected reaction to switch, got %+v", counts)
	}

	var storedCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM chat_message_reactions WHERE message_id = ? AND user_id = ?", "message-1", "user-1").Scan(&storedCount); err != nil {
		t.Fatal(err)
	}
	if storedCount != 1 {
		t.Fatalf("expected exactly one stored reaction, got %d", storedCount)
	}

	counts, err = repository.ToggleMessageReaction("message-1", "user-1", "❤️")
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 0 {
		t.Fatalf("expected selecting the same reaction to clear it, got %+v", counts)
	}
}
