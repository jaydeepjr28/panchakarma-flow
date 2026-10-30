package domain

import (
	"encoding/json"
	"errors"
	"time"
)

// Common domain errors
var (
	ErrNotificationNotFound  = errors.New("notification not found")
	ErrDuplicateEvent        = errors.New("event has already been processed by consumer")
	ErrReminderAlreadyExists = errors.New("scheduled reminder already registered for target event")
	ErrInvalidPreference     = errors.New("invalid notification preference specification")
)

type Priority string

const (
	PriorityLow       Priority = "LOW"
	PriorityNormal    Priority = "NORMAL"
	PriorityHigh      Priority = "HIGH"
	PriorityEmergency Priority = "EMERGENCY"
)

type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "PENDING"
	DeliveryStatusDelivered DeliveryStatus = "DELIVERED"
	DeliveryStatusFailed    DeliveryStatus = "FAILED"
	DeliveryStatusCancelled DeliveryStatus = "CANCELLED"
	DeliveryStatusSkipped   DeliveryStatus = "SKIPPED"
)

type ReminderStatus string

const (
	ReminderStatusScheduled  ReminderStatus = "SCHEDULED"
	ReminderStatusDispatched ReminderStatus = "DISPATCHED"
	ReminderStatusCancelled  ReminderStatus = "CANCELLED"
	ReminderStatusSkipped    ReminderStatus = "SKIPPED"
)

// Notification represents an in-app or cross-channel alert record for a user.
type Notification struct {
	ID                string          `json:"id"`
	UserID            string          `json:"user_id"`
	Title             string          `json:"title"`
	Message           string          `json:"message"`
	NotificationType  string          `json:"notification_type"`
	Priority          Priority        `json:"priority"`
	ReferenceID       *string         `json:"reference_id,omitempty"`
	ReferenceType     *string         `json:"reference_type,omitempty"`
	IsRead            bool            `json:"is_read"`
	ReadAt            *time.Time      `json:"read_at,omitempty"`
	ChannelDispatches json.RawMessage `json:"channel_dispatches"`
	CreatedAt         time.Time       `json:"created_at"`
}

