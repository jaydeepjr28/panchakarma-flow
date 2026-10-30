package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// AppointmentStatus represents the lifecycle state of an appointment.
type AppointmentStatus string

const (
	StatusHeld            AppointmentStatus = "HELD"
	StatusPendingApproval AppointmentStatus = "PENDING_APPROVAL"
	StatusConfirmed       AppointmentStatus = "CONFIRMED"
	StatusCheckedIn       AppointmentStatus = "CHECKED_IN"
	StatusInProgress      AppointmentStatus = "IN_PROGRESS"
	StatusCompleted       AppointmentStatus = "COMPLETED"
	StatusCancelled       AppointmentStatus = "CANCELLED"
	StatusRescheduled     AppointmentStatus = "RESCHEDULED"
	StatusNoShow          AppointmentStatus = "NO_SHOW"
	StatusRejected        AppointmentStatus = "REJECTED"
)

// HoldStatus defines the status of provisional reservation holds.
type HoldStatus string

const (
	HoldStatusActive    HoldStatus = "ACTIVE"
	HoldStatusConfirmed HoldStatus = "CONFIRMED"
	HoldStatusExpired   HoldStatus = "EXPIRED"
	HoldStatusReleased  HoldStatus = "RELEASED"
)

// WaitlistStatus defines the state of a waitlist request.
type WaitlistStatus string

const (
	WaitlistStatusWaiting   WaitlistStatus = "WAITING"
	WaitlistStatusOffered   WaitlistStatus = "OFFERED"
	WaitlistStatusExpired   WaitlistStatus = "EXPIRED"
	WaitlistStatusFulfilled WaitlistStatus = "FULFILLED"
	WaitlistStatusCancelled WaitlistStatus = "CANCELLED"
)

// OutboxStatus defines status of transactional outbox records.
type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "PENDING"
	OutboxStatusProcessed OutboxStatus = "PROCESSED"
	OutboxStatusFailed    OutboxStatus = "FAILED"
)

var (
	ErrInvalidStateTransition = errors.New("invalid appointment state transition")
	ErrHoldExpired            = errors.New("appointment hold has expired")
	ErrHoldNotFound           = errors.New("appointment hold not found or inactive")
	ErrSlotDoubleBooked       = errors.New("doctor or room already booked for this slot range")
	ErrAppointmentNotFound    = errors.New("appointment not found")
	ErrWaitlistNotFound       = errors.New("waitlist entry not found")
	ErrWaitlistOfferExpired   = errors.New("waitlist offer has expired")
	ErrUnauthorizedAccess     = errors.New("unauthorized to access or modify this appointment")
)

// validTransitions maps permitted from-state to acceptable to-states.
var validTransitions = map[AppointmentStatus][]AppointmentStatus{
	StatusHeld: {
		StatusPendingApproval,
		StatusConfirmed,
		StatusCancelled,
	},
	StatusPendingApproval: {
		StatusConfirmed,
		StatusRejected,
		StatusCancelled,
	},
	StatusConfirmed: {
		StatusCheckedIn,
		StatusCancelled,
		StatusRescheduled,
		StatusNoShow,
	},
	StatusCheckedIn: {
		StatusInProgress,
		StatusCancelled,
	},
	StatusInProgress: {
		StatusCompleted,
	},
	// Terminal states: no further transitions permitted
	StatusCompleted:   {},
	StatusCancelled:   {},
	StatusRescheduled: {},
	StatusNoShow:      {},
	StatusRejected:    {},
}

// CanTransitionTo checks whether an appointment in fromStatus can move to toStatus.
func CanTransitionTo(current AppointmentStatus, target AppointmentStatus) bool {
	// Terminal states cannot transition to anything
	if current == StatusCompleted || current == StatusCancelled || current == StatusRescheduled || current == StatusNoShow || current == StatusRejected {
		return false
	}
	if current == target {
		return true // No-op idempotent transition for non-terminal states
	}
	allowed, ok := validTransitions[current]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == target {
			return true
		}
	}
	return false
}

// ValidateTransition returns ErrInvalidStateTransition if the transition is prohibited.
func ValidateTransition(current AppointmentStatus, target AppointmentStatus) error {
	if !CanTransitionTo(current, target) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStateTransition, current, target)
	}
	return nil
}

