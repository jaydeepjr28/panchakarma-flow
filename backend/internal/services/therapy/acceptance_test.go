package therapy_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/repository"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ThreadSafeMockTherapyRepository provides thread-safe data access for acceptance testing.
type ThreadSafeMockTherapyRepository struct {
	mu            sync.Mutex
	sessions      map[string]*domain.TherapySession
	sessionByApp  map[string]*domain.TherapySession
	steps         map[string]map[int]*domain.StepExecution
	materials     map[string][]domain.SessionMaterial
	complications map[string]*domain.ComplicationEvent
	followUps     map[string]*domain.SessionFollowUp
	auditLogs     map[string][]domain.SessionAuditLog
	outboxEvents  []*domain.OutboxEvent
}

func NewThreadSafeMockTherapyRepository() *ThreadSafeMockTherapyRepository {
	return &ThreadSafeMockTherapyRepository{
		sessions:      make(map[string]*domain.TherapySession),
		sessionByApp:  make(map[string]*domain.TherapySession),
		steps:         make(map[string]map[int]*domain.StepExecution),
		materials:     make(map[string][]domain.SessionMaterial),
		complications: make(map[string]*domain.ComplicationEvent),
		followUps:     make(map[string]*domain.SessionFollowUp),
		auditLogs:     make(map[string][]domain.SessionAuditLog),
		outboxEvents:  make([]*domain.OutboxEvent, 0),
	}
}

func (m *ThreadSafeMockTherapyRepository) CreateSession(ctx context.Context, session *domain.TherapySession, steps []domain.StepExecution, outbox *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessionByApp[session.AppointmentID]; exists {
		return domain.ErrDuplicateActiveSession
	}

	copySession := *session
	m.sessions[session.ID] = &copySession
	m.sessionByApp[session.AppointmentID] = &copySession

	m.steps[session.ID] = make(map[int]*domain.StepExecution)
	for _, s := range steps {
		sCopy := s
		sCopy.SessionID = session.ID
		m.steps[session.ID][s.StepOrder] = &sCopy
	}

	if outbox != nil {
		m.outboxEvents = append(m.outboxEvents, outbox)
	}
	return nil
}

func (m *ThreadSafeMockTherapyRepository) GetSessionByID(ctx context.Context, id string) (*domain.TherapySession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, exists := m.sessions[id]
	if !exists {
		return nil, domain.ErrSessionNotFound
	}
	copySession := *s

	stepMap := m.steps[id]
	var steps []domain.StepExecution
	for i := 1; i <= len(stepMap); i++ {
		if st, ok := stepMap[i]; ok {
			steps = append(steps, *st)
		}
	}
	copySession.Steps = steps
	return &copySession, nil
}

func (m *ThreadSafeMockTherapyRepository) GetSessionByAppointmentID(ctx context.Context, appointmentID string) (*domain.TherapySession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, exists := m.sessionByApp[appointmentID]
	if !exists {
		return nil, domain.ErrSessionNotFound
	}
	copySession := *s
	return &copySession, nil
}

func (m *ThreadSafeMockTherapyRepository) ListSessions(ctx context.Context, filter repository.ListFilter) ([]*domain.TherapySession, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []*domain.TherapySession
	for _, s := range m.sessions {
		if filter.ClinicID != "" && s.ClinicID != filter.ClinicID {
			continue
		}
		if filter.TherapistID != nil && s.TherapistID != *filter.TherapistID {
			continue
		}
		if filter.PatientID != nil && s.PatientID != *filter.PatientID {
			continue
		}
		if filter.Status != nil && s.Status != *filter.Status {
			continue
		}
		copyS := *s
		list = append(list, &copyS)
	}
	return list, len(list), nil
}

func (m *ThreadSafeMockTherapyRepository) UpdateSessionStatusWithTx(ctx context.Context, sessionID string, fromStatus, toStatus domain.SessionStatus, notes, observations string, startedAt, completedAt *time.Time, outbox *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, exists := m.sessions[sessionID]
	if !exists {
		return domain.ErrSessionNotFound
	}

	if s.Status == domain.SessionStatusCompleted || s.Status == domain.SessionStatusAbortedClinical {
		return domain.ErrSessionCompleted
	}

	if fromStatus != "" && s.Status != fromStatus {
		return domain.ErrInvalidSessionTransition
	}

	if err := domain.ValidateSessionTransition(s.Status, toStatus); err != nil {
		return err
	}

	s.Status = toStatus
	if notes != "" {
		s.TherapistNotes = notes
	}
	if observations != "" {
		s.Observations = observations
	}
	if startedAt != nil {
		s.StartedAt = startedAt
	}
	if completedAt != nil {
		s.CompletedAt = completedAt
	}
	s.UpdatedAt = time.Now().UTC()

	if outbox != nil {
		m.outboxEvents = append(m.outboxEvents, outbox)
	}
	return nil
}

