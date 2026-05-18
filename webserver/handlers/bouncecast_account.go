package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/owncast/owncast/config"
	"github.com/owncast/owncast/core/chat"
	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/notificationsrepository"
	"github.com/owncast/owncast/persistence/userrepository"
	"github.com/owncast/owncast/utils"
	webutils "github.com/owncast/owncast/webserver/utils"
	"github.com/teris-io/shortid"
)

const bounceCastAccountPasswordMinLength = 8
const bounceCastMessengerChannel = "messenger"

type bounceCastAccountRegisterRequest struct {
	DisplayName             string                                    `json:"displayName"`
	Email                   string                                    `json:"email"`
	Password                string                                    `json:"password"`
	ProfileImageURL         string                                    `json:"profileImageUrl"`
	NotificationPreferences *bounceCastNotificationPreferencesRequest `json:"notificationPreferences"`
}

type bounceCastAccountLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type bounceCastAccountProfileRequest struct {
	DisplayName             string                                    `json:"displayName"`
	ProfileImageURL         string                                    `json:"profileImageUrl"`
	NotificationPreferences *bounceCastNotificationPreferencesRequest `json:"notificationPreferences"`
}

type bounceCastAccountResponse struct {
	User                    *models.User                             `json:"user"`
	NotificationPreferences bounceCastAccountNotificationPreferences `json:"notificationPreferences"`
	AccessToken             string                                   `json:"accessToken,omitempty"`
	Message                 string                                   `json:"message,omitempty"`
}

type bounceCastNotificationPreferencesRequest struct {
	Email                bool   `json:"email"`
	BrowserPush          bool   `json:"browserPush"`
	Messenger            bool   `json:"messenger"`
	MessengerDestination string `json:"messengerDestination"`
	BrowserPushEndpoint  string `json:"browserPushEndpoint"`
}

type bounceCastAccountNotificationPreferences struct {
	Email                bool   `json:"email"`
	BrowserPush          bool   `json:"browserPush"`
	Messenger            bool   `json:"messenger"`
	MessengerDestination string `json:"messengerDestination,omitempty"`
}

// BounceCastAccountOptions handles CORS preflight for public account APIs.
func BounceCastAccountOptions(w http.ResponseWriter, r *http.Request) {
	setBounceCastAccountHeaders(w)
	w.WriteHeader(http.StatusNoContent)
}

// BounceCastAccountRegister creates a registered BounceCast chat account. When
// a valid anonymous chat access token is supplied it upgrades that identity in
// place so chat history, color, wallet, and moderation state are preserved.
func BounceCastAccountRegister(w http.ResponseWriter, r *http.Request) {
	setBounceCastAccountHeaders(w)

	var request bounceCastAccountRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	displayName := utils.MakeSafeStringOfLength(request.DisplayName, config.MaxChatDisplayNameLength)
	if displayName == "" {
		webutils.BadRequestHandler(w, errors.New("displayName is required"))
		return
	}

	email, err := normalizeBounceCastAccountEmail(request.Email)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	if err := validateBounceCastAccountPassword(request.Password); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	profileImageURL, err := normalizeBounceCastProfileImageURL(request.ProfileImageURL)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	notificationPreferences, err := normalizeBounceCastNotificationPreferences(request.NotificationPreferences)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	passwordHash, err := utils.HashPassword(strings.TrimSpace(request.Password))
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	accessToken := strings.TrimSpace(r.URL.Query().Get("accessToken"))
	if accessToken != "" {
		user := userrepository.Get().GetUserByToken(accessToken)
		if user != nil && user.Email == "" {
			updatedUser, err := upgradeBounceCastAccount(user.ID, displayName, email, passwordHash, profileImageURL)
			if err != nil {
				writeBounceCastAccountSaveError(w, err)
				return
			}
			if err := saveBounceCastAccountNotificationPreferences(updatedUser, notificationPreferences); err != nil {
				webutils.InternalErrorHandler(w, err)
				return
			}
			notificationPreferences, _ = getBounceCastAccountNotificationPreferences(updatedUser.ID)
			_ = chat.SendConnectedClientInfoToUser(user.ID)
			webutils.WriteResponse(w, bounceCastAccountResponse{
				User:                    updatedUser,
				NotificationPreferences: notificationPreferences,
				AccessToken:             accessToken,
				Message:                 "Account registered.",
			})
			return
		}
	}

	user, token, err := createBounceCastAccount(displayName, email, passwordHash, profileImageURL)
	if err != nil {
		writeBounceCastAccountSaveError(w, err)
		return
	}
	if err := saveBounceCastAccountNotificationPreferences(user, notificationPreferences); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	notificationPreferences, _ = getBounceCastAccountNotificationPreferences(user.ID)

	webutils.WriteResponse(w, bounceCastAccountResponse{
		User:                    user,
		NotificationPreferences: notificationPreferences,
		AccessToken:             token,
		Message:                 "Account registered.",
	})
}

