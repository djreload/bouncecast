package admin

import "testing"

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
			name:        "invalid push",
			channel:     "push",
			destination: `{"keys":{"p256dh":"key"}}`,
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
