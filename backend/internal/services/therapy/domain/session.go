package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// SessionStatus defines the lifecycle of a therapy execution.
type SessionStatus string

const (
	SessionStatusCheckedIn         SessionStatus = "CHECKED_IN"
	SessionStatusInProgress        SessionStatus = "IN_PROGRESS"
	SessionStatusComplicationAlert SessionStatus = "COMPLICATION_ALERT"
	SessionStatusCompleted         SessionStatus = "COMPLETED"
	SessionStatusAbortedClinical   SessionStatus = "ABORTED_CLINICAL"
)

// StepStatus defines the state of a single therapy step.
type StepStatus string

const (
	StepStatusNotStarted StepStatus = "NOT_STARTED"
	StepStatusInProgress StepStatus = "IN_PROGRESS"
	StepStatusPaused     StepStatus = "PAUSED"
	StepStatusCompleted  StepStatus = "COMPLETED"
	StepStatusSkipped    StepStatus = "SKIPPED"
)

// ComplicationSeverity indicates clinical risk level.
type ComplicationSeverity string

const (
	SeverityLow      ComplicationSeverity = "LOW"
	SeverityMedium   ComplicationSeverity = "MEDIUM"
	SeverityHigh     ComplicationSeverity = "HIGH"
	SeverityCritical ComplicationSeverity = "CRITICAL"
)

// ComplicationStatus tracks review state by attending physician.
type ComplicationStatus string

const (
	ComplicationUnresolved     ComplicationStatus = "UNRESOLVED"
	ComplicationDoctorReviewed ComplicationStatus = "DOCTOR_REVIEWED"
	ComplicationResolved       ComplicationStatus = "RESOLVED"
)

var (
	ErrInvalidSessionTransition = errors.New("invalid session state transition")
	ErrInvalidStepTransition    = errors.New("invalid step state transition")
	ErrSessionCompleted         = errors.New("cannot modify completed or clinically aborted session")
	ErrSessionNotFound          = errors.New("therapy session not found")
	ErrDuplicateActiveSession   = errors.New("active session already exists for this appointment")
	ErrStepNotFound             = errors.New("therapy step not found")
	ErrStepOutOfOrder           = errors.New("previous therapy step must be completed before starting next step")
	ErrUnauthorizedAccess       = errors.New("unauthorized to access or modify this therapy session")
	ErrPreChecklistRequired     = errors.New("pre-therapy checklist verification is required before starting session")
	ErrCorrectionReasonRequired = errors.New("justification reason is required for clinical record correction")
	ErrComplicationNotFound     = errors.New("complication record not found")
	ErrIncompleteSteps          = errors.New("all therapy steps must be completed or skipped before completing session")
	ErrUnauthorizedCrossClinic  = errors.New("unauthorized cross-clinic access")
	ErrUnauthorizedPatientAccess = errors.New("unauthorized patient access to therapy session")
)

// validSessionTransitions maps allowable transitions for therapy sessions.
var validSessionTransitions = map[SessionStatus][]SessionStatus{
	SessionStatusCheckedIn: {
		SessionStatusInProgress,
		SessionStatusAbortedClinical,
	},
	SessionStatusInProgress: {
		SessionStatusComplicationAlert,
		SessionStatusCompleted,
		SessionStatusAbortedClinical,
	},
	SessionStatusComplicationAlert: {
		SessionStatusInProgress,
		SessionStatusAbortedClinical,
	},
	// Terminal states: immutable
	SessionStatusCompleted:       {},
	SessionStatusAbortedClinical: {},
}