// BounceCastAccountLogin verifies a registered account and issues a normal chat
// access token for the existing user.
func BounceCastAccountLogin(w http.ResponseWriter, r *http.Request) {
	setBounceCastAccountHeaders(w)

	var request bounceCastAccountLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	email, err := normalizeBounceCastAccountEmail(request.Email)
	if err != nil {
		writeBounceCastAccountUnauthorized(w)
		return
	}
	password := strings.TrimSpace(request.Password)
	if password == "" {
		writeBounceCastAccountUnauthorized(w)
		return
	}

	userID, passwordHash, err := getBounceCastAccountLogin(email)
	if err != nil || passwordHash == "" {
		writeBounceCastAccountUnauthorized(w)
		return
	}
	if err := utils.CompareHash(passwordHash, password); err != nil {
		writeBounceCastAccountUnauthorized(w)
		return
	}

	token, err := utils.GenerateAccessToken()
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	datastore := data.GetDatastore()
	if err := func() error {
		datastore.DbLock.Lock()
		defer datastore.DbLock.Unlock()

		tx, err := datastore.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback() //nolint

		if _, err := tx.Exec("INSERT INTO user_access_tokens(token, user_id) VALUES(?, ?)", token, userID); err != nil {
			return err
		}
		if _, err := tx.Exec("UPDATE users SET authenticated_at = COALESCE(authenticated_at, CURRENT_TIMESTAMP), last_login_at = CURRENT_TIMESTAMP WHERE id = ?", userID); err != nil {
			return err
		}
		return tx.Commit()
	}(); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	user := userrepository.Get().GetUserByID(userID)
	notificationPreferences, _ := getBounceCastAccountNotificationPreferences(userID)
	webutils.WriteResponse(w, bounceCastAccountResponse{
		User:                    user,
		NotificationPreferences: notificationPreferences,
		AccessToken:             token,
		Message:                 "Logged in.",
	})
}

// BounceCastAccountMe returns the authenticated chat account.
func BounceCastAccountMe(user models.User, w http.ResponseWriter, r *http.Request) {
	setBounceCastAccountHeaders(w)
	notificationPreferences, _ := getBounceCastAccountNotificationPreferences(user.ID)
	webutils.WriteResponse(w, bounceCastAccountResponse{User: &user, NotificationPreferences: notificationPreferences})
}

