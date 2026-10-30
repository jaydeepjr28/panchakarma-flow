package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/repository"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/kafka"
)

// MockAppointmentRepository implements repository.AppointmentRepository for unit tests.
type MockAppointmentRepository struct {
	holds        map[string]*domain.AppointmentHold
	appointments map[string]*domain.Appointment
	idempotent   map[string]*domain.Appointment
	waitlists    map[string]*domain.AppointmentWaitlist
	outbox       []*domain.OutboxEvent
	bookedRanges []repository.TimeRange

	SimulateExclusionError bool
}

func NewMockAppointmentRepository() *MockAppointmentRepository {
	return &MockAppointmentRepository{
		holds:        make(map[string]*domain.AppointmentHold),
		appointments: make(map[string]*domain.Appointment),
		idempotent:   make(map[string]*domain.Appointment),
		waitlists:    make(map[string]*domain.AppointmentWaitlist),
	}
}

func (m *MockAppointmentRepository) CreateHold(ctx context.Context, hold *domain.AppointmentHold) error {
	m.holds[hold.HoldToken] = hold
	m.holds[hold.ID] = hold
	return nil
}

func (m *MockAppointmentRepository) GetHoldByToken(ctx context.Context, token string) (*domain.AppointmentHold, error) {
	h, ok := m.holds[token]
	if !ok {
		return nil, domain.ErrHoldNotFound
	}
	return h, nil
}

func (m *MockAppointmentRepository) GetHoldByID(ctx context.Context, holdID string) (*domain.AppointmentHold, error) {
	h, ok := m.holds[holdID]
	if !ok {
		return nil, domain.ErrHoldNotFound
	}
	return h, nil
}

func (m *MockAppointmentRepository) UpdateHoldStatus(ctx context.Context, holdID string, status domain.HoldStatus) error {
	h, ok := m.holds[holdID]
	if !ok {
		return domain.ErrHoldNotFound
	}
	h.Status = status
	return nil
}

func (m *MockAppointmentRepository) ExpireStaleHolds(ctx context.Context) (int64, error) {
	var count int64
	now := time.Now().UTC()
	for _, h := range m.holds {
		if h.Status == domain.HoldStatusActive && h.ExpiresAt.Before(now) {
			h.Status = domain.HoldStatusExpired
			count++
		}
	}
	return count, nil
}

