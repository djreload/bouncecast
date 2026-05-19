package models

// BounceCastSocialLink is a public, sanitized profile link for a DJ profile.
type BounceCastSocialLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}
