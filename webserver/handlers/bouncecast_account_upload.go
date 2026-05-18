package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/owncast/owncast/config"
	"github.com/owncast/owncast/core/chat"
	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/userrepository"
	webutils "github.com/owncast/owncast/webserver/utils"
)

const (
	bounceCastProfileImageFieldName = "image"
	bounceCastProfileImageMaxBytes  = 2 * 1024 * 1024
	bounceCastProfileImageMaxPixels = 4096
)

var bounceCastProfileImageContentTypes = map[string]string{
	"image/gif":  ".gif",
	"image/jpeg": ".jpg",
	"image/png":  ".png",
}

// BounceCastAccountUploadProfileImage validates and stores a local profile
// picture for the current chat account under data/public/profiles.
func BounceCastAccountUploadProfileImage(user models.User, w http.ResponseWriter, r *http.Request) {
	setBounceCastAccountHeaders(w)
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	if !enforceBounceCastRateLimit(w, r, bounceCastAccountProfileImageRateLimit, bounceCastRateLimitUserSubject(user.ID), bounceCastRateLimitIPSubject(r)) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, bounceCastProfileImageMaxBytes+1024)
	if err := r.ParseMultipartForm(bounceCastProfileImageMaxBytes + 1024); err != nil {
		webutils.BadRequestHandler(w, errors.New("profile image upload must be a multipart image up to 2 MB"))
		return
	}

	file, header, err := r.FormFile(bounceCastProfileImageFieldName)
	if err != nil {
		webutils.BadRequestHandler(w, errors.New("profile image file is required"))
		return
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(io.LimitReader(file, bounceCastProfileImageMaxBytes+1))
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if len(imageBytes) == 0 {
		webutils.BadRequestHandler(w, errors.New("profile image cannot be empty"))
		return
	}
	if len(imageBytes) > bounceCastProfileImageMaxBytes || header.Size > bounceCastProfileImageMaxBytes {
		webutils.BadRequestHandler(w, errors.New("profile image must be 2 MB or smaller"))
		return
	}

	contentType := http.DetectContentType(imageBytes)
	extension, ok := bounceCastProfileImageContentTypes[contentType]
	if !ok {
		webutils.BadRequestHandler(w, errors.New("profile image must be a PNG, JPG, or GIF"))
		return
	}
	if err := validateBounceCastProfileImageDimensions(imageBytes); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	profileImageURL, err := saveBounceCastProfileImage(user.ID, imageBytes, extension)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if err := updateBounceCastAccountProfileImage(user, profileImageURL); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	updatedUser := userrepository.Get().GetUserByID(user.ID)
	notificationPreferences, _ := getBounceCastAccountNotificationPreferences(user.ID)
	_ = chat.SendConnectedClientInfoToUser(user.ID)
	writeJSON(w, bounceCastAccountResponse{
		User:                    updatedUser,
		NotificationPreferences: notificationPreferences,
		Message:                 "Profile image uploaded.",
	})
}

func validateBounceCastProfileImageDimensions(imageBytes []byte) error {
	config, _, err := image.DecodeConfig(bytes.NewReader(imageBytes))
	if err != nil {
		return errors.New("profile image could not be decoded")
	}
	if config.Width <= 0 || config.Height <= 0 {
		return errors.New("profile image dimensions are invalid")
	}
	if config.Width > bounceCastProfileImageMaxPixels || config.Height > bounceCastProfileImageMaxPixels {
		return fmt.Errorf("profile image must be %d x %d pixels or smaller", bounceCastProfileImageMaxPixels, bounceCastProfileImageMaxPixels)
	}
	return nil
}

func saveBounceCastProfileImage(userID string, imageBytes []byte, extension string) (string, error) {
	profilesDir := filepath.Join(config.PublicFilesPath, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		return "", err
	}

	safeUserID := safeBounceCastProfileImageUserID(userID)
	sum := sha256.Sum256(imageBytes)
	filename := fmt.Sprintf("%s-%s%s", safeUserID, hex.EncodeToString(sum[:])[:16], extension)
	destination := filepath.Join(profilesDir, filename)

	profilesAbs, err := filepath.Abs(profilesDir)
	if err != nil {
		return "", err
	}
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(destinationAbs, profilesAbs+string(os.PathSeparator)) {
		return "", errors.New("profile image destination resolved outside public profiles directory")
	}

	if err := os.WriteFile(destinationAbs, imageBytes, 0o644); err != nil {
		return "", err
	}
	removeOldBounceCastProfileImages(profilesDir, safeUserID, destinationAbs)
	return "/public/profiles/" + filename, nil
}

func removeOldBounceCastProfileImages(profilesDir string, safeUserID string, keepPath string) {
	matches, err := filepath.Glob(filepath.Join(profilesDir, safeUserID+"-*"))
	if err != nil {
		return
	}
	keepPath, _ = filepath.Abs(keepPath)
	for _, match := range matches {
		matchAbs, err := filepath.Abs(match)
		if err != nil || matchAbs == keepPath {
			continue
		}
		_ = os.Remove(matchAbs)
	}
}

func safeBounceCastProfileImageUserID(userID string) string {
	builder := strings.Builder{}
	for _, char := range strings.TrimSpace(userID) {
		isAllowed := (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_'
		if isAllowed {
			builder.WriteRune(char)
		}
	}
	if builder.Len() == 0 {
		return "user"
	}
	return builder.String()
}

func updateBounceCastAccountProfileImage(user models.User, profileImageURL string) error {
	if _, err := data.GetDatabase().Exec(`
		UPDATE users
		SET profile_image_url = NULLIF(?, '')
		WHERE id = ?
	`, profileImageURL, user.ID); err != nil {
		return err
	}
	if strings.TrimSpace(user.Email) == "" {
		return nil
	}
	_, err := data.GetDatabase().Exec(`
		UPDATE bouncecast_streamer_accounts
		SET avatar_url = NULLIF(?, ''), updated_at = CURRENT_TIMESTAMP
		WHERE LOWER(COALESCE(email, '')) = LOWER(?)
	`, profileImageURL, user.Email)
	return err
}
