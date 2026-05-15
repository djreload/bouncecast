package stars

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/owncast/owncast/core/chat"
	"github.com/owncast/owncast/core/chat/events"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/starsrepository"
)

const (
	MaxStarMessageLength = 120

	EffectSparkle   = "sparkle"
	EffectFireworks = "fireworks"
	EffectHearts    = "hearts"
	EffectHype      = "hype"
	EffectDJDrop    = "dj_drop"
)

var allowedEffects = map[string]struct{}{
	EffectSparkle:   {},
	EffectFireworks: {},
	EffectHearts:    {},
	EffectHype:      {},
	EffectDJDrop:    {},
}

type Service struct {
	repository starsrepository.Repository
	payPal     func(models.StarSettings) payPalProcessor
}

type payPalProcessor interface {
	CreateOrder(ctx context.Context, pkg models.StarPackage, requestID string) (PayPalCreateOrderResponse, error)
	CaptureOrder(ctx context.Context, orderID string, requestID string) (PayPalCaptureResponse, error)
	VerifyWebhookSignature(ctx context.Context, headers http.Header, body []byte) (bool, error)
}

type CreateOrderResult struct {
	PayPalOrderID string `json:"paypalOrderId"`
	ClientID      string `json:"clientId"`
	Environment   string `json:"environment"`
}

type CaptureOrderResult struct {
	Order       models.StarPayPalOrder `json:"order"`
	Credited    bool                   `json:"credited"`
	Wallet      models.StarWallet      `json:"wallet"`
	RawPayPalID string                 `json:"rawPayPalId,omitempty"`
}

type WebhookResult struct {
	EventID    string `json:"eventId"`
	EventType  string `json:"eventType"`
	ResourceID string `json:"resourceId,omitempty"`
	Processed  bool   `json:"processed"`
	Message    string `json:"message,omitempty"`
}

func NewService(repository starsrepository.Repository) *Service {
	return &Service{
		repository: repository,
		payPal: func(settings models.StarSettings) payPalProcessor {
			return NewPayPalClient(settings)
		},
	}
}

func GetService() *Service {
	return NewService(starsrepository.Get())
}

func PublicSettings(settings models.StarSettings, packages []models.StarPackage) models.PublicStarSettings {
	return models.PublicStarSettings{
		Enabled:               settings.Enabled,
		PayPalEnvironment:     settings.PayPalEnvironment,
		PayPalClientID:        settings.PayPalClientID,
		Currency:              settings.Currency,
		SupportMessage:        settings.SupportMessage,
		MinimumSendAmount:     settings.MinimumSendAmount,
		MaximumSendAmount:     settings.MaximumSendAmount,
		SendCooldownSeconds:   settings.SendCooldownSeconds,
		OverlayEffectsEnabled: settings.OverlayEffectsEnabled,
		SoundEffectsEnabled:   settings.SoundEffectsEnabled,
		Packages:              packages,
	}
}

func (s *Service) GetPublicConfig() (models.PublicStarSettings, error) {
	settings, err := s.repository.GetSettings()
	if err != nil {
		return models.PublicStarSettings{}, err
	}
	packages, err := s.repository.ListPackages(false)
	if err != nil {
		return models.PublicStarSettings{}, err
	}
	return PublicSettings(settings, packages), nil
}

func (s *Service) CreatePayPalOrder(ctx context.Context, userID string, packageID int64) (CreateOrderResult, error) {
	settings, err := s.repository.GetSettings()
	if err != nil {
		return CreateOrderResult{}, err
	}
	if !settings.Enabled {
		return CreateOrderResult{}, errors.New("Stars are disabled")
	}

	pkg, err := s.repository.GetPackage(packageID)
	if err != nil {
		return CreateOrderResult{}, err
	}
	if !pkg.Enabled {
		return CreateOrderResult{}, errors.New("star package is disabled")
	}

	requestID := fmt.Sprintf("stars-%s-%d-%d", userID, packageID, time.Now().UnixNano())
	order, err := s.payPal(settings).CreateOrder(ctx, pkg, requestID)
	if err != nil {
		return CreateOrderResult{}, err
	}
	if _, err := s.repository.CreatePayPalOrder(userID, pkg, order.ID); err != nil {
		return CreateOrderResult{}, err
	}

	return CreateOrderResult{
		PayPalOrderID: order.ID,
		ClientID:      settings.PayPalClientID,
		Environment:   settings.PayPalEnvironment,
	}, nil
}

func (s *Service) CapturePayPalOrder(ctx context.Context, userID string, paypalOrderID string) (CaptureOrderResult, error) {
	settings, err := s.repository.GetSettings()
	if err != nil {
		return CaptureOrderResult{}, err
	}
	if !settings.Enabled {
		return CaptureOrderResult{}, errors.New("Stars are disabled")
	}

	requestID := fmt.Sprintf("stars-capture-%s-%d", paypalOrderID, time.Now().UnixNano())
	capture, err := s.payPal(settings).CaptureOrder(ctx, paypalOrderID, requestID)
	if err != nil {
		_ = s.repository.FailPayPalOrder(paypalOrderID, err.Error())
		return CaptureOrderResult{}, err
	}
	if !strings.EqualFold(capture.CaptureStatus, "COMPLETED") && !strings.EqualFold(capture.Status, "COMPLETED") {
		_ = s.repository.FailPayPalOrder(paypalOrderID, capture.RawStatus)
		return CaptureOrderResult{}, errors.New("PayPal capture was not completed")
	}

	order, credited, err := s.repository.CompletePayPalOrder(paypalOrderID, capture.CaptureID, capture.PayerID, capture.RawStatus)
	if err != nil {
		return CaptureOrderResult{}, err
	}
	if order.UserID != userID {
		return CaptureOrderResult{}, errors.New("PayPal order belongs to a different user")
	}
	wallet, err := s.repository.GetWalletSummary(userID)
	if err != nil {
		return CaptureOrderResult{}, err
	}
	return CaptureOrderResult{Order: order, Credited: credited, Wallet: wallet.Wallet, RawPayPalID: capture.ID}, nil
}

