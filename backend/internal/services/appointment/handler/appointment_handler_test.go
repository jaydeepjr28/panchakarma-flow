package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/handler"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/repository"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	appointmentv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/appointment/v1"
)

type MockService struct {
	holds        map[string]*domain.AppointmentHold
	appointments map[string]*domain.Appointment
	waitlists    map[string]*domain.AppointmentWaitlist
}

func NewMockService() *MockService {
	return &MockService{
		holds:        make(map[string]*domain.AppointmentHold),
		appointments: make(map[string]*domain.Appointment),
		waitlists:    make(map[string]*domain.AppointmentWaitlist),
	}
}

func (m *MockService) SearchAvailableSlots(ctx context.Context, req service.SearchSlotsRequest) (*service.SearchSlotsResponse, error) {
	return &service.SearchSlotsResponse{
		ClinicID: req.ClinicID,
		AvailableSlots: []domain.SearchSlot{
			{
				ClinicID:  req.ClinicID,
				DoctorID:  "doc_1",
				RoomID:    "room_1",
				PackageID: req.PackageID,
				StartTime: time.Now().UTC().Add(24 * time.Hour),
				EndTime:   time.Now().UTC().Add(25 * time.Hour),
				DurationM: 60,
			},
		},
	}, nil
}

func (m *MockService) CreateHold(ctx context.Context, req service.CreateHoldRequest) (*service.CreateHoldResponse, error) {
	holdToken := "hld_" + uuid.New().String()
	hold := &domain.AppointmentHold{
		ID:        uuid.New().String(),
		HoldToken: holdToken,
		PatientID: req.PatientID,
		ClinicID:  req.ClinicID,
		DoctorID:  req.DoctorID,
		RoomID:    req.RoomID,
		PackageID: req.PackageID,
		StartTime: req.StartTime,
		EndTime:   req.StartTime.Add(time.Duration(req.DurationMinutes) * time.Minute),
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
		Status:    domain.HoldStatusActive,
	}
	m.holds[holdToken] = hold
	return &service.CreateHoldResponse{
		HoldID:     hold.ID,
		HoldToken:  holdToken,
		ExpiresAt:  hold.ExpiresAt,
		TTLSeconds: 600,
		DoctorID:   req.DoctorID,
		RoomID:     req.RoomID,
	}, nil
}

func (m *MockService) ConfirmAppointment(ctx context.Context, req service.ConfirmAppointmentRequest) (*domain.Appointment, error) {
	hold, ok := m.holds[req.HoldToken]
	if !ok {
		return nil, domain.ErrHoldNotFound
	}
	appt := &domain.Appointment{
		ID:        uuid.New().String(),
		PatientID: hold.PatientID,
		ClinicID:  hold.ClinicID,
		DoctorID:  hold.DoctorID,
		RoomID:    hold.RoomID,
		PackageID: hold.PackageID,
		StartTime: hold.StartTime,
		EndTime:   hold.EndTime,
		Status:    domain.StatusConfirmed,
	}
	m.appointments[appt.ID] = appt
	return appt, nil
}

func (m *MockService) BookDirectAppointment(ctx context.Context, req service.DirectBookingRequest) (*domain.Appointment, error) {
	appt := &domain.Appointment{
		ID:        uuid.New().String(),
		PatientID: req.PatientID,
		ClinicID:  req.ClinicID,
		DoctorID:  req.DoctorID,
		RoomID:    req.RoomID,
		PackageID: req.PackageID,
		StartTime: req.StartTime,
		EndTime:   req.StartTime.Add(time.Duration(req.DurationMinutes) * time.Minute),
		Status:    domain.StatusConfirmed,
	}
	m.appointments[appt.ID] = appt
	return appt, nil
}

func (m *MockService) BookMultiSessionPackage(ctx context.Context, req service.MultiSessionPackageRequest) ([]*domain.Appointment, error) {
	pkgBookingID := uuid.New().String()
	var list []*domain.Appointment
	for idx, start := range req.SessionStarts {
		appt := &domain.Appointment{
			ID:               uuid.New().String(),
			PatientID:        req.PatientID,
			ClinicID:         req.ClinicID,
			PackageID:        req.PackageID,
			StartTime:        start,
			EndTime:          start.Add(time.Duration(req.DurationMinutes) * time.Minute),
			Status:           domain.StatusConfirmed,
			SessionNumber:    idx + 1,
			TotalSessions:    len(req.SessionStarts),
			PackageBookingID: &pkgBookingID,
		}
		list = append(list, appt)
	}
	return list, nil
}

