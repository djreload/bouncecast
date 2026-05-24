package rewards

import (
	"fmt"
	"html"
	"strings"

	"github.com/owncast/owncast/core/chat"
	"github.com/owncast/owncast/core/chat/events"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/rewardsrepository"
)

const (
	EffectSparkle   = "sparkle"
	EffectFireworks = "fireworks"
	EffectHearts    = "hearts"
	EffectHype      = "hype"
	EffectDJDrop    = "dj_drop"
)

type Service struct {
	repository rewardsrepository.Repository
}

func NewService(repository rewardsrepository.Repository) *Service {
	return &Service{repository: repository}
}

func GetService() *Service {
	return NewService(rewardsrepository.Get())
}

func (s *Service) GetWheelData(userID string) (models.RewardWheelData, error) {
	return s.repository.GetWheelData(userID)
}

func (s *Service) AwardSpinCredits(userID string, amount int, source string, referenceID string, note string) (models.RewardSpinBalance, error) {
	return s.repository.AwardSpinCredits(userID, amount, source, referenceID, note)
}

func (s *Service) SpinWheel(user models.User) (models.RewardSpinResult, error) {
	result, err := s.repository.SpinWheel(user)
	if err != nil {
		return models.RewardSpinResult{}, err
	}

	if result.Winner != nil {
		settings, _ := s.repository.GetSettings()
		message := renderOverlayTemplate(settings.OverlayTemplate, user.DisplayName, result.Prize.Name)
		_ = chat.SendSystemAction(message, false)
		if settings.OverlayEnabled {
			if err := BroadcastRewardOverlay(user.DisplayName, result.Prize.Name, result.Prize.Image, message, EffectFireworks, settings.OverlaySoundEnabled); err == nil {
				result.OverlaySent = true
			}
		}
		go s.sendAdminWinEmails(result)
	}
	return result, nil
}

func BroadcastRewardOverlay(displayName string, prizeName string, prizeImage string, message string, effect string, soundEnabled bool) error {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = "A viewer"
	}
	prizeName = strings.TrimSpace(prizeName)
	if prizeName == "" {
		prizeName = "a prize"
	}
	message = sanitizeOverlayText(message)
	if message == "" {
		message = fmt.Sprintf("%s just won %s on the Rewards Wheel!", displayName, prizeName)
	}

	overlayEvent := events.RewardWheelWinEvent{
		Event:        events.Event{},
		DisplayName:  html.EscapeString(displayName),
		PrizeName:    html.EscapeString(prizeName),
		PrizeImage:   strings.TrimSpace(prizeImage),
		Message:      message,
		Effect:       normalizeEffect(effect),
		SoundEnabled: soundEnabled,
	}
	overlayEvent.SetDefaults()
	return chat.Broadcast(&overlayEvent)
}

func renderOverlayTemplate(template string, viewer string, prize string) string {
	template = strings.TrimSpace(template)
	if template == "" {
		template = "{viewer} just won {prize} on the Rewards Wheel!"
	}
	replacer := strings.NewReplacer(
		"{viewer}", strings.TrimSpace(viewer),
		"{prize}", strings.TrimSpace(prize),
	)
	return sanitizeOverlayText(replacer.Replace(template))
}

func sanitizeOverlayText(value string) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > 180 {
		value = string([]rune(value)[:180])
	}
	return html.EscapeString(value)
}

func normalizeEffect(effect string) string {
	switch strings.TrimSpace(effect) {
	case EffectSparkle, EffectFireworks, EffectHearts, EffectHype, EffectDJDrop:
		return strings.TrimSpace(effect)
	default:
		return EffectSparkle
	}
}
