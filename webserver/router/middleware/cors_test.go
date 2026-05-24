package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetBounceCastCORSHeadersAllowsSameHost(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "http://k-nrg.co.uk/api/bouncecast/auth/login", nil)
	request.Host = "k-nrg.co.uk"
	request.Header.Set("Origin", "https://k-nrg.co.uk")
	recorder := httptest.NewRecorder()

	SetBounceCastCORSHeaders(recorder, request, "GET, POST, OPTIONS")

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://k-nrg.co.uk" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want same origin", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
	}
}

func TestSetBounceCastCORSHeadersAllowsForwardedHost(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "http://127.0.0.1:8080/api/bouncecast/auth/login", nil)
	request.Host = "127.0.0.1:8080"
	request.Header.Set("X-Forwarded-Host", "k-nrg.co.uk:443")
	request.Header.Set("Origin", "https://k-nrg.co.uk")
	recorder := httptest.NewRecorder()

	SetBounceCastCORSHeaders(recorder, request, "GET, POST, OPTIONS")

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://k-nrg.co.uk" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want forwarded host origin", got)
	}
}

func TestSetBounceCastCORSHeadersBlocksLocalhostOriginForLiveHost(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "http://k-nrg.co.uk/api/bouncecast/auth/login", nil)
	request.Host = "k-nrg.co.uk"
	request.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()

	SetBounceCastCORSHeaders(recorder, request, "GET, POST, OPTIONS")

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty for untrusted localhost origin", got)
	}
}

func TestSetBounceCastCORSHeadersAllowsLocalhostDevAgainstLocalBackend(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "http://127.0.0.1:8080/api/bouncecast/auth/login", nil)
	request.Host = "127.0.0.1:8080"
	request.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()

	SetBounceCastCORSHeaders(recorder, request, "GET, POST, OPTIONS")

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want localhost dev origin", got)
	}
}

func TestSetBounceCastCORSHeadersAllowsConfiguredOrigin(t *testing.T) {
	t.Setenv("BOUNCECAST_ALLOWED_ORIGINS", "https://app.example.com")
	request := httptest.NewRequest(http.MethodOptions, "http://k-nrg.co.uk/api/bouncecast/auth/login", nil)
	request.Host = "k-nrg.co.uk"
	request.Header.Set("Origin", "https://app.example.com")
	recorder := httptest.NewRecorder()

	SetBounceCastCORSHeaders(recorder, request, "GET, POST, OPTIONS")

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want configured origin", got)
	}
}
