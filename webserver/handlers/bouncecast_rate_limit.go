package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/owncast/owncast/utils"
	webutils "github.com/owncast/owncast/webserver/utils"
)

type bounceCastRateLimitRule struct {
	Name        string
	MaxRequests int
	Window      time.Duration
	Message     string
}

type bounceCastRateLimitBucket struct {
	Count   int
	ResetAt time.Time
}

var (
	bounceCastRateLimitLock    sync.Mutex
	bounceCastRateLimitBuckets = map[string]bounceCastRateLimitBucket{}

	bounceCastAccountRegisterIPRateLimit    = bounceCastRateLimitRule{Name: "account-register-ip", MaxRequests: 8, Window: 15 * time.Minute, Message: "Too many registration attempts. Please wait before trying again."}
	bounceCastAccountRegisterEmailRateLimit = bounceCastRateLimitRule{Name: "account-register-email", MaxRequests: 4, Window: time.Hour, Message: "Too many registration attempts for this email. Please wait before trying again."}
	bounceCastAccountLoginIPRateLimit       = bounceCastRateLimitRule{Name: "account-login-ip", MaxRequests: 20, Window: 10 * time.Minute, Message: "Too many login attempts. Please wait before trying again."}
	bounceCastAccountLoginEmailRateLimit    = bounceCastRateLimitRule{Name: "account-login-email", MaxRequests: 8, Window: 10 * time.Minute, Message: "Too many login attempts for this account. Please wait before trying again."}
	bounceCastAccountProfileRateLimit       = bounceCastRateLimitRule{Name: "account-profile", MaxRequests: 20, Window: time.Hour, Message: "Too many profile updates. Please wait before trying again."}
	bounceCastAccountProfileImageRateLimit  = bounceCastRateLimitRule{Name: "account-profile-image", MaxRequests: 10, Window: time.Hour, Message: "Too many profile image uploads. Please wait before trying again."}
	bounceCastStarsCheckoutRateLimit        = bounceCastRateLimitRule{Name: "stars-checkout", MaxRequests: 12, Window: 10 * time.Minute, Message: "Too many Stars checkout attempts. Please wait before trying again."}
	bounceCastStarsCaptureRateLimit         = bounceCastRateLimitRule{Name: "stars-capture", MaxRequests: 20, Window: 10 * time.Minute, Message: "Too many Stars payment confirmations. Please wait before trying again."}
	bounceCastStarsSendRateLimit            = bounceCastRateLimitRule{Name: "stars-send", MaxRequests: 10, Window: time.Minute, Message: "Too many Stars sends. Please wait before trying again."}
	bounceCastMessengerWebhookRateLimit     = bounceCastRateLimitRule{Name: "messenger-webhook", MaxRequests: 120, Window: time.Minute, Message: "Too many Messenger webhook requests. Please wait before trying again."}
)

func enforceBounceCastRateLimit(w http.ResponseWriter, r *http.Request, rule bounceCastRateLimitRule, subjects ...string) bool {
	now := time.Now().UTC()
	if len(subjects) == 0 {
		subjects = []string{bounceCastRateLimitIPSubject(r)}
	}

	for _, subject := range subjects {
		subject = strings.TrimSpace(subject)
		if subject == "" {
			continue
		}
		allowed, retryAfter := takeBounceCastRateLimit(rule, subject, now)
		if !allowed {
			writeBounceCastRateLimitExceeded(w, rule.Message, retryAfter)
			return false
		}
	}
	return true
}

func takeBounceCastRateLimit(rule bounceCastRateLimitRule, subject string, now time.Time) (bool, time.Duration) {
	bounceCastRateLimitLock.Lock()
	defer bounceCastRateLimitLock.Unlock()

	for key, bucket := range bounceCastRateLimitBuckets {
		if !bucket.ResetAt.After(now) {
			delete(bounceCastRateLimitBuckets, key)
		}
	}

	key := fmt.Sprintf("%s:%s", rule.Name, subject)
	bucket := bounceCastRateLimitBuckets[key]
	if bucket.ResetAt.IsZero() || !bucket.ResetAt.After(now) {
		bounceCastRateLimitBuckets[key] = bounceCastRateLimitBucket{Count: 1, ResetAt: now.Add(rule.Window)}
		return true, 0
	}

	if bucket.Count >= rule.MaxRequests {
		return false, time.Until(bucket.ResetAt)
	}

	bucket.Count++
	bounceCastRateLimitBuckets[key] = bucket
	return true, 0
}

func bounceCastRateLimitIPSubject(r *http.Request) string {
	ipAddress := strings.TrimSpace(utils.GetIPAddressFromRequest(r))
	if ipAddress == "" {
		ipAddress = strings.TrimSpace(r.RemoteAddr)
	}
	if ipAddress == "" {
		return "ip:unknown"
	}
	return "ip:" + ipAddress
}

func bounceCastRateLimitUserSubject(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}
	return "user:" + userID
}

func bounceCastRateLimitEmailSubject(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return ""
	}
	return "email:" + email
}

func writeBounceCastRateLimitExceeded(w http.ResponseWriter, message string, retryAfter time.Duration) {
	if retryAfter < time.Second {
		retryAfter = time.Second
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(webutils.J{
		"success": false,
		"error":   "rate_limited",
		"message": message,
	})
}

func resetBounceCastRateLimitersForTesting() {
	bounceCastRateLimitLock.Lock()
	defer bounceCastRateLimitLock.Unlock()
	bounceCastRateLimitBuckets = map[string]bounceCastRateLimitBucket{}
}
