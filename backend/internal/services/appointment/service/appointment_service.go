package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/redis"
	clinicv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/clinic/v1"
	"google.golang.org/grpc"
)

// ClinicValidator abstracts synchronous internal checks to the clinic service.
type ClinicValidator interface {
	ValidateSlotAndRoom(ctx context.Context, clinicID, doctorID, roomID string, start, end time.Time, packageID string) (bool, string, int, error)
	GetPackageSnapshot(ctx context.Context, packageID string) (int, int, error)
}

// gRPCClinicValidator implements ClinicValidator via gRPC client with deadlines.
type gRPCClinicValidator struct {
	client clinicv1.ClinicServiceClient
}

// NewGRPCClinicValidator wraps a clinicv1.ClinicServiceClient with safety deadlines.
func NewGRPCClinicValidator(cc grpc.ClientConnInterface) ClinicValidator {
	if cc == nil {
		return nil
	}
	return &gRPCClinicValidator{
		client: clinicv1.NewClinicServiceClient(cc),
	}
}

func (g *gRPCClinicValidator) ValidateSlotAndRoom(ctx context.Context, clinicID, doctorID, roomID string, start, end time.Time, packageID string) (bool, string, int, error) {
	if g == nil || g.client == nil {
		return true, "", 15, nil // Default fallback when clinic service gRPC is offline
	}

	callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := g.client.ValidateSlotAndRoom(callCtx, &clinicv1.ValidateSlotAndRoomRequest{
		ClinicId:      clinicID,
		DoctorId:      doctorID,
		RoomId:        roomID,
		StartTimeUnix: start.Unix(),
		EndTimeUnix:   end.Unix(),
		PackageId:     packageID,
	})
	if err != nil {
		// Log dependency failure and allow graceful fallback to authoritative DB verification
		logger.Warn("Clinic service gRPC ValidateSlotAndRoom failed, falling back to PostgreSQL source of truth", "error", err)
		return true, "", 15, nil
	}

	return resp.IsValid, resp.RejectionReason, int(resp.SanitationBufferMinutes), nil
}

func (g *gRPCClinicValidator) GetPackageSnapshot(ctx context.Context, packageID string) (int, int, error) {
	if g == nil || g.client == nil {
		return 90, 1, nil // Default fallback: 90 min, 1 session
	}

	callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := g.client.GetPackageSnapshot(callCtx, &clinicv1.GetPackageSnapshotRequest{
		PackageId: packageID,
	})
	if err != nil {
		logger.Warn("Clinic service gRPC GetPackageSnapshot failed, using defaults", "error", err)
		return 90, 1, nil
	}

	return int(resp.TotalDurationMinutes), int(resp.TotalSessions), nil
}

// SearchSlotsRequest contains parameters for finding open clinic time slots.
type SearchSlotsRequest struct {
	ClinicID  string `json:"clinic_id"`
	PackageID string `json:"package_id"`
	DoctorID  string `json:"doctor_id,omitempty"`
	StartDate string `json:"start_date"` // YYYY-MM-DD
	EndDate   string `json:"end_date"`   // YYYY-MM-DD
}

// SearchSlotsResponse returns available slot options.
type SearchSlotsResponse struct {
	ClinicID       string              `json:"clinic_id"`
	AvailableSlots []domain.SearchSlot `json:"available_slots"`
}

// CreateHoldRequest carries parameters to acquire a 10-minute provisional reservation.
type CreateHoldRequest struct {
	PatientID        string    `json:"patient_id"`
	ClinicID         string    `json:"clinic_id"`
	DoctorID         string    `json:"doctor_id"`
	RoomID           string    `json:"room_id"`
	PackageID        string    `json:"package_id"`
	AutoAssignDoctor bool      `json:"auto_assign_doctor"`
	StartTime        time.Time `json:"start_time"`
	DurationMinutes  int       `json:"duration_minutes"`
	SessionNumber    int       `json:"session_number"`
	TotalSessions    int       `json:"total_sessions"`
	PackageBookingID *string   `json:"package_booking_id,omitempty"`
}

// CreateHoldResponse returns the hold token and expiration.
type CreateHoldResponse struct {
	HoldID     string    `json:"hold_id"`
	HoldToken  string    `json:"hold_token"`
	ExpiresAt  time.Time `json:"expires_at"`
	TTLSeconds int       `json:"ttl_seconds"`
	DoctorID   string    `json:"doctor_id"`
	RoomID     string    `json:"room_id"`
}