func (m *MockAppointmentRepository) CheckActiveHoldConflict(ctx context.Context, doctorID, roomID string, start, end time.Time) (bool, error) {
	now := time.Now().UTC()
	for _, h := range m.holds {
		if h.Status == domain.HoldStatusActive && h.ExpiresAt.After(now) {
			if (h.DoctorID == doctorID || h.RoomID == roomID) && !(end.Before(h.StartTime) || start.After(h.EndTime)) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *MockAppointmentRepository) CreateAppointmentWithTx(ctx context.Context, appt *domain.Appointment, holdID *string, outbox *domain.OutboxEvent) error {
	if m.SimulateExclusionError {
		return domain.ErrSlotDoubleBooked
	}

	// Check if any existing appointment overlaps doctor or room
	for _, existing := range m.appointments {
		if existing.Status == domain.StatusConfirmed || existing.Status == domain.StatusCheckedIn || existing.Status == domain.StatusInProgress {
			if (existing.DoctorID == appt.DoctorID || existing.RoomID == appt.RoomID) &&
				!(appt.EndTime.Before(existing.StartTime) || appt.StartTime.After(existing.EndTime)) {
				return domain.ErrSlotDoubleBooked
			}
		}
	}

	if holdID != nil && *holdID != "" {
		if h, ok := m.holds[*holdID]; ok {
			h.Status = domain.HoldStatusConfirmed
		}
	}

	m.appointments[appt.ID] = appt
	if appt.IdempotencyKey != nil && *appt.IdempotencyKey != "" {
		m.idempotent[*appt.IdempotencyKey] = appt
	}
	if outbox != nil {
		m.outbox = append(m.outbox, outbox)
	}
	return nil
}

func (m *MockAppointmentRepository) GetAppointmentByID(ctx context.Context, id string) (*domain.Appointment, error) {
	a, ok := m.appointments[id]
	if !ok {
		return nil, domain.ErrAppointmentNotFound
	}
	return a, nil
}

func (m *MockAppointmentRepository) GetAppointmentByIdempotencyKey(ctx context.Context, key string) (*domain.Appointment, error) {
	a, ok := m.idempotent[key]
	if !ok {
		return nil, domain.ErrAppointmentNotFound
	}
	return a, nil
}

func (m *MockAppointmentRepository) ListAppointments(ctx context.Context, filter repository.ListFilter) ([]*domain.Appointment, int, error) {
	var res []*domain.Appointment
	for _, a := range m.appointments {
		if filter.ClinicID != "" && a.ClinicID != filter.ClinicID {
			continue
		}
		if filter.PatientID != nil && a.PatientID != *filter.PatientID {
			continue
		}
		if filter.DoctorID != nil && a.DoctorID != *filter.DoctorID {
			continue
		}
		res = append(res, a)
	}
	return res, len(res), nil
}

func (m *MockAppointmentRepository) UpdateAppointmentStatusWithTx(ctx context.Context, apptID string, fromStatus, toStatus domain.AppointmentStatus, reason string, changedBy *string, outbox *domain.OutboxEvent) error {
	a, ok := m.appointments[apptID]
	if !ok {
		return domain.ErrAppointmentNotFound
	}
	if fromStatus != "" && a.Status != fromStatus {
		return domain.ErrInvalidStateTransition
	}
	if err := domain.ValidateTransition(a.Status, toStatus); err != nil {
		return err
	}
	a.Status = toStatus
	a.CancellationReason = reason
	if outbox != nil {
		m.outbox = append(m.outbox, outbox)
	}
	return nil
}

func (m *MockAppointmentRepository) RescheduleAppointmentWithTx(ctx context.Context, oldApptID string, newAppt *domain.Appointment, outbox *domain.OutboxEvent) error {
	old, ok := m.appointments[oldApptID]
	if !ok {
		return domain.ErrAppointmentNotFound
	}
	if m.SimulateExclusionError {
		return domain.ErrSlotDoubleBooked
	}

	// Exclusion check on new slot
	for id, existing := range m.appointments {
		if id == oldApptID {
			continue
		}
		if existing.Status == domain.StatusConfirmed {
			if (existing.DoctorID == newAppt.DoctorID || existing.RoomID == newAppt.RoomID) &&
				!(newAppt.EndTime.Before(existing.StartTime) || newAppt.StartTime.After(existing.EndTime)) {
				return domain.ErrSlotDoubleBooked
			}
		}
	}

	old.Status = domain.StatusRescheduled
	old.RescheduledToID = &newAppt.ID
	m.appointments[newAppt.ID] = newAppt
	if outbox != nil {
		m.outbox = append(m.outbox, outbox)
	}
	return nil
}

func (m *MockAppointmentRepository) CreateWaitlistEntry(ctx context.Context, entry *domain.AppointmentWaitlist) error {
	m.waitlists[entry.ID] = entry
	return nil
}

func (m *MockAppointmentRepository) GetWaitlistEntry(ctx context.Context, id string) (*domain.AppointmentWaitlist, error) {
	w, ok := m.waitlists[id]
	if !ok {
		return nil, domain.ErrWaitlistNotFound
	}
	return w, nil
}

func (m *MockAppointmentRepository) ListWaitlistEntries(ctx context.Context, clinicID string, patientID *string, status *domain.WaitlistStatus) ([]*domain.AppointmentWaitlist, error) {
	var res []*domain.AppointmentWaitlist
	for _, w := range m.waitlists {
		if w.ClinicID == clinicID {
			if patientID != nil && w.PatientID != *patientID {
				continue
			}
			if status != nil && w.Status != *status {
				continue
			}
			res = append(res, w)
		}
	}
	return res, nil
}

func (m *MockAppointmentRepository) FindNextWaitlistMatch(ctx context.Context, clinicID, packageID string, doctorID *string, dateStr string) (*domain.AppointmentWaitlist, error) {
	for _, w := range m.waitlists {
		if w.ClinicID == clinicID && w.PackageID == packageID && w.Status == domain.WaitlistStatusWaiting {
			return w, nil
		}
	}
	return nil, domain.ErrWaitlistNotFound
}

func (m *MockAppointmentRepository) OfferWaitlistSlot(ctx context.Context, waitlistID string, start, end, expiresAt time.Time, outbox *domain.OutboxEvent) error {
	w, ok := m.waitlists[waitlistID]
	if !ok {
		return domain.ErrWaitlistNotFound
	}
	w.Status = domain.WaitlistStatusOffered
	w.OfferedSlotStart = &start
	w.OfferedSlotEnd = &end
	w.OfferExpiresAt = &expiresAt
	if outbox != nil {
		m.outbox = append(m.outbox, outbox)
	}
	return nil
}

func (m *MockAppointmentRepository) AcceptWaitlistOffer(ctx context.Context, waitlistID string) error {
	w, ok := m.waitlists[waitlistID]
	if !ok {
		return domain.ErrWaitlistNotFound
	}
	w.Status = domain.WaitlistStatusFulfilled
	return nil
}

func (m *MockAppointmentRepository) CreateOutboxEvent(ctx context.Context, event *domain.OutboxEvent) error {
	m.outbox = append(m.outbox, event)
	return nil
}

func (m *MockAppointmentRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	var pending []*domain.OutboxEvent
	for _, e := range m.outbox {
		if e.Status == domain.OutboxStatusPending {
			pending = append(pending, e)
			if len(pending) >= limit {
				break
			}
		}
	}
	return pending, nil
}

func (m *MockAppointmentRepository) MarkOutboxEventProcessed(ctx context.Context, id string) error {
	for _, e := range m.outbox {
		if e.ID == id {
			e.Status = domain.OutboxStatusProcessed
			return nil
		}
	}
	return nil
}

func (m *MockAppointmentRepository) MarkOutboxEventFailed(ctx context.Context, id string, reason string) error {
	for _, e := range m.outbox {
		if e.ID == id {
			e.RetryCount++
			return nil
		}
	}
	return nil
}

func (m *MockAppointmentRepository) GetAppointmentHistory(ctx context.Context, appointmentID string) ([]*domain.AppointmentHistory, error) {
	return nil, nil
}

func (m *MockAppointmentRepository) GetBookedRanges(ctx context.Context, clinicID string, doctorID, roomID *string, start, end time.Time) ([]repository.TimeRange, error) {
	return m.bookedRanges, nil
}

// MockClinicValidator implements service.ClinicValidator for tests.
type MockClinicValidator struct {
	IsValid   bool
	Rejection string
}

func (v *MockClinicValidator) ValidateSlotAndRoom(ctx context.Context, clinicID, doctorID, roomID string, start, end time.Time, packageID string) (bool, string, int, error) {
	return v.IsValid, v.Rejection, 15, nil
}

func (v *MockClinicValidator) GetPackageSnapshot(ctx context.Context, packageID string) (int, int, error) {
	return 90, 1, nil
}

func TestAppointmentService_HoldAndConfirmFlow(t *testing.T) {
	repo := NewMockAppointmentRepository()
	validator := &MockClinicValidator{IsValid: true}
	// Note: redis is passed as nil to demonstrate that zero double-booking works without Redis!
	svc := service.NewAppointmentService(repo, nil, validator)
	ctx := context.Background()

	clinicID := uuid.New().String()
	patientID := uuid.New().String()
	packageID := uuid.New().String()
	doctorID := uuid.New().String()
	roomID := uuid.New().String()
	slotStart := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Hour)

	// 1. Create Hold
	holdResp, err := svc.CreateHold(ctx, service.CreateHoldRequest{
		PatientID:       patientID,
		ClinicID:        clinicID,
		DoctorID:        doctorID,
		RoomID:          roomID,
		PackageID:       packageID,
		StartTime:       slotStart,
		DurationMinutes: 90,
	})
	if err != nil {
		t.Fatalf("unexpected error creating hold: %v", err)
	}

	if holdResp.HoldToken == "" {
		t.Fatalf("expected non-empty hold token")
	}

	// 2. Confirm Appointment
	idempKey := "idemp-booking-001"
	appt, err := svc.ConfirmAppointment(ctx, service.ConfirmAppointmentRequest{
		HoldToken:       holdResp.HoldToken,
		PatientID:       patientID,
		ConsentRecordID: "consent-123",
		Notes:           "Patient requested warm sesame oil",
		IdempotencyKey:  &idempKey,
	})
	if err != nil {
		t.Fatalf("unexpected error confirming appointment: %v", err)
	}

	if appt.Status != domain.StatusConfirmed {
		t.Fatalf("expected appointment status CONFIRMED, got %s", appt.Status)
	}

	// Check outbox event was generated
	if len(repo.outbox) == 0 {
		t.Fatalf("expected transactional outbox event to be created")
	}
	if repo.outbox[0].EventType != "panchakarma.appointment.confirmed.v1" {
		t.Fatalf("expected event type panchakarma.appointment.confirmed.v1, got %s", repo.outbox[0].EventType)
	}

	// 3. Test Idempotency: Retrying with identical key returns cached appointment without recreation
	retryAppt, err := svc.ConfirmAppointment(ctx, service.ConfirmAppointmentRequest{
		HoldToken:      holdResp.HoldToken,
		PatientID:      patientID,
		IdempotencyKey: &idempKey,
	})
	if err != nil {
		t.Fatalf("unexpected error on idempotent retry: %v", err)
	}
	if retryAppt.ID != appt.ID {
		t.Fatalf("idempotent call should return identical appointment ID %s, got %s", appt.ID, retryAppt.ID)
	}
}

func TestAppointmentService_DoubleBookingPreventionWithoutRedis(t *testing.T) {
	repo := NewMockAppointmentRepository()
	validator := &MockClinicValidator{IsValid: true}
	// Redis is nil! All locking is verified authoritatively by PostgreSQL invariants
	svc := service.NewAppointmentService(repo, nil, validator)
	ctx := context.Background()

	clinicID := uuid.New().String()
	patientA := uuid.New().String()
	patientB := uuid.New().String()
	packageID := uuid.New().String()
	doctorID := uuid.New().String()
	roomID := uuid.New().String()
	slotStart := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Hour)

	// Patient A books direct
	_, err := svc.BookDirectAppointment(ctx, service.DirectBookingRequest{
		PatientID:       patientA,
		ClinicID:        clinicID,
		DoctorID:        doctorID,
		RoomID:          roomID,
		PackageID:       packageID,
		StartTime:       slotStart,
		DurationMinutes: 90,
	})
	if err != nil {
		t.Fatalf("patient A direct booking failed: %v", err)
	}

	// Patient B attempts to book the same doctor and slot simultaneously
	_, err = svc.BookDirectAppointment(ctx, service.DirectBookingRequest{
		PatientID:       patientB,
		ClinicID:        clinicID,
		DoctorID:        doctorID,
		RoomID:          roomID,
		PackageID:       packageID,
		StartTime:       slotStart.Add(30 * time.Minute), // Overlapping slot!
		DurationMinutes: 90,
	})
	if err == nil {
		t.Fatalf("expected double booking to fail, but it succeeded!")
	}
	if !errors.Is(err, domain.ErrSlotDoubleBooked) {
		t.Fatalf("expected ErrSlotDoubleBooked, got %v", err)
	}
}

