package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/owncast/owncast/config"
	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
)

const bounceCastProfileImageTestPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAAAAAA6fptVAAAACklEQVR42mP8z8AABQMBgAFG6PIAAAAASUVORK5CYII="

func cleanupBounceCastAccountUploadTestUsers(t *testing.T) {
	t.Helper()
	db := data.GetDatabase()
	_, _ = db.Exec(`DELETE FROM user_access_tokens WHERE user_id IN (
		SELECT id FROM users WHERE email LIKE '%@upload-test.example'
	)`)
	_, _ = db.Exec(`DELETE FROM users WHERE email LIKE '%@upload-test.example'`)
}

func TestBounceCastAccountUploadProfileImageStoresPublicImage(t *testing.T) {
	resetBounceCastRateLimitersForTesting()
	cleanupBounceCastAccountUploadTestUsers(t)
	t.Cleanup(func() { cleanupBounceCastAccountUploadTestUsers(t) })

	oldPublicFilesPath := config.PublicFilesPath
	publicDir := t.TempDir()
	config.PublicFilesPath = publicDir
	t.Cleanup(func() { config.PublicFilesPath = oldPublicFilesPath })

	if _, err := data.GetDatabase().Exec(`
		INSERT INTO users(id, display_name, display_color, previous_names, created_at, authenticated_at, email, password_hash, registered_at)
		VALUES('upload-profile-user', 'Upload Profile User', 1, 'Upload Profile User', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'user@upload-test.example', 'hash', CURRENT_TIMESTAMP)
	`); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	imageBytes, err := base64.StdEncoding.DecodeString(bounceCastProfileImageTestPNG)
	if err != nil {
		t.Fatalf("decode image: %v", err)
	}
	body, contentType := makeBounceCastProfileImageUploadBody(t, "avatar.png", imageBytes)
	request := httptest.NewRequest(http.MethodPost, "/api/bouncecast/account/profile-image", body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()

	BounceCastAccountUploadProfileImage(models.User{
		ID:          "upload-profile-user",
		DisplayName: "Upload Profile User",
		Email:       "user@upload-test.example",
	}, recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("upload status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	var response bounceCastAccountResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.User == nil || !strings.HasPrefix(response.User.ProfileImageURL, "/public/profiles/upload-profile-user-") || !strings.HasSuffix(response.User.ProfileImageURL, ".png") {
		t.Fatalf("unexpected profile image url: %#v", response.User)
	}

	storedPath := filepath.Join(publicDir, strings.TrimPrefix(response.User.ProfileImageURL, "/public/"))
	if _, err := os.Stat(storedPath); err != nil {
		t.Fatalf("stored profile image not found at %s: %v", storedPath, err)
	}
}

func TestBounceCastAccountUploadProfileImageRejectsNonImage(t *testing.T) {
	resetBounceCastRateLimitersForTesting()

	body, contentType := makeBounceCastProfileImageUploadBody(t, "avatar.txt", []byte("not an image"))
	request := httptest.NewRequest(http.MethodPost, "/api/bouncecast/account/profile-image", body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()

	BounceCastAccountUploadProfileImage(models.User{ID: "upload-bad-image-user"}, recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("upload status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
}

func makeBounceCastProfileImageUploadBody(t *testing.T, filename string, imageBytes []byte) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(bounceCastProfileImageFieldName, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(imageBytes); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}