func (m *ThreadSafeMockTherapyRepository) UpdateSessionPostCare(ctx context.Context, sessionID string, instructions string, items json.RawMessage, dispatched bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, exists := m.sessions[sessionID]
	if !exists {
		return domain.ErrSessionNotFound
	}
	s.PostCareInstructions = instructions
	s.PostCareItems = items
	s.PostCareDispatched = dispatched
	s.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *ThreadSafeMockTherapyRepository) GetStepExecution(ctx context.Context, sessionID string, stepOrder int) (*domain.StepExecution, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stepMap, exists := m.steps[sessionID]
	if !exists {
		return nil, domain.ErrStepNotFound
	}
	step, exists := stepMap[stepOrder]
	if !exists {
		return nil, domain.ErrStepNotFound
	}
	copyStep := *step
	return &copyStep, nil
}

func (m *ThreadSafeMockTherapyRepository) UpdateStepStatus(ctx context.Context, sessionID string, stepOrder int, fromStatus, toStatus domain.StepStatus, durationSeconds int, notes, obs string, startedAt, pausedAt, completedAt *time.Time, outbox *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	stepMap, exists := m.steps[sessionID]
	if !exists {
		return domain.ErrStepNotFound
	}
	step, exists := stepMap[stepOrder]
	if !exists {
		return domain.ErrStepNotFound
	}

	step.Status = toStatus
	if durationSeconds > 0 {
		step.ActualDurationSeconds += durationSeconds
	}
	if notes != "" {
		step.TherapistNotes = notes
	}
	if obs != "" {
		step.Observations = obs
	}
	if startedAt != nil {
		step.StartedAt = startedAt
	}
	if pausedAt != nil {
		step.PausedAt = pausedAt
	}
	if completedAt != nil {
		step.CompletedAt = completedAt
	}

	if outbox != nil {
		m.outboxEvents = append(m.outboxEvents, outbox)
	}
	return nil
}

func (m *ThreadSafeMockTherapyRepository) RecordMaterial(ctx context.Context, material *domain.SessionMaterial) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.materials[material.SessionID] = append(m.materials[material.SessionID], *material)
	return nil
}

func (m *ThreadSafeMockTherapyRepository) GetSessionMaterials(ctx context.Context, sessionID string) ([]domain.SessionMaterial, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.materials[sessionID], nil
}

func (m *ThreadSafeMockTherapyRepository) FlagComplicationWithTx(ctx context.Context, comp *domain.ComplicationEvent, outbox *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, exists := m.sessions[comp.SessionID]
	if !exists {
		return domain.ErrSessionNotFound
	}
	s.Status = domain.SessionStatusComplicationAlert
	s.UpdatedAt = time.Now().UTC()

	m.complications[comp.ID] = comp
	if outbox != nil {
		m.outboxEvents = append(m.outboxEvents, outbox)
	}
	return nil
}

func (m *ThreadSafeMockTherapyRepository) ResolveComplicationWithTx(ctx context.Context, compID string, doctorID string, notes string, resumeStatus domain.SessionStatus, outbox *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	comp, exists := m.complications[compID]
	if !exists {
		return domain.ErrComplicationNotFound
	}

	now := time.Now().UTC()
	comp.Status = domain.ComplicationResolved
	comp.AttendingDoctorID = doctorID
	comp.DoctorNotes = notes
	comp.DoctorClearedAt = &now

	if s, ok := m.sessions[comp.SessionID]; ok {
		s.Status = resumeStatus
		s.UpdatedAt = now
	}

	if outbox != nil {
		m.outboxEvents = append(m.outboxEvents, outbox)
	}
	return nil
}