func TestAppointmentService_CancellationAndWaitlistOffer(t *testing.T) {
	repo := NewMockAppointmentRepository()
	validator := &MockClinicValidator{IsValid: true}
	svc := service.NewAppointmentService(repo, nil, validator)
	ctx := context.Background()

	clinicID := uuid.New().String()
	patientA := uuid.New().String()
	patientB := uuid.New().String()
	packageID := uuid.New().String()
	doctorID := uuid.New().String()
	roomID := uuid.New().String()
	slotStart := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Hour)

	// Patient A books appointment
	appt, err := svc.BookDirectAppointment(ctx, service.DirectBookingRequest{
		PatientID:       patientA,
		ClinicID:        clinicID,
		DoctorID:        doctorID,
		RoomID:          roomID,
		PackageID:       packageID,
		StartTime:       slotStart,
		DurationMinutes: 90,
	})
	if err != nil {
		t.Fatalf("booking failed: %v", err)
	}

	// Patient B joins waitlist for that day
	dateStr := slotStart.Format("2006-01-02")
	waitlistEntry, err := svc.JoinWaitlist(ctx, service.JoinWaitlistRequest{
		ClinicID:           clinicID,
		PatientID:          patientB,
		PackageID:          packageID,
		PreferredDoctorID:  &doctorID,
		PreferredDateStart: dateStr,
		PreferredDateEnd:   dateStr,
	})
	if err != nil {
		t.Fatalf("waitlist join failed: %v", err)
	}

	// Patient A cancels appointment
	err = svc.CancelAppointment(ctx, appt.ID, patientA, "patient", clinicID, "Family emergency")
	if err != nil {
		t.Fatalf("cancel failed: %v", err)
	}

	// Verify Patient B received an OFFERED slot
	updatedWaitlist, err := repo.GetWaitlistEntry(ctx, waitlistEntry.ID)
	if err != nil {
		t.Fatalf("failed to fetch waitlist entry: %v", err)
	}
	if updatedWaitlist.Status != domain.WaitlistStatusOffered {
		t.Fatalf("expected waitlist status OFFERED, got %s", updatedWaitlist.Status)
	}
	if updatedWaitlist.OfferedSlotStart == nil || !updatedWaitlist.OfferedSlotStart.Equal(slotStart) {
		t.Fatalf("expected offered start to match cancelled appointment slot")
	}

	// Patient B accepts the waitlist offer!
	acceptedAppt, err := svc.AcceptWaitlistOffer(ctx, waitlistEntry.ID, patientB)
	if err != nil {
		t.Fatalf("accept waitlist offer failed: %v", err)
	}
	if acceptedAppt.PatientID != patientB {
		t.Fatalf("expected appointment patient to be Patient B, got %s", acceptedAppt.PatientID)
	}
	if acceptedAppt.Status != domain.StatusConfirmed {
		t.Fatalf("expected confirmed appointment, got %s", acceptedAppt.Status)
	}
}

