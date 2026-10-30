package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	appointmentv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/appointment/v1"
	clinicv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/clinic/v1"
	"google.golang.org/grpc"
)

// AppointmentVerifier validates appointment eligibility before therapy check-in.
type AppointmentVerifier interface {
	VerifyAppointmentForSession(ctx context.Context, appointmentID, clinicID string) (isEligible bool, rejectionReason string, patientID, doctorID, roomID, packageID string, err error)
}

// PackageSnapshotProvider retrieves therapy step sequence for a package.
type PackageSnapshotProvider interface {
	GetPackageSteps(ctx context.Context, packageID string) ([]domain.StepExecution, error)
}

// gRPCAppointmentVerifier calls Appointment Service via gRPC with deadlines.
type gRPCAppointmentVerifier struct {
	client appointmentv1.AppointmentServiceClient
}

func NewGRPCAppointmentVerifier(cc grpc.ClientConnInterface) AppointmentVerifier {
	if cc == nil {
		return nil
	}
	return &gRPCAppointmentVerifier{client: appointmentv1.NewAppointmentServiceClient(cc)}
}

func (g *gRPCAppointmentVerifier) VerifyAppointmentForSession(ctx context.Context, appointmentID, clinicID string) (bool, string, string, string, string, string, error) {
	if g == nil || g.client == nil {
		return true, "", "pat_fallback", "doc_fallback", "room_fallback", "pkg_fallback", nil
	}

	callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := g.client.VerifyAppointmentForSession(callCtx, &appointmentv1.VerifyAppointmentRequest{
		AppointmentId: appointmentID,
		ClinicId:      clinicID,
	})
	if err != nil {
		logger.Warn("Appointment service gRPC call failed, allowing fallback", "error", err)
		return true, "", "", "", "", "", nil
	}

	return resp.IsEligible, resp.RejectionReason, resp.PatientId, resp.DoctorId, resp.RoomId, resp.PackageId, nil
}

// gRPCPackageSnapshotProvider retrieves ordered steps from Clinic Service.
type gRPCPackageSnapshotProvider struct {
	client clinicv1.ClinicServiceClient
}

func NewGRPCPackageSnapshotProvider(cc grpc.ClientConnInterface) PackageSnapshotProvider {
	if cc == nil {
		return nil
	}
	return &gRPCPackageSnapshotProvider{client: clinicv1.NewClinicServiceClient(cc)}
}

func (g *gRPCPackageSnapshotProvider) GetPackageSteps(ctx context.Context, packageID string) ([]domain.StepExecution, error) {
	if g == nil || g.client == nil {
		return defaultSteps(), nil
	}

	callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := g.client.GetPackageSnapshot(callCtx, &clinicv1.GetPackageSnapshotRequest{
		PackageId: packageID,
	})
	if err != nil || len(resp.Steps) == 0 {
		return defaultSteps(), nil
	}

	var steps []domain.StepExecution
	for _, s := range resp.Steps {
		steps = append(steps, domain.StepExecution{
			ID:                     uuid.New().String(),
			StepID:                 s.StepId,
			StepOrder:              int(s.StepOrder),
			Name:                   s.Name,
			PlannedDurationMinutes: int(s.DurationMinutes),
			Status:                 domain.StepStatusNotStarted,
		})
	}
	return steps, nil
}

func defaultSteps() []domain.StepExecution {
	return []domain.StepExecution{
		{
			ID:                     uuid.New().String(),
			StepID:                 "step-1-abhyanga",
			StepOrder:              1,
			Name:                   "Abhyanga (Herbal Oil Massage)",
			PlannedDurationMinutes: 45,
			Status:                 domain.StepStatusNotStarted,
		},
		{
			ID:                     uuid.New().String(),
			StepID:                 "step-2-svedana",
			StepOrder:              2,
			Name:                   "Svedana (Medicated Steam Bath)",
			PlannedDurationMinutes: 20,
			Status:                 domain.StepStatusNotStarted,
		},
	}
}