// BounceCastAccountUpdateProfile updates public chat profile details for a
// connected chat identity.
func BounceCastAccountUpdateProfile(user models.User, w http.ResponseWriter, r *http.Request) {
	setBounceCastAccountHeaders(w)

	var request bounceCastAccountProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	displayName := utils.MakeSafeStringOfLength(request.DisplayName, config.MaxChatDisplayNameLength)
	if displayName == "" {
		webutils.BadRequestHandler(w, errors.New("displayName is required"))
		return
	}
	profileImageURL, err := normalizeBounceCastProfileImageURL(request.ProfileImageURL)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	notificationPreferences, err := getBounceCastAccountNotificationPreferences(user.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if request.NotificationPreferences != nil {
		notificationPreferences, err = normalizeBounceCastNotificationPreferences(request.NotificationPreferences)
		if err != nil {
			webutils.BadRequestHandler(w, err)
			return
		}
	}

	if err := ensureBounceCastDisplayNameAvailable(user.ID, displayName); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	datastore := data.GetDatastore()
	if err := func() error {
		datastore.DbLock.Lock()
		defer datastore.DbLock.Unlock()

		tx, err := datastore.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback() //nolint

		if displayName != user.DisplayName {
			if _, err := tx.Exec(`
				UPDATE users
				SET display_name = ?, previous_names = previous_names || ?, namechanged_at = CURRENT_TIMESTAMP,
					profile_image_url = NULLIF(?, '')
				WHERE id = ?
			`, displayName, ","+displayName, profileImageURL, user.ID); err != nil {
				return err
			}
		} else if _, err := tx.Exec(`
			UPDATE users
			SET profile_image_url = NULLIF(?, '')
			WHERE id = ?
		`, profileImageURL, user.ID); err != nil {
			return err
		}

		return tx.Commit()
	}(); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}

	updatedUser := userrepository.Get().GetUserByID(user.ID)
	if err := saveBounceCastAccountNotificationPreferences(updatedUser, notificationPreferences); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	notificationPreferences, _ = getBounceCastAccountNotificationPreferences(user.ID)
	_ = chat.SendConnectedClientInfoToUser(user.ID)
	webutils.WriteResponse(w, bounceCastAccountResponse{
		User:                    updatedUser,
		NotificationPreferences: notificationPreferences,
		Message:                 "Profile updated.",
	})
}

// BounceCastAccountUpdateNotifications updates opt-in preferences for the
// current public viewer account. It does not grant admin or Studio access.
func BounceCastAccountUpdateNotifications(user models.User, w http.ResponseWriter, r *http.Request) {
	setBounceCastAccountHeaders(w)

	var request bounceCastNotificationPreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}

	notificationPreferences, err := normalizeBounceCastNotificationPreferences(&request)
	if err != nil {
		webutils.BadRequestHandler(w, err)
		return
	}
	notificationPreferences.BrowserPush = request.BrowserPush

	if err := saveBounceCastAccountNotificationPreferences(&user, notificationPreferences); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	if strings.TrimSpace(request.BrowserPushEndpoint) != "" {
		if err := saveBounceCastBrowserPushDestination(request.BrowserPushEndpoint, request.BrowserPush); err != nil {
			webutils.BadRequestHandler(w, err)
			return
		}
	}

	notificationPreferences, _ = getBounceCastAccountNotificationPreferences(user.ID)
	webutils.WriteResponse(w, bounceCastAccountResponse{
		User:                    &user,
		NotificationPreferences: notificationPreferences,
		Message:                 "Notification preferences updated.",
	})
}

func createBounceCastAccount(displayName string, email string, passwordHash string, profileImageURL string) (*models.User, string, error) {
	datastore := data.GetDatastore()

	token, err := utils.GenerateAccessToken()
	if err != nil {
		return nil, "", err
	}

	userID := shortid.MustGenerate()
	displayColor := utils.GenerateRandomDisplayColor(config.MaxUserColor)
	now := time.Now().UTC()

	if err := func() error {
		datastore.DbLock.Lock()
		defer datastore.DbLock.Unlock()

		tx, err := datastore.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback() //nolint

		if _, err := tx.Exec(`
			INSERT INTO users(id, display_name, display_color, previous_names, created_at, authenticated_at, scopes, email, password_hash, profile_image_url, registered_at, last_login_at)
			VALUES(?, ?, ?, ?, ?, ?, NULL, ?, ?, NULLIF(?, ''), ?, ?)
		`, userID, displayName, displayColor, displayName, now, now, email, passwordHash, profileImageURL, now, now); err != nil {
			return err
		}
		if _, err := tx.Exec("INSERT INTO user_access_tokens(token, user_id) VALUES(?, ?)", token, userID); err != nil {
			return err
		}
		return tx.Commit()
	}(); err != nil {
		return nil, "", err
	}

	return userrepository.Get().GetUserByID(userID), token, nil
}