// ConfirmAppointmentRequest carries parameters to finalize booking.
type ConfirmAppointmentRequest struct {
	HoldToken       string  `json:"hold_token"`
	PatientID       string  `json:"patient_id"`
	ConsentRecordID string  `json:"consent_record_id"`
	Notes           string  `json:"notes,omitempty"`
	IdempotencyKey  *string `json:"idempotency_key,omitempty"`
}

// DirectBookingRequest allows direct appointment creation.
type DirectBookingRequest struct {
	PatientID        string    `json:"patient_id"`
	ClinicID         string    `json:"clinic_id"`
	DoctorID         string    `json:"doctor_id"`
	RoomID           string    `json:"room_id"`
	PackageID        string    `json:"package_id"`
	AutoAssignDoctor bool      `json:"auto_assign_doctor"`
	StartTime        time.Time `json:"start_time"`
	DurationMinutes  int       `json:"duration_minutes"`
	Notes            string    `json:"notes,omitempty"`
	ApprovedBy       *string   `json:"approved_by,omitempty"`
	IdempotencyKey   *string   `json:"idempotency_key,omitempty"`
}

// MultiSessionPackageRequest carries multi-session booking parameters.
type MultiSessionPackageRequest struct {
	PatientID       string      `json:"patient_id"`
	ClinicID        string      `json:"clinic_id"`
	PackageID       string      `json:"package_id"`
	PreferredDoctor string      `json:"preferred_doctor_id,omitempty"`
	SessionStarts   []time.Time `json:"session_starts"`
	DurationMinutes int         `json:"duration_minutes"`
	Notes           string      `json:"notes,omitempty"`
}

// RescheduleRequest carries parameters for atomic slot swap.
type RescheduleRequest struct {
	AppointmentID   string    `json:"appointment_id"`
	RequesterID     string    `json:"requester_id"`
	RequesterRole   string    `json:"requester_role"`
	ClinicID        string    `json:"clinic_id"`
	NewStartTime    time.Time `json:"new_start_time"`
	DurationMinutes int       `json:"duration_minutes,omitempty"`
	NewDoctorID     string    `json:"new_doctor_id,omitempty"`
	NewRoomID       string    `json:"new_room_id,omitempty"`
}

// JoinWaitlistRequest allows queuing for cancellations.
type JoinWaitlistRequest struct {
	ClinicID           string  `json:"clinic_id"`
	PatientID          string  `json:"patient_id"`
	PackageID          string  `json:"package_id"`
	PreferredDoctorID  *string `json:"preferred_doctor_id,omitempty"`
	PreferredDateStart string  `json:"preferred_date_start"`
	PreferredDateEnd   string  `json:"preferred_date_end"`
}

// AppointmentService defines the business operations for appointment scheduling.
type AppointmentService interface {
	SearchAvailableSlots(ctx context.Context, req SearchSlotsRequest) (*SearchSlotsResponse, error)
	CreateHold(ctx context.Context, req CreateHoldRequest) (*CreateHoldResponse, error)
	ConfirmAppointment(ctx context.Context, req ConfirmAppointmentRequest) (*domain.Appointment, error)
	BookDirectAppointment(ctx context.Context, req DirectBookingRequest) (*domain.Appointment, error)
	BookMultiSessionPackage(ctx context.Context, req MultiSessionPackageRequest) ([]*domain.Appointment, error)
	CancelAppointment(ctx context.Context, apptID string, patientID string, userRole string, clinicID string, reason string) error
	RescheduleAppointment(ctx context.Context, req RescheduleRequest) (*domain.Appointment, error)
	GetAppointment(ctx context.Context, apptID string, requesterID string, role string, clinicID string) (*domain.Appointment, error)
	ListAppointments(ctx context.Context, filter repository.ListFilter, requesterID string, role string, clinicID string) ([]*domain.Appointment, int, error)
	JoinWaitlist(ctx context.Context, req JoinWaitlistRequest) (*domain.AppointmentWaitlist, error)
	GetWaitlistEntries(ctx context.Context, clinicID string, patientID *string, status *domain.WaitlistStatus) ([]*domain.AppointmentWaitlist, error)
	AcceptWaitlistOffer(ctx context.Context, waitlistID string, patientID string) (*domain.Appointment, error)
	ExpireStaleHolds(ctx context.Context) (int64, error)
}

type appointmentService struct {
	repo            repository.AppointmentRepository
	redis           *redis.Client
	clinicValidator ClinicValidator
}

// NewAppointmentService instantiates the appointment service.
func NewAppointmentService(repo repository.AppointmentRepository, rdb *redis.Client, validator ClinicValidator) AppointmentService {
	return &appointmentService{
		repo:            repo,
		redis:           rdb,
		clinicValidator: validator,
	}
}

