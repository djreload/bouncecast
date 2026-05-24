package rewards

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
)

type rewardEmailSettings struct {
	enabled     bool
	host        string
	port        int
	username    string
	password    string
	fromAddress string
	fromName    string
	startTLS    bool
}

func (s *Service) DispatchOrder(adminUserID string, orderID int64, courier string, trackingReference string, trackingURL string, dispatchNote string) (models.RewardOrder, models.RewardUserNotification, error) {
	order, notification, err := s.repository.MarkOrderDispatched(adminUserID, orderID, courier, trackingReference, trackingURL, dispatchNote)
	if err != nil {
		return models.RewardOrder{}, models.RewardUserNotification{}, err
	}
	emailErr := s.sendDispatchEmail(order, notification)
	recordRewardNotificationEmailStatus(notification.ID, emailErr == nil, emailErrorString(emailErr))
	if emailErr == nil {
		notification.EmailSent = true
	} else {
		notification.EmailError = emailErr.Error()
	}
	return order, notification, nil
}

func (s *Service) sendAdminWinEmails(result models.RewardSpinResult) {
	if result.Winner == nil {
		return
	}
	recipients := rewardAdminEmailRecipients()
	if len(recipients) == 0 {
		recordRewardAdminEmailLog(result, "No admin email recipients with stored email addresses were found.")
		return
	}
	body := fmt.Sprintf("Rewards Wheel win\n\nViewer: %s\nPrize: %s\nSpin ID: %d\nClaim ID: %d\nOrder ID: %d\n\nOpen the Rewards Wheel admin fulfilment dashboard to process this prize.",
		result.Winner.UsernameSnapshot, result.Winner.PrizeSnapshot, result.Winner.SpinID, result.Winner.ClaimID, result.Winner.OrderID)
	failures := []string{}
	for _, recipient := range recipients {
		if err := sendRewardEmail(recipient, "BounceCast Rewards Wheel win", body); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", recipient, err.Error()))
		}
	}
	if len(failures) == 0 {
		recordRewardAdminEmailLog(result, fmt.Sprintf("Reward win email sent to %d admin recipient(s).", len(recipients)))
		return
	}
	recordRewardAdminEmailLog(result, "Reward win email failures: "+strings.Join(failures, "; "))
}

func (s *Service) sendDispatchEmail(order models.RewardOrder, notification models.RewardUserNotification) error {
	email := rewardUserEmail(order.WinnerUserID)
	if email == "" {
		return fmt.Errorf("winner has no stored email address")
	}
	body := fmt.Sprintf("%s\n\nPrize: %s\nOrder ID: %d\n", notification.Message, order.PrizeSnapshot, order.ID)
	if order.Courier != "" {
		body += "Courier: " + order.Courier + "\n"
	}
	if order.TrackingReference != "" {
		body += "Tracking reference: " + order.TrackingReference + "\n"
	}
	if order.TrackingURL != "" {
		body += "Tracking URL: " + order.TrackingURL + "\n"
	}
	if order.DispatchNote != "" {
		body += "\n" + order.DispatchNote + "\n"
	}
	body += "\nContact the BounceCast site team if you need help with this prize."
	return sendRewardEmail(email, "Your BounceCast reward has been dispatched", body)
}

