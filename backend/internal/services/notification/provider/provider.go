package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/domain"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
)

// EmailSender abstracts email transmission providers (SendGrid, AWS SES, SMTP, Mock).
type EmailSender interface {
	SendEmail(ctx context.Context, notifID, recipient, subject, body string) (*domain.DeliveryAttempt, error)
}

// SMSSender abstracts SMS gateways (Twilio, Gupshup, Mock).
type SMSSender interface {
	SendSMS(ctx context.Context, notifID, recipientPhone, message string) (*domain.DeliveryAttempt, error)
}

// PushSender abstracts mobile push messaging (Firebase Cloud Messaging, Apple APNs, Mock).
type PushSender interface {
	SendPush(ctx context.Context, notifID, deviceToken, title, message string, data map[string]string) (*domain.DeliveryAttempt, error)
}

// DeliveryProvider aggregates all outbound communication channels.
type DeliveryProvider interface {
	EmailSender
	SMSSender
	PushSender
}

// SentMessage represents a record of a dispatched message in the development adapter.
type SentMessage struct {
	ID        string            `json:"id"`
	Channel   string            `json:"channel"`
	Recipient string            `json:"recipient"` // Masked for privacy
	Subject   string            `json:"subject,omitempty"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data,omitempty"`
	SentAt    time.Time         `json:"sent_at"`
}

// DevelopmentDeliveryAdapter safely records sanitized deliveries without external calls.
type DevelopmentDeliveryAdapter struct {
	mu           sync.RWMutex
	sentMessages []SentMessage
	failures     map[string]bool // channel -> shouldFail
}

// NewDevelopmentDeliveryAdapter initializes the development adapter.
func NewDevelopmentDeliveryAdapter() *DevelopmentDeliveryAdapter {
	return &DevelopmentDeliveryAdapter{
		sentMessages: make([]SentMessage, 0),
		failures:     make(map[string]bool),
	}
}

// MaskEmail masks email addresses for privacy (e.g., k****a@example.com).
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || len(parts[0]) <= 1 {
		return "***@masked.domain"
	}
	user := parts[0]
	if len(user) <= 2 {
		return fmt.Sprintf("%c*@%s", user[0], parts[1])
	}
	return fmt.Sprintf("%c%s%c@%s", user[0], strings.Repeat("*", len(user)-2), user[len(user)-1], parts[1])
}

// MaskPhone masks phone numbers for privacy (e.g., +9198****3210).
func MaskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) < 6 {
		return "******"
	}
	prefix := phone[:4]
	suffix := phone[len(phone)-3:]
	maskedLen := len(phone) - 7
	if maskedLen < 2 {
		maskedLen = 3
	}
	return fmt.Sprintf("%s%s%s", prefix, strings.Repeat("*", maskedLen), suffix)
}

// SendEmail records a simulated email.
func (d *DevelopmentDeliveryAdapter) SendEmail(ctx context.Context, notifID, recipient, subject, body string) (*domain.DeliveryAttempt, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC()
	sanitized := MaskEmail(recipient)

	attempt := &domain.DeliveryAttempt{
		ID:             uuid.New().String(),
		NotificationID: notifID,
		Channel:        "email",
		Recipient:      sanitized,
		Provider:       "dev-email-adapter",
		AttemptNumber:  1,
		DispatchedAt:   &now,
		CreatedAt:      now,
	}

	if d.failures["email"] {
		attempt.Status = domain.DeliveryStatusFailed
		errMsg := "simulated email provider downtime / rate-limit"
		attempt.ErrorMessage = &errMsg
		logger.Warn("DevelopmentDeliveryAdapter simulated email failure", "recipient", sanitized, "subject", subject)
		return attempt, fmt.Errorf("email delivery failed: %s", errMsg)
	}

	attempt.Status = domain.DeliveryStatusDelivered
	d.sentMessages = append(d.sentMessages, SentMessage{
		ID:        attempt.ID,
		Channel:   "email",
		Recipient: sanitized,
		Subject:   subject,
		Body:      body,
		SentAt:    now,
	})

	logger.Info("DevelopmentDeliveryAdapter [EMAIL SENT]", "recipient", sanitized, "subject", subject)
	return attempt, nil
}

