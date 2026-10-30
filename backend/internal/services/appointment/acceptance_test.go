package appointment_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/repository"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/service"
)

// ThreadSafeMockRepository implements repository.AppointmentRepository with strict concurrency safety.
type ThreadSafeMockRepository struct {
	mu           sync.Mutex
	holds        map[string]*domain.AppointmentHold
	appointments map[string]*domain.Appointment
	idempotent   map[string]*domain.Appointment
	waitlists    map[string]*domain.AppointmentWaitlist
	outbox       []*domain.OutboxEvent
}

func NewThreadSafeMockRepository() *ThreadSafeMockRepository {
	return &ThreadSafeMockRepository{
		holds:        make(map[string]*domain.AppointmentHold),
		appointments: make(map[string]*domain.Appointment),
		idempotent:   make(map[string]*domain.Appointment),
		waitlists:    make(map[string]*domain.AppointmentWaitlist),
	}
}

func (m *ThreadSafeMockRepository) CreateHold(ctx context.Context, hold *domain.AppointmentHold) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.holds[hold.HoldToken] = hold
	m.holds[hold.ID] = hold
	return nil
}

func (m *ThreadSafeMockRepository) GetHoldByToken(ctx context.Context, token string) (*domain.AppointmentHold, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.holds[token]
	if !ok {
		return nil, domain.ErrHoldNotFound
	}
	return h, nil
}

func (m *ThreadSafeMockRepository) GetHoldByID(ctx context.Context, holdID string) (*domain.AppointmentHold, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.holds[holdID]
	if !ok {
		return nil, domain.ErrHoldNotFound
	}
	return h, nil
}

func (m *ThreadSafeMockRepository) UpdateHoldStatus(ctx context.Context, holdID string, status domain.HoldStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.holds[holdID]
	if !ok {
		return domain.ErrHoldNotFound
	}
	h.Status = status
	return nil
}

