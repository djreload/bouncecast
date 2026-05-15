package stars

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/owncast/owncast/models"
)

const (
	payPalSandboxBaseURL = "https://api-m.sandbox.paypal.com"
	payPalLiveBaseURL    = "https://api-m.paypal.com"
)

type PayPalClient struct {
	settings   models.StarSettings
	baseURL    string
	httpClient *http.Client
}

type PayPalCreateOrderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type PayPalCaptureResponse struct {
	ID            string
	Status        string
	CaptureID     string
	CaptureStatus string
	PayerID       string
	RawStatus     string
}

type payPalTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func NewPayPalClient(settings models.StarSettings) *PayPalClient {
	return newPayPalClient(settings, payPalBaseURL(settings.PayPalEnvironment), http.DefaultClient)
}

func newPayPalClient(settings models.StarSettings, baseURL string, httpClient *http.Client) *PayPalClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &PayPalClient{
		settings:   settings,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func payPalBaseURL(environment string) string {
	if strings.EqualFold(environment, "live") {
		return payPalLiveBaseURL
	}
	return payPalSandboxBaseURL
}

func (c *PayPalClient) CreateOrder(ctx context.Context, pkg models.StarPackage, requestID string) (PayPalCreateOrderResponse, error) {
	if err := c.validateCredentials(); err != nil {
		return PayPalCreateOrderResponse{}, err
	}
	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return PayPalCreateOrderResponse{}, err
	}

	payload := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"description": fmt.Sprintf("%s for BounceCast Stars", pkg.Name),
				"custom_id":   fmt.Sprintf("star_package:%d", pkg.ID),
				"amount": map[string]string{
					"currency_code": pkg.Currency,
					"value":         centsToDecimal(pkg.PriceCents),
				},
			},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v2/checkout/orders", bytes.NewReader(body))
	if err != nil {
		return PayPalCreateOrderResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	if requestID != "" {
		req.Header.Set("PayPal-Request-Id", requestID)
	}

	var response PayPalCreateOrderResponse
	if err := c.doJSON(req, &response); err != nil {
		return PayPalCreateOrderResponse{}, err
	}
	if response.ID == "" {
		return PayPalCreateOrderResponse{}, errors.New("PayPal did not return an order ID")
	}
	return response, nil
}

func (c *PayPalClient) CaptureOrder(ctx context.Context, orderID string, requestID string) (PayPalCaptureResponse, error) {
	if err := c.validateCredentials(); err != nil {
		return PayPalCaptureResponse{}, err
	}
	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return PayPalCaptureResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v2/checkout/orders/"+url.PathEscape(orderID)+"/capture", bytes.NewReader([]byte("{}")))
	if err != nil {
		return PayPalCaptureResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	if requestID != "" {
		req.Header.Set("PayPal-Request-Id", requestID)
	}

	var raw map[string]interface{}
	if err := c.doJSON(req, &raw); err != nil {
		return PayPalCaptureResponse{}, err
	}

	response := PayPalCaptureResponse{
		ID:        stringFromMap(raw, "id"),
		Status:    stringFromMap(raw, "status"),
		RawStatus: stringFromMap(raw, "status"),
	}
	response.PayerID = stringFromNested(raw, "payer", "payer_id")
	response.CaptureID, response.CaptureStatus = captureDetails(raw)
	if response.CaptureStatus != "" {
		response.RawStatus = response.CaptureStatus
	}
	return response, nil
}

func (c *PayPalClient) VerifyWebhookSignature(ctx context.Context, headers http.Header, body []byte) (bool, error) {
	if err := c.validateCredentials(); err != nil {
		return false, err
	}
	if strings.TrimSpace(c.settings.PayPalWebhookID) == "" {
		return false, errors.New("missing PayPal webhook ID")
	}

	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		return false, err
	}

	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return false, err
	}

	payload := map[string]interface{}{
		"auth_algo":         headers.Get("PAYPAL-AUTH-ALGO"),
		"cert_url":          headers.Get("PAYPAL-CERT-URL"),
		"transmission_id":   headers.Get("PAYPAL-TRANSMISSION-ID"),
		"transmission_sig":  headers.Get("PAYPAL-TRANSMISSION-SIG"),
		"transmission_time": headers.Get("PAYPAL-TRANSMISSION-TIME"),
		"webhook_id":        c.settings.PayPalWebhookID,
		"webhook_event":     event,
	}
	requestBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/notifications/verify-webhook-signature", bytes.NewReader(requestBody))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	var response struct {
		VerificationStatus string `json:"verification_status"`
	}
	if err := c.doJSON(req, &response); err != nil {
		return false, err
	}
	return response.VerificationStatus == "SUCCESS", nil
}

func (c *PayPalClient) validateCredentials() error {
	if strings.TrimSpace(c.settings.PayPalClientID) == "" || strings.TrimSpace(c.settings.PayPalClientSecret) == "" {
		return errors.New("PayPal client ID and secret are required")
	}
	return nil
}

func (c *PayPalClient) getAccessToken(ctx context.Context) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	basic := base64.StdEncoding.EncodeToString([]byte(c.settings.PayPalClientID + ":" + c.settings.PayPalClientSecret))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var response payPalTokenResponse
	if err := c.doJSON(req, &response); err != nil {
		return "", err
	}
	if response.AccessToken == "" {
		return "", errors.New("PayPal did not return an access token")
	}
	return response.AccessToken, nil
}

func (c *PayPalClient) doJSON(req *http.Request, target interface{}) error {
	response, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("PayPal request failed with %d: %s", response.StatusCode, safePayPalError(responseBody))
	}
	if len(responseBody) == 0 {
		return nil
	}
	return json.Unmarshal(responseBody, target)
}

func safePayPalError(body []byte) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "unreadable PayPal response"
	}
	delete(parsed, "debug_id")
	delete(parsed, "links")
	safe, _ := json.Marshal(parsed)
	return string(safe)
}

func centsToDecimal(cents int) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func stringFromMap(value map[string]interface{}, key string) string {
	raw, _ := value[key].(string)
	return raw
}

func stringFromNested(value map[string]interface{}, keys ...string) string {
	var current interface{} = value
	for _, key := range keys {
		asMap, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = asMap[key]
	}
	result, _ := current.(string)
	return result
}

func captureDetails(raw map[string]interface{}) (string, string) {
	units, _ := raw["purchase_units"].([]interface{})
	for _, unitValue := range units {
		unit, _ := unitValue.(map[string]interface{})
		payments, _ := unit["payments"].(map[string]interface{})
		captures, _ := payments["captures"].([]interface{})
		for _, captureValue := range captures {
			capture, _ := captureValue.(map[string]interface{})
			return stringFromMap(capture, "id"), stringFromMap(capture, "status")
		}
	}
	return "", ""
}