func (m *ThreadSafeMockTherapyRepository) GetComplicationByID(ctx context.Context, id string) (*domain.ComplicationEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	comp, exists := m.complications[id]
	if !exists {
		return nil, domain.ErrComplicationNotFound
	}
	copyC := *comp
	return &copyC, nil
}

func (m *ThreadSafeMockTherapyRepository) ListComplications(ctx context.Context, sessionID string) ([]*domain.ComplicationEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []*domain.ComplicationEvent
	for _, c := range m.complications {
		if c.SessionID == sessionID {
			copyC := *c
			list = append(list, &copyC)
		}
	}
	return list, nil
}

func (m *ThreadSafeMockTherapyRepository) CreateFollowUp(ctx context.Context, followUp *domain.SessionFollowUp) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.followUps[followUp.ID] = followUp
	return nil
}

func (m *ThreadSafeMockTherapyRepository) GetSessionFollowUps(ctx context.Context, sessionID string) ([]domain.SessionFollowUp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []domain.SessionFollowUp
	for _, f := range m.followUps {
		if f.SessionID == sessionID {
			list = append(list, *f)
		}
	}
	return list, nil
}

func (m *ThreadSafeMockTherapyRepository) LinkFollowUpAppointment(ctx context.Context, followUpID, appointmentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	f, exists := m.followUps[followUpID]
	if !exists {
		return errors.New("follow-up record not found")
	}
	f.LinkedAppointmentID = &appointmentID
	f.Status = "SCHEDULED"
	return nil
}

func (m *ThreadSafeMockTherapyRepository) RecordAuditLog(ctx context.Context, audit *domain.SessionAuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.auditLogs[audit.SessionID] = append(m.auditLogs[audit.SessionID], *audit)
	return nil
}

func (m *ThreadSafeMockTherapyRepository) GetSessionAuditLogs(ctx context.Context, sessionID string) ([]domain.SessionAuditLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.auditLogs[sessionID], nil
}

func (m *ThreadSafeMockTherapyRepository) UpdateTherapistNotesWithAudit(ctx context.Context, sessionID string, newNotes string, reason string, correctedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, exists := m.sessions[sessionID]
	if !exists {
		return domain.ErrSessionNotFound
	}

	oldNotes := s.TherapistNotes
	s.TherapistNotes = newNotes
	s.UpdatedAt = time.Now().UTC()

	audit := domain.SessionAuditLog{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		FieldName:   "therapist_notes",
		OldValue:    oldNotes,
		NewValue:    newNotes,
		Reason:      reason,
		CorrectedBy: correctedBy,
		CreatedAt:   time.Now().UTC(),
	}
	m.auditLogs[sessionID] = append(m.auditLogs[sessionID], audit)
	return nil
}

func (m *ThreadSafeMockTherapyRepository) CreateOutboxEvent(ctx context.Context, event *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.outboxEvents = append(m.outboxEvents, event)
	return nil
}

func (m *ThreadSafeMockTherapyRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var pending []*domain.OutboxEvent
	for _, e := range m.outboxEvents {
		if e.Status == "PENDING" {
			pending = append(pending, e)
			if len(pending) >= limit {
				break
			}
		}
	}
	return pending, nil
}

func (m *ThreadSafeMockTherapyRepository) MarkOutboxEventProcessed(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, e := range m.outboxEvents {
		if e.ID == id {
			e.Status = "PROCESSED"
			now := time.Now().UTC()
			e.ProcessedAt = &now
			return nil
		}
	}
	return nil
}

func (m *ThreadSafeMockTherapyRepository) MarkOutboxEventFailed(ctx context.Context, id string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, e := range m.outboxEvents {
		if e.ID == id {
			e.RetryCount++
			return nil
		}
	}
	return nil
}

