package events

// MessageReactionEvent is sent when a user toggles an emoji reaction on a chat message.
type MessageReactionEvent struct {
	Event
	MessageID string         `json:"messageId"`
	Reaction  string         `json:"reaction,omitempty"`
	Counts    map[string]int `json:"counts,omitempty"`
}

var allowedMessageReactions = map[string]struct{}{
	"\U0001F525":   {}, // fire
	"\u2764\ufe0f": {}, // heart
	"\U0001F602":   {}, // laugh
	"\U0001F44D":   {}, // thumbs up
	"\U0001F622":   {}, // cry
}

// IsAllowedMessageReaction returns if a reaction is part of the public picker.
func IsAllowedMessageReaction(reaction string) bool {
	_, ok := allowedMessageReactions[reaction]
	return ok
}

// GetBroadcastPayload will return the reaction update sent to all chat users.
func (e *MessageReactionEvent) GetBroadcastPayload() EventPayload {
	return EventPayload{
		"id":        e.ID,
		"timestamp": e.Timestamp,
		"type":      MessageReactionUpdate,
		"messageId": e.MessageID,
		"counts":    e.Counts,
	}
}

// GetMessageType will return the event type for this message.
func (e *MessageReactionEvent) GetMessageType() EventType {
	return MessageReactionUpdate
}