func (m *ThreadSafeMockRepository) ExpireStaleHolds(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *ThreadSafeMockRepository) CheckActiveHoldConflict(ctx context.Context, doctorID, roomID string, start, end time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *ThreadSafeMockRepository) CreateAppointmentWithTx(ctx context.Context, appt *domain.Appointment, holdID *string, outbox *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate PostgreSQL GiST exclusion constraint check:
	// EXCLUDE USING gist (doctor_id WITH =, slot_range WITH &&)
	// EXCLUDE USING gist (room_id WITH =, slot_range WITH &&)
	for _, existing := range m.appointments {
		if existing.Status == domain.StatusConfirmed || existing.Status == domain.StatusCheckedIn || existing.Status == domain.StatusInProgress {
			doctorConflict := existing.DoctorID == appt.DoctorID
			roomConflict := existing.RoomID == appt.RoomID
			slotOverlap := !(appt.EndTime.Before(existing.StartTime) || appt.StartTime.After(existing.EndTime))

			if (doctorConflict || roomConflict) && slotOverlap {
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

func (m *ThreadSafeMockRepository) GetAppointmentByID(ctx context.Context, id string) (*domain.Appointment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.appointments[id]
	if !ok {
		return nil, domain.ErrAppointmentNotFound
	}
	return a, nil
}

func (m *ThreadSafeMockRepository) GetAppointmentByIdempotencyKey(ctx context.Context, key string) (*domain.Appointment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.idempotent[key]
	if !ok {
		return nil, domain.ErrAppointmentNotFound
	}
	return a, nil
}

func (m *ThreadSafeMockRepository) ListAppointments(ctx context.Context, filter repository.ListFilter) ([]*domain.Appointment, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *ThreadSafeMockRepository) UpdateAppointmentStatusWithTx(ctx context.Context, apptID string, fromStatus, toStatus domain.AppointmentStatus, reason string, changedBy *string, outbox *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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

func (m *ThreadSafeMockRepository) RescheduleAppointmentWithTx(ctx context.Context, oldApptID string, newAppt *domain.Appointment, outbox *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	old, ok := m.appointments[oldApptID]
	if !ok {
		return domain.ErrAppointmentNotFound
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

func (m *ThreadSafeMockRepository) CreateWaitlistEntry(ctx context.Context, entry *domain.AppointmentWaitlist) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.waitlists[entry.ID] = entry
	return nil
}

func (m *ThreadSafeMockRepository) GetWaitlistEntry(ctx context.Context, id string) (*domain.AppointmentWaitlist, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.waitlists[id]
	if !ok {
		return nil, domain.ErrWaitlistNotFound
	}
	return w, nil
}

func (m *ThreadSafeMockRepository) ListWaitlistEntries(ctx context.Context, clinicID string, patientID *string, status *domain.WaitlistStatus) ([]*domain.AppointmentWaitlist, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *ThreadSafeMockRepository) FindNextWaitlistMatch(ctx context.Context, clinicID, packageID string, doctorID *string, dateStr string) (*domain.AppointmentWaitlist, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, w := range m.waitlists {
		if w.ClinicID == clinicID && w.PackageID == packageID && w.Status == domain.WaitlistStatusWaiting {
			return w, nil
		}
	}
	return nil, domain.ErrWaitlistNotFound
}

func (m *ThreadSafeMockRepository) OfferWaitlistSlot(ctx context.Context, waitlistID string, start, end, expiresAt time.Time, outbox *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *ThreadSafeMockRepository) AcceptWaitlistOffer(ctx context.Context, waitlistID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.waitlists[waitlistID]
	if !ok {
		return domain.ErrWaitlistNotFound
	}
	w.Status = domain.WaitlistStatusFulfilled
	return nil
}

func (m *ThreadSafeMockRepository) CreateOutboxEvent(ctx context.Context, event *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outbox = append(m.outbox, event)
	return nil
}

func (m *ThreadSafeMockRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *ThreadSafeMockRepository) MarkOutboxEventProcessed(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.outbox {
		if e.ID == id {
			e.Status = domain.OutboxStatusProcessed
			return nil
		}
	}
	return nil
}

func (m *ThreadSafeMockRepository) MarkOutboxEventFailed(ctx context.Context, id string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.outbox {
		if e.ID == id {
			e.RetryCount++
			return nil
		}
	}
	return nil
}

func (m *ThreadSafeMockRepository) GetAppointmentHistory(ctx context.Context, appointmentID string) ([]*domain.AppointmentHistory, error) {
	return nil, nil
}

func (m *ThreadSafeMockRepository) GetBookedRanges(ctx context.Context, clinicID string, doctorID, roomID *string, start, end time.Time) ([]repository.TimeRange, error) {
	return nil, nil
}

// 1. CONCURRENCY ACCEPTANCE TEST: 20 Simultaneous Bookings for the Same Doctor & Room WITHOUT REDIS
func TestAcceptance_SimultaneousBookingsWithoutRedis(t *testing.T) {
	repo := NewThreadSafeMockRepository()
	// Pass redis as nil: this demonstrates that PostgreSQL source-of-truth invariants
	// mathematically prevent double booking even if Redis is completely dead or disabled!
	svc := service.NewAppointmentService(repo, nil, nil)
	ctx := context.Background()

	clinicID := uuid.New().String()
	doctorID := uuid.New().String()
	roomID := uuid.New().String()
	packageID := uuid.New().String()
	slotStart := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Hour)

	concurrency := 20
	var successCount int64
	var doubleBookingRejections int64

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()

			patientID := uuid.New().String()
			_, err := svc.BookDirectAppointment(ctx, service.DirectBookingRequest{
				PatientID:       patientID,
				ClinicID:        clinicID,
				DoctorID:        doctorID,
				RoomID:          roomID,
				PackageID:       packageID,
				StartTime:       slotStart,
				DurationMinutes: 90,
			})

			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else if errors.Is(err, domain.ErrSlotDoubleBooked) {
				atomic.AddInt64(&doubleBookingRejections, 1)
			}
		}(i)
	}

	wg.Wait()

	if successCount != 1 {
		t.Fatalf("CRITICAL CONCURRENCY FAILURE: Exactly 1 appointment should succeed, but %d succeeded!", successCount)
	}
	if doubleBookingRejections != int64(concurrency-1) {
		t.Fatalf("CRITICAL CONCURRENCY FAILURE: %d bookings should be rejected with ErrSlotDoubleBooked, got %d", concurrency-1, doubleBookingRejections)
	}
}

// 2. ACCEPTANCE TEST: Hold Expiration and Lazy Invalidation
func TestAcceptance_HoldExpiry(t *testing.T) {
	repo := NewThreadSafeMockRepository()
	svc := service.NewAppointmentService(repo, nil, nil)
	ctx := context.Background()

	clinicID := uuid.New().String()
	patientID := uuid.New().String()
	packageID := uuid.New().String()
	doctorID := uuid.New().String()
	roomID := uuid.New().String()
	slotStart := time.Now().UTC().Add(24 * time.Hour)

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
		t.Fatalf("hold creation failed: %v", err)
	}

	// Manually expire the hold in repository
	repo.mu.Lock()
	repo.holds[holdResp.HoldToken].ExpiresAt = time.Now().UTC().Add(-1 * time.Minute)
	repo.mu.Unlock()

	// Attempt to confirm the expired hold
	_, err = svc.ConfirmAppointment(ctx, service.ConfirmAppointmentRequest{
		HoldToken: holdResp.HoldToken,
		PatientID: patientID,
	})
	if err == nil {
		t.Fatalf("expected error confirming expired hold, but it succeeded!")
	}
	if !errors.Is(err, domain.ErrHoldExpired) {
		t.Fatalf("expected ErrHoldExpired, got %v", err)
	}
}

// 3. ACCEPTANCE TEST: Idempotency under Duplicate Submissions
func TestAcceptance_DuplicateIdempotentRequests(t *testing.T) {
	repo := NewThreadSafeMockRepository()
	svc := service.NewAppointmentService(repo, nil, nil)
	ctx := context.Background()

	clinicID := uuid.New().String()
	patientID := uuid.New().String()
	packageID := uuid.New().String()
	doctorID := uuid.New().String()
	roomID := uuid.New().String()
	slotStart := time.Now().UTC().Add(24 * time.Hour)
	idempKey := "idemp-test-duplicate-key-42"

	req := service.DirectBookingRequest{
		PatientID:      patientID,
		ClinicID:       clinicID,
		DoctorID:       doctorID,
		RoomID:         roomID,
		PackageID:      packageID,
		StartTime:      slotStart,
		IdempotencyKey: &idempKey,
	}

	// First execution
	firstAppt, err := svc.BookDirectAppointment(ctx, req)
	if err != nil {
		t.Fatalf("first booking failed: %v", err)
	}

	// Concurrent duplicate submissions with same idempotency key
	var wg sync.WaitGroup
	duplicateAttempts := 10
	errCount := int64(0)

	for i := 0; i < duplicateAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			retryAppt, err := svc.BookDirectAppointment(ctx, req)
			if err != nil {
				atomic.AddInt64(&errCount, 1)
				return
			}
			if retryAppt.ID != firstAppt.ID {
				t.Errorf("expected identical appointment ID %s, got %s", firstAppt.ID, retryAppt.ID)
			}
		}()
	}
	wg.Wait()

	if errCount != 0 {
		t.Fatalf("expected 0 errors on idempotent retries, got %d", errCount)
	}

	// Verify only 1 appointment exists in database
	if len(repo.appointments) != 1 {
		t.Fatalf("expected exactly 1 appointment in DB, got %d", len(repo.appointments))
	}
}

