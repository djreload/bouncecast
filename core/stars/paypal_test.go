package stars

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/owncast/owncast/models"
)

func TestPayPalCreateOrderUsesServerPackageValues(t *testing.T) {
	var createPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/oauth2/token":
			if r.Header.Get("Authorization") == "" {
				t.Fatal("missing basic auth")
			}
			_, _ = w.Write([]byte(`{"access_token":"token","token_type":"Bearer"}`))
		case "/v2/checkout/orders":
			if r.Header.Get("Authorization") != "Bearer token" {
				t.Fatal("missing bearer token")
			}
			if err := json.NewDecoder(r.Body).Decode(&createPayload); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"id":"ORDER-123","status":"CREATED"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newPayPalClient(models.StarSettings{PayPalClientID: "client", PayPalClientSecret: "secret"}, server.URL, server.Client())
	response, err := client.CreateOrder(context.Background(), models.StarPackage{ID: 7, Name: "1200 Stars", StarAmount: 1200, PriceCents: 1000, Currency: "GBP"}, "request-1")
	if err != nil {
		t.Fatal(err)
	}
	if response.ID != "ORDER-123" {
		t.Fatalf("unexpected order response: %+v", response)
	}
	payload, _ := json.Marshal(createPayload)
	if !strings.Contains(string(payload), `"value":"10.00"`) || !strings.Contains(string(payload), `"currency_code":"GBP"`) {
		t.Fatalf("server package values not used in PayPal payload: %s", payload)
	}
}

func TestPayPalCaptureParsesCaptureDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/oauth2/token":
			_, _ = w.Write([]byte(`{"access_token":"token","token_type":"Bearer"}`))
		case "/v2/checkout/orders/ORDER-123/capture":
			_, _ = w.Write([]byte(`{"id":"ORDER-123","status":"COMPLETED","payer":{"payer_id":"PAYER-1"},"purchase_units":[{"payments":{"captures":[{"id":"CAPTURE-1","status":"COMPLETED"}]}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newPayPalClient(models.StarSettings{PayPalClientID: "client", PayPalClientSecret: "secret"}, server.URL, server.Client())
	response, err := client.CaptureOrder(context.Background(), "ORDER-123", "request-2")
	if err != nil {
		t.Fatal(err)
	}
	if response.CaptureID != "CAPTURE-1" || response.PayerID != "PAYER-1" || response.CaptureStatus != "COMPLETED" {
		t.Fatalf("unexpected capture response: %+v", response)
	}
}

func TestPayPalWebhookSignatureVerificationPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/oauth2/token":
			_, _ = w.Write([]byte(`{"access_token":"token","token_type":"Bearer"}`))
		case "/v1/notifications/verify-webhook-signature":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload["webhook_id"] != "WEBHOOK-1" {
				t.Fatalf("unexpected webhook id: %+v", payload)
			}
			_, _ = w.Write([]byte(`{"verification_status":"SUCCESS"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	headers := http.Header{}
	headers.Set("PAYPAL-AUTH-ALGO", "SHA256withRSA")
	headers.Set("PAYPAL-CERT-URL", "https://example.com/cert")
	headers.Set("PAYPAL-TRANSMISSION-ID", "abc")
	headers.Set("PAYPAL-TRANSMISSION-SIG", "sig")
	headers.Set("PAYPAL-TRANSMISSION-TIME", "now")

	client := newPayPalClient(models.StarSettings{PayPalClientID: "client", PayPalClientSecret: "secret", PayPalWebhookID: "WEBHOOK-1"}, server.URL, server.Client())
	verified, err := client.VerifyWebhookSignature(context.Background(), headers, []byte(`{"id":"EVT-1","event_type":"CHECKOUT.ORDER.APPROVED"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Fatal("expected webhook verification to succeed")
	}
}