func upgradeBounceCastAccount(userID string, displayName string, email string, passwordHash string, profileImageURL string) (*models.User, error) {
	if err := ensureBounceCastDisplayNameAvailable(userID, displayName); err != nil {
		return nil, err
	}

	datastore := data.GetDatastore()
	if err := func() error {
		datastore.DbLock.Lock()
		defer datastore.DbLock.Unlock()

		tx, err := datastore.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback() //nolint

		if _, err := tx.Exec(`
			UPDATE users
			SET display_name = ?, previous_names = previous_names || ?, namechanged_at = CURRENT_TIMESTAMP,
				authenticated_at = COALESCE(authenticated_at, CURRENT_TIMESTAMP),
				email = ?, password_hash = ?, profile_image_url = NULLIF(?, ''),
				registered_at = COALESCE(registered_at, CURRENT_TIMESTAMP),
				last_login_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, displayName, ","+displayName, email, passwordHash, profileImageURL, userID); err != nil {
			return err
		}
		return tx.Commit()
	}(); err != nil {
		return nil, err
	}

	return userrepository.Get().GetUserByID(userID), nil
}

func getBounceCastAccountLogin(email string) (string, string, error) {
	user, err := getBounceCastAccountRoleUserForLogin(email)
	if err != nil {
		return "", "", err
	}
	return user.ID, user.PasswordHash, nil
}

func normalizeBounceCastAccountEmail(value string) (string, error) {
	email := strings.TrimSpace(value)
	if email == "" {
		return "", errors.New("email is required")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil {
		return "", errors.New("email must be a valid email address")
	}
	return strings.ToLower(parsed.Address), nil
}

func validateBounceCastAccountPassword(password string) error {
	if strings.ContainsAny(password, "\r\n") {
		return errors.New("password cannot contain line breaks")
	}
	if len(strings.TrimSpace(password)) < bounceCastAccountPasswordMinLength {
		return fmt.Errorf("password must be at least %d characters", bounceCastAccountPasswordMinLength)
	}
	return nil
}

func normalizeBounceCastProfileImageURL(value string) (string, error) {
	imageURL := utils.MakeSafeStringOfLength(value, 500)
	if imageURL == "" {
		return "", nil
	}
	if strings.HasPrefix(imageURL, "/") && !strings.HasPrefix(imageURL, "//") {
		return imageURL, nil
	}
	parsed, err := url.ParseRequestURI(imageURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("profile image must be an http(s) URL or a local path")
	}
	return imageURL, nil
}

func normalizeBounceCastNotificationPreferences(request *bounceCastNotificationPreferencesRequest) (bounceCastAccountNotificationPreferences, error) {
	if request == nil {
		return bounceCastAccountNotificationPreferences{}, nil
	}
	messengerDestination := utils.MakeSafeStringOfLength(request.MessengerDestination, 500)
	if request.Messenger && messengerDestination == "" {
		return bounceCastAccountNotificationPreferences{}, errors.New("messenger destination is required when Messenger notifications are enabled")
	}

	return bounceCastAccountNotificationPreferences{
		Email:                request.Email,
		BrowserPush:          request.BrowserPush,
		Messenger:            request.Messenger,
		MessengerDestination: messengerDestination,
	}, nil
}

func getBounceCastAccountNotificationPreferences(userID string) (bounceCastAccountNotificationPreferences, error) {
	var emailOptIn int
	var browserPushOptIn int
	var messengerOptIn int
	var messengerDestination sql.NullString
	err := data.GetDatabase().QueryRow(`
		SELECT notification_email_opt_in, notification_browser_push_opt_in,
			notification_messenger_opt_in, notification_messenger_destination
		FROM users
		WHERE id = ?
	`, userID).Scan(&emailOptIn, &browserPushOptIn, &messengerOptIn, &messengerDestination)
	if err != nil {
		return bounceCastAccountNotificationPreferences{}, err
	}

	return bounceCastAccountNotificationPreferences{
		Email:                emailOptIn == 1,
		BrowserPush:          browserPushOptIn == 1,
		Messenger:            messengerOptIn == 1,
		MessengerDestination: messengerDestination.String,
	}, nil
}

func saveBounceCastAccountNotificationPreferences(user *models.User, preferences bounceCastAccountNotificationPreferences) error {
	if user == nil {
		return errors.New("user is required")
	}

	currentPreferences, _ := getBounceCastAccountNotificationPreferences(user.ID)
	datastore := data.GetDatastore()
	if err := func() error {
		datastore.DbLock.Lock()
		defer datastore.DbLock.Unlock()

		_, err := datastore.DB.Exec(`
			UPDATE users
			SET notification_email_opt_in = ?,
				notification_browser_push_opt_in = ?,
				notification_messenger_opt_in = ?,
				notification_messenger_destination = NULLIF(?, '')
			WHERE id = ?
		`, bounceCastBoolToInt(preferences.Email), bounceCastBoolToInt(preferences.BrowserPush), bounceCastBoolToInt(preferences.Messenger), preferences.MessengerDestination, user.ID)
		return err
	}(); err != nil {
		return err
	}

	if user.Email != "" {
		if err := syncBounceCastNotificationDestination("email", user.Email, preferences.Email); err != nil {
			return err
		}
	}
	if currentPreferences.MessengerDestination != "" && currentPreferences.MessengerDestination != preferences.MessengerDestination {
		if err := notificationsrepository.Get().RemoveNotificationForChannel(bounceCastMessengerChannel, currentPreferences.MessengerDestination); err != nil {
			return err
		}
	}
	return syncBounceCastNotificationDestination(bounceCastMessengerChannel, preferences.MessengerDestination, preferences.Messenger)
}

func saveBounceCastBrowserPushDestination(endpoint string, enabled bool) error {
	endpoint = utils.MakeSafeStringOfLength(endpoint, 2000)
	if endpoint == "" {
		return nil
	}
	return syncBounceCastNotificationDestination(notificationsrepository.BrowserPushNotification, endpoint, enabled)
}

func syncBounceCastNotificationDestination(channel string, destination string, enabled bool) error {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return nil
	}

	repository := notificationsrepository.Get()
	if err := repository.RemoveNotificationForChannel(channel, destination); err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	return repository.AddNotification(channel, destination)
}

func bounceCastBoolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func ensureBounceCastDisplayNameAvailable(userID string, displayName string) error {
	var existingID string
	err := data.GetDatabase().QueryRow(`
		SELECT id
		FROM users
		WHERE LOWER(display_name) = LOWER(?)
			AND id != ?
			AND disabled_at IS NULL
			AND (type = 'API' OR authenticated_at IS NOT NULL)
		LIMIT 1
	`, displayName, userID).Scan(&existingID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("display name is already in use")
}

func writeBounceCastAccountSaveError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	if strings.Contains(strings.ToLower(fmt.Sprint(err)), "unique constraint failed") {
		webutils.BadRequestHandler(w, errors.New("email is already registered"))
		return
	}
	webutils.InternalErrorHandler(w, err)
}

func writeBounceCastAccountUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(webutils.J{"error": "invalid email or password"})
}

func setBounceCastAccountHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}
