package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBounceCastRateLimiterAllowsUntilLimit(t *testing.T) {
	resetBounceCastRateLimitersForTesting()

	rule := bounceCastRateLimitRule{
		Name:        "test-limit",
		MaxRequests: 2,
		Window:      time.Hour,
		Message:     "slow down",
	}
	request := httptest.NewRequest(http.MethodPost, "/limited", nil)
	request.RemoteAddr = "198.51.100.10:1234"

	for attempt := 0; attempt < 2; attempt++ {
		recorder := httptest.NewRecorder()
		if !enforceBounceCastRateLimit(recorder, request, rule, bounceCastRateLimitIPSubject(request)) {
			t.Fatalf("attempt %d was unexpectedly rate limited", attempt+1)
		}
		if recorder.Code != http.StatusOK {
			t.Fatalf("attempt %d wrote status %d", attempt+1, recorder.Code)
		}
	}

	recorder := httptest.NewRecorder()
	if enforceBounceCastRateLimit(recorder, request, rule, bounceCastRateLimitIPSubject(request)) {
		t.Fatal("third attempt was allowed, want rate limited")
	}
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", recorder.Code)
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestBounceCastRateLimiterSeparatesSubjects(t *testing.T) {
	resetBounceCastRateLimitersForTesting()

	rule := bounceCastRateLimitRule{
		Name:        "test-subjects",
		MaxRequests: 1,
		Window:      time.Hour,
		Message:     "slow down",
	}
	request := httptest.NewRequest(http.MethodPost, "/limited", nil)

	if !enforceBounceCastRateLimit(httptest.NewRecorder(), request, rule, "user:first") {
		t.Fatal("first subject should be allowed")
	}
	if !enforceBounceCastRateLimit(httptest.NewRecorder(), request, rule, "user:second") {
		t.Fatal("second subject should be allowed independently")
	}

	recorder := httptest.NewRecorder()
	if enforceBounceCastRateLimit(recorder, request, rule, "user:first") {
		t.Fatal("first subject second request should be rate limited")
	}
}
