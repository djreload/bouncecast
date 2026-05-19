package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/core/facebookmessenger"
	webutils "github.com/owncast/owncast/webserver/utils"
)

type facebookMessengerWebhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		Messaging []struct {
			Sender struct {
				ID string `json:"id"`
			} `json:"sender"`
			Message struct {
				Text string `json:"text"`
			} `json:"message"`
			Postback struct {
				Payload string `json:"payload"`
				Title   string `json:"title"`
			} `json:"postback"`
		} `json:"messaging"`
	} `json:"entry"`
}

// GetBounceCastMessengerAlertsPublicConfig exposes only the public CTA data
// needed by the stream/DJ pages.
func GetBounceCastMessengerAlertsPublicConfig(w http.ResponseWriter, r *http.Request) {
	setBounceCastPublicHeaders(w)
	webutils.WriteResponse(w, facebookmessenger.GetPublicConfig(data.GetDatabase()))
}

// FacebookMessengerWebhookVerify implements Meta's GET webhook verification
// challenge flow.
func FacebookMessengerWebhookVerify(w http.ResponseWriter, r *http.Request) {
	settings := facebookmessenger.ReadSettings(data.GetDatabase())
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token != "" && token == settings.WebhookVerifyToken {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(challenge))
		return
	}
	http.Error(w, "Facebook Messenger webhook verification failed", http.StatusForbidden)
}

// FacebookMessengerWebhook receives Messenger Page messages and records
// keyword-based opt-ins/opt-outs. It responds quickly and sends helper replies
// in a goroutine so Meta webhook retries are not held up by Graph API latency.
func FacebookMessengerWebhook(w http.ResponseWriter, r *http.Request) {
	if !enforceBounceCastRateLimit(w, r, bounceCastMessengerWebhookRateLimit, bounceCastRateLimitIPSubject(r)) {
		return
	}

	settings := facebookmessenger.ReadSettings(data.GetDatabase())
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if settings.ValidateAppSecret {
		if !facebookmessenger.VerifySignature(settings.AppSecret, body, r.Header.Get("X-Hub-Signature-256")) {
			http.Error(w, "invalid Facebook Messenger webhook signature", http.StatusForbidden)
			return
		}
	}

	var payload facebookMessengerWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	service := facebookmessenger.NewService(settings)
	for _, entry := range payload.Entry {
		for _, event := range entry.Messaging {
			psid := strings.TrimSpace(event.Sender.ID)
			text := strings.TrimSpace(event.Message.Text)
			if text == "" {
				text = strings.TrimSpace(event.Postback.Payload)
			}
			if psid == "" || text == "" {
				continue
			}
			_, response, err := facebookmessenger.HandleIncomingMessage(data.GetDatabase(), psid, text, "page_message")
			if err != nil || strings.TrimSpace(response) == "" {
				continue
			}
			go func(recipient string, message string) {
				if sendErr := service.SendText(recipient, message); sendErr != nil {
					// Webhook acknowledgement should not fail just because a helper
					// reply could not be sent.
					return
				}
			}(psid, response)
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("EVENT_RECEIVED"))
}
