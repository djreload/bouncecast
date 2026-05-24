package events

import "testing"

func TestRewardWheelWinEventPayload(t *testing.T) {
	event := RewardWheelWinEvent{
		DisplayName:  "Kevin",
		PrizeName:    "VIP T-shirt",
		PrizeImage:   "/public/prizes/vip.png",
		Message:      "Kevin just won VIP T-shirt on the Rewards Wheel!",
		Effect:       "fireworks",
		SoundEnabled: true,
	}
	event.SetDefaults()

	payload := event.GetBroadcastPayload()
	if payload["type"] != RewardWheelWin || payload["displayName"] != "Kevin" || payload["prizeName"] != "VIP T-shirt" || payload["soundEnabled"] != true {
		t.Fatalf("unexpected reward payload: %+v", payload)
	}
}
