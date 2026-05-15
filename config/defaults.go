package config

import (
	"time"

	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/webserver/handlers/generated"
)

// Defaults will hold default configuration values.
type Defaults struct {
	PageBodyContent string

	FederationGoLiveMessage string

	Summary              string
	ServerWelcomeMessage string
	Logo                 string
	YPServer             string

	Title string

	DatabaseFilePath string

	FederationUsername string
	WebServerIP        string
	Name               string
	AdminPassword      string
	StreamKeys         []generated.StreamKey

	StreamVariants []models.StreamOutputVariant

	Tags               []string
	RTMPServerPort     int
	SegmentsInPlaylist int

	SegmentLengthSeconds int
	WebServerPort        int

	ChatEstablishedUserModeTimeDuration time.Duration

	YPEnabled bool
}

// GetDefaults will return default configuration values.
func GetDefaults() Defaults {
	defaultStreamKey := "abc123"
	defaultStreamKeyComment := "Default stream key"
	return Defaults{
		Name:                 "New BounceCast Server",
		Summary:              "BounceCast is a self-hosted livestreaming and chat server forked from Owncast, focused on custom branding, creator tools, and future modular streaming features.",
		ServerWelcomeMessage: "",
		Logo:                 "logo.png",
		AdminPassword:        "abc123",
		StreamKeys: []generated.StreamKey{
			{Key: &defaultStreamKey, Comment: &defaultStreamKeyComment},
		},
		Tags: []string{
			"bouncecast",
			"streaming",
		},

		PageBodyContent: `
# Welcome to BounceCast!

- This is a live stream powered by BounceCast, a self-hosted livestreaming and chat server forked from [Owncast](https://owncast.online).

- Customize this page, your logo, and your stream details in the admin.

- If you're the owner of this server you should visit the admin and customize the content on this page.

<hr/>
	`,

		DatabaseFilePath: "data/owncast.db",

		YPEnabled: false,
		YPServer:  "https://owncast.directory",

		WebServerPort:  8080,
		WebServerIP:    "0.0.0.0",
		RTMPServerPort: 1935,

		ChatEstablishedUserModeTimeDuration: time.Minute * 15,

		StreamVariants: []models.StreamOutputVariant{
			{
				IsAudioPassthrough: true,
				VideoBitrate:       1200,
				Framerate:          24,
				CPUUsageLevel:      2,
			},
		},

		FederationUsername:      "streamer",
		FederationGoLiveMessage: "I've gone live!",
	}
}