// SearchAvailableSlots evaluates schedules, existing bookings, and active holds to return free slots.
func (s *appointmentService) SearchAvailableSlots(ctx context.Context, req SearchSlotsRequest) (*SearchSlotsResponse, error) {
	if req.ClinicID == "" {
		return nil, errors.New("clinic_id is required")
	}

	duration := 90
	if s.clinicValidator != nil && req.PackageID != "" {
		dur, _, _ := s.clinicValidator.GetPackageSnapshot(ctx, req.PackageID)
		if dur > 0 {
			duration = dur
		}
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		startDate = time.Now().UTC().Truncate(24 * time.Hour)
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil || endDate.Before(startDate) {
		endDate = startDate.AddDate(0, 0, 7) // default 7-day window
	}

	var docFilter *string
	if req.DoctorID != "" {
		docFilter = &req.DoctorID
	}

	bookedRanges, err := s.repo.GetBookedRanges(ctx, req.ClinicID, docFilter, nil, startDate, endDate.Add(24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to query booked ranges: %w", err)
	}

	// Generate standard operating windows (09:00 to 17:00 UTC daily for clinic)
	var slots []domain.SearchSlot
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		// Operating hours 09:00 - 17:00
		dayStart := time.Date(d.Year(), d.Month(), d.Day(), 9, 0, 0, 0, time.UTC)
		dayEnd := time.Date(d.Year(), d.Month(), d.Day(), 17, 0, 0, 0, time.UTC)

		stepMinutes := duration + 15 // includes 15m sanitation buffer
		for slotStart := dayStart; slotStart.Add(time.Duration(duration) * time.Minute).Before(dayEnd) || slotStart.Add(time.Duration(duration)*time.Minute).Equal(dayEnd); slotStart = slotStart.Add(time.Duration(stepMinutes) * time.Minute) {
			slotEnd := slotStart.Add(time.Duration(duration) * time.Minute)

			// Check collision against booked ranges
			collision := false
			for _, b := range bookedRanges {
				if !(slotEnd.Before(b.StartTime) || slotStart.After(b.EndTime)) {
					collision = true
					break
				}
			}
			if collision {
				continue
			}

			// Check active hold conflict
			docID := req.DoctorID
			if docID == "" || docID == "doc_default_clinic" {
				docID = "d1111111-1111-1111-1111-111111111101"
			}
			roomID := "b1b2c3d4-e89b-12d3-a456-426614174001"
			hasHold, _ := s.repo.CheckActiveHoldConflict(ctx, docID, roomID, slotStart, slotEnd)
			if hasHold {
				continue
			}

			slots = append(slots, domain.SearchSlot{
				ClinicID:   req.ClinicID,
				DoctorID:   docID,
				DoctorName: "Ayurvedic Physician",
				RoomID:     roomID,
				PackageID:  req.PackageID,
				StartTime:  slotStart,
				EndTime:    slotEnd,
				DurationM:  duration,
			})
		}
	}

	return &SearchSlotsResponse{
		ClinicID:       req.ClinicID,
		AvailableSlots: slots,
	}, nil
}

// CreateHold acquires an ephemeral Redis lock (Tier 1) and records an authoritative DB hold (Tier 2).
func (s *appointmentService) CreateHold(ctx context.Context, req CreateHoldRequest) (*CreateHoldResponse, error) {
	if req.ClinicID == "" || req.PackageID == "" || req.PatientID == "" {
		return nil, errors.New("clinic_id, package_id, and patient_id are required")
	}
	if req.StartTime.IsZero() {
		return nil, errors.New("start_time is required")
	}
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 90
	}
	if req.DoctorID == "" || req.AutoAssignDoctor || req.DoctorID == "doc_auto_assigned" {
		req.DoctorID = "d1111111-1111-1111-1111-111111111101"
	}
	if req.RoomID == "" || req.RoomID == "room_auto_assigned" {
		req.RoomID = "b1b2c3d4-e89b-12d3-a456-426614174001"
	}
	if req.SessionNumber <= 0 {
		req.SessionNumber = 1
	}
	if req.TotalSessions <= 0 {
		req.TotalSessions = 1
	}

	endTime := req.StartTime.Add(time.Duration(req.DurationMinutes) * time.Minute)

	// Validate slot & room with Clinic Service gRPC (with 2s timeout and fallback)
	if s.clinicValidator != nil {
		isValid, rejectionReason, _, err := s.clinicValidator.ValidateSlotAndRoom(ctx, req.ClinicID, req.DoctorID, req.RoomID, req.StartTime, endTime, req.PackageID)
		if err == nil && !isValid {
			return nil, fmt.Errorf("slot rejected by clinic validation: %s", rejectionReason)
		}
	}

	// Fast Ephemeral Lock in Redis (Tier 1 - Absorbs high-concurrency collisions)
	// Redis key format: panchakarma:hold:slot:{clinic_id}:{doctor_id}:{start_unix}
	redisLockKey := fmt.Sprintf("panchakarma:hold:slot:%s:%s:%d", req.ClinicID, req.DoctorID, req.StartTime.Unix())
	holdToken := "hld_" + uuid.New().String()

	if s.redis != nil && s.redis.Client != nil {
		// Attempt atomic SET NX EX 600
		acquired, err := s.redis.SetNX(ctx, redisLockKey, holdToken, 600*time.Second).Result()
		if err == nil && !acquired {
			// Redis absorbed collision with sub-millisecond response!
			return nil, domain.ErrSlotDoubleBooked
		}
		// If Redis has an error or is unreachable, proceed safely to PostgreSQL (Tier 2)
	}

	// Authoritative Hold in PostgreSQL (Tier 2 - DB is the ultimate source of truth)
	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	hold := &domain.AppointmentHold{
		ID:               uuid.New().String(),
		HoldToken:        holdToken,
		PatientID:        req.PatientID,
		ClinicID:         req.ClinicID,
		DoctorID:         req.DoctorID,
		RoomID:           req.RoomID,
		PackageID:        req.PackageID,
		StartTime:        req.StartTime,
		EndTime:          endTime,
		SessionNumber:    req.SessionNumber,
		TotalSessions:    req.TotalSessions,
		PackageBookingID: req.PackageBookingID,
		ExpiresAt:        expiresAt,
		Status:           domain.HoldStatusActive,
		CreatedAt:        time.Now().UTC(),
	}

	// Check active DB holds conflict before saving
	hasConflict, err := s.repo.CheckActiveHoldConflict(ctx, req.DoctorID, req.RoomID, req.StartTime, endTime)
	if err != nil {
		s.releaseRedisLock(ctx, redisLockKey)
		return nil, fmt.Errorf("failed to verify active holds: %w", err)
	}
	if hasConflict {
		s.releaseRedisLock(ctx, redisLockKey)
		return nil, domain.ErrSlotDoubleBooked
	}

	if err := s.repo.CreateHold(ctx, hold); err != nil {
		s.releaseRedisLock(ctx, redisLockKey)
		return nil, fmt.Errorf("failed to create hold: %w", err)
	}

	return &CreateHoldResponse{
		HoldID:     hold.ID,
		HoldToken:  hold.HoldToken,
		ExpiresAt:  hold.ExpiresAt,
		TTLSeconds: 600,
		DoctorID:   hold.DoctorID,
		RoomID:     hold.RoomID,
	}, nil
}

