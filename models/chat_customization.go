package models

// ChatCustomization contains public chat appearance and GIF picker settings.
type ChatCustomization struct {
	BackgroundImageURL string  `json:"backgroundImageUrl"`
	BackgroundOpacity  float64 `json:"backgroundOpacity"`
	TenorAPIKey        string  `json:"tenorApiKey"`
}