func TestAppointmentService_RescheduleAtomicSwap(t *testing.T) {
	repo := NewMockAppointmentRepository()
	validator := &MockClinicValidator{IsValid: true}
	svc := service.NewAppointmentService(repo, nil, validator)
	ctx := context.Background()

	clinicID := uuid.New().String()
	patientID := uuid.New().String()
	packageID := uuid.New().String()
	doctorID := uuid.New().String()
	roomID := uuid.New().String()
	oldStart := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Hour)
	newStart := oldStart.Add(24 * time.Hour)

	// Initial confirmed appointment
	oldAppt, err := svc.BookDirectAppointment(ctx, service.DirectBookingRequest{
		PatientID:       patientID,
		ClinicID:        clinicID,
		DoctorID:        doctorID,
		RoomID:          roomID,
		PackageID:       packageID,
		StartTime:       oldStart,
		DurationMinutes: 90,
	})
	if err != nil {
		t.Fatalf("booking failed: %v", err)
	}

	// Reschedule to new slot
	newAppt, err := svc.RescheduleAppointment(ctx, service.RescheduleRequest{
		AppointmentID: oldAppt.ID,
		RequesterID:   patientID,
		RequesterRole: "patient",
		ClinicID:      clinicID,
		NewStartTime:  newStart,
	})
	if err != nil {
		t.Fatalf("reschedule failed: %v", err)
	}

	if newAppt.Status != domain.StatusConfirmed {
		t.Fatalf("expected new appointment status CONFIRMED, got %s", newAppt.Status)
	}

	// Verify old appointment was marked RESCHEDULED and links to new appointment
	oldReloaded, _ := repo.GetAppointmentByID(ctx, oldAppt.ID)
	if oldReloaded.Status != domain.StatusRescheduled {
		t.Fatalf("expected old appointment status RESCHEDULED, got %s", oldReloaded.Status)
	}
	if oldReloaded.RescheduledToID == nil || *oldReloaded.RescheduledToID != newAppt.ID {
		t.Fatalf("expected rescheduled_to_id to point to new appointment ID")
	}
}