// 1. ACCEPTANCE TEST: Invalid State Transitions
func TestAcceptance_TherapyInvalidTransitions(t *testing.T) {
	repo := NewThreadSafeMockTherapyRepository()
	svc := service.NewTherapyService(repo, nil, nil)
	ctx := context.Background()

	session, err := svc.CheckInAppointment(ctx, service.CheckInRequest{
		AppointmentID:        "appt-invalid-trans",
		ClinicID:             "clinic-1",
		TherapistID:          "therapist-1",
		PreChecklistVerified: true,
	})
	require.NoError(t, err)

	// Attempt to directly complete without starting
	_, err = svc.CompleteSession(ctx, session.ID, "therapist-1", "Done", "Good", "Drink warm water", nil)
	// Must fail because session is CHECKED_IN and steps are not completed
	assert.Error(t, err)

	// Start session: CHECKED_IN -> IN_PROGRESS
	started, err := svc.StartSession(ctx, session.ID, "therapist-1")
	require.NoError(t, err)
	assert.Equal(t, domain.SessionStatusInProgress, started.Status)

	// Attempt invalid transition: e.g. StartSession again on IN_PROGRESS
	_, err = svc.StartSession(ctx, session.ID, "therapist-1")
	assert.Error(t, err)

	// Complete all steps
	require.NoError(t, svc.StartStep(ctx, session.ID, 1, "therapist-1"))
	require.NoError(t, svc.CompleteStep(ctx, session.ID, 1, "therapist-1", 900, "Done 1", ""))
	require.NoError(t, svc.StartStep(ctx, session.ID, 2, "therapist-1"))
	require.NoError(t, svc.CompleteStep(ctx, session.ID, 2, "therapist-1", 900, "Done 2", ""))

	// Complete session: IN_PROGRESS -> COMPLETED
	completed, err := svc.CompleteSession(ctx, session.ID, "therapist-1", "Notes", "Obs", "Rest", nil)
	require.NoError(t, err)
	assert.Equal(t, domain.SessionStatusCompleted, completed.Status)

	// Terminal immutable state: Attempting to start step or start session fails with ErrSessionCompleted
	err = svc.StartStep(ctx, session.ID, 1, "therapist-1")
	assert.ErrorIs(t, err, domain.ErrSessionCompleted)

	_, err = svc.StartSession(ctx, session.ID, "therapist-1")
	assert.ErrorIs(t, err, domain.ErrSessionCompleted)
}

// 2. ACCEPTANCE TEST: Duplicate Active Session Prevention Under High Concurrency
func TestAcceptance_DuplicateSessionStarts(t *testing.T) {
	repo := NewThreadSafeMockTherapyRepository()
	svc := service.NewTherapyService(repo, nil, nil)
	ctx := context.Background()

	appointmentID := "appt-race-unique-123"
	concurrency := 15
	var wg sync.WaitGroup
	var successCount int64
	var duplicateErrors int64

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			req := service.CheckInRequest{
				AppointmentID:        appointmentID,
				ClinicID:             "clinic-alpha",
				TherapistID:          "therapist-alpha",
				PreChecklistVerified: true,
			}
			_, err := svc.CheckInAppointment(ctx, req)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else if errors.Is(err, domain.ErrDuplicateActiveSession) {
				atomic.AddInt64(&duplicateErrors, 1)
			}
		}(i)
	}

	wg.Wait()

	if successCount != 1 {
		t.Fatalf("CRITICAL CONCURRENCY FAILURE: Exactly 1 check-in must succeed, but %d succeeded!", successCount)
	}
	if duplicateErrors != int64(concurrency-1) {
		t.Fatalf("CRITICAL CONCURRENCY FAILURE: Expected %d duplicate session rejections, got %d", concurrency-1, duplicateErrors)
	}
}

// 3. ACCEPTANCE TEST: Ordered Step Completion and Timestamp Verification
func TestAcceptance_OrderedStepCompletionAndTimestamps(t *testing.T) {
	repo := NewThreadSafeMockTherapyRepository()
	svc := service.NewTherapyService(repo, nil, nil)
	ctx := context.Background()

	session, err := svc.CheckInAppointment(ctx, service.CheckInRequest{
		AppointmentID:        "appt-steps-order",
		ClinicID:             "clinic-1",
		TherapistID:          "therapist-1",
		PreChecklistVerified: true,
	})
	require.NoError(t, err)

	started, err := svc.StartSession(ctx, session.ID, "therapist-1")
	require.NoError(t, err)
	require.NotNil(t, started.StartedAt)
	assert.WithinDuration(t, time.Now().UTC(), *started.StartedAt, 2*time.Second)

	// Step 2 cannot start before Step 1
	err = svc.StartStep(ctx, session.ID, 2, "therapist-1")
	assert.ErrorIs(t, err, domain.ErrStepOutOfOrder)

	// Step 1: Start -> Pause -> Resume -> Complete
	require.NoError(t, svc.StartStep(ctx, session.ID, 1, "therapist-1"))
	require.NoError(t, svc.PauseStep(ctx, session.ID, 1, "therapist-1"))
	require.NoError(t, svc.ResumeStep(ctx, session.ID, 1, "therapist-1"))
	require.NoError(t, svc.CompleteStep(ctx, session.ID, 1, "therapist-1", 1200, "Done Step 1", "Good response"))

	// Step 2 can now start
	require.NoError(t, svc.StartStep(ctx, session.ID, 2, "therapist-1"))
	require.NoError(t, svc.CompleteStep(ctx, session.ID, 2, "therapist-1", 600, "Done Step 2", "Well tolerated"))

	// Complete session with server timestamp
	completed, err := svc.CompleteSession(ctx, session.ID, "therapist-1", "Session smooth", "No issues", "Drink herbal tea", nil)
	require.NoError(t, err)
	require.NotNil(t, completed.CompletedAt)
	assert.WithinDuration(t, time.Now().UTC(), *completed.CompletedAt, 2*time.Second)
}

