package middleware

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

// EnableCors enables the CORS header on the responses.
func EnableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

// SetBounceCastCORSHeaders applies restrictive CORS headers for BounceCast's
// browser-facing account, Studio, and admin session APIs. Production requests
// should be same-origin; localhost dev is allowed only for localhost backends.
// Extra trusted origins may be supplied through BOUNCECAST_ALLOWED_ORIGINS as a
// comma-separated list, for reverse proxies or split frontend deployments.
func SetBounceCastCORSHeaders(w http.ResponseWriter, r *http.Request, methods string) {
	w.Header().Add("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", methods)

	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" || !isBounceCastCORSOriginAllowed(r, origin) {
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

func isBounceCastCORSOriginAllowed(r *http.Request, origin string) bool {
	parsedOrigin, err := url.Parse(origin)
	if err != nil || parsedOrigin.Scheme == "" || parsedOrigin.Host == "" {
		return false
	}

	if hostMatchesOrigin(parsedOrigin, r.Host) || hostMatchesOrigin(parsedOrigin, r.Header.Get("X-Forwarded-Host")) {
		return true
	}

	if isLocalHost(parsedOrigin.Hostname()) && isLocalHost(requestHostName(r)) {
		return true
	}

	for _, allowedOrigin := range strings.Split(os.Getenv("BOUNCECAST_ALLOWED_ORIGINS"), ",") {
		allowedOrigin = strings.TrimSpace(allowedOrigin)
		if allowedOrigin == "" {
			continue
		}
		parsedAllowedOrigin, err := url.Parse(allowedOrigin)
		if err != nil || parsedAllowedOrigin.Scheme == "" || parsedAllowedOrigin.Host == "" {
			continue
		}
		if strings.EqualFold(strings.TrimRight(allowedOrigin, "/"), strings.TrimRight(origin, "/")) {
			return true
		}
	}

	return false
}

func hostMatchesOrigin(origin *url.URL, requestHost string) bool {
	requestHost = strings.TrimSpace(requestHost)
	if requestHost == "" {
		return false
	}
	parsedRequestHost, err := url.Parse("//" + requestHost)
	if err != nil || parsedRequestHost.Hostname() == "" {
		return strings.EqualFold(origin.Host, requestHost)
	}
	if !strings.EqualFold(origin.Hostname(), parsedRequestHost.Hostname()) {
		return false
	}
	originPort := origin.Port()
	requestPort := parsedRequestHost.Port()
	if originPort == requestPort {
		return true
	}
	if originPort == "" && isDefaultPortForScheme(requestPort, origin.Scheme) {
		return true
	}
	return requestPort == "" && isDefaultPortForScheme(originPort, origin.Scheme)
}

func isDefaultPortForScheme(port string, scheme string) bool {
	return (scheme == "https" && port == "443") || (scheme == "http" && port == "80")
}

func requestHostName(r *http.Request) string {
	host := strings.TrimSpace(r.Host)
	if parsedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); parsedHost != "" {
		host = parsedHost
	}
	if host == "" {
		return ""
	}
	parsedURL, err := url.Parse("//" + host)
	if err != nil || parsedURL.Hostname() == "" {
		return host
	}
	return parsedURL.Hostname()
}

func isLocalHost(host string) bool {
	host = strings.ToLower(strings.Trim(host, "[] "))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
