package domain

import (
	"encoding/json"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
)

// Common Domain Errors
var (
	ErrDuplicateFeedback       = errors.Conflict("", "Feedback has already been submitted for this appointment", nil)
	ErrAppointmentNotEligible  = errors.BadRequest("Feedback can only be submitted for completed appointments", nil)
	ErrUnauthorizedPatient     = errors.Forbidden("You can only submit feedback for your own appointments", nil)
	ErrFeedbackNotFound        = errors.NotFound("Feedback record not found", nil)
	ErrAdverseEventNotFound    = errors.NotFound("Adverse event record not found", nil)
	ErrCrossClinicAccessDenied = errors.Forbidden("Access denied: You cannot view reports or analytics for another clinic", nil)
	ErrInvalidDateRange        = errors.BadRequest("Invalid date range: start_date cannot be after end_date", nil)
	ErrInvalidRating           = errors.BadRequest("Ratings must be an integer between 1 and 5", nil)
)

// PatientFeedback represents patient rating, comments, and clinic review.
type PatientFeedback struct {
	ID                          string     `json:"id"`
	AppointmentID               string     `json:"appointment_id"`
	SessionID                   *string    `json:"session_id,omitempty"`
	PatientID                   string     `json:"patient_id"`
	ClinicID                    string     `json:"clinic_id"`
	DoctorID                    string     `json:"doctor_id"`
	TherapistID                 string     `json:"therapist_id"`
	OverallRating               int        `json:"overall_rating"`
	TherapistRating             int        `json:"therapist_rating"`
	FacilityCleanlinessRating   int        `json:"facility_cleanliness_rating"`
	Comments                    *string    `json:"comments,omitempty"`
	IsAnonymous                 bool       `json:"is_anonymous"`
	Status                      string     `json:"status"` // 'SUBMITTED', 'ACKNOWLEDGED', 'RESOLVED'
	AcknowledgedAt              *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy              *string    `json:"acknowledged_by,omitempty"`
	StaffNotes                  *string    `json:"staff_notes,omitempty"`
	CreatedAt                   time.Time  `json:"created_at"`
}

// SubmitFeedbackRequest is the payload sent by a patient to rate their completed appointment.
type SubmitFeedbackRequest struct {
	AppointmentID               string  `json:"appointment_id" binding:"required"`
	SessionID                   *string `json:"session_id,omitempty"`
	ClinicID                    string  `json:"clinic_id"`
	DoctorID                    string  `json:"doctor_id"`
	TherapistID                 string  `json:"therapist_id"`
	OverallRating               int     `json:"overall_rating" binding:"required,min=1,max=5"`
	TherapistRating             int     `json:"therapist_rating" binding:"required,min=1,max=5"`
	FacilityCleanlinessRating   int     `json:"facility_cleanliness_rating" binding:"required,min=1,max=5"`
	Comments                    *string `json:"comments,omitempty"`
	IsAnonymous                 bool    `json:"is_anonymous"`
}

// AcknowledgeFeedbackRequest represents clinic staff review/acknowledgment.
type AcknowledgeFeedbackRequest struct {
	StaffNotes string `json:"staff_notes" binding:"required"`
	Status     string `json:"status,omitempty"` // 'ACKNOWLEDGED' or 'RESOLVED'
}