// 4. ACCEPTANCE TEST: Unauthorized Cross-Clinic and Cross-Patient Access Checks
func TestAcceptance_UnauthorizedAccessChecks(t *testing.T) {
	repo := NewThreadSafeMockTherapyRepository()
	svc := service.NewTherapyService(repo, nil, nil)
	ctx := context.Background()

	session, err := svc.CheckInAppointment(ctx, service.CheckInRequest{
		AppointmentID:        "appt-access-check",
		ClinicID:             "clinic-alpha",
		TherapistID:          "therapist-alpha",
		PreChecklistVerified: true,
	})
	require.NoError(t, err)

	// 1. Therapist from Clinic Beta attempts to access Clinic Alpha session
	_, err = svc.GetSession(ctx, session.ID, "therapist-beta", "therapist", "clinic-beta")
	assert.ErrorIs(t, err, domain.ErrUnauthorizedCrossClinic)

	// 2. Unrelated patient attempts to access session
	_, err = svc.GetSession(ctx, session.ID, "patient-mallory", "patient", "")
	assert.ErrorIs(t, err, domain.ErrUnauthorizedPatientAccess)

	// 3. Therapist from Clinic Beta attempts to start session
	_, err = svc.StartSession(ctx, session.ID, "therapist-beta")
	assert.ErrorIs(t, err, domain.ErrUnauthorizedCrossClinic)

	// 4. Authorized owner patient accesses session: therapist notes MUST be masked
	s, err := svc.GetSession(ctx, session.ID, session.PatientID, "patient", "")
	require.NoError(t, err)
	assert.Empty(t, s.TherapistNotes, "Patient must not have access to therapist clinical notes")
}

// 5. ACCEPTANCE TEST: Follow-up Creation and Appointment Linking
func TestAcceptance_FollowUpCreationAndLinking(t *testing.T) {
	repo := NewThreadSafeMockTherapyRepository()
	svc := service.NewTherapyService(repo, nil, nil)
	ctx := context.Background()

	session, err := svc.CheckInAppointment(ctx, service.CheckInRequest{
		AppointmentID:        "appt-fu-test",
		ClinicID:             "clinic-alpha",
		TherapistID:          "therapist-alpha",
		PreChecklistVerified: true,
	})
	require.NoError(t, err)

	interval := 14
	fu, err := svc.CreateFollowUp(ctx, session.ID, "doc-ayur", "doctor", service.CreateFollowUpRequest{
		RecommendedIntervalDay: &interval,
		Instructions:           "Follow up in 2 weeks for Nasya treatment plan",
	})
	require.NoError(t, err)
	assert.Equal(t, "PENDING", fu.Status)

	// Link with future booked appointment
	newAppointmentID := "appt-nasya-session-2"
	err = svc.LinkFollowUpAppointment(ctx, fu.ID, newAppointmentID)
	require.NoError(t, err)

	followUps, err := repo.GetSessionFollowUps(ctx, session.ID)
	require.NoError(t, err)
	require.Len(t, followUps, 1)
	assert.Equal(t, "SCHEDULED", followUps[0].Status)
	assert.Equal(t, newAppointmentID, *followUps[0].LinkedAppointmentID)
}

