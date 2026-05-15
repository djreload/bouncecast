package events

import "testing"

func TestStarsSentEventPayload(t *testing.T) {
	event := StarsSentEvent{
		DisplayName: "Kevin",
		Amount:      100,
		Message:     "Big up",
		Effect:      "sparkle",
	}
	event.SetDefaults()

	payload := event.GetBroadcastPayload()
	if payload["type"] != StarsSent || payload["displayName"] != "Kevin" || payload["amount"] != 100 || payload["effect"] != "sparkle" {
		t.Fatalf("unexpected Stars payload: %+v", payload)
	}
}