// Appointment represents a booked or confirmed Panchakarma clinical session.
type Appointment struct {
	ID                 string            `json:"id"`
	PatientID          string            `json:"patient_id"`
	ClinicID           string            `json:"clinic_id"`
	DoctorID           string            `json:"doctor_id"`
	RoomID             string            `json:"room_id"`
	PackageID          string            `json:"package_id"`
	StartTime          time.Time         `json:"start_time"`
	EndTime            time.Time         `json:"end_time"`
	Status             AppointmentStatus `json:"status"`
	SessionNumber      int               `json:"session_number"`
	TotalSessions      int               `json:"total_sessions"`
	PackageBookingID   *string           `json:"package_booking_id,omitempty"`
	IdempotencyKey     *string           `json:"idempotency_key,omitempty"`
	Notes              string            `json:"notes,omitempty"`
	CancellationReason string            `json:"cancellation_reason,omitempty"`
	ApprovedBy         *string           `json:"approved_by,omitempty"`
	ApprovedAt         *time.Time        `json:"approved_at,omitempty"`
	RejectionReason    string            `json:"rejection_reason,omitempty"`
	RescheduledToID    *string           `json:"rescheduled_to_id,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// AppointmentHold represents a temporary 10-minute slot lock prior to confirmation.
type AppointmentHold struct {
	ID               string     `json:"id"`
	HoldToken        string     `json:"hold_token"`
	PatientID        string     `json:"patient_id"`
	ClinicID         string     `json:"clinic_id"`
	DoctorID         string     `json:"doctor_id"`
	RoomID           string     `json:"room_id"`
	PackageID        string     `json:"package_id"`
	StartTime        time.Time  `json:"start_time"`
	EndTime          time.Time  `json:"end_time"`
	SessionNumber    int        `json:"session_number"`
	TotalSessions    int        `json:"total_sessions"`
	PackageBookingID *string    `json:"package_booking_id,omitempty"`
	ExpiresAt        time.Time  `json:"expires_at"`
	Status           HoldStatus `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
}

// IsExpired returns true if hold expired based on given current time.
func (h *AppointmentHold) IsExpired(now time.Time) bool {
	return h.Status != HoldStatusActive || !now.Before(h.ExpiresAt)
}

// AppointmentWaitlist allows patients to queue for cancelled slots.
type AppointmentWaitlist struct {
	ID                 string         `json:"id"`
	ClinicID           string         `json:"clinic_id"`
	PatientID          string         `json:"patient_id"`
	PackageID          string         `json:"package_id"`
	PreferredDoctorID  *string        `json:"preferred_doctor_id,omitempty"`
	PreferredDateStart string         `json:"preferred_date_start"` // YYYY-MM-DD
	PreferredDateEnd   string         `json:"preferred_date_end"`   // YYYY-MM-DD
	Status             WaitlistStatus `json:"status"`
	OfferedSlotStart   *time.Time     `json:"offered_slot_start,omitempty"`
	OfferedSlotEnd     *time.Time     `json:"offered_slot_end,omitempty"`
	OfferExpiresAt     *time.Time     `json:"offer_expires_at,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// AppointmentHistory logs audit state transitions.
type AppointmentHistory struct {
	ID            string             `json:"id"`
	AppointmentID string             `json:"appointment_id"`
	FromStatus    *AppointmentStatus `json:"from_status,omitempty"`
	ToStatus      AppointmentStatus  `json:"to_status"`
	ChangedBy     *string            `json:"changed_by,omitempty"`
	Reason        string             `json:"reason,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
}

// OutboxEvent represents an asynchronous domain event to be delivered to Kafka.
type OutboxEvent struct {
	ID            string          `json:"id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	TraceID       string          `json:"trace_id"`
	Status        OutboxStatus    `json:"status"`
	RetryCount    int             `json:"retry_count"`
	CreatedAt     time.Time       `json:"created_at"`
	ProcessedAt   *time.Time      `json:"processed_at,omitempty"`
}

// SearchSlot represents an available slot candidate.
type SearchSlot struct {
	ClinicID   string    `json:"clinic_id"`
	DoctorID   string    `json:"doctor_id"`
	DoctorName string    `json:"doctor_name,omitempty"`
	RoomID     string    `json:"room_id"`
	PackageID  string    `json:"package_id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	DurationM  int       `json:"duration_minutes"`
}