// AdverseEvent represents a recorded clinical complication or reaction.
type AdverseEvent struct {
	ID                    string                 `json:"id"`
	SessionID             *string                `json:"session_id,omitempty"`
	AppointmentID         *string                `json:"appointment_id,omitempty"`
	PatientID             string                 `json:"patient_id"`
	ClinicID              string                 `json:"clinic_id"`
	Severity              string                 `json:"severity"` // 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'
	Symptoms              string                 `json:"symptoms"`
	VitalsAtIncident      map[string]interface{} `json:"vitals_at_incident,omitempty"`
	ImmediateActionsTaken *string                `json:"immediate_actions_taken,omitempty"`
	Status                string                 `json:"status"` // 'REPORTED', 'UNDER_REVIEW', 'ACKNOWLEDGED', 'RESOLVED'
	ReportedBy            string                 `json:"reported_by"`
	ReporterRole          string                 `json:"reporter_role"`
	AttendingDoctorID     *string                `json:"attending_doctor_id,omitempty"`
	AcknowledgedAt        *time.Time             `json:"acknowledged_at,omitempty"`
	AcknowledgedBy        *string                `json:"acknowledged_by,omitempty"`
	StaffNotes            *string                `json:"staff_notes,omitempty"`
	ResolvedAt            *time.Time             `json:"resolved_at,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// ReportAdverseEventRequest is the payload to report a clinical adverse incident.
type ReportAdverseEventRequest struct {
	SessionID             *string                `json:"session_id,omitempty"`
	AppointmentID         *string                `json:"appointment_id,omitempty"`
	PatientID             string                 `json:"patient_id"`
	ClinicID              string                 `json:"clinic_id" binding:"required"`
	Severity              string                 `json:"severity" binding:"required,oneof=LOW MEDIUM HIGH CRITICAL"`
	Symptoms              string                 `json:"symptoms" binding:"required"`
	VitalsAtIncident      map[string]interface{} `json:"vitals_at_incident,omitempty"`
	ImmediateActionsTaken *string                `json:"immediate_actions_taken,omitempty"`
	AttendingDoctorID     *string                `json:"attending_doctor_id,omitempty"`
}

// AcknowledgeAdverseEventRequest is sent by doctor or clinic admin to acknowledge an issue.
type AcknowledgeAdverseEventRequest struct {
	StaffNotes string `json:"staff_notes" binding:"required"`
	Status     string `json:"status" binding:"required,oneof=UNDER_REVIEW ACKNOWLEDGED RESOLVED"`
}

// AnalyticsProjections represent cached operational data for rapid metric computation.

type AnalyticsAppointment struct {
	ID                  string    `json:"id"`
	ClinicID            string    `json:"clinic_id"`
	PatientID           string    `json:"patient_id"`
	DoctorID            *string   `json:"doctor_id,omitempty"`
	RoomID              *string   `json:"room_id,omitempty"`
	PackageID           *string   `json:"package_id,omitempty"`
	PackageName         *string   `json:"package_name,omitempty"`
	Status              string    `json:"status"`
	ScheduledStart      time.Time `json:"scheduled_start"`
	ScheduledEnd        time.Time `json:"scheduled_end"`
	IsFollowUp          bool      `json:"is_follow_up"`
	ParentAppointmentID *string   `json:"parent_appointment_id,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type AnalyticsSession struct {
	ID                  string     `json:"id"`
	AppointmentID       string     `json:"appointment_id"`
	ClinicID            string     `json:"clinic_id"`
	PatientID           string     `json:"patient_id"`
	TherapistID         *string    `json:"therapist_id,omitempty"`
	Status              string     `json:"status"`
	AllStepsCompleted   bool       `json:"all_steps_completed"`
	TotalSteps          int        `json:"total_steps"`
	CompletedSteps      int        `json:"completed_steps"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	DurationMinutes     *float64   `json:"duration_minutes,omitempty"`
	FollowUpRecommended bool       `json:"follow_up_recommended"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// Metric Structures with Numerator, Denominator, and Documentation

type MetricResult struct {
	Value       float64 `json:"value"`
	Numerator   float64 `json:"numerator"`
	Denominator float64 `json:"denominator"`
	Unit        string  `json:"unit"`
	Description string  `json:"description"`
}

type ComplicationBreakdown struct {
	TotalCount   int64            `json:"total_count"`
	ResolvedRate float64          `json:"resolved_rate_pct"`
	BySeverity   map[string]int64 `json:"by_severity"`
	ByStatus     map[string]int64 `json:"by_status"`
}

type DailyMetricRollup struct {
	Date                 string   `json:"date"`
	TotalAppointments    int64    `json:"total_appointments"`
	CompletedSessions    int64    `json:"completed_sessions"`
	CancelledCount       int64    `json:"cancelled_count"`
	NoShowCount          int64    `json:"no_show_count"`
	AvgSessionMinutes    float64  `json:"avg_session_minutes"`
	AverageRating        *float64 `json:"average_rating,omitempty"`
	ComplicationsReported int64   `json:"complications_reported"`
}

type ClinicDashboardMetrics struct {
	ClinicID             string                 `json:"clinic_id"`
	StartDate            string                 `json:"start_date"`
	EndDate              string                 `json:"end_date"`
	Timezone             string                 `json:"timezone"`
	BookingConversion    MetricResult           `json:"booking_conversion"`
	CancellationRate     MetricResult           `json:"cancellation_rate"`
	NoShowRate           MetricResult           `json:"no_show_rate"`
	ChecklistAdherence   MetricResult           `json:"checklist_adherence"`
	AverageSessionDuration MetricResult         `json:"average_session_duration"`
	AverageFeedbackRating MetricResult          `json:"average_feedback_rating"`
	TherapistUtilization MetricResult           `json:"therapist_utilization"`
	FollowUpCompletion   MetricResult           `json:"follow_up_completion"`
	Complications        ComplicationBreakdown  `json:"complications"`
	DailyTrends          []DailyMetricRollup    `json:"daily_trends"`
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
type FeedbackSubmittedEventPayload struct {
	FeedbackID    string `json:"feedback_id"`
	AppointmentID string `json:"appointment_id"`
	PatientID     string `json:"patient_id"`
	ClinicID      string `json:"clinic_id"`
	Rating        int    `json:"rating"`
	Comments      string `json:"comments,omitempty"`
}
