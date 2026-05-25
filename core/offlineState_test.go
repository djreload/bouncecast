package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/owncast/owncast/config"
)

func TestAppendOfflineToVariantPlaylistUpdatesTargetDuration(t *testing.T) {
	tempDir := t.TempDir()
	previousTempDir := config.TempDir
	config.TempDir = tempDir
	t.Cleanup(func() {
		config.TempDir = previousTempDir
	})

	playlistFilePath := filepath.Join(tempDir, "stream.m3u8")
	initialPlaylist := strings.Join([]string{
		"#EXTM3U",
		"#EXT-X-VERSION:6",
		"#EXT-X-TARGETDURATION:3",
		"#EXT-X-MEDIA-SEQUENCE:7",
		"#EXTINF:3.000000,",
		"stream-7.ts",
		"",
	}, "\n")

	if err := os.WriteFile(playlistFilePath, []byte(initialPlaylist), 0o600); err != nil {
		t.Fatalf("write playlist: %v", err)
	}

	appendOfflineToVariantPlaylist(0, playlistFilePath)

	updatedPlaylistBytes, err := os.ReadFile(playlistFilePath)
	if err != nil {
		t.Fatalf("read updated playlist: %v", err)
	}
	updatedPlaylist := string(updatedPlaylistBytes)

	if !strings.Contains(updatedPlaylist, "#EXT-X-TARGETDURATION:9") {
		t.Fatalf("target duration was not updated: %s", updatedPlaylist)
	}

	if !strings.Contains(updatedPlaylist, "#EXTINF:8.128000,") {
		t.Fatalf("offline segment duration was not written: %s", updatedPlaylist)
	}
}