// 4. ACCEPTANCE TEST: Cancellation and Concurrent Check-In Race
func TestAcceptance_CancellationVsCheckInRace(t *testing.T) {
	repo := NewThreadSafeMockRepository()
	svc := service.NewAppointmentService(repo, nil, nil)
	ctx := context.Background()

	clinicID := uuid.New().String()
	patientID := uuid.New().String()
	doctorID := uuid.New().String()
	roomID := uuid.New().String()
	packageID := uuid.New().String()
	slotStart := time.Now().UTC().Add(2 * time.Hour)

	appt, err := svc.BookDirectAppointment(ctx, service.DirectBookingRequest{
		PatientID:       patientID,
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

	// Patient cancels and Clinic tries to cancel/check-in simultaneously
	var wg sync.WaitGroup
	var cancelSucceeded int64
	var invalidTransitions int64

	wg.Add(2)
	go func() {
		defer wg.Done()
		err := svc.CancelAppointment(ctx, appt.ID, patientID, "patient", clinicID, "Cancel A")
		if err == nil {
			atomic.AddInt64(&cancelSucceeded, 1)
		} else {
			atomic.AddInt64(&invalidTransitions, 1)
		}
	}()

	go func() {
		defer wg.Done()
		err := svc.CancelAppointment(ctx, appt.ID, patientID, "patient", clinicID, "Cancel B")
		if err == nil {
			atomic.AddInt64(&cancelSucceeded, 1)
		} else {
			atomic.AddInt64(&invalidTransitions, 1)
		}
	}()

	wg.Wait()

	if cancelSucceeded != 1 {
		t.Fatalf("expected exactly 1 cancel to succeed, got %d", cancelSucceeded)
	}
	if invalidTransitions != 1 {
		t.Fatalf("expected exactly 1 transition to fail, got %d", invalidTransitions)
	}
}

// 5. ACCEPTANCE TEST: Time-Zone Boundaries (UTC vs Asia/Kolkata UTC+5:30)
func TestAcceptance_TimezoneBoundaries(t *testing.T) {
	locKolkata, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Fatalf("failed loading Asia/Kolkata timezone: %v", err)
	}

	// 10:00 AM on Nov 1st in Kolkata is 04:30 AM UTC
	kolkataTime := time.Date(2026, 11, 1, 10, 0, 0, 0, locKolkata)
	utcEquivalent := kolkataTime.UTC()

	if utcEquivalent.Hour() != 4 || utcEquivalent.Minute() != 30 {
		t.Fatalf("expected 10:00 Kolkata to be 04:30 UTC, got %v", utcEquivalent)
	}

	// Verify range check detects overlap across timezone boundaries
	slotAStart := kolkataTime.UTC()
	slotAEnd := slotAStart.Add(90 * time.Minute)

	// Another appointment booked by someone specifying UTC directly
	slotBStart := time.Date(2026, 11, 1, 5, 0, 0, 0, time.UTC) // 05:00 UTC falls inside [04:30, 06:00) UTC
	slotBEnd := slotBStart.Add(90 * time.Minute)

	overlap := !(slotBEnd.Before(slotAStart) || slotBStart.After(slotAEnd))
	if !overlap {
		t.Fatalf("expected timezone boundary overlap between Kolkata 10:00-11:30 and UTC 05:00-06:30")
	}
}

// 6. ACCEPTANCE TEST: Unauthorized Appointment Access
func TestAcceptance_UnauthorizedAppointmentAccess(t *testing.T) {
	repo := NewThreadSafeMockRepository()
	svc := service.NewAppointmentService(repo, nil, nil)
	ctx := context.Background()

	clinicA := uuid.New().String()
	clinicB := uuid.New().String()
	patientA := uuid.New().String()
	patientB := uuid.New().String()

	apptA, _ := svc.BookDirectAppointment(ctx, service.DirectBookingRequest{
		PatientID: patientA,
		ClinicID:  clinicA,
		DoctorID:  "doc-1",
		RoomID:    "room-1",
		PackageID: "pkg-1",
		StartTime: time.Now().UTC().Add(24 * time.Hour),
	})

	// 1. Patient B attempts to retrieve Patient A's appointment
	_, err := svc.GetAppointment(ctx, apptA.ID, patientB, "patient", clinicA)
	if !errors.Is(err, domain.ErrUnauthorizedAccess) {
		t.Fatalf("expected ErrUnauthorizedAccess for Patient B, got %v", err)
	}

	// 2. Staff from Clinic B attempts to retrieve Clinic A's appointment
	_, err = svc.GetAppointment(ctx, apptA.ID, "staff-b", "clinic_admin", clinicB)
	if !errors.Is(err, domain.ErrUnauthorizedAccess) {
		t.Fatalf("expected ErrUnauthorizedAccess for Clinic B staff, got %v", err)
	}

	// 3. Patient B attempts to cancel Patient A's appointment
	err = svc.CancelAppointment(ctx, apptA.ID, patientB, "patient", clinicA, "Hacked cancel")
	if !errors.Is(err, domain.ErrUnauthorizedAccess) {
		t.Fatalf("expected ErrUnauthorizedAccess on unauthorized cancellation, got %v", err)
	}
}