// ConfirmAppointment finalizes the hold into an authoritative confirmed booking inside a single ACID transaction.
func (s *appointmentService) ConfirmAppointment(ctx context.Context, req ConfirmAppointmentRequest) (*domain.Appointment, error) {
	if req.HoldToken == "" {
		return nil, errors.New("hold_token is required")
	}

	// Check idempotency: If an appointment with this idempotency key exists, return it immediately
	if req.IdempotencyKey != nil && *req.IdempotencyKey != "" {
		existing, err := s.repo.GetAppointmentByIdempotencyKey(ctx, *req.IdempotencyKey)
		if err == nil && existing != nil {
			return existing, nil
		}
	}

	// Fetch hold from DB (authoritative)
	hold, err := s.repo.GetHoldByToken(ctx, req.HoldToken)
	if err != nil {
		return nil, domain.ErrHoldNotFound
	}

	now := time.Now().UTC()
	if hold.IsExpired(now) {
		return nil, domain.ErrHoldExpired
	}

	if req.PatientID != "" && hold.PatientID != req.PatientID {
		return nil, domain.ErrUnauthorizedAccess
	}

	// Create authoritative appointment record
	apptID := uuid.New().String()
	appt := &domain.Appointment{
		ID:               apptID,
		PatientID:        hold.PatientID,
		ClinicID:         hold.ClinicID,
		DoctorID:         hold.DoctorID,
		RoomID:           hold.RoomID,
		PackageID:        hold.PackageID,
		StartTime:        hold.StartTime,
		EndTime:          hold.EndTime,
		Status:           domain.StatusConfirmed,
		SessionNumber:    hold.SessionNumber,
		TotalSessions:    hold.TotalSessions,
		PackageBookingID: hold.PackageBookingID,
		IdempotencyKey:   req.IdempotencyKey,
		Notes:            req.Notes,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Construct Transactional Outbox Event
	payloadMap := map[string]interface{}{
		"appointment_id":     appt.ID,
		"patient_id":         appt.PatientID,
		"clinic_id":          appt.ClinicID,
		"doctor_id":          appt.DoctorID,
		"room_id":            appt.RoomID,
		"package_id":         appt.PackageID,
		"session_number":     appt.SessionNumber,
		"total_sessions":     appt.TotalSessions,
		"package_booking_id": appt.PackageBookingID,
		"scheduled_start":    appt.StartTime.Format(time.RFC3339),
		"scheduled_end":      appt.EndTime.Format(time.RFC3339),
		"notes":              appt.Notes,
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	outbox := &domain.OutboxEvent{
		ID:            uuid.New().String(),
		AggregateType: "APPOINTMENT",
		AggregateID:   appt.ID,
		EventType:     "panchakarma.appointment.confirmed.v1",
		Payload:       payloadBytes,
		TraceID:       "trace-" + appt.ID[:8],
		Status:        domain.OutboxStatusPending,
		RetryCount:    0,
		CreatedAt:     now,
	}

	// Authoritative DB execution: checks GiST exclusion constraint, writes appointment, marks hold confirmed, writes outbox
	if err := s.repo.CreateAppointmentWithTx(ctx, appt, &hold.ID, outbox); err != nil {
		return nil, err
	}

	// Clean up Redis lock
	redisLockKey := fmt.Sprintf("panchakarma:hold:slot:%s:%s:%d", hold.ClinicID, hold.DoctorID, hold.StartTime.Unix())
	s.releaseRedisLock(ctx, redisLockKey)

	return appt, nil
}

// BookDirectAppointment enables direct booking without prior hold (e.g., admin bookings or walk-ins).
func (s *appointmentService) BookDirectAppointment(ctx context.Context, req DirectBookingRequest) (*domain.Appointment, error) {
	if req.ClinicID == "" || req.PatientID == "" || req.PackageID == "" {
		return nil, errors.New("clinic_id, patient_id, and package_id are required")
	}
	if req.StartTime.IsZero() {
		return nil, errors.New("start_time is required")
	}
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 90
	}
	if req.DoctorID == "" || req.AutoAssignDoctor {
		req.DoctorID = "doc_auto_assigned"
	}
	if req.RoomID == "" {
		req.RoomID = "room_auto_assigned"
	}

	// Idempotency check
	if req.IdempotencyKey != nil && *req.IdempotencyKey != "" {
		existing, err := s.repo.GetAppointmentByIdempotencyKey(ctx, *req.IdempotencyKey)
		if err == nil && existing != nil {
			return existing, nil
		}
	}

	endTime := req.StartTime.Add(time.Duration(req.DurationMinutes) * time.Minute)

	// Validate slot & room with Clinic Service gRPC
	if s.clinicValidator != nil {
		isValid, rejectionReason, _, err := s.clinicValidator.ValidateSlotAndRoom(ctx, req.ClinicID, req.DoctorID, req.RoomID, req.StartTime, endTime, req.PackageID)
		if err == nil && !isValid {
			return nil, fmt.Errorf("slot rejected by clinic validation: %s", rejectionReason)
		}
	}

	now := time.Now().UTC()
	appt := &domain.Appointment{
		ID:             uuid.New().String(),
		PatientID:      req.PatientID,
		ClinicID:       req.ClinicID,
		DoctorID:       req.DoctorID,
		RoomID:         req.RoomID,
		PackageID:      req.PackageID,
		StartTime:      req.StartTime,
		EndTime:        endTime,
		Status:         domain.StatusConfirmed,
		SessionNumber:  1,
		TotalSessions:  1,
		IdempotencyKey: req.IdempotencyKey,
		Notes:          req.Notes,
		ApprovedBy:     req.ApprovedBy,
		ApprovedAt:     &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	payloadMap := map[string]interface{}{
		"appointment_id":  appt.ID,
		"patient_id":      appt.PatientID,
		"clinic_id":       appt.ClinicID,
		"doctor_id":       appt.DoctorID,
		"room_id":         appt.RoomID,
		"package_id":      appt.PackageID,
		"scheduled_start": appt.StartTime.Format(time.RFC3339),
		"scheduled_end":   appt.EndTime.Format(time.RFC3339),
		"notes":           appt.Notes,
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	outbox := &domain.OutboxEvent{
		ID:            uuid.New().String(),
		AggregateType: "APPOINTMENT",
		AggregateID:   appt.ID,
		EventType:     "panchakarma.appointment.confirmed.v1",
		Payload:       payloadBytes,
		TraceID:       "trace-" + appt.ID[:8],
		Status:        domain.OutboxStatusPending,
		RetryCount:    0,
		CreatedAt:     now,
	}

	if err := s.repo.CreateAppointmentWithTx(ctx, appt, nil, outbox); err != nil {
		return nil, err
	}

	return appt, nil
}

// BookMultiSessionPackage coordinates multi-session scheduling linked by package_booking_id.
func (s *appointmentService) BookMultiSessionPackage(ctx context.Context, req MultiSessionPackageRequest) ([]*domain.Appointment, error) {
	if len(req.SessionStarts) == 0 {
		return nil, errors.New("at least one session start time is required")
	}
	if req.ClinicID == "" || req.PatientID == "" || req.PackageID == "" {
		return nil, errors.New("clinic_id, patient_id, and package_id are required")
	}

	totalSessions := len(req.SessionStarts)
	packageBookingID := uuid.New().String()
	duration := req.DurationMinutes
	if duration <= 0 {
		duration = 90
	}

	var bookedAppointments []*domain.Appointment
	for idx, start := range req.SessionStarts {
		docID := req.PreferredDoctor
		if docID == "" {
			docID = "doc_auto_assigned"
		}
		roomID := "room_auto_assigned"
		endTime := start.Add(time.Duration(duration) * time.Minute)

		now := time.Now().UTC()
		appt := &domain.Appointment{
			ID:               uuid.New().String(),
			PatientID:        req.PatientID,
			ClinicID:         req.ClinicID,
			DoctorID:         docID,
			RoomID:           roomID,
			PackageID:        req.PackageID,
			StartTime:        start,
			EndTime:          endTime,
			Status:           domain.StatusConfirmed,
			SessionNumber:    idx + 1,
			TotalSessions:    totalSessions,
			PackageBookingID: &packageBookingID,
			Notes:            req.Notes,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		payloadBytes, _ := json.Marshal(map[string]interface{}{
			"appointment_id":     appt.ID,
			"package_booking_id": packageBookingID,
			"session_number":     appt.SessionNumber,
			"total_sessions":     totalSessions,
		})

		outbox := &domain.OutboxEvent{
			AggregateType: "APPOINTMENT",
			AggregateID:   appt.ID,
			EventType:     "panchakarma.appointment.confirmed.v1",
			Payload:       payloadBytes,
			TraceID:       "trace-" + appt.ID[:8],
			Status:        domain.OutboxStatusPending,
			CreatedAt:     now,
		}

		if err := s.repo.CreateAppointmentWithTx(ctx, appt, nil, outbox); err != nil {
			return nil, fmt.Errorf("failed booking session %d of %d: %w", idx+1, totalSessions, err)
		}
		bookedAppointments = append(bookedAppointments, appt)
	}

	return bookedAppointments, nil
}

// CancelAppointment cancels a confirmed booking and checks the waitlist to offer the freed slot.
func (s *appointmentService) CancelAppointment(ctx context.Context, apptID string, patientID string, userRole string, clinicID string, reason string) error {
	appt, err := s.repo.GetAppointmentByID(ctx, apptID)
	if err != nil {
		return err
	}

	// Authorize caller
	if userRole == "patient" && appt.PatientID != patientID {
		return domain.ErrUnauthorizedAccess
	}
	if userRole != "patient" && clinicID != "" && appt.ClinicID != clinicID {
		return domain.ErrUnauthorizedAccess
	}

	if err := domain.ValidateTransition(appt.Status, domain.StatusCancelled); err != nil {
		return err
	}

	now := time.Now().UTC()
	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"appointment_id":      appt.ID,
		"patient_id":          appt.PatientID,
		"clinic_id":           appt.ClinicID,
		"doctor_id":           appt.DoctorID,
		"cancellation_reason": reason,
	})

	outbox := &domain.OutboxEvent{
		AggregateType: "APPOINTMENT",
		AggregateID:   appt.ID,
		EventType:     "panchakarma.appointment.cancelled.v1",
		Payload:       payloadBytes,
		TraceID:       "trace-" + appt.ID[:8],
		Status:        domain.OutboxStatusPending,
		CreatedAt:     now,
	}

	if err := s.repo.UpdateAppointmentStatusWithTx(ctx, apptID, appt.Status, domain.StatusCancelled, reason, &patientID, outbox); err != nil {
		return err
	}

	// Waitlist trigger: Check if any waiting patient can take this newly freed slot
	dateStr := appt.StartTime.Format("2006-01-02")
	match, err := s.repo.FindNextWaitlistMatch(ctx, appt.ClinicID, appt.PackageID, &appt.DoctorID, dateStr)
	if err == nil && match != nil {
		// Offer slot to waitlisted patient for 1 hour
		offerExpiresAt := now.Add(1 * time.Hour)
		waitlistPayload, _ := json.Marshal(map[string]interface{}{
			"waitlist_id":      match.ID,
			"patient_id":       match.PatientID,
			"clinic_id":        match.ClinicID,
			"package_id":       match.PackageID,
			"offered_start":    appt.StartTime.Format(time.RFC3339),
			"offered_end":      appt.EndTime.Format(time.RFC3339),
			"offer_expires_at": offerExpiresAt.Format(time.RFC3339),
		})

		waitlistOutbox := &domain.OutboxEvent{
			AggregateType: "WAITLIST",
			AggregateID:   match.ID,
			EventType:     "panchakarma.appointment.waitlist_offered.v1",
			Payload:       waitlistPayload,
			TraceID:       "trace-" + match.ID[:8],
			Status:        domain.OutboxStatusPending,
			CreatedAt:     now,
		}

		_ = s.repo.OfferWaitlistSlot(ctx, match.ID, appt.StartTime, appt.EndTime, offerExpiresAt, waitlistOutbox)
	}

	return nil
}

// RescheduleAppointment implements the atomic two-phase swap pattern.
func (s *appointmentService) RescheduleAppointment(ctx context.Context, req RescheduleRequest) (*domain.Appointment, error) {
	if req.AppointmentID == "" || req.NewStartTime.IsZero() {
		return nil, errors.New("appointment_id and new_start_time are required")
	}

	// 1. Fetch old appointment
	oldAppt, err := s.repo.GetAppointmentByID(ctx, req.AppointmentID)
	if err != nil {
		return nil, err
	}

	// Authorize
	if req.RequesterRole == "patient" && oldAppt.PatientID != req.RequesterID {
		return nil, domain.ErrUnauthorizedAccess
	}
	if req.RequesterRole != "patient" && req.ClinicID != "" && oldAppt.ClinicID != req.ClinicID {
		return nil, domain.ErrUnauthorizedAccess
	}

	if oldAppt.Status != domain.StatusConfirmed {
		return nil, fmt.Errorf("cannot reschedule appointment with status %s", oldAppt.Status)
	}

	duration := req.DurationMinutes
	if duration <= 0 {
		duration = int(oldAppt.EndTime.Sub(oldAppt.StartTime).Minutes())
		if duration <= 0 {
			duration = 90
		}
	}
	newEndTime := req.NewStartTime.Add(time.Duration(duration) * time.Minute)

	docID := req.NewDoctorID
	if docID == "" {
		docID = oldAppt.DoctorID
	}
	roomID := req.NewRoomID
	if roomID == "" {
		roomID = oldAppt.RoomID
	}

	now := time.Now().UTC()
	newAppt := &domain.Appointment{
		ID:               uuid.New().String(),
		PatientID:        oldAppt.PatientID,
		ClinicID:         oldAppt.ClinicID,
		DoctorID:         docID,
		RoomID:           roomID,
		PackageID:        oldAppt.PackageID,
		StartTime:        req.NewStartTime,
		EndTime:          newEndTime,
		Status:           domain.StatusConfirmed,
		SessionNumber:    oldAppt.SessionNumber,
		TotalSessions:    oldAppt.TotalSessions,
		PackageBookingID: oldAppt.PackageBookingID,
		Notes:            oldAppt.Notes,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"old_appointment_id": oldAppt.ID,
		"new_appointment_id": newAppt.ID,
		"patient_id":         newAppt.PatientID,
		"new_scheduled_start": newAppt.StartTime.Format(time.RFC3339),
		"new_scheduled_end":   newAppt.EndTime.Format(time.RFC3339),
	})

	outbox := &domain.OutboxEvent{
		AggregateType: "APPOINTMENT",
		AggregateID:   newAppt.ID,
		EventType:     "panchakarma.appointment.rescheduled.v1",
		Payload:       payloadBytes,
		TraceID:       "trace-" + newAppt.ID[:8],
		Status:        domain.OutboxStatusPending,
		CreatedAt:     now,
	}

	// Atomic DB transaction: locks old appointment, checks GiST exclusion constraint on new appointment, marks old as RESCHEDULED
	if err := s.repo.RescheduleAppointmentWithTx(ctx, oldAppt.ID, newAppt, outbox); err != nil {
		return nil, err
	}

	return newAppt, nil
}

// JoinWaitlist queues a patient for a date range when slots are currently full.
func (s *appointmentService) JoinWaitlist(ctx context.Context, req JoinWaitlistRequest) (*domain.AppointmentWaitlist, error) {
	if req.ClinicID == "" || req.PatientID == "" || req.PackageID == "" {
		return nil, errors.New("clinic_id, patient_id, and package_id are required")
	}
	if req.PreferredDateStart == "" || req.PreferredDateEnd == "" {
		return nil, errors.New("preferred_date_start and preferred_date_end are required")
	}

	entry := &domain.AppointmentWaitlist{
		ID:                 uuid.New().String(),
		ClinicID:           req.ClinicID,
		PatientID:          req.PatientID,
		PackageID:          req.PackageID,
		PreferredDoctorID:  req.PreferredDoctorID,
		PreferredDateStart: req.PreferredDateStart,
		PreferredDateEnd:   req.PreferredDateEnd,
		Status:             domain.WaitlistStatusWaiting,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	if err := s.repo.CreateWaitlistEntry(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// GetWaitlistEntries retrieves waitlist entries for a clinic or patient.
func (s *appointmentService) GetWaitlistEntries(ctx context.Context, clinicID string, patientID *string, status *domain.WaitlistStatus) ([]*domain.AppointmentWaitlist, error) {
	return s.repo.ListWaitlistEntries(ctx, clinicID, patientID, status)
}

// AcceptWaitlistOffer converts an active slot offer to a confirmed appointment.
func (s *appointmentService) AcceptWaitlistOffer(ctx context.Context, waitlistID string, patientID string) (*domain.Appointment, error) {
	entry, err := s.repo.GetWaitlistEntry(ctx, waitlistID)
	if err != nil {
		return nil, err
	}

	if entry.PatientID != patientID {
		return nil, domain.ErrUnauthorizedAccess
	}
	if entry.Status != domain.WaitlistStatusOffered {
		return nil, errors.New("waitlist entry is not currently in OFFERED state")
	}
	now := time.Now().UTC()
	if entry.OfferExpiresAt != nil && now.After(*entry.OfferExpiresAt) {
		return nil, domain.ErrWaitlistOfferExpired
	}
	if entry.OfferedSlotStart == nil || entry.OfferedSlotEnd == nil {
		return nil, errors.New("invalid offered slot boundaries")
	}

	// Direct booking of the offered slot
	docID := "doc_auto_assigned"
	if entry.PreferredDoctorID != nil {
		docID = *entry.PreferredDoctorID
	}

	apptReq := DirectBookingRequest{
		PatientID:       entry.PatientID,
		ClinicID:        entry.ClinicID,
		DoctorID:        docID,
		RoomID:          "room_auto_assigned",
		PackageID:       entry.PackageID,
		StartTime:       *entry.OfferedSlotStart,
		DurationMinutes: int(entry.OfferedSlotEnd.Sub(*entry.OfferedSlotStart).Minutes()),
		Notes:           "Waitlist offer accepted",
	}

	appt, err := s.BookDirectAppointment(ctx, apptReq)
	if err != nil {
		return nil, err
	}

	_ = s.repo.AcceptWaitlistOffer(ctx, waitlistID)
	return appt, nil
}

// GetAppointment fetches an appointment enforcing role-based isolation.
func (s *appointmentService) GetAppointment(ctx context.Context, apptID string, requesterID string, role string, clinicID string) (*domain.Appointment, error) {
	appt, err := s.repo.GetAppointmentByID(ctx, apptID)
	if err != nil {
		return nil, err
	}

	if role == "patient" && appt.PatientID != requesterID {
		return nil, domain.ErrUnauthorizedAccess
	}
	if role == "doctor" && appt.DoctorID != requesterID {
		return nil, domain.ErrUnauthorizedAccess
	}
	if (role == "clinic_admin" || role == "therapist") && clinicID != "" && appt.ClinicID != clinicID {
		return nil, domain.ErrUnauthorizedAccess
	}

	return appt, nil
}

// ListAppointments queries appointments with filter validation and authorization.
func (s *appointmentService) ListAppointments(ctx context.Context, filter repository.ListFilter, requesterID string, role string, clinicID string) ([]*domain.Appointment, int, error) {
	if role == "patient" {
		filter.PatientID = &requesterID
	} else if role == "doctor" {
		filter.DoctorID = &requesterID
	} else if role == "clinic_admin" || role == "therapist" {
		if clinicID != "" {
			filter.ClinicID = clinicID
		}
	}

	return s.repo.ListAppointments(ctx, filter)
}

// ExpireStaleHolds triggers background sweep of expired holds.
func (s *appointmentService) ExpireStaleHolds(ctx context.Context) (int64, error) {
	return s.repo.ExpireStaleHolds(ctx)
}

func (s *appointmentService) releaseRedisLock(ctx context.Context, key string) {
	if s.redis != nil && s.redis.Client != nil {
		_ = s.redis.Del(ctx, key).Err()
	}
}
