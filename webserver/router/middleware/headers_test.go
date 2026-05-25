package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetHeadersIncludesPayPalAndHardeningHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()

	SetHeaders(recorder, "nonce-test")

	csp := recorder.Header().Get("Content-Security-Policy")
	for _, expected := range []string{
		"script-src 'nonce-test' 'self' https://www.paypal.com",
		"connect-src 'self' https://*.paypal.com",
		"https://tenor.googleapis.com",
		"frame-src 'self' https://*.paypal.com",
		"object-src 'none'",
		"base-uri 'self'",
	} {
		if !strings.Contains(csp, expected) {
			t.Fatalf("CSP %q missing %q", csp, expected)
		}
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := recorder.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("Referrer-Policy = %q, want strict-origin-when-cross-origin", got)
	}
}
