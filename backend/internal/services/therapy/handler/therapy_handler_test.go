package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockTherapyService implements service.TherapyService
type mockTherapyService struct {
	mock.Mock
}

func (m *mockTherapyService) CheckInAppointment(ctx context.Context, req service.CheckInRequest) (*domain.TherapySession, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TherapySession), args.Error(1)
}

func (m *mockTherapyService) StartSession(ctx context.Context, sessionID, therapistID string) (*domain.TherapySession, error) {
	args := m.Called(ctx, sessionID, therapistID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TherapySession), args.Error(1)
}

func (m *mockTherapyService) StartStep(ctx context.Context, sessionID string, stepOrder int, therapistID string) error {
	args := m.Called(ctx, sessionID, stepOrder, therapistID)
	return args.Error(0)
}

func (m *mockTherapyService) PauseStep(ctx context.Context, sessionID string, stepOrder int, therapistID string) error {
	args := m.Called(ctx, sessionID, stepOrder, therapistID)
	return args.Error(0)
}

func (m *mockTherapyService) ResumeStep(ctx context.Context, sessionID string, stepOrder int, therapistID string) error {
	args := m.Called(ctx, sessionID, stepOrder, therapistID)
	return args.Error(0)
}

func (m *mockTherapyService) CompleteStep(ctx context.Context, sessionID string, stepOrder int, therapistID string, actualSeconds int, notes, obs string) error {
	args := m.Called(ctx, sessionID, stepOrder, therapistID, actualSeconds, notes, obs)
	return args.Error(0)
}

func (m *mockTherapyService) RecordMaterial(ctx context.Context, sessionID, therapistID string, req service.RecordMaterialRequest) (*domain.SessionMaterial, error) {
	args := m.Called(ctx, sessionID, therapistID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SessionMaterial), args.Error(1)
}

func (m *mockTherapyService) CompleteSession(ctx context.Context, sessionID, therapistID, notes, obs, postCare string, postCareItems json.RawMessage) (*domain.TherapySession, error) {
	args := m.Called(ctx, sessionID, therapistID, notes, obs, postCare, postCareItems)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TherapySession), args.Error(1)
}

func (m *mockTherapyService) FlagComplication(ctx context.Context, sessionID, reporterID, role string, req service.FlagComplicationRequest) (*domain.ComplicationEvent, error) {
	args := m.Called(ctx, sessionID, reporterID, role, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ComplicationEvent), args.Error(1)
}

func (m *mockTherapyService) ResolveComplication(ctx context.Context, compID, doctorID, notes string, resumeStatus domain.SessionStatus) error {
	args := m.Called(ctx, compID, doctorID, notes, resumeStatus)
	return args.Error(0)
}

func (m *mockTherapyService) CreateFollowUp(ctx context.Context, sessionID, authorID, role string, req service.CreateFollowUpRequest) (*domain.SessionFollowUp, error) {
	args := m.Called(ctx, sessionID, authorID, role, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SessionFollowUp), args.Error(1)
}

func (m *mockTherapyService) LinkFollowUpAppointment(ctx context.Context, followUpID, appointmentID string) error {
	args := m.Called(ctx, followUpID, appointmentID)
	return args.Error(0)
}

func (m *mockTherapyService) CorrectTherapistNotes(ctx context.Context, sessionID, correctedBy, newNotes, reason string) error {
	args := m.Called(ctx, sessionID, correctedBy, newNotes, reason)
	return args.Error(0)
}

func (m *mockTherapyService) GetSession(ctx context.Context, sessionID, requesterID, role, clinicID string) (*domain.TherapySession, error) {
	args := m.Called(ctx, sessionID, requesterID, role, clinicID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TherapySession), args.Error(1)
}

func (m *mockTherapyService) ListAssignedSessionsForTherapist(ctx context.Context, therapistID, clinicID string, status *domain.SessionStatus) ([]*domain.TherapySession, error) {
	args := m.Called(ctx, therapistID, clinicID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TherapySession), args.Error(1)
}

func (m *mockTherapyService) ListPatientSessions(ctx context.Context, patientID string) ([]*domain.TherapySession, error) {
	args := m.Called(ctx, patientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TherapySession), args.Error(1)
}

func setupTestRouter(svc service.TherapyService, userID, role, clinicID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Inject auth context simulation
	r.Use(func(c *gin.Context) {
		if userID != "" {
			c.Set(middleware.ContextUserIDKey, userID)
		}
		if role != "" {
			c.Set(middleware.ContextUserRoleKey, role)
		}
		if clinicID != "" {
			c.Set(middleware.ContextClinicIDKey, clinicID)
		}
		c.Next()
	})

	handler := NewTherapyHTTPHandler(svc, nil)
	v1 := r.Group("/api/v1")
	handler.RegisterRoutes(v1)

	return r
}