func sendRewardEmail(destination string, subject string, body string) error {
	settings := readRewardEmailSettings()
	if !settings.enabled {
		return fmt.Errorf("SMTP email notifications are not enabled")
	}
	if settings.host == "" || settings.fromAddress == "" {
		return fmt.Errorf("SMTP host and from address are required")
	}
	fromAddress := strings.TrimSpace(settings.fromAddress)
	if _, err := mail.ParseAddress(fromAddress); err != nil {
		return fmt.Errorf("invalid SMTP from address: %w", err)
	}
	to, err := mail.ParseAddress(strings.TrimSpace(destination))
	if err != nil {
		return fmt.Errorf("invalid email destination: %w", err)
	}

	from := mail.Address{Name: sanitizeRewardEmailHeader(settings.fromName), Address: fromAddress}
	var message strings.Builder
	message.WriteString(fmt.Sprintf("From: %s\r\n", from.String()))
	message.WriteString(fmt.Sprintf("To: %s\r\n", to.String()))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", sanitizeRewardEmailHeader(subject)))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	message.WriteString("\r\n")
	message.WriteString(body)

	address := net.JoinHostPort(settings.host, strconv.Itoa(settings.port))
	dialer := net.Dialer{Timeout: 10 * time.Second}
	var connection net.Conn
	var connectionErr error
	if settings.port == 465 {
		connection, connectionErr = tls.DialWithDialer(&dialer, "tcp", address, &tls.Config{ServerName: settings.host, MinVersion: tls.VersionTLS12})
	} else {
		connection, connectionErr = dialer.Dial("tcp", address)
	}
	if connectionErr != nil {
		return connectionErr
	}
	defer connection.Close()

	client, err := smtp.NewClient(connection, settings.host)
	if err != nil {
		return err
	}
	defer client.Close()

	if settings.startTLS && settings.port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: settings.host, MinVersion: tls.VersionTLS12}); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("SMTP server does not support STARTTLS")
		}
	}

	if settings.username != "" {
		if err := client.Auth(smtp.PlainAuth("", settings.username, settings.password, settings.host)); err != nil {
			return err
		}
	}
	if err := client.Mail(fromAddress); err != nil {
		return err
	}
	if err := client.Rcpt(to.Address); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write([]byte(message.String())); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func readRewardEmailSettings() rewardEmailSettings {
	get := func(key string) string {
		var value string
		_ = data.GetDatastore().DB.QueryRow(`SELECT value FROM bouncecast_notification_settings WHERE key=?`, key).Scan(&value)
		return strings.TrimSpace(value)
	}
	provider := get("email_provider")
	port, _ := strconv.Atoi(get("email_port"))
	host := get("email_host")
	startTLS := get("email_start_tls") == "true"
	if provider == "brevo" {
		host = "smtp-relay.brevo.com"
		if port == 0 {
			port = 587
		}
		startTLS = true
	}
	if port == 0 {
		port = 587
	}
	fromName := get("email_from_name")
	if fromName == "" {
		fromName = "BounceCast"
	}
	return rewardEmailSettings{
		enabled:     get("email_enabled") == "true",
		host:        host,
		port:        port,
		username:    get("email_username"),
		password:    get("email_password"),
		fromAddress: get("email_from_address"),
		fromName:    fromName,
		startTLS:    startTLS,
	}
}

func rewardAdminEmailRecipients() []string {
	rows, err := data.GetDatastore().DB.Query(`SELECT DISTINCT email FROM users
		WHERE email IS NOT NULL AND email != '' AND (
			(',' || COALESCE(scopes, '') || ',') LIKE '%,' || ? || ',%' OR
			(',' || COALESCE(scopes, '') || ',') LIKE '%,' || ? || ',%'
		)`, models.BounceCastOwnerScopeKey, models.BounceCastAdminScopeKey)
	if err != nil {
		return nil
	}
	defer rows.Close()
	recipients := []string{}
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err == nil && strings.TrimSpace(email) != "" {
			recipients = append(recipients, strings.TrimSpace(email))
		}
	}
	return recipients
}

func rewardUserEmail(userID string) string {
	var email string
	_ = data.GetDatastore().DB.QueryRow(`SELECT COALESCE(email, '') FROM users WHERE id=?`, userID).Scan(&email)
	return strings.TrimSpace(email)
}

func recordRewardAdminEmailLog(result models.RewardSpinResult, body string) {
	if result.Winner == nil {
		return
	}
	_, _ = data.GetDatastore().DB.Exec(`INSERT INTO reward_admin_messages(type, severity, title, body, user_id, prize_id, spin_id, claim_id, order_id)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		models.RewardAdminMessageTypeWin, "info", "Rewards email status", body, result.Winner.UserID, nullableRewardID(result.Winner.PrizeID), result.Winner.SpinID, nullableRewardID(result.Winner.ClaimID), nullableRewardID(result.Winner.OrderID))
}

func recordRewardNotificationEmailStatus(notificationID int64, sent bool, errText string) {
	_, _ = data.GetDatastore().DB.Exec(`UPDATE reward_user_notifications SET email_sent=?, email_error=? WHERE id=?`, boolToInt(sent), nullableRewardString(errText), notificationID)
}

func sanitizeRewardEmailHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func emailErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullableRewardString(value string) interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableRewardID(value int64) interface{} {
	if value <= 0 {
		return nil
	}
	return value
}