func (m *MockService) CancelAppointment(ctx context.Context, apptID string, patientID string, userRole string, clinicID string, reason string) error {
	appt, ok := m.appointments[apptID]
	if !ok {
		return domain.ErrAppointmentNotFound
	}
	appt.Status = domain.StatusCancelled
	appt.CancellationReason = reason
	return nil
}

func (m *MockService) RescheduleAppointment(ctx context.Context, req service.RescheduleRequest) (*domain.Appointment, error) {
	oldAppt, ok := m.appointments[req.AppointmentID]
	if !ok {
		return nil, domain.ErrAppointmentNotFound
	}
	oldAppt.Status = domain.StatusRescheduled
	newAppt := &domain.Appointment{
		ID:        uuid.New().String(),
		PatientID: oldAppt.PatientID,
		ClinicID:  oldAppt.ClinicID,
		DoctorID:  oldAppt.DoctorID,
		RoomID:    oldAppt.RoomID,
		PackageID: oldAppt.PackageID,
		StartTime: req.NewStartTime,
		EndTime:   req.NewStartTime.Add(90 * time.Minute),
		Status:    domain.StatusConfirmed,
	}
	m.appointments[newAppt.ID] = newAppt
	return newAppt, nil
}

func (m *MockService) GetAppointment(ctx context.Context, apptID string, requesterID string, role string, clinicID string) (*domain.Appointment, error) {
	appt, ok := m.appointments[apptID]
	if !ok {
		return nil, domain.ErrAppointmentNotFound
	}
	return appt, nil
}

func (m *MockService) ListAppointments(ctx context.Context, filter repository.ListFilter, requesterID string, role string, clinicID string) ([]*domain.Appointment, int, error) {
	var list []*domain.Appointment
	for _, a := range m.appointments {
		list = append(list, a)
	}
	return list, len(list), nil
}

func (m *MockService) JoinWaitlist(ctx context.Context, req service.JoinWaitlistRequest) (*domain.AppointmentWaitlist, error) {
	w := &domain.AppointmentWaitlist{
		ID:                 uuid.New().String(),
		ClinicID:           req.ClinicID,
		PatientID:          req.PatientID,
		PackageID:          req.PackageID,
		PreferredDateStart: req.PreferredDateStart,
		PreferredDateEnd:   req.PreferredDateEnd,
		Status:             domain.WaitlistStatusWaiting,
	}
	m.waitlists[w.ID] = w
	return w, nil
}

func (m *MockService) GetWaitlistEntries(ctx context.Context, clinicID string, patientID *string, status *domain.WaitlistStatus) ([]*domain.AppointmentWaitlist, error) {
	var list []*domain.AppointmentWaitlist
	for _, w := range m.waitlists {
		list = append(list, w)
	}
	return list, nil
}

func (m *MockService) AcceptWaitlistOffer(ctx context.Context, waitlistID string, patientID string) (*domain.Appointment, error) {
	w, ok := m.waitlists[waitlistID]
	if !ok {
		return nil, domain.ErrWaitlistNotFound
	}
	w.Status = domain.WaitlistStatusFulfilled
	appt := &domain.Appointment{
		ID:        uuid.New().String(),
		PatientID: patientID,
		ClinicID:  w.ClinicID,
		PackageID: w.PackageID,
		Status:    domain.StatusConfirmed,
	}
	m.appointments[appt.ID] = appt
	return appt, nil
}

func (m *MockService) ExpireStaleHolds(ctx context.Context) (int64, error) {
	return 0, nil
}

func setupTestRouter(svc service.AppointmentService, tm *auth.TokenManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/api/v1")
	h := handler.NewAppointmentHTTPHandler(svc, tm)
	h.RegisterRoutes(rg)
	return r
}