func TestTherapyHandler_CheckIn(t *testing.T) {
	mockSvc := new(mockTherapyService)
	router := setupTestRouter(mockSvc, "therapist-10", "therapist", "clinic-10")

	t.Run("successful check-in returns 201 Created", func(t *testing.T) {
		expectedSession := &domain.TherapySession{
			ID:                   "ses-100",
			AppointmentID:        "appt-100",
			ClinicID:             "clinic-10",
			TherapistID:          "therapist-10",
			Status:               domain.SessionStatusCheckedIn,
			PreChecklistVerified: true,
		}

		mockSvc.On("CheckInAppointment", mock.Anything, mock.MatchedBy(func(req service.CheckInRequest) bool {
			return req.AppointmentID == "appt-100" && req.PreChecklistVerified
		})).Return(expectedSession, nil).Once()

		body := map[string]interface{}{
			"appointment_id":         "appt-100",
			"pre_checklist_verified": true,
			"baseline_vitals": map[string]interface{}{
				"bp_systolic":   120,
				"bp_diastolic":  80,
				"pulse":         72,
				"temperature_c": 36.8,
			},
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/check-in", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("missing checklist returns 400 Bad Request", func(t *testing.T) {
		mockSvc.On("CheckInAppointment", mock.Anything, mock.Anything).
			Return(nil, domain.ErrPreChecklistRequired).Once()

		body := map[string]interface{}{
			"appointment_id":         "appt-200",
			"pre_checklist_verified": false,
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/check-in", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("duplicate active session returns 409 Conflict", func(t *testing.T) {
		mockSvc.On("CheckInAppointment", mock.Anything, mock.Anything).
			Return(nil, domain.ErrDuplicateActiveSession).Once()

		body := map[string]interface{}{
			"appointment_id":         "appt-300",
			"pre_checklist_verified": true,
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/check-in", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestTherapyHandler_StepSequencingAndOrdering(t *testing.T) {
	mockSvc := new(mockTherapyService)
	router := setupTestRouter(mockSvc, "therapist-1", "therapist", "clinic-1")

	t.Run("step out of order returns 409 Conflict", func(t *testing.T) {
		mockSvc.On("StartStep", mock.Anything, "ses-1", 2, "therapist-1").
			Return(domain.ErrStepOutOfOrder).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/ses-1/steps/2/start", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("start step in order returns 200 OK", func(t *testing.T) {
		mockSvc.On("StartStep", mock.Anything, "ses-1", 1, "therapist-1").
			Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/ses-1/steps/1/start", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("complete step returns 200 OK", func(t *testing.T) {
		mockSvc.On("CompleteStep", mock.Anything, "ses-1", 1, "therapist-1", 1800, "Good massage", "None").
			Return(nil).Once()

		body := map[string]interface{}{
			"actual_duration_seconds": 1800,
			"therapist_notes":         "Good massage",
			"observations":            "None",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/ses-1/steps/1/complete", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTherapyHandler_ComplicationsAndFollowUps(t *testing.T) {
	mockSvc := new(mockTherapyService)
	router := setupTestRouter(mockSvc, "therapist-1", "therapist", "clinic-1")

	t.Run("flag complication returns 201 Created", func(t *testing.T) {
		comp := &domain.ComplicationEvent{
			ID:           "comp-1",
			SessionID:    "ses-1",
			ReporterID:   "therapist-1",
			ReporterRole: "therapist",
			Severity:     domain.SeverityHigh,
			Symptoms:     "Nausea and dizziness",
			Status:       domain.ComplicationUnresolved,
			CreatedAt:    time.Now().UTC(),
		}

		mockSvc.On("FlagComplication", mock.Anything, "ses-1", "therapist-1", "therapist", mock.Anything).
			Return(comp, nil).Once()

		body := map[string]interface{}{
			"severity": domain.SeverityHigh,
			"symptoms": "Nausea and dizziness",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/ses-1/complications", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("create follow-up recommendation returns 201 Created", func(t *testing.T) {
		fu := &domain.SessionFollowUp{
			ID:           "fu-1",
			SessionID:    "ses-1",
			Instructions: "Come back next week",
			Status:       "PENDING",
		}

		mockSvc.On("CreateFollowUp", mock.Anything, "ses-1", "therapist-1", "therapist", mock.Anything).
			Return(fu, nil).Once()

		body := map[string]interface{}{
			"instructions": "Come back next week",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/ses-1/follow-ups", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestTherapyHandler_NotesCorrectionWithAudit(t *testing.T) {
	mockSvc := new(mockTherapyService)
	router := setupTestRouter(mockSvc, "therapist-1", "therapist", "clinic-1")

	t.Run("notes correction without justification reason fails with 400 Bad Request", func(t *testing.T) {
		body := map[string]interface{}{
			"therapist_notes": "Corrected note",
			"reason":          "",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/sessions/ses-1/notes", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("notes correction with reason returns 200 OK", func(t *testing.T) {
		mockSvc.On("CorrectTherapistNotes", mock.Anything, "ses-1", "therapist-1", "Corrected note", "Typo fixed").
			Return(nil).Once()

		body := map[string]interface{}{
			"therapist_notes": "Corrected note",
			"reason":          "Typo fixed",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/sessions/ses-1/notes", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestTherapyHandler_AccessControl(t *testing.T) {
	mockSvc := new(mockTherapyService)
	router := setupTestRouter(mockSvc, "patient-mallory", "patient", "")

	t.Run("unauthorized patient access returns 403 Forbidden", func(t *testing.T) {
		mockSvc.On("GetSession", mock.Anything, "ses-secret", "patient-mallory", "patient", "").
			Return(nil, domain.ErrUnauthorizedPatientAccess).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/ses-secret", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("authorized access returns 200 OK", func(t *testing.T) {
		session := &domain.TherapySession{
			ID:        "ses-allowed",
			PatientID: "patient-mallory",
			Status:    domain.SessionStatusCompleted,
		}

		mockSvc.On("GetSession", mock.Anything, "ses-allowed", "patient-mallory", "patient", "").
			Return(session, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/ses-allowed", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
