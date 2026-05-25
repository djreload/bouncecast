package admin

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"github.com/owncast/owncast/core/data"
	mobilecore "github.com/owncast/owncast/core/mobile"
	"github.com/owncast/owncast/models"
	webutils "github.com/owncast/owncast/webserver/utils"
)

const (
	mobileAssetUploadFieldName = "asset"
	mobileAssetMaxBytes        = 5 * 1024 * 1024
	mobileAssetMaxPixels       = 8192
)

var mobileAssetContentTypes = map[string]string{
	"image/gif":  ".gif",
	"image/jpeg": ".jpg",
	"image/png":  ".png",
}

func GetMobileAdmin(w http.ResponseWriter, r *http.Request) {
	settings, err := mobilecore.GetAdminSettings(data.GetDatabase())
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, settings)
}

func SetMobileAdminSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	var request models.MobileAdminSettings
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	settings, err := mobilecore.SaveAdminSettings(data.GetDatabase(), request)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	recordBounceCastAuditEvent(r, "mobile_settings_updated", "mobile", "settings", map[string]interface{}{
		"app":                     settings.App.Name,
		"adsEnabled":              settings.Ads.Enabled,
		"pushNotifications":       settings.Features.PushNotifications,
		"maintenanceMode":         settings.App.MaintenanceMode,
		"minimumSupportedVersion": settings.App.MinimumSupportedVersion,
	})
	writeJSON(w, settings)
}

func UploadMobileAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}

	assetType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	r.Body = http.MaxBytesReader(w, r.Body, mobileAssetMaxBytes+1024)
	if err := r.ParseMultipartForm(mobileAssetMaxBytes + 1024); err != nil {
		webutils.BadRequestHandler(w, errors.New("mobile asset must be a multipart image up to 5 MB"))
		return
	}
	if assetType == "" {
		assetType = strings.ToLower(strings.TrimSpace(r.FormValue("type")))
	}
	if assetType == "" {
		assetType = "app-background"
	}
	if !isAllowedMobileAssetType(assetType) {
		webutils.BadRequestHandler(w, fmt.Errorf("unsupported mobile asset type %q", assetType))
		return
	}

	file, header, err := r.FormFile(mobileAssetUploadFieldName)
	if err != nil {
		webutils.BadRequestHandler(w, errors.New("mobile asset file is required"))
		return
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(io.LimitReader(file, mobileAssetMaxBytes+1))
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if len(imageBytes) == 0 {
		webutils.BadRequestHandler(w, errors.New("mobile asset cannot be empty"))
		return
	}
	if len(imageBytes) > mobileAssetMaxBytes || header.Size > mobileAssetMaxBytes {
		webutils.BadRequestHandler(w, errors.New("mobile asset must be 5 MB or smaller"))
		return
	}

	contentType := http.DetectContentType(imageBytes)
	extension, ok := mobileAssetContentTypes[contentType]
	if !ok {
		webutils.BadRequestHandler(w, errors.New("mobile asset must be a PNG, JPG, or GIF"))
		return
	}
	if err := validateMobileAssetDimensions(imageBytes); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	assetURL, sizeBytes, err := saveMobileAsset(assetType, imageBytes, extension)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	_, _ = data.GetDatabase().Exec(`
		INSERT INTO mobile_asset_uploads(asset_type, url, content_type, size_bytes, created_by)
		VALUES(?, ?, ?, ?, ?)
	`, assetType, assetURL, contentType, sizeBytes, mobileAdminSubject(r))

	writeJSON(w, webutils.J{"url": assetURL, "asset_type": assetType})
}

func validateMobileAssetDimensions(imageBytes []byte) error {
	imageConfig, _, err := image.DecodeConfig(bytes.NewReader(imageBytes))
	if err != nil {
		return errors.New("mobile asset could not be decoded")
	}
	if imageConfig.Width <= 0 || imageConfig.Height <= 0 {
		return errors.New("mobile asset dimensions are invalid")
	}
	if imageConfig.Width > mobileAssetMaxPixels || imageConfig.Height > mobileAssetMaxPixels {
		return fmt.Errorf("mobile asset must be %d x %d pixels or smaller", mobileAssetMaxPixels, mobileAssetMaxPixels)
	}
	return nil
}

func saveMobileAsset(assetType string, imageBytes []byte, extension string) (string, int, error) {
	assetDir := filepath.Join(config.PublicFilesPath, "mobile-assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		return "", 0, err
	}

	sum := sha256.Sum256(imageBytes)
	filename := fmt.Sprintf("%s-%s%s", safeMobileAssetType(assetType), hex.EncodeToString(sum[:])[:16], extension)
	destination := filepath.Join(assetDir, filename)

	assetDirAbs, err := filepath.Abs(assetDir)
	if err != nil {
		return "", 0, err
	}
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return "", 0, err
	}
	if !strings.HasPrefix(destinationAbs, assetDirAbs+string(os.PathSeparator)) {
		return "", 0, errors.New("mobile asset destination resolved outside public mobile asset directory")
	}

	if err := os.WriteFile(destinationAbs, imageBytes, 0o644); err != nil {
		return "", 0, err
	}
	return "/public/mobile-assets/" + filename, len(imageBytes), nil
}

func isAllowedMobileAssetType(assetType string) bool {
	switch assetType {
	case "app-background", "logo", "splash", "app-icon":
		return true
	default:
		return false
	}
}

func safeMobileAssetType(assetType string) string {
	var builder strings.Builder
	for _, char := range strings.TrimSpace(assetType) {
		isAllowed := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-'
		if isAllowed {
			builder.WriteRune(char)
		}
	}
	if builder.Len() == 0 {
		return "asset"
	}
	return builder.String()
}

func mobileAdminSubject(r *http.Request) string {
	if user := r.Context().Value("user"); user != nil {
		return fmt.Sprint(user)
	}
	return strings.TrimSpace(r.Header.Get("X-Forwarded-User"))
}