// CanTransitionSession checks if a session transition is allowed.
func CanTransitionSession(current, target SessionStatus) bool {
	if current == target {
		return true // Idempotent
	}
	allowed, ok := validSessionTransitions[current]
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

// ValidateSessionTransition returns error if session transition is prohibited.
func ValidateSessionTransition(current, target SessionStatus) error {
	if !CanTransitionSession(current, target) {
		return fmt.Errorf("%w: cannot transition session from %s to %s", ErrInvalidSessionTransition, current, target)
	}
	return nil
}

// validStepTransitions maps allowable transitions for procedural steps.
var validStepTransitions = map[StepStatus][]StepStatus{
	StepStatusNotStarted: {
		StepStatusInProgress,
		StepStatusSkipped,
	},
	StepStatusInProgress: {
		StepStatusPaused,
		StepStatusCompleted,
	},
	StepStatusPaused: {
		StepStatusInProgress,
		StepStatusCompleted,
		StepStatusSkipped,
	},
	// Terminal states for steps
	StepStatusCompleted: {},
	StepStatusSkipped:   {},
}

// CanTransitionStep checks if step transition is allowed.
func CanTransitionStep(current, target StepStatus) bool {
	if current == target {
		return true
	}
	allowed, ok := validStepTransitions[current]
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

// ValidateStepTransition returns error if step transition is prohibited.
func ValidateStepTransition(current, target StepStatus) error {
	if !CanTransitionStep(current, target) {
		return fmt.Errorf("%w: cannot transition step from %s to %s", ErrInvalidStepTransition, current, target)
	}
	return nil
}

// BaselineVitals captures vital signs taken at check-in or incident.
type BaselineVitals struct {
	BPSystolic   int     `json:"bp_systolic"`
	BPDiastolic  int     `json:"bp_diastolic"`
	Pulse        int     `json:"pulse"`
	TemperatureC float64 `json:"temperature_c"`
}

// TherapySession represents the hands-on clinical execution of a scheduled appointment.
type TherapySession struct {
	ID                   string          `json:"id"`
	AppointmentID        string          `json:"appointment_id"`
	PatientID            string          `json:"patient_id"`
	ClinicID             string          `json:"clinic_id"`
	DoctorID             string          `json:"doctor_id"`
	TherapistID          string          `json:"therapist_id"`
	RoomID               string          `json:"room_id"`
	PackageID            string          `json:"package_id"`
	PackageVersion       int             `json:"package_version"`
	Status               SessionStatus   `json:"status"`
	BaselineVitals       json.RawMessage `json:"baseline_vitals"`
	PreChecklistVerified bool            `json:"pre_checklist_verified"`
	PreChecklistData     json.RawMessage `json:"pre_checklist_data,omitempty"`
	TherapistNotes       string          `json:"therapist_notes,omitempty"`
	Observations         string          `json:"observations,omitempty"`
	PostCareInstructions string          `json:"post_care_instructions,omitempty"`
	PostCareItems        json.RawMessage `json:"post_care_items,omitempty"`
	PostCareDispatched   bool            `json:"post_care_dispatched"`
	StartedAt            *time.Time      `json:"started_at,omitempty"`
	CompletedAt          *time.Time      `json:"completed_at,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`

	Steps     []StepExecution    `json:"steps,omitempty"`
	Materials []SessionMaterial  `json:"materials,omitempty"`
	FollowUps []SessionFollowUp  `json:"follow_ups,omitempty"`
}

// IsTerminal returns true if session has finished and is immutable.
func (s *TherapySession) IsTerminal() bool {
	return s.Status == SessionStatusCompleted || s.Status == SessionStatusAbortedClinical
}

// StepExecution represents the execution and timing of an individual therapy step.
type StepExecution struct {
	ID                     string     `json:"id"`
	SessionID              string     `json:"session_id"`
	StepID                 string     `json:"step_id"`
	StepOrder              int        `json:"step_order"`
	Name                   string     `json:"name"`
	PlannedDurationMinutes int        `json:"planned_duration_minutes"`
	ActualDurationSeconds  int        `json:"actual_duration_seconds"`
	Status                 StepStatus `json:"status"`
	StartedAt              *time.Time `json:"started_at,omitempty"`
	PausedAt               *time.Time `json:"paused_at,omitempty"`
	CompletedAt            *time.Time `json:"completed_at,omitempty"`
	TherapistNotes         string     `json:"therapist_notes,omitempty"`
	Observations           string     `json:"observations,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
}

// SessionMaterial records oils, decoctions, and medical supplies consumed.
type SessionMaterial struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	MaterialName string    `json:"material_name"`
	Quantity     float64   `json:"quantity"`
	Unit         string    `json:"unit"` // 'ml', 'grams', 'units'
	BatchNumber  string    `json:"batch_number,omitempty"`
	Notes        string    `json:"notes,omitempty"`
	RecordedBy   string    `json:"recorded_by"`
	CreatedAt    time.Time `json:"created_at"`
}

// ComplicationEvent documents adverse symptoms requiring clinical intervention.
type ComplicationEvent struct {
	ID                    string               `json:"id"`
	SessionID             string               `json:"session_id"`
	ReporterID            string               `json:"reporter_id"`
	ReporterRole          string               `json:"reporter_role"`
	Severity              ComplicationSeverity `json:"severity"`
	Symptoms              string               `json:"symptoms"`
	VitalsAtIncident      json.RawMessage      `json:"vitals_at_incident"`
	ImmediateActionsTaken string               `json:"immediate_actions_taken"`
	Status                ComplicationStatus   `json:"status"`
	AttendingDoctorID     string               `json:"attending_doctor_id"`
	DoctorNotes           string               `json:"doctor_notes,omitempty"`
	DoctorClearedAt       *time.Time           `json:"doctor_cleared_at,omitempty"`
	CreatedAt             time.Time            `json:"created_at"`
}

// SessionFollowUp records practitioner follow-up recommendations.
type SessionFollowUp struct {
	ID                     string     `json:"id"`
	SessionID              string     `json:"session_id"`
	RecommendedBy          string     `json:"recommended_by"`
	RecommendedPackageID   *string    `json:"recommended_package_id,omitempty"`
	RecommendedIntervalDay *int       `json:"recommended_interval_days,omitempty"`
	RecommendedDate        *string    `json:"recommended_date,omitempty"` // YYYY-MM-DD
	Instructions           string     `json:"instructions"`
	LinkedAppointmentID    *string    `json:"linked_appointment_id,omitempty"`
	Status                 string     `json:"status"` // 'PENDING', 'SCHEDULED', 'DISMISSED'
	CreatedAt              time.Time  `json:"created_at"`
}

// SessionAuditLog tracks historical corrections to clinical notes or vitals.
type SessionAuditLog struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	FieldName   string    `json:"field_name"`
	OldValue    string    `json:"old_value,omitempty"`
	NewValue    string    `json:"new_value"`
	Reason      string    `json:"reason"`
	CorrectedBy string    `json:"corrected_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// OutboxEvent represents an asynchronous event for Kafka.
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
