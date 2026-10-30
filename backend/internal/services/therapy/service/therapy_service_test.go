package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTherapyRepository is an in-memory thread-safe mock for TherapyRepository.
type mockTherapyRepository struct {
	mu           sync.Mutex
	sessions     map[string]*domain.TherapySession
	sessionByApp map[string]*domain.TherapySession
	steps        map[string]map[int]*domain.StepExecution // sessionID -> stepOrder -> step
	materials    map[string][]domain.SessionMaterial
	complications map[string]*domain.ComplicationEvent
	followUps    map[string]*domain.SessionFollowUp
	auditLogs    map[string][]domain.SessionAuditLog
	outboxEvents []*domain.OutboxEvent
}

func newMockTherapyRepository() *mockTherapyRepository {
	return &mockTherapyRepository{
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

func (m *mockTherapyRepository) CreateSession(ctx context.Context, session *domain.TherapySession, steps []domain.StepExecution, outbox *domain.OutboxEvent) error {
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

func (m *mockTherapyRepository) GetSessionByID(ctx context.Context, id string) (*domain.TherapySession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, exists := m.sessions[id]
	if !exists {
		return nil, domain.ErrSessionNotFound
	}
	copySession := *s

	// attach steps
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

func (m *mockTherapyRepository) GetSessionByAppointmentID(ctx context.Context, appointmentID string) (*domain.TherapySession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, exists := m.sessionByApp[appointmentID]
	if !exists {
		return nil, domain.ErrSessionNotFound
	}
	copySession := *s
	return &copySession, nil
}

func (m *mockTherapyRepository) ListSessions(ctx context.Context, filter repository.ListFilter) ([]*domain.TherapySession, int, error) {
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

func (m *mockTherapyRepository) UpdateSessionStatusWithTx(ctx context.Context, sessionID string, fromStatus, toStatus domain.SessionStatus, notes, observations string, startedAt, completedAt *time.Time, outbox *domain.OutboxEvent) error {
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

func (m *mockTherapyRepository) UpdateSessionPostCare(ctx context.Context, sessionID string, instructions string, items json.RawMessage, dispatched bool) error {
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

func (m *mockTherapyRepository) GetStepExecution(ctx context.Context, sessionID string, stepOrder int) (*domain.StepExecution, error) {
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

func (m *mockTherapyRepository) UpdateStepStatus(ctx context.Context, sessionID string, stepOrder int, fromStatus, toStatus domain.StepStatus, durationSeconds int, notes, obs string, startedAt, pausedAt, completedAt *time.Time, outbox *domain.OutboxEvent) error {
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

func (m *mockTherapyRepository) RecordMaterial(ctx context.Context, material *domain.SessionMaterial) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.materials[material.SessionID] = append(m.materials[material.SessionID], *material)
	return nil
}

func (m *mockTherapyRepository) GetSessionMaterials(ctx context.Context, sessionID string) ([]domain.SessionMaterial, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.materials[sessionID], nil
}

func (m *mockTherapyRepository) FlagComplicationWithTx(ctx context.Context, comp *domain.ComplicationEvent, outbox *domain.OutboxEvent) error {
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

func (m *mockTherapyRepository) ResolveComplicationWithTx(ctx context.Context, compID string, doctorID string, notes string, resumeStatus domain.SessionStatus, outbox *domain.OutboxEvent) error {
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

func (m *mockTherapyRepository) GetComplicationByID(ctx context.Context, id string) (*domain.ComplicationEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	comp, exists := m.complications[id]
	if !exists {
		return nil, domain.ErrComplicationNotFound
	}
	copyC := *comp
	return &copyC, nil
}

func (m *mockTherapyRepository) ListComplications(ctx context.Context, sessionID string) ([]*domain.ComplicationEvent, error) {
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

func (m *mockTherapyRepository) CreateFollowUp(ctx context.Context, followUp *domain.SessionFollowUp) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.followUps[followUp.ID] = followUp
	return nil
}

func (m *mockTherapyRepository) GetSessionFollowUps(ctx context.Context, sessionID string) ([]domain.SessionFollowUp, error) {
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

func (m *mockTherapyRepository) LinkFollowUpAppointment(ctx context.Context, followUpID, appointmentID string) error {
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

func (m *mockTherapyRepository) RecordAuditLog(ctx context.Context, audit *domain.SessionAuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.auditLogs[audit.SessionID] = append(m.auditLogs[audit.SessionID], *audit)
	return nil
}

func (m *mockTherapyRepository) GetSessionAuditLogs(ctx context.Context, sessionID string) ([]domain.SessionAuditLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.auditLogs[sessionID], nil
}

func (m *mockTherapyRepository) UpdateTherapistNotesWithAudit(ctx context.Context, sessionID string, newNotes string, reason string, correctedBy string) error {
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

func (m *mockTherapyRepository) CreateOutboxEvent(ctx context.Context, event *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.outboxEvents = append(m.outboxEvents, event)
	return nil
}

func (m *mockTherapyRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
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

func (m *mockTherapyRepository) MarkOutboxEventProcessed(ctx context.Context, id string) error {
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

func (m *mockTherapyRepository) MarkOutboxEventFailed(ctx context.Context, id string, reason string) error {
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

// mockAppointmentVerifier implements AppointmentVerifier
type mockAppointmentVerifier struct {
	isEligible bool
	reason     string
	patientID  string
	doctorID   string
	roomID     string
	packageID  string
	err        error
}

func (m *mockAppointmentVerifier) VerifyAppointmentForSession(ctx context.Context, appointmentID, clinicID string) (bool, string, string, string, string, string, error) {
	if m.err != nil {
		return false, "", "", "", "", "", m.err
	}
	return m.isEligible, m.reason, m.patientID, m.doctorID, m.roomID, m.packageID, nil
}

// mockPackageSnapshotProvider implements PackageSnapshotProvider
type mockPackageSnapshotProvider struct {
	steps []domain.StepExecution
	err   error
}

func (m *mockPackageSnapshotProvider) GetPackageSteps(ctx context.Context, packageID string) ([]domain.StepExecution, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.steps, nil
}

func TestTherapyService_CheckIn(t *testing.T) {
	ctx := context.Background()
	repo := newMockTherapyRepository()
	verifier := &mockAppointmentVerifier{
		isEligible: true,
		patientID:  "pat-100",
		doctorID:   "doc-100",
		roomID:     "room-100",
		packageID:  "pkg-abhyanga",
	}
	provider := &mockPackageSnapshotProvider{
		steps: []domain.StepExecution{
			{StepID: "s1", StepOrder: 1, Name: "Step 1", PlannedDurationMinutes: 30, Status: domain.StepStatusNotStarted},
			{StepID: "s2", StepOrder: 2, Name: "Step 2", PlannedDurationMinutes: 20, Status: domain.StepStatusNotStarted},
		},
	}

	svc := NewTherapyService(repo, verifier, provider)

	t.Run("check-in fails when checklist not verified", func(t *testing.T) {
		req := CheckInRequest{
			AppointmentID:        "appt-1",
			ClinicID:             "clinic-1",
			TherapistID:          "therapist-1",
			PreChecklistVerified: false,
		}
		_, err := svc.CheckInAppointment(ctx, req)
		assert.ErrorIs(t, err, domain.ErrPreChecklistRequired)
	})

	t.Run("check-in fails when appointment ineligible", func(t *testing.T) {
		ineligibleVerifier := &mockAppointmentVerifier{
			isEligible: false,
			reason:     "appointment status is CANCELLED",
		}
		ineligibleSvc := NewTherapyService(repo, ineligibleVerifier, provider)
		req := CheckInRequest{
			AppointmentID:        "appt-2",
			ClinicID:             "clinic-1",
			TherapistID:          "therapist-1",
			PreChecklistVerified: true,
		}
		_, err := ineligibleSvc.CheckInAppointment(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "appointment not eligible for check-in")
	})

	t.Run("check-in successful initializes session and ordered steps", func(t *testing.T) {
		req := CheckInRequest{
			AppointmentID:        "appt-100",
			ClinicID:             "clinic-1",
			TherapistID:          "therapist-1",
			PreChecklistVerified: true,
			BaselineVitals: domain.BaselineVitals{
				BPSystolic:   120,
				BPDiastolic:  80,
				Pulse:        72,
				TemperatureC: 36.8,
			},
		}

		session, err := svc.CheckInAppointment(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, session.ID)
		assert.Equal(t, domain.SessionStatusCheckedIn, session.Status)
		assert.Equal(t, "pat-100", session.PatientID)
		assert.Equal(t, "clinic-1", session.ClinicID)
		assert.Equal(t, 2, len(session.Steps))
		assert.Equal(t, 1, session.Steps[0].StepOrder)
		assert.Equal(t, 2, session.Steps[1].StepOrder)

		// Verify duplicate active session prevention
		_, err = svc.CheckInAppointment(ctx, req)
		assert.ErrorIs(t, err, domain.ErrDuplicateActiveSession)
	})
}

func TestTherapyService_StartAndStepSequencing(t *testing.T) {
	ctx := context.Background()
	repo := newMockTherapyRepository()
	provider := &mockPackageSnapshotProvider{
		steps: []domain.StepExecution{
			{StepID: "s1", StepOrder: 1, Name: "Step 1", PlannedDurationMinutes: 30, Status: domain.StepStatusNotStarted},
			{StepID: "s2", StepOrder: 2, Name: "Step 2", PlannedDurationMinutes: 20, Status: domain.StepStatusNotStarted},
		},
	}
	svc := NewTherapyService(repo, nil, provider)

	session, err := svc.CheckInAppointment(ctx, CheckInRequest{
		AppointmentID:        "appt-seq",
		ClinicID:             "clinic-1",
		TherapistID:          "therapist-1",
		PreChecklistVerified: true,
	})
	require.NoError(t, err)

	t.Run("unauthorized therapist cannot start session", func(t *testing.T) {
		_, err := svc.StartSession(ctx, session.ID, "wrong-therapist")
		assert.ErrorIs(t, err, domain.ErrUnauthorizedCrossClinic)
	})

	t.Run("start session sets status to IN_PROGRESS and server started_at", func(t *testing.T) {
		started, err := svc.StartSession(ctx, session.ID, "therapist-1")
		require.NoError(t, err)
		assert.Equal(t, domain.SessionStatusInProgress, started.Status)
		assert.NotNil(t, started.StartedAt)
	})

	t.Run("starting step 2 before step 1 completion fails with ErrStepOutOfOrder", func(t *testing.T) {
		err := svc.StartStep(ctx, session.ID, 2, "therapist-1")
		assert.ErrorIs(t, err, domain.ErrStepOutOfOrder)
	})

	t.Run("start step 1, pause and resume", func(t *testing.T) {
		err := svc.StartStep(ctx, session.ID, 1, "therapist-1")
		require.NoError(t, err)

		step, err := repo.GetStepExecution(ctx, session.ID, 1)
		require.NoError(t, err)
		assert.Equal(t, domain.StepStatusInProgress, step.Status)

		err = svc.PauseStep(ctx, session.ID, 1, "therapist-1")
		require.NoError(t, err)

		step, err = repo.GetStepExecution(ctx, session.ID, 1)
		require.NoError(t, err)
		assert.Equal(t, domain.StepStatusPaused, step.Status)

		err = svc.ResumeStep(ctx, session.ID, 1, "therapist-1")
		require.NoError(t, err)

		step, err = repo.GetStepExecution(ctx, session.ID, 1)
		require.NoError(t, err)
		assert.Equal(t, domain.StepStatusInProgress, step.Status)
	})

	t.Run("complete step 1 unlocks step 2", func(t *testing.T) {
		err := svc.CompleteStep(ctx, session.ID, 1, "therapist-1", 1800, "Step 1 completed smoothly", "No erythema")
		require.NoError(t, err)

		step1, err := repo.GetStepExecution(ctx, session.ID, 1)
		require.NoError(t, err)
		assert.Equal(t, domain.StepStatusCompleted, step1.Status)

		// Step 2 can now be started
		err = svc.StartStep(ctx, session.ID, 2, "therapist-1")
		require.NoError(t, err)

		step2, err := repo.GetStepExecution(ctx, session.ID, 2)
		require.NoError(t, err)
		assert.Equal(t, domain.StepStatusInProgress, step2.Status)
	})
}

func TestTherapyService_CompleteSessionAndImmutability(t *testing.T) {
	ctx := context.Background()
	repo := newMockTherapyRepository()
	provider := &mockPackageSnapshotProvider{
		steps: []domain.StepExecution{
			{StepID: "s1", StepOrder: 1, Name: "Step 1", PlannedDurationMinutes: 15, Status: domain.StepStatusNotStarted},
			{StepID: "s2", StepOrder: 2, Name: "Step 2", PlannedDurationMinutes: 15, Status: domain.StepStatusNotStarted},
		},
	}
	svc := NewTherapyService(repo, nil, provider)

	session, err := svc.CheckInAppointment(ctx, CheckInRequest{
		AppointmentID:        "appt-complete",
		ClinicID:             "clinic-1",
		TherapistID:          "therapist-1",
		PreChecklistVerified: true,
	})
	require.NoError(t, err)

	_, err = svc.StartSession(ctx, session.ID, "therapist-1")
	require.NoError(t, err)

	t.Run("cannot complete session when steps incomplete", func(t *testing.T) {
		_, err := svc.CompleteSession(ctx, session.ID, "therapist-1", "Done", "Good", "Drink warm water", nil)
		assert.ErrorIs(t, err, domain.ErrIncompleteSteps)
	})

	t.Run("complete both steps and complete session successfully", func(t *testing.T) {
		// Step 1
		require.NoError(t, svc.StartStep(ctx, session.ID, 1, "therapist-1"))
		require.NoError(t, svc.CompleteStep(ctx, session.ID, 1, "therapist-1", 900, "Done 1", "Good 1"))

		// Step 2
		require.NoError(t, svc.StartStep(ctx, session.ID, 2, "therapist-1"))
		require.NoError(t, svc.CompleteStep(ctx, session.ID, 2, "therapist-1", 900, "Done 2", "Good 2"))

		completed, err := svc.CompleteSession(ctx, session.ID, "therapist-1", "All done", "Patient relaxed", "Drink warm water", nil)
		require.NoError(t, err)
		assert.Equal(t, domain.SessionStatusCompleted, completed.Status)
		assert.NotNil(t, completed.CompletedAt)
	})

	t.Run("completed session is immutable against step modification", func(t *testing.T) {
		err := svc.StartStep(ctx, session.ID, 1, "therapist-1")
		assert.ErrorIs(t, err, domain.ErrSessionCompleted)
	})

	t.Run("modifying therapist notes on completed session requires justification and creates audit record", func(t *testing.T) {
		err := svc.CorrectTherapistNotes(ctx, session.ID, "therapist-1", "Updated clinical observation", "")
		assert.ErrorIs(t, err, domain.ErrCorrectionReasonRequired)

		err = svc.CorrectTherapistNotes(ctx, session.ID, "therapist-1", "Updated observation: patient advised to rest 24h", "Typo correction in discharge notes")
		require.NoError(t, err)

		auditLogs, err := repo.GetSessionAuditLogs(ctx, session.ID)
		require.NoError(t, err)
		require.Len(t, auditLogs, 1)
		assert.Equal(t, "therapist_notes", auditLogs[0].FieldName)
		assert.Equal(t, "Typo correction in discharge notes", auditLogs[0].Reason)
		assert.Equal(t, "therapist-1", auditLogs[0].CorrectedBy)
	})
}

func TestTherapyService_MaterialsAndFollowUps(t *testing.T) {
	ctx := context.Background()
	repo := newMockTherapyRepository()
	svc := NewTherapyService(repo, nil, nil)

	session, err := svc.CheckInAppointment(ctx, CheckInRequest{
		AppointmentID:        "appt-mat",
		ClinicID:             "clinic-1",
		TherapistID:          "therapist-1",
		PreChecklistVerified: true,
	})
	require.NoError(t, err)

	t.Run("record materials used", func(t *testing.T) {
		mat, err := svc.RecordMaterial(ctx, session.ID, "therapist-1", RecordMaterialRequest{
			MaterialName: "Mahanarayana Taila",
			Quantity:     150,
			Unit:         "ml",
			BatchNumber:  "BAT-2026-X",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, mat.ID)
		assert.Equal(t, "Mahanarayana Taila", mat.MaterialName)
		assert.Equal(t, 150.0, mat.Quantity)

		items, err := repo.GetSessionMaterials(ctx, session.ID)
		require.NoError(t, err)
		assert.Len(t, items, 1)
	})

	t.Run("create follow-up recommendation and link appointment", func(t *testing.T) {
		days := 7
		fu, err := svc.CreateFollowUp(ctx, session.ID, "therapist-1", "therapist", CreateFollowUpRequest{
			RecommendedIntervalDay: &days,
			Instructions:           "Follow up in one week for Virechana preparation",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, fu.ID)
		assert.Equal(t, "PENDING", fu.Status)

		err = svc.LinkFollowUpAppointment(ctx, fu.ID, "appt-follow-up-200")
		require.NoError(t, err)

		followUps, err := repo.GetSessionFollowUps(ctx, session.ID)
		require.NoError(t, err)
		require.Len(t, followUps, 1)
		assert.Equal(t, "SCHEDULED", followUps[0].Status)
		assert.Equal(t, "appt-follow-up-200", *followUps[0].LinkedAppointmentID)
	})
}

func TestTherapyService_ComplicationLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := newMockTherapyRepository()
	svc := NewTherapyService(repo, nil, nil)

	session, err := svc.CheckInAppointment(ctx, CheckInRequest{
		AppointmentID:        "appt-comp",
		ClinicID:             "clinic-1",
		TherapistID:          "therapist-1",
		PreChecklistVerified: true,
	})
	require.NoError(t, err)

	_, err = svc.StartSession(ctx, session.ID, "therapist-1")
	require.NoError(t, err)

	var compID string
	t.Run("flag complication transitions session to COMPLICATION_ALERT", func(t *testing.T) {
		comp, err := svc.FlagComplication(ctx, session.ID, "therapist-1", "therapist", FlagComplicationRequest{
			Severity:              domain.SeverityHigh,
			Symptoms:              "Sudden drop in blood pressure and acute dizziness",
			ImmediateActionsTaken: "Therapy halted immediately, patient placed in supine position, administered warm electrolytes",
			AttendingDoctorID:     "doc-on-call",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, comp.ID)
		compID = comp.ID

		s, err := repo.GetSessionByID(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.SessionStatusComplicationAlert, s.Status)
	})

	t.Run("doctor resolves complication and resumes session", func(t *testing.T) {
		err := svc.ResolveComplication(ctx, compID, "doc-on-call", "Vitals stabilized at 115/75. Patient safe to resume gentle massage.", domain.SessionStatusInProgress)
		require.NoError(t, err)

		s, err := repo.GetSessionByID(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.SessionStatusInProgress, s.Status)

		comp, err := repo.GetComplicationByID(ctx, compID)
		require.NoError(t, err)
		assert.Equal(t, domain.ComplicationResolved, comp.Status)
		assert.NotNil(t, comp.DoctorClearedAt)
		assert.Equal(t, "doc-on-call", comp.AttendingDoctorID)
	})
}

func TestTherapyService_AccessControl(t *testing.T) {
	ctx := context.Background()
	repo := newMockTherapyRepository()
	verifier := &mockAppointmentVerifier{
		isEligible: true,
		patientID:  "patient-alice",
		doctorID:   "doctor-bob",
		roomID:     "room-1",
		packageID:  "pkg-1",
	}
	svc := NewTherapyService(repo, verifier, nil)

	session, err := svc.CheckInAppointment(ctx, CheckInRequest{
		AppointmentID:        "appt-sec",
		ClinicID:             "clinic-alpha",
		TherapistID:          "therapist-charlie",
		PreChecklistVerified: true,
	})
	require.NoError(t, err)

	t.Run("cross clinic staff denied access", func(t *testing.T) {
		_, err := svc.GetSession(ctx, session.ID, "staff-other", "therapist", "clinic-beta")
		assert.ErrorIs(t, err, domain.ErrUnauthorizedCrossClinic)
	})

	t.Run("unrelated patient cannot access session", func(t *testing.T) {
		_, err := svc.GetSession(ctx, session.ID, "patient-mallory", "patient", "")
		assert.ErrorIs(t, err, domain.ErrUnauthorizedPatientAccess)
	})

	t.Run("owner patient can access session, notes are masked", func(t *testing.T) {
		s, err := svc.GetSession(ctx, session.ID, "patient-alice", "patient", "")
		require.NoError(t, err)
		assert.Equal(t, session.ID, s.ID)
		// Patient should not see internal therapist clinical notes
		assert.Empty(t, s.TherapistNotes)
	})

	t.Run("assigned therapist can access session with therapist notes", func(t *testing.T) {
		s, err := svc.GetSession(ctx, session.ID, "therapist-charlie", "therapist", "clinic-alpha")
		require.NoError(t, err)
		assert.Equal(t, session.ID, s.ID)
	})
}
