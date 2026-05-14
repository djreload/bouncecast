package admin

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/utils"
)

func TestNormalizeBounceCastSubscriberDestination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		channel     string
		destination string
		want        string
		wantErr     bool
	}{
		{
			name:        "email",
			channel:     "email",
			destination: "DJ Alerts <alerts@example.com>",
			want:        "alerts@example.com",
		},
		{
			name:        "webhook",
			channel:     "webhook",
			destination: "https://example.com/hooks/live",
			want:        "https://example.com/hooks/live",
		},
		{
			name:        "push",
			channel:     "push",
			destination: `{"endpoint":"https://push.example.com/subscription","keys":{"p256dh":"key","auth":"auth"}}`,
			want:        `{"endpoint":"https://push.example.com/subscription","keys":{"p256dh":"key","auth":"auth"}}`,
		},
		{
			name:        "invalid email",
			channel:     "email",
			destination: "not-an-email",
			wantErr:     true,
		},
		{
			name:        "invalid webhook scheme",
			channel:     "webhook",
			destination: "ftp://example.com/live",
			wantErr:     true,
		},
		{
			name:        "invalid push endpoint",
			channel:     "push",
			destination: `{"keys":{"p256dh":"key"}}`,
			wantErr:     true,
		},
		{
			name:        "invalid push keys",
			channel:     "push",
			destination: `{"endpoint":"https://push.example.com/subscription","keys":{"p256dh":"key"}}`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeBounceCastSubscriberDestination(tt.channel, tt.destination)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateBounceCastEmailHeaderRejectsLineBreaks(t *testing.T) {
	t.Parallel()

	if err := validateBounceCastEmailHeader("BounceCast", "fromName"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateBounceCastEmailHeader("BounceCast\r\nBcc: test@example.com", "subject"); err == nil {
		t.Fatalf("expected line break error")
	}
}

func TestNormalizeBounceCastStreamerRoleAndStatus(t *testing.T) {
	t.Parallel()

	role, err := normalizeBounceCastStreamerRole("")
	if err != nil {
		t.Fatalf("unexpected role error: %v", err)
	}
	if role != "streamer" {
		t.Fatalf("role = %q, want streamer", role)
	}

	status, err := normalizeBounceCastStreamerStatus("")
	if err != nil {
		t.Fatalf("unexpected status error: %v", err)
	}
	if status != "active" {
		t.Fatalf("status = %q, want active", status)
	}

	if _, err := normalizeBounceCastStreamerRole("super-admin"); err == nil {
		t.Fatal("expected invalid role error")
	}
	if _, err := normalizeBounceCastStreamerStatus("deleted"); err == nil {
		t.Fatal("expected invalid status error")
	}
}

func TestValidateBounceCastStreamerPassword(t *testing.T) {
	t.Parallel()

	if err := validateBounceCastStreamerPassword("long-enough"); err != nil {
		t.Fatalf("unexpected password error: %v", err)
	}
	if err := validateBounceCastStreamerPassword("short"); err == nil {
		t.Fatal("expected short password error")
	}
	if err := validateBounceCastStreamerPassword("long-enough\r\n"); err == nil {
		t.Fatal("expected line break password error")
	}
}

func TestSetBounceCastStreamerPasswordStoresHash(t *testing.T) {
	db := data.GetDatabase()
	handle := "password-test-dj"
	_, _ = db.Exec(`DELETE FROM bouncecast_streamer_accounts WHERE handle = ?`, handle)

	result, err := db.Exec(`
		INSERT INTO bouncecast_streamer_accounts(display_name, handle, role, status)
		VALUES('Password Test DJ', ?, 'streamer', 'invited')
	`, handle)
	if err != nil {
		t.Fatalf("insert streamer: %v", err)
	}
	streamerID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read streamer id: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM bouncecast_streamer_accounts WHERE id = ?`, streamerID)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/admin/bouncecast/streamers/password", strings.NewReader(`{"id":`+strconv.FormatInt(streamerID, 10)+`,"password":"new-password"}`))
	recorder := httptest.NewRecorder()

	SetBounceCastStreamerPassword(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	var passwordHash string
	var status string
	if err := db.QueryRow(`SELECT password_hash, status FROM bouncecast_streamer_accounts WHERE id = ?`, streamerID).Scan(&passwordHash, &status); err != nil {
		t.Fatalf("read streamer password: %v", err)
	}
	if passwordHash == "" {
		t.Fatal("expected password hash")
	}
	if err := utils.CompareHash(passwordHash, "new-password"); err != nil {
		t.Fatalf("stored password hash did not match: %v", err)
	}
	if status != "active" {
		t.Fatalf("status = %q, want active after password set", status)
	}
}