// CheckInRequest carries check-in parameters.
type CheckInRequest struct {
	AppointmentID        string                `json:"appointment_id"`
	ClinicID             string                `json:"clinic_id"`
	TherapistID          string                `json:"therapist_id"`
	BaselineVitals       domain.BaselineVitals `json:"baseline_vitals"`
	PreChecklistVerified bool                  `json:"pre_checklist_verified"`
	PreChecklistData     json.RawMessage       `json:"pre_checklist_data,omitempty"`
}

// RecordMaterialRequest carries consumable supply details.
type RecordMaterialRequest struct {
	MaterialName string  `json:"material_name"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
	BatchNumber  string  `json:"batch_number,omitempty"`
	Notes        string  `json:"notes,omitempty"`
}

// FlagComplicationRequest carries adverse incident data.
type FlagComplicationRequest struct {
	Severity              domain.ComplicationSeverity `json:"severity"`
	Symptoms              string                      `json:"symptoms"`
	VitalsAtIncident      domain.BaselineVitals       `json:"vitals_at_incident"`
	ImmediateActionsTaken string                      `json:"immediate_actions_taken"`
	AttendingDoctorID     string                      `json:"attending_doctor_id"`
}

// CreateFollowUpRequest carries doctor/therapist recommendations.
type CreateFollowUpRequest struct {
	RecommendedPackageID   *string `json:"recommended_package_id,omitempty"`
	RecommendedIntervalDay *int    `json:"recommended_interval_days,omitempty"`
	RecommendedDate        *string `json:"recommended_date,omitempty"`
	Instructions           string  `json:"instructions"`
}

// TherapyService defines business operations for therapy execution.
type TherapyService interface {
	CheckInAppointment(ctx context.Context, req CheckInRequest) (*domain.TherapySession, error)
	StartSession(ctx context.Context, sessionID, therapistID string) (*domain.TherapySession, error)
	StartStep(ctx context.Context, sessionID string, stepOrder int, therapistID string) error
	PauseStep(ctx context.Context, sessionID string, stepOrder int, therapistID string) error
	ResumeStep(ctx context.Context, sessionID string, stepOrder int, therapistID string) error
	CompleteStep(ctx context.Context, sessionID string, stepOrder int, therapistID string, actualSeconds int, notes, obs string) error
	RecordMaterial(ctx context.Context, sessionID, therapistID string, req RecordMaterialRequest) (*domain.SessionMaterial, error)
	CompleteSession(ctx context.Context, sessionID, therapistID, notes, obs, postCare string, postCareItems json.RawMessage) (*domain.TherapySession, error)
	FlagComplication(ctx context.Context, sessionID, reporterID, role string, req FlagComplicationRequest) (*domain.ComplicationEvent, error)
	ResolveComplication(ctx context.Context, compID, doctorID, notes string, resumeStatus domain.SessionStatus) error
	CreateFollowUp(ctx context.Context, sessionID, authorID, role string, req CreateFollowUpRequest) (*domain.SessionFollowUp, error)
	LinkFollowUpAppointment(ctx context.Context, followUpID, appointmentID string) error
	CorrectTherapistNotes(ctx context.Context, sessionID, correctedBy, newNotes, reason string) error
	GetSession(ctx context.Context, sessionID, requesterID, role, clinicID string) (*domain.TherapySession, error)
	ListAssignedSessionsForTherapist(ctx context.Context, therapistID, clinicID string, status *domain.SessionStatus) ([]*domain.TherapySession, error)
	ListPatientSessions(ctx context.Context, patientID string) ([]*domain.TherapySession, error)
}

type therapyService struct {
	repo            repository.TherapyRepository
	apptVerifier    AppointmentVerifier
	pkgProvider     PackageSnapshotProvider
}

// NewTherapyService instantiates the therapy management service.
func NewTherapyService(repo repository.TherapyRepository, verifier AppointmentVerifier, provider PackageSnapshotProvider) TherapyService {
	return &therapyService{
		repo:         repo,
		apptVerifier: verifier,
		pkgProvider:  provider,
	}
}

// CheckInAppointment verifies appointment readiness and initializes therapy session and steps.
func (s *therapyService) CheckInAppointment(ctx context.Context, req CheckInRequest) (*domain.TherapySession, error) {
	if req.AppointmentID == "" || req.ClinicID == "" || req.TherapistID == "" {
		return nil, errors.New("appointment_id, clinic_id, and therapist_id are required")
	}
	if !req.PreChecklistVerified {
		return nil, domain.ErrPreChecklistRequired
	}

	// 1. Check Appointment Service via gRPC
	patientID := "pat_auto"
	doctorID := "doc_auto"
	roomID := "room_auto"
	packageID := "pkg_auto"

	if s.apptVerifier != nil {
		isEligible, reason, pID, dID, rID, pkgID, err := s.apptVerifier.VerifyAppointmentForSession(ctx, req.AppointmentID, req.ClinicID)
		if err == nil {
			if !isEligible {
				return nil, fmt.Errorf("appointment not eligible for check-in: %s", reason)
			}
			if pID != "" {
				patientID = pID
			}
			if dID != "" {
				doctorID = dID
			}
			if rID != "" {
				roomID = rID
			}
			if pkgID != "" {
				packageID = pkgID
			}
		}
	}

	// 2. Load Package Steps
	var steps []domain.StepExecution
	if s.pkgProvider != nil {
		st, err := s.pkgProvider.GetPackageSteps(ctx, packageID)
		if err == nil && len(st) > 0 {
			steps = st
		}
	}
	if len(steps) == 0 {
		steps = defaultSteps()
	}

	now := time.Now().UTC()
	vitalsJSON, err := json.Marshal(req.BaselineVitals)
	if err != nil {
		return nil, errors.New("invalid baseline vitals format")
	}

	sessionID := uuid.New().String()
	session := &domain.TherapySession{
		ID:                   sessionID,
		AppointmentID:        req.AppointmentID,
		PatientID:            patientID,
		ClinicID:             req.ClinicID,
		DoctorID:             doctorID,
		TherapistID:          req.TherapistID,
		RoomID:               roomID,
		PackageID:            packageID,
		PackageVersion:       1,
		Status:               domain.SessionStatusCheckedIn,
		BaselineVitals:       vitalsJSON,
		PreChecklistVerified: req.PreChecklistVerified,
		PreChecklistData:     req.PreChecklistData,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	// Minimal outbox event
	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"session_id":     sessionID,
		"appointment_id": req.AppointmentID,
		"patient_id":     patientID,
		"clinic_id":      req.ClinicID,
		"status":         "CHECKED_IN",
	})

	outbox := &domain.OutboxEvent{
		AggregateType: "SESSION",
		AggregateID:   sessionID,
		EventType:     "panchakarma.session.checked_in.v1",
		Payload:       payloadBytes,
		TraceID:       "trace-" + sessionID[:8],
		Status:        "PENDING",
		CreatedAt:     now,
	}

	if err := s.repo.CreateSession(ctx, session, steps, outbox); err != nil {
		return nil, err
	}

	session.Steps = steps
	return session, nil
}

// StartSession sets status to IN_PROGRESS with server-generated start timestamp.
func (s *therapyService) StartSession(ctx context.Context, sessionID, therapistID string) (*domain.TherapySession, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session.IsTerminal() {
		return nil, domain.ErrSessionCompleted
	}
	if therapistID != "" && session.TherapistID != therapistID {
		return nil, domain.ErrUnauthorizedCrossClinic
	}
	if session.Status != domain.SessionStatusCheckedIn {
		return nil, fmt.Errorf("cannot start session in status %s", session.Status)
	}

	now := time.Now().UTC()
	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"session_id":     session.ID,
		"appointment_id": session.AppointmentID,
		"patient_id":     session.PatientID,
		"clinic_id":      session.ClinicID,
		"started_at":     now.Format(time.RFC3339),
		"status":         "IN_PROGRESS",
	})

	outbox := &domain.OutboxEvent{
		AggregateType: "SESSION",
		AggregateID:   session.ID,
		EventType:     "panchakarma.session.started.v1",
		Payload:       payloadBytes,
		TraceID:       "trace-" + session.ID[:8],
		Status:        "PENDING",
		CreatedAt:     now,
	}

	err = s.repo.UpdateSessionStatusWithTx(ctx, sessionID, domain.SessionStatusCheckedIn, domain.SessionStatusInProgress, "", "", &now, nil, outbox)
	if err != nil {
		return nil, err
	}

	session.Status = domain.SessionStatusInProgress
	session.StartedAt = &now
	return session, nil
}

// StartStep transitions step to IN_PROGRESS after verifying step ordering.
func (s *therapyService) StartStep(ctx context.Context, sessionID string, stepOrder int, therapistID string) error {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.IsTerminal() {
		return domain.ErrSessionCompleted
	}
	if session.Status != domain.SessionStatusInProgress {
		return fmt.Errorf("session must be IN_PROGRESS to start a step, current status is %s", session.Status)
	}

	// Verify step order: previous step must be COMPLETED or SKIPPED
	if stepOrder > 1 {
		prevStep, err := s.repo.GetStepExecution(ctx, sessionID, stepOrder-1)
		if err != nil {
			return err
		}
		if prevStep.Status != domain.StepStatusCompleted && prevStep.Status != domain.StepStatusSkipped {
			return domain.ErrStepOutOfOrder
		}
	}

	now := time.Now().UTC()
	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"session_id": sessionID,
		"step_order": stepOrder,
		"started_at": now.Format(time.RFC3339),
	})

	outbox := &domain.OutboxEvent{
		AggregateType: "SESSION_STEP",
		AggregateID:   sessionID,
		EventType:     "panchakarma.session.step_started.v1",
		Payload:       payloadBytes,
		TraceID:       "trace-" + sessionID[:8],
		Status:        "PENDING",
		CreatedAt:     now,
	}

	return s.repo.UpdateStepStatus(ctx, sessionID, stepOrder, domain.StepStatusNotStarted, domain.StepStatusInProgress, 0, "", "", &now, nil, nil, outbox)
}

// PauseStep pauses an in-progress step timer.
func (s *therapyService) PauseStep(ctx context.Context, sessionID string, stepOrder int, therapistID string) error {
	now := time.Now().UTC()
	return s.repo.UpdateStepStatus(ctx, sessionID, stepOrder, domain.StepStatusInProgress, domain.StepStatusPaused, 0, "", "", nil, &now, nil, nil)
}

// ResumeStep resumes a paused step timer.
func (s *therapyService) ResumeStep(ctx context.Context, sessionID string, stepOrder int, therapistID string) error {
	return s.repo.UpdateStepStatus(ctx, sessionID, stepOrder, domain.StepStatusPaused, domain.StepStatusInProgress, 0, "", "", nil, nil, nil, nil)
}

// CompleteStep marks a step completed with actual duration and practitioner notes.
func (s *therapyService) CompleteStep(ctx context.Context, sessionID string, stepOrder int, therapistID string, actualSeconds int, notes, obs string) error {
	now := time.Now().UTC()

	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"session_id":              sessionID,
		"step_order":              stepOrder,
		"actual_duration_seconds": actualSeconds,
		"completed_at":            now.Format(time.RFC3339),
	})

	outbox := &domain.OutboxEvent{
		AggregateType: "SESSION_STEP",
		AggregateID:   sessionID,
		EventType:     "panchakarma.session.step_completed.v1",
		Payload:       payloadBytes,
		TraceID:       "trace-" + sessionID[:8],
		Status:        "PENDING",
		CreatedAt:     now,
	}

	return s.repo.UpdateStepStatus(ctx, sessionID, stepOrder, "", domain.StepStatusCompleted, actualSeconds, notes, obs, nil, nil, &now, outbox)
}

// RecordMaterial logs supplies and oils consumed during therapy.
func (s *therapyService) RecordMaterial(ctx context.Context, sessionID, therapistID string, req RecordMaterialRequest) (*domain.SessionMaterial, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.IsTerminal() {
		return nil, domain.ErrSessionCompleted
	}

	material := &domain.SessionMaterial{
		ID:           uuid.New().String(),
		SessionID:    sessionID,
		MaterialName: req.MaterialName,
		Quantity:     req.Quantity,
		Unit:         req.Unit,
		BatchNumber:  req.BatchNumber,
		Notes:        req.Notes,
		RecordedBy:   therapistID,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.RecordMaterial(ctx, material); err != nil {
		return nil, err
	}
	return material, nil
}

// CompleteSession finishes the procedure, dispatches post-care, and publishes durable Kafka event.
func (s *therapyService) CompleteSession(ctx context.Context, sessionID, therapistID, notes, obs, postCare string, postCareItems json.RawMessage) (*domain.TherapySession, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session.IsTerminal() {
		return nil, domain.ErrSessionCompleted
	}
	if session.Status != domain.SessionStatusInProgress && session.Status != domain.SessionStatusCheckedIn {
		return nil, fmt.Errorf("cannot complete session in status %s", session.Status)
	}

	// Verify all steps are completed or skipped
	for _, step := range session.Steps {
		if step.Status != domain.StepStatusCompleted && step.Status != domain.StepStatusSkipped {
			return nil, domain.ErrIncompleteSteps
		}
	}

	now := time.Now().UTC()

	// Minimal Kafka event payload (no unnecessary health data)
	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"session_id":     session.ID,
		"appointment_id": session.AppointmentID,
		"patient_id":     session.PatientID,
		"clinic_id":      session.ClinicID,
		"completed_at":   now.Format(time.RFC3339),
		"status":         "COMPLETED",
	})

	outbox := &domain.OutboxEvent{
		AggregateType: "SESSION",
		AggregateID:   session.ID,
		EventType:     "panchakarma.session.completed.v1",
		Payload:       payloadBytes,
		TraceID:       "trace-" + session.ID[:8],
		Status:        "PENDING",
		CreatedAt:     now,
	}

	err = s.repo.UpdateSessionStatusWithTx(ctx, sessionID, session.Status, domain.SessionStatusCompleted, notes, obs, nil, &now, outbox)
	if err != nil {
		return nil, err
	}

	// Dispatch post care instructions
	_ = s.repo.UpdateSessionPostCare(ctx, sessionID, postCare, postCareItems, true)

	session.Status = domain.SessionStatusCompleted
	session.CompletedAt = &now
	session.PostCareDispatched = true
	session.PostCareInstructions = postCare
	return session, nil
}

// FlagComplication reports an adverse clinical event and triggers high-priority escalation.
func (s *therapyService) FlagComplication(ctx context.Context, sessionID, reporterID, role string, req FlagComplicationRequest) (*domain.ComplicationEvent, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.IsTerminal() {
		return nil, domain.ErrSessionCompleted
	}

	vitalsJSON, _ := json.Marshal(req.VitalsAtIncident)
	now := time.Now().UTC()

	docID := req.AttendingDoctorID
	if docID == "" {
		docID = session.DoctorID
	}

	comp := &domain.ComplicationEvent{
		ID:                    uuid.New().String(),
		SessionID:             sessionID,
		ReporterID:            reporterID,
		ReporterRole:          role,
		Severity:              req.Severity,
		Symptoms:              req.Symptoms,
		VitalsAtIncident:      vitalsJSON,
		ImmediateActionsTaken: req.ImmediateActionsTaken,
		Status:                domain.ComplicationUnresolved,
		AttendingDoctorID:     docID,
		CreatedAt:             now,
	}

	// High priority emergency Kafka event
	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"complication_id":     comp.ID,
		"session_id":          sessionID,
		"appointment_id":      session.AppointmentID,
		"patient_id":          session.PatientID,
		"severity":            comp.Severity,
		"attending_doctor_id": docID,
		"clinic_id":           session.ClinicID,
		"timestamp":           now.Format(time.RFC3339),
	})

	outbox := &domain.OutboxEvent{
		AggregateType: "COMPLICATION",
		AggregateID:   comp.ID,
		EventType:     "panchakarma.complication.flagged.v1",
		Payload:       payloadBytes,
		TraceID:       "trace-" + comp.ID[:8],
		Status:        "PENDING",
		CreatedAt:     now,
	}

	if err := s.repo.FlagComplicationWithTx(ctx, comp, outbox); err != nil {
		return nil, err
	}

	return comp, nil
}

// ResolveComplication documents doctor clearance and stabilizes session.
func (s *therapyService) ResolveComplication(ctx context.Context, compID, doctorID, notes string, resumeStatus domain.SessionStatus) error {
	if resumeStatus == "" {
		resumeStatus = domain.SessionStatusInProgress
	}
	now := time.Now().UTC()

	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"complication_id": compID,
		"doctor_id":       doctorID,
		"resume_status":   resumeStatus,
		"resolved_at":     now.Format(time.RFC3339),
	})

	outbox := &domain.OutboxEvent{
		AggregateType: "COMPLICATION",
		AggregateID:   compID,
		EventType:     "panchakarma.complication.resolved.v1",
		Payload:       payloadBytes,
		TraceID:       "trace-" + compID[:8],
		Status:        "PENDING",
		CreatedAt:     now,
	}

	return s.repo.ResolveComplicationWithTx(ctx, compID, doctorID, notes, resumeStatus, outbox)
}

// CreateFollowUp creates post-therapy follow-up recommendations.
func (s *therapyService) CreateFollowUp(ctx context.Context, sessionID, authorID, role string, req CreateFollowUpRequest) (*domain.SessionFollowUp, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	followUp := &domain.SessionFollowUp{
		ID:                     uuid.New().String(),
		SessionID:              session.ID,
		RecommendedBy:          authorID,
		RecommendedPackageID:   req.RecommendedPackageID,
		RecommendedIntervalDay: req.RecommendedIntervalDay,
		RecommendedDate:        req.RecommendedDate,
		Instructions:           req.Instructions,
		Status:                 "PENDING",
		CreatedAt:              time.Now().UTC(),
	}

	if err := s.repo.CreateFollowUp(ctx, followUp); err != nil {
		return nil, err
	}
	return followUp, nil
}

// LinkFollowUpAppointment associates a recommended follow-up with a booked appointment.
func (s *therapyService) LinkFollowUpAppointment(ctx context.Context, followUpID, appointmentID string) error {
	return s.repo.LinkFollowUpAppointment(ctx, followUpID, appointmentID)
}

// CorrectTherapistNotes modifies clinical documentation with mandatory audit justification.
func (s *therapyService) CorrectTherapistNotes(ctx context.Context, sessionID, correctedBy, newNotes, reason string) error {
	if reason == "" {
		return domain.ErrCorrectionReasonRequired
	}
	return s.repo.UpdateTherapistNotesWithAudit(ctx, sessionID, newNotes, reason, correctedBy)
}

// GetSession enforces privacy and authorization for session access.
func (s *therapyService) GetSession(ctx context.Context, sessionID, requesterID, role, clinicID string) (*domain.TherapySession, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if role == "patient" {
		if session.PatientID != requesterID {
			return nil, domain.ErrUnauthorizedPatientAccess
		}
		// Mask internal therapist clinical notes from patient
		session.TherapistNotes = ""
	}
	if role == "therapist" && session.TherapistID != requesterID && session.ClinicID != clinicID {
		return nil, domain.ErrUnauthorizedCrossClinic
	}
	if (role == "clinic_admin" || role == "doctor") && clinicID != "" && session.ClinicID != clinicID {
		return nil, domain.ErrUnauthorizedCrossClinic
	}

	return session, nil
}

// ListAssignedSessionsForTherapist returns active and past sessions for practitioner.
func (s *therapyService) ListAssignedSessionsForTherapist(ctx context.Context, therapistID, clinicID string, status *domain.SessionStatus) ([]*domain.TherapySession, error) {
	filter := repository.ListFilter{
		ClinicID:    clinicID,
		TherapistID: &therapistID,
		Status:      status,
		Limit:       50,
	}
	sessions, _, err := s.repo.ListSessions(ctx, filter)
	return sessions, err
}

// ListPatientSessions returns patient-facing historical sessions.
func (s *therapyService) ListPatientSessions(ctx context.Context, patientID string) ([]*domain.TherapySession, error) {
	filter := repository.ListFilter{
		PatientID: &patientID,
		Limit:     50,
	}
	sessions, _, err := s.repo.ListSessions(ctx, filter)
	return sessions, err
}