// 6. ACCEPTANCE TEST: Clinical Correction History and Audit Logging
func TestAcceptance_CorrectionHistoryAndAudit(t *testing.T) {
	repo := NewThreadSafeMockTherapyRepository()
	svc := service.NewTherapyService(repo, nil, nil)
	ctx := context.Background()

	session, err := svc.CheckInAppointment(ctx, service.CheckInRequest{
		AppointmentID:        "appt-audit-test",
		ClinicID:             "clinic-1",
		TherapistID:          "therapist-1",
		PreChecklistVerified: true,
	})
	require.NoError(t, err)

	_, err = svc.StartSession(ctx, session.ID, "therapist-1")
	require.NoError(t, err)

	// Complete all steps and finish session
	require.NoError(t, svc.StartStep(ctx, session.ID, 1, "therapist-1"))
	require.NoError(t, svc.CompleteStep(ctx, session.ID, 1, "therapist-1", 600, "Step 1 initial note", ""))
	require.NoError(t, svc.StartStep(ctx, session.ID, 2, "therapist-1"))
	require.NoError(t, svc.CompleteStep(ctx, session.ID, 2, "therapist-1", 600, "Step 2 initial note", ""))
	_, err = svc.CompleteSession(ctx, session.ID, "therapist-1", "Initial discharge notes", "", "Rest", nil)
	require.NoError(t, err)

	// 1. Correction attempt without reason fails
	err = svc.CorrectTherapistNotes(ctx, session.ID, "therapist-1", "Updated note", "")
	assert.ErrorIs(t, err, domain.ErrCorrectionReasonRequired)

	// 2. Correction with valid justification succeeds and produces audit trail
	err = svc.CorrectTherapistNotes(ctx, session.ID, "therapist-1", "Initial discharge notes: Patient notified about oil absorption time.", "Added missing patient discharge guidance")
	require.NoError(t, err)

	auditLogs, err := repo.GetSessionAuditLogs(ctx, session.ID)
	require.NoError(t, err)
	require.Len(t, auditLogs, 1)
	assert.Equal(t, "therapist_notes", auditLogs[0].FieldName)
	assert.Equal(t, "Initial discharge notes", auditLogs[0].OldValue)
	assert.Equal(t, "Initial discharge notes: Patient notified about oil absorption time.", auditLogs[0].NewValue)
	assert.Equal(t, "Added missing patient discharge guidance", auditLogs[0].Reason)
	assert.Equal(t, "therapist-1", auditLogs[0].CorrectedBy)
}

// 7. ACCEPTANCE TEST: Complication Reporting and Attending Doctor Clearance
func TestAcceptance_ComplicationEscalationAndResolution(t *testing.T) {
	repo := NewThreadSafeMockTherapyRepository()
	svc := service.NewTherapyService(repo, nil, nil)
	ctx := context.Background()

	session, err := svc.CheckInAppointment(ctx, service.CheckInRequest{
		AppointmentID:        "appt-comp-test",
		ClinicID:             "clinic-1",
		TherapistID:          "therapist-1",
		PreChecklistVerified: true,
	})
	require.NoError(t, err)

	_, err = svc.StartSession(ctx, session.ID, "therapist-1")
	require.NoError(t, err)

	// Flag complication
	comp, err := svc.FlagComplication(ctx, session.ID, "therapist-1", "therapist", service.FlagComplicationRequest{
		Severity:              domain.SeverityCritical,
		Symptoms:              "Extreme erythema and acute pruritus during medicated steam",
		ImmediateActionsTaken: "Steam stopped immediately, patient wiped down with cool wet towels, doctor summoned",
		AttendingDoctorID:     "doc-emergency",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.ComplicationUnresolved, comp.Status)

	// Session is placed in alert status
	s, err := repo.GetSessionByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SessionStatusComplicationAlert, s.Status)

	// Doctor reviews and safely aborts or resumes session
	err = svc.ResolveComplication(ctx, comp.ID, "doc-emergency", "Patient diagnosed with hypersensitivity to taila. Therapy discontinued for safety.", domain.SessionStatusAbortedClinical)
	require.NoError(t, err)

	s, err = repo.GetSessionByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SessionStatusAbortedClinical, s.Status)

	resolvedComp, err := repo.GetComplicationByID(ctx, comp.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.ComplicationResolved, resolvedComp.Status)
	assert.NotNil(t, resolvedComp.DoctorClearedAt)
}