// SendSMS records a simulated SMS.
func (d *DevelopmentDeliveryAdapter) SendSMS(ctx context.Context, notifID, recipientPhone, message string) (*domain.DeliveryAttempt, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC()
	sanitized := MaskPhone(recipientPhone)

	attempt := &domain.DeliveryAttempt{
		ID:             uuid.New().String(),
		NotificationID: notifID,
		Channel:        "sms",
		Recipient:      sanitized,
		Provider:       "dev-sms-adapter",
		AttemptNumber:  1,
		DispatchedAt:   &now,
		CreatedAt:      now,
	}

	if d.failures["sms"] {
		attempt.Status = domain.DeliveryStatusFailed
		errMsg := "simulated SMS gateway gateway-timeout"
		attempt.ErrorMessage = &errMsg
		logger.Warn("DevelopmentDeliveryAdapter simulated SMS failure", "recipient", sanitized)
		return attempt, fmt.Errorf("sms delivery failed: %s", errMsg)
	}

	attempt.Status = domain.DeliveryStatusDelivered
	d.sentMessages = append(d.sentMessages, SentMessage{
		ID:        attempt.ID,
		Channel:   "sms",
		Recipient: sanitized,
		Body:      message,
		SentAt:    now,
	})

	logger.Info("DevelopmentDeliveryAdapter [SMS SENT]", "recipient", sanitized, "message_preview", message)
	return attempt, nil
}

// SendPush records a simulated Push notification.
func (d *DevelopmentDeliveryAdapter) SendPush(ctx context.Context, notifID, deviceToken, title, message string, data map[string]string) (*domain.DeliveryAttempt, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now().UTC()
	maskedToken := "push-token-***"
	if len(deviceToken) > 6 {
		maskedToken = deviceToken[:3] + "***" + deviceToken[len(deviceToken)-3:]
	}

	attempt := &domain.DeliveryAttempt{
		ID:             uuid.New().String(),
		NotificationID: notifID,
		Channel:        "push",
		Recipient:      maskedToken,
		Provider:       "dev-push-adapter",
		AttemptNumber:  1,
		DispatchedAt:   &now,
		CreatedAt:      now,
	}

	if d.failures["push"] {
		attempt.Status = domain.DeliveryStatusFailed
		errMsg := "simulated APNs/FCM connection reset"
		attempt.ErrorMessage = &errMsg
		return attempt, fmt.Errorf("push delivery failed: %s", errMsg)
	}

	attempt.Status = domain.DeliveryStatusDelivered
	d.sentMessages = append(d.sentMessages, SentMessage{
		ID:        attempt.ID,
		Channel:   "push",
		Recipient: maskedToken,
		Subject:   title,
		Body:      message,
		Data:      data,
		SentAt:    now,
	})

	logger.Info("DevelopmentDeliveryAdapter [PUSH SENT]", "recipient", maskedToken, "title", title)
	return attempt, nil
}

// SetFailureSimulation toggles simulated delivery failures for testing retries and resilience.
func (d *DevelopmentDeliveryAdapter) SetFailureSimulation(channel string, shouldFail bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.failures[channel] = shouldFail
}

// GetSentMessages returns a clone of recorded messages.
func (d *DevelopmentDeliveryAdapter) GetSentMessages() []SentMessage {
	d.mu.RLock()
	defer d.mu.RUnlock()
	copied := make([]SentMessage, len(d.sentMessages))
	copy(copied, d.sentMessages)
	return copied
}

// ClearLogs clears recorded delivery logs.
func (d *DevelopmentDeliveryAdapter) ClearLogs() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sentMessages = make([]SentMessage, 0)
}