// NotificationPreference stores per-user delivery channel settings.
type NotificationPreference struct {
	UserID          string    `json:"user_id"`
	EmailEnabled    bool      `json:"email_enabled"`
	SMSEnabled      bool      `json:"sms_enabled"`
	WhatsAppEnabled bool      `json:"whatsapp_enabled"`
	InAppEnabled    bool      `json:"in_app_enabled"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// DeliveryAttempt logs individual dispatch actions with sanitized audit records.
type DeliveryAttempt struct {
	ID             string         `json:"id"`
	NotificationID string         `json:"notification_id"`
	Channel        string         `json:"channel"`
	Recipient      string         `json:"recipient"` // Sanitized / masked
	Status         DeliveryStatus `json:"status"`
	Provider       string         `json:"provider"`
	AttemptNumber  int            `json:"attempt_number"`
	ErrorMessage   *string        `json:"error_message,omitempty"`
	DispatchedAt   *time.Time     `json:"dispatched_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// ScheduledReminder holds delayed clinical and preparation reminders.
type ScheduledReminder struct {
	ID              string         `json:"id"`
	AppointmentID   string         `json:"appointment_id"`
	PatientID       string         `json:"patient_id"`
	ClinicID        string         `json:"clinic_id"`
	ReminderType    string         `json:"reminder_type"`
	ScheduledTime   time.Time      `json:"scheduled_time"`
	TargetEventTime time.Time      `json:"target_event_time"`
	Status          ReminderStatus `json:"status"`
	IdempotencyKey  string         `json:"idempotency_key"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// OutboxEvent represents transactional outbox records.
type OutboxEvent struct {
	ID            string          `json:"id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	TraceID       string          `json:"trace_id"`
	Status        string          `json:"status"`
	RetryCount    int             `json:"retry_count"`
	CreatedAt     time.Time       `json:"created_at"`
	ProcessedAt   *time.Time      `json:"processed_at,omitempty"`
}

// Event Payloads

type AppointmentConfirmedPayload struct {
	AppointmentID  string `json:"appointment_id"`
	PatientID      string `json:"patient_id"`
	ClinicID       string `json:"clinic_id"`
	DoctorID       string `json:"doctor_id"`
	RoomID         string `json:"room_id"`
	PackageID      string `json:"package_id"`
	PackageName    string `json:"package_name"`
	ScheduledStart string `json:"scheduled_start"`
	ScheduledEnd   string `json:"scheduled_end"`
	Notes          string `json:"notes,omitempty"`
}

type AppointmentCancelledPayload struct {
	AppointmentID string `json:"appointment_id"`
	PatientID     string `json:"patient_id"`
	ClinicID      string `json:"clinic_id"`
	DoctorID      string `json:"doctor_id"`
	Reason        string `json:"reason,omitempty"`
	CancelledBy   string `json:"cancelled_by,omitempty"`
}

type AppointmentRescheduledPayload struct {
	AppointmentID string `json:"appointment_id"`
	PatientID     string `json:"patient_id"`
	ClinicID      string `json:"clinic_id"`
	DoctorID      string `json:"doctor_id"`
	OldStartTime  string `json:"old_start_time"`
	NewStartTime  string `json:"new_start_time"`
	Reason        string `json:"reason,omitempty"`
}

type SessionStartedPayload struct {
	SessionID     string `json:"session_id"`
	AppointmentID string `json:"appointment_id"`
	PatientID     string `json:"patient_id"`
	ClinicID      string `json:"clinic_id"`
	StartedAt     string `json:"started_at"`
}

type SessionCompletedPayload struct {
	SessionID             string `json:"session_id"`
	AppointmentID         string `json:"appointment_id"`
	PatientID             string `json:"patient_id"`
	ClinicID              string `json:"clinic_id"`
	CompletedAt           string `json:"completed_at"`
	PostCareInstructions string `json:"post_care_instructions,omitempty"`
}

type ComplicationFlaggedPayload struct {
	ComplicationID    string `json:"complication_id"`
	SessionID         string `json:"session_id"`
	AppointmentID     string `json:"appointment_id"`
	PatientID         string `json:"patient_id"`
	ClinicID          string `json:"clinic_id"`
	AttendingDoctorID string `json:"attending_doctor_id"`
	Severity          string `json:"severity"`
	Symptoms          string `json:"symptoms"`
	Timestamp         string `json:"timestamp"`
}

type ComplicationResolvedPayload struct {
	ComplicationID string `json:"complication_id"`
	SessionID      string `json:"session_id"`
	DoctorID       string `json:"doctor_id"`
	ResolvedAt     string `json:"resolved_at"`
}

type FeedbackRequestedPayload struct {
	AppointmentID string `json:"appointment_id"`
	PatientID     string `json:"patient_id"`
	ClinicID      string `json:"clinic_id"`
}

type FeedbackSubmittedPayload struct {
	FeedbackID    string `json:"feedback_id"`
	AppointmentID string `json:"appointment_id"`
	PatientID     string `json:"patient_id"`
	ClinicID      string `json:"clinic_id"`
	Rating        int    `json:"rating"`
	Comments      string `json:"comments,omitempty"`
}

type ReminderDuePayload struct {
	ReminderID    string `json:"reminder_id"`
	AppointmentID string `json:"appointment_id"`
	PatientID     string `json:"patient_id"`
	ClinicID      string `json:"clinic_id"`
	ReminderType  string `json:"reminder_type"`
	ScheduledTime string `json:"scheduled_time"`
}

type FollowUpReminderPayload struct {
	FollowUpID             string `json:"follow_up_id"`
	SessionID              string `json:"session_id"`
	PatientID              string `json:"patient_id"`
	ClinicID               string `json:"clinic_id"`
	RecommendedDate        string `json:"recommended_date"`
	Instructions           string `json:"instructions,omitempty"`
	RecommendedPackageName string `json:"recommended_package_name,omitempty"`
}
