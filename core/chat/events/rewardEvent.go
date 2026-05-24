package events

// RewardWheelWinEvent is broadcast when the Rewards Wheel creates a real prize win.
type RewardWheelWinEvent struct {
	Event
	DisplayName  string `json:"displayName"`
	PrizeName    string `json:"prizeName"`
	PrizeImage   string `json:"prizeImage,omitempty"`
	Message      string `json:"message"`
	Effect       string `json:"effect"`
	SoundEnabled bool   `json:"soundEnabled"`
}

// GetBroadcastPayload will return the object to send to all chat users.
func (e *RewardWheelWinEvent) GetBroadcastPayload() EventPayload {
	return EventPayload{
		"id":           e.ID,
		"timestamp":    e.Timestamp,
		"type":         RewardWheelWin,
		"displayName":  e.DisplayName,
		"prizeName":    e.PrizeName,
		"prizeImage":   e.PrizeImage,
		"message":      e.Message,
		"effect":       e.Effect,
		"soundEnabled": e.SoundEnabled,
	}
}

// GetMessageType will return the event type for this message.
func (e *RewardWheelWinEvent) GetMessageType() EventType {
	return RewardWheelWin
}