func TestAppointmentHTTPHandler_SearchAndHold(t *testing.T) {
	svc := NewMockService()
	tm, _ := auth.NewTokenManager("super-secret-key-that-is-at-least-32-bytes-long", time.Hour, 24*time.Hour, nil)
	r := setupTestRouter(svc, tm)

	token, _, _, _ := tm.GenerateAccessToken("pat-123", "pat@example.com", "patient", "")

	t.Run("SearchAvailability_Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/availability?clinic_id=cln-1&start_date=2026-11-01&end_date=2026-11-05", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
	})

	t.Run("CreateHold_Success", func(t *testing.T) {
		body := map[string]interface{}{
			"clinic_id":        "cln-1",
			"package_id":       "pkg-1",
			"doctor_id":        "doc-1",
			"room_id":          "room-1",
			"auto_assign_doctor": false,
			"start_time":       time.Now().UTC().Add(24 * time.Hour),
			"duration_minutes": 90,
		}
		data, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings/hold", bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				HoldToken  string `json:"hold_token"`
				TTLSeconds int    `json:"ttl_seconds"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Data.HoldToken == "" {
			t.Fatalf("expected hold token in response")
		}

		// Now confirm the appointment with the hold token
		confirmBody := map[string]interface{}{
			"hold_token":        resp.Data.HoldToken,
			"consent_record_id": "consent-001",
			"notes":             "No nuts allergy",
		}
		confirmData, _ := json.Marshal(confirmBody)
		wConfirm := httptest.NewRecorder()
		reqConfirm, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings/confirm", bytes.NewReader(confirmData))
		reqConfirm.Header.Set("Authorization", "Bearer "+token)
		reqConfirm.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(wConfirm, reqConfirm)

		if wConfirm.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", wConfirm.Code, wConfirm.Body.String())
		}
	})

	t.Run("UnauthorizedAccess_WithoutToken", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings/hold", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})
}

// MockRepoForGRPC implements repository for testing gRPC handler.
type MockRepoForGRPC struct {
	repository.AppointmentRepository
	appts map[string]*domain.Appointment
}

func (m *MockRepoForGRPC) GetAppointmentByID(ctx context.Context, id string) (*domain.Appointment, error) {
	a, ok := m.appts[id]
	if !ok {
		return nil, domain.ErrAppointmentNotFound
	}
	return a, nil
}

func TestAppointmentGRPCHandler_VerifyAndSummary(t *testing.T) {
	repo := &MockRepoForGRPC{
		appts: map[string]*domain.Appointment{
			"appt-confirmed": {
				ID:        "appt-confirmed",
				PatientID: "pat-1",
				ClinicID:  "cln-1",
				DoctorID:  "doc-1",
				RoomID:    "room-1",
				PackageID: "pkg-1",
				Status:    domain.StatusConfirmed,
				StartTime: time.Now().UTC(),
				EndTime:   time.Now().UTC().Add(time.Hour),
			},
			"appt-cancelled": {
				ID:        "appt-cancelled",
				PatientID: "pat-2",
				ClinicID:  "cln-1",
				Status:    domain.StatusCancelled,
			},
		},
	}

	grpcHandler := handler.NewAppointmentGRPCHandler(repo)
	ctx := context.Background()

	t.Run("VerifyAppointment_ConfirmedEligible", func(t *testing.T) {
		resp, err := grpcHandler.VerifyAppointmentForSession(ctx, &appointmentv1.VerifyAppointmentRequest{
			AppointmentId: "appt-confirmed",
			ClinicId:      "cln-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.IsEligible {
			t.Fatalf("expected appointment to be eligible, got rejection: %s", resp.RejectionReason)
		}
		if resp.DoctorId != "doc-1" {
			t.Errorf("expected doctor_id doc-1, got %s", resp.DoctorId)
		}
	})

	t.Run("VerifyAppointment_CancelledNotEligible", func(t *testing.T) {
		resp, err := grpcHandler.VerifyAppointmentForSession(ctx, &appointmentv1.VerifyAppointmentRequest{
			AppointmentId: "appt-cancelled",
			ClinicId:      "cln-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.IsEligible {
			t.Fatalf("expected cancelled appointment not to be eligible")
		}
		if resp.RejectionReason != "CANCELLED" {
			t.Errorf("expected rejection reason CANCELLED, got %s", resp.RejectionReason)
		}
	})

	t.Run("VerifyAppointment_ClinicMismatch", func(t *testing.T) {
		resp, err := grpcHandler.VerifyAppointmentForSession(ctx, &appointmentv1.VerifyAppointmentRequest{
			AppointmentId: "appt-confirmed",
			ClinicId:      "cln-other-clinic",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.IsEligible {
			t.Fatalf("expected clinic mismatch not to be eligible")
		}
		if resp.RejectionReason != "CLINIC_MISMATCH" {
			t.Errorf("expected CLINIC_MISMATCH, got %s", resp.RejectionReason)
		}
	})

	t.Run("GetAppointmentSummary_Success", func(t *testing.T) {
		resp, err := grpcHandler.GetAppointmentSummary(ctx, &appointmentv1.GetAppointmentSummaryRequest{
			AppointmentId: "appt-confirmed",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Status != "CONFIRMED" {
			t.Errorf("expected status CONFIRMED, got %s", resp.Status)
		}
	})
}
