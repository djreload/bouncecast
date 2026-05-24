package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

// SetHeaders will set our global headers for web resources.
func SetHeaders(w http.ResponseWriter, nonce string) {
	// Content security policy
	csp := []string{
		fmt.Sprintf("script-src '%s' 'self' https://www.paypal.com https://www.sandbox.paypal.com https://www.paypalobjects.com", nonce),
		"connect-src 'self' https://*.paypal.com https://*.paypalobjects.com",
		"frame-src 'self' https://*.paypal.com",
		"img-src 'self' data: blob: http: https:",
		"object-src 'none'",
		"base-uri 'self'",
		"worker-src 'self' blob:", // No single quotes around blob:
	}
	w.Header().Set("Content-Security-Policy", strings.Join(csp, "; "))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}