func (s *Service) ProcessPayPalWebhook(ctx context.Context, headers http.Header, body []byte) (WebhookResult, error) {
	settings, err := s.repository.GetSettings()
	if err != nil {
		return WebhookResult{}, err
	}
	verified, err := s.payPal(settings).VerifyWebhookSignature(ctx, headers, body)
	if err != nil {
		return WebhookResult{}, err
	}
	if !verified {
		return WebhookResult{}, errors.New("PayPal webhook signature verification failed")
	}

	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		return WebhookResult{}, err
	}
	eventID := stringFromMap(event, "id")
	eventType := stringFromMap(event, "event_type")
	resource, _ := event["resource"].(map[string]interface{})
	resourceID := stringFromMap(resource, "id")
	result := WebhookResult{EventID: eventID, EventType: eventType, ResourceID: resourceID}

	switch eventType {
	case "CHECKOUT.ORDER.APPROVED":
		result.Processed = true
		result.Message = "order approval noted"
	case "PAYMENT.CAPTURE.COMPLETED":
		orderID := stringFromNested(resource, "supplementary_data", "related_ids", "order_id")
		if orderID == "" {
			orderID = stringFromMap(resource, "invoice_id")
		}
		if orderID == "" {
			_ = s.repository.RecordWebhookEvent(eventID, eventType, resourceID, "ignored", "missing related order id")
			return result, errors.New("completed capture webhook missing related order id")
		}
		_, credited, err := s.repository.CompletePayPalOrder(orderID, resourceID, stringFromNested(resource, "payer", "payer_id"), "WEBHOOK_COMPLETED")
		if err != nil {
			_ = s.repository.RecordWebhookEvent(eventID, eventType, resourceID, "error", err.Error())
			return result, err
		}
		result.Processed = true
		if credited {
			result.Message = "wallet credited"
		} else {
			result.Message = "already credited"
		}
	case "PAYMENT.CAPTURE.DENIED", "PAYMENT.CAPTURE.DECLINED":
		orderID := stringFromNested(resource, "supplementary_data", "related_ids", "order_id")
		if orderID != "" {
			_ = s.repository.FailPayPalOrder(orderID, eventType)
		}
		result.Processed = true
		result.Message = "payment marked failed"
	case "PAYMENT.CAPTURE.REFUNDED", "PAYMENT.CAPTURE.REVERSED":
		if err := s.repository.RefundPayPalCapture(resourceID, eventType); err != nil {
			_ = s.repository.RecordWebhookEvent(eventID, eventType, resourceID, "error", err.Error())
			return result, err
		}
		result.Processed = true
		result.Message = "refund recorded"
	default:
		result.Message = "event ignored"
	}

	if err := s.repository.RecordWebhookEvent(eventID, eventType, resourceID, "processed", ""); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) SendStars(user models.User, amount int, message string, effect string) (models.StarSendEvent, error) {
	settings, err := s.repository.GetSettings()
	if err != nil {
		return models.StarSendEvent{}, err
	}
	if !settings.Enabled {
		return models.StarSendEvent{}, errors.New("Stars are disabled")
	}
	if amount < settings.MinimumSendAmount || amount > settings.MaximumSendAmount {
		return models.StarSendEvent{}, fmt.Errorf("send amount must be between %d and %d", settings.MinimumSendAmount, settings.MaximumSendAmount)
	}
	if lastSend, err := s.repository.GetLastSendTime(user.ID); err != nil {
		return models.StarSendEvent{}, err
	} else if lastSend != nil && settings.SendCooldownSeconds > 0 && time.Since(*lastSend) < time.Duration(settings.SendCooldownSeconds)*time.Second {
		return models.StarSendEvent{}, errors.New("please wait before sending Stars again")
	}

	message = sanitizeStarMessage(message)
	effect = normalizeEffect(effect)
	sendEvent, err := s.repository.SendStars(user.ID, user.DisplayName, amount, message, effect)
	if err != nil {
		return models.StarSendEvent{}, err
	}

	chatLine := fmt.Sprintf("%s sent %d Stars", user.DisplayName, amount)
	if message != "" {
		chatLine += " - " + message
	}
	_ = chat.SendSystemAction(chatLine, false)

	if settings.OverlayEffectsEnabled {
		overlayEvent := events.StarsSentEvent{
			Event:        events.Event{},
			DisplayName:  user.DisplayName,
			Amount:       amount,
			Message:      message,
			Effect:       effect,
			SoundEnabled: settings.SoundEffectsEnabled,
		}
		overlayEvent.SetDefaults()
		_ = chat.Broadcast(&overlayEvent)
	}

	return sendEvent, nil
}

func sanitizeStarMessage(message string) string {
	message = strings.TrimSpace(message)
	if len([]rune(message)) > MaxStarMessageLength {
		message = string([]rune(message)[:MaxStarMessageLength])
	}
	return html.EscapeString(message)
}

func normalizeEffect(effect string) string {
	effect = strings.TrimSpace(effect)
	if _, ok := allowedEffects[effect]; ok {
		return effect
	}
	return EffectSparkle
}
