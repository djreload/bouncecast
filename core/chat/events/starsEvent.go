package events

// StarsSentEvent is broadcast when a viewer sends Stars to support the site.
type StarsSentEvent struct {
	Event
	DisplayName  string `json:"displayName"`
	Amount       int    `json:"amount"`
	Message      string `json:"message,omitempty"`
	Effect       string `json:"effect"`
	SoundEnabled bool   `json:"soundEnabled"`
}

// GetBroadcastPayload will return the object to send to all chat users.
func (e *StarsSentEvent) GetBroadcastPayload() EventPayload {
	return EventPayload{
		"id":           e.ID,
		"timestamp":    e.Timestamp,
		"type":         StarsSent,
		"displayName":  e.DisplayName,
		"amount":       e.Amount,
		"message":      e.Message,
		"effect":       e.Effect,
		"soundEnabled": e.SoundEnabled,
	}
}

// GetMessageType will return the event type for this message.
func (e *StarsSentEvent) GetMessageType() EventType {
	return StarsSent
}