// MockKafkaPublisher simulates Kafka publishing for the outbox worker.
type MockKafkaPublisher struct {
	published []*kafka.EventEnvelope
}

func (m *MockKafkaPublisher) PublishEvent(ctx context.Context, topic, partitionKey string, event *kafka.EventEnvelope) error {
	m.published = append(m.published, event)
	return nil
}

func TestOutboxPublisher_BatchProcessing(t *testing.T) {
	repo := NewMockAppointmentRepository()
	kafkaMock := &MockKafkaPublisher{}

	// Add 2 pending events
	repo.outbox = append(repo.outbox,
		&domain.OutboxEvent{
			ID:            uuid.New().String(),
			AggregateType: "APPOINTMENT",
			AggregateID:   uuid.New().String(),
			EventType:     "panchakarma.appointment.confirmed.v1",
			Payload:       []byte(`{"test":1}`),
			Status:        domain.OutboxStatusPending,
			CreatedAt:     time.Now().UTC(),
		},
		&domain.OutboxEvent{
			ID:            uuid.New().String(),
			AggregateType: "APPOINTMENT",
			AggregateID:   uuid.New().String(),
			EventType:     "panchakarma.appointment.cancelled.v1",
			Payload:       []byte(`{"test":2}`),
			Status:        domain.OutboxStatusPending,
			CreatedAt:     time.Now().UTC(),
		},
	)

	publisher := service.NewOutboxPublisher(repo, kafkaMock, 1*time.Second, 10)
	ctx := context.Background()

	processed, err := publisher.ProcessBatch(ctx)
	if err != nil {
		t.Fatalf("process batch failed: %v", err)
	}
	if processed != 2 {
		t.Fatalf("expected 2 processed events, got %d", processed)
	}
	if len(kafkaMock.published) != 2 {
		t.Fatalf("expected 2 published Kafka envelopes, got %d", len(kafkaMock.published))
	}
	if repo.outbox[0].Status != domain.OutboxStatusProcessed {
		t.Fatalf("expected event 0 to be marked PROCESSED")
	}
}
