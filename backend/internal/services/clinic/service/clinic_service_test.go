package service_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/service"
)

// inMemoryClinicRepo provides a thread-safe mock for service layer tests.
type inMemoryClinicRepo struct {
	mu         sync.RWMutex
	clinics    map[string]*domain.Clinic
	rooms      map[string]*domain.TreatmentRoom
	equipment  map[string][]domain.RoomEquipment
	doctors    map[string]*domain.Doctor
	therapists map[string]*domain.Therapist
	schedules  map[string]*domain.DoctorSchedule
	leaves     map[string]*domain.DoctorLeave
	packages   map[string]*domain.TherapyPackage
	steps      map[string][]domain.PackageStep
	checklists map[string][]domain.CareChecklist
}

func newInMemoryClinicRepo() *inMemoryClinicRepo {
	return &inMemoryClinicRepo{
		clinics:    make(map[string]*domain.Clinic),
		rooms:      make(map[string]*domain.TreatmentRoom),
		equipment:  make(map[string][]domain.RoomEquipment),
		doctors:    make(map[string]*domain.Doctor),
		therapists: make(map[string]*domain.Therapist),
		schedules:  make(map[string]*domain.DoctorSchedule),
		leaves:     make(map[string]*domain.DoctorLeave),
		packages:   make(map[string]*domain.TherapyPackage),
		steps:      make(map[string][]domain.PackageStep),
		checklists: make(map[string][]domain.CareChecklist),
	}
}

func (m *inMemoryClinicRepo) CreateClinic(ctx context.Context, c *domain.Clinic) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clinics[c.ID] = c
	return nil
}

func (m *inMemoryClinicRepo) GetClinicByID(ctx context.Context, id string) (*domain.Clinic, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clinics[id], nil
}

func (m *inMemoryClinicRepo) GetClinicByRegistrationNumber(ctx context.Context, regNum string) (*domain.Clinic, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.clinics {
		if strings.EqualFold(c.RegistrationNumber, regNum) {
			return c, nil
		}
	}
	return nil, nil
}

func (m *inMemoryClinicRepo) ListClinics(ctx context.Context, city string, activeOnly bool) ([]*domain.Clinic, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*domain.Clinic
	for _, c := range m.clinics {
		if city != "" && !strings.EqualFold(c.AddressCity, city) {
			continue
		}
		if activeOnly && !c.IsActive {
			continue
		}
		res = append(res, c)
	}
	return res, nil
}

func (m *inMemoryClinicRepo) UpdateClinic(ctx context.Context, c *domain.Clinic) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clinics[c.ID] = c
	return nil
}

func (m *inMemoryClinicRepo) DeactivateClinic(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.clinics[id]; ok {
		c.IsActive = false
	}
	return nil
}

func (m *inMemoryClinicRepo) CreateRoom(ctx context.Context, r *domain.TreatmentRoom) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rooms[r.ID] = r
	return nil
}

func (m *inMemoryClinicRepo) GetRoomByID(ctx context.Context, id string) (*domain.TreatmentRoom, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rooms[id], nil
}

func (m *inMemoryClinicRepo) ListRooms(ctx context.Context, clinicID string) ([]*domain.TreatmentRoom, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*domain.TreatmentRoom
	for _, r := range m.rooms {
		if r.ClinicID == clinicID {
			res = append(res, r)
		}
	}
	return res, nil
}

func (m *inMemoryClinicRepo) UpdateRoom(ctx context.Context, room *domain.TreatmentRoom) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rooms[room.ID] = room
	return nil
}

func (m *inMemoryClinicRepo) DeactivateRoom(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.rooms[id]; ok {
		r.IsActive = false
	}
	return nil
}

func (m *inMemoryClinicRepo) AddRoomEquipment(ctx context.Context, eq *domain.RoomEquipment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.equipment[eq.RoomID] = append(m.equipment[eq.RoomID], *eq)
	return nil
}

func (m *inMemoryClinicRepo) ListRoomEquipment(ctx context.Context, roomID string) ([]domain.RoomEquipment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.equipment[roomID], nil
}

func (m *inMemoryClinicRepo) CreateDoctor(ctx context.Context, d *domain.Doctor) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.doctors[d.ID] = d
	return nil
}

func (m *inMemoryClinicRepo) GetDoctorByID(ctx context.Context, id string) (*domain.Doctor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.doctors[id], nil
}

func (m *inMemoryClinicRepo) GetDoctorByUserID(ctx context.Context, userID string) (*domain.Doctor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, d := range m.doctors {
		if d.UserID == userID {
			return d, nil
		}
	}
	return nil, nil
}

func (m *inMemoryClinicRepo) ListDoctors(ctx context.Context, clinicID string) ([]*domain.Doctor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*domain.Doctor
	for _, d := range m.doctors {
		if d.ClinicID == clinicID {
			res = append(res, d)
		}
	}
	return res, nil
}

func (m *inMemoryClinicRepo) UpdateDoctor(ctx context.Context, d *domain.Doctor) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.doctors[d.ID] = d
	return nil
}

func (m *inMemoryClinicRepo) DeactivateDoctor(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.doctors[id]; ok {
		d.IsActive = false
	}
	return nil
}

func (m *inMemoryClinicRepo) CreateTherapist(ctx context.Context, t *domain.Therapist) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.therapists[t.ID] = t
	return nil
}

func (m *inMemoryClinicRepo) GetTherapistByID(ctx context.Context, id string) (*domain.Therapist, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.therapists[id], nil
}

func (m *inMemoryClinicRepo) GetTherapistByUserID(ctx context.Context, userID string) (*domain.Therapist, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.therapists {
		if t.UserID == userID {
			return t, nil
		}
	}
	return nil, nil
}

func (m *inMemoryClinicRepo) ListTherapists(ctx context.Context, clinicID string) ([]*domain.Therapist, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*domain.Therapist
	for _, t := range m.therapists {
		if t.ClinicID == clinicID {
			res = append(res, t)
		}
	}
	return res, nil
}

func (m *inMemoryClinicRepo) UpdateTherapist(ctx context.Context, t *domain.Therapist) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.therapists[t.ID] = t
	return nil
}

func (m *inMemoryClinicRepo) DeactivateTherapist(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.therapists[id]; ok {
		t.IsActive = false
	}
	return nil
}

func (m *inMemoryClinicRepo) CreateSchedule(ctx context.Context, s *domain.DoctorSchedule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.schedules[s.ID] = s
	return nil
}

func (m *inMemoryClinicRepo) GetScheduleByID(ctx context.Context, id string) (*domain.DoctorSchedule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.schedules[id], nil
}

func (m *inMemoryClinicRepo) ListDoctorSchedules(ctx context.Context, doctorID string) ([]*domain.DoctorSchedule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*domain.DoctorSchedule
	for _, s := range m.schedules {
		if s.DoctorID == doctorID && s.IsActive {
			res = append(res, s)
		}
	}
	return res, nil
}

func (m *inMemoryClinicRepo) ListDoctorSchedulesForDay(ctx context.Context, doctorID string, dayOfWeek int) ([]*domain.DoctorSchedule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*domain.DoctorSchedule
	for _, s := range m.schedules {
		if s.DoctorID == doctorID && s.DayOfWeek == dayOfWeek && s.IsActive {
			res = append(res, s)
		}
	}
	return res, nil
}

func (m *inMemoryClinicRepo) DeleteSchedule(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.schedules, id)
	return nil
}

func (m *inMemoryClinicRepo) CreateLeave(ctx context.Context, l *domain.DoctorLeave) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leaves[l.ID] = l
	return nil
}

func (m *inMemoryClinicRepo) GetLeaveByID(ctx context.Context, id string) (*domain.DoctorLeave, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.leaves[id], nil
}

func (m *inMemoryClinicRepo) ListDoctorLeaves(ctx context.Context, doctorID string) ([]*domain.DoctorLeave, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*domain.DoctorLeave
	for _, l := range m.leaves {
		if l.DoctorID == doctorID {
			res = append(res, l)
		}
	}
	return res, nil
}

func (m *inMemoryClinicRepo) GetApprovedLeavesInRange(ctx context.Context, doctorID string, start, end time.Time) ([]*domain.DoctorLeave, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*domain.DoctorLeave
	for _, l := range m.leaves {
		if l.DoctorID == doctorID && l.Status == "APPROVED" {
			// Overlap: l.StartTime < end AND l.EndTime > start
			if l.StartTime.Before(end) && l.EndTime.After(start) {
				res = append(res, l)
			}
		}
	}
	return res, nil
}

func (m *inMemoryClinicRepo) UpdateLeaveStatus(ctx context.Context, id string, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.leaves[id]; ok {
		l.Status = status
	}
	return nil
}

func (m *inMemoryClinicRepo) CreatePackage(ctx context.Context, p *domain.TherapyPackage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.packages[p.ID] = p
	return nil
}

func (m *inMemoryClinicRepo) GetPackageByID(ctx context.Context, id string) (*domain.TherapyPackage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pkg, exists := m.packages[id]
	if !exists {
		return nil, nil
	}
	pkgCopy := *pkg
	pkgCopy.Steps = m.steps[id]
	pkgCopy.Checklists = m.checklists[id]
	return &pkgCopy, nil
}

func (m *inMemoryClinicRepo) ListPackages(ctx context.Context, clinicID string, publishedOnly bool) ([]*domain.TherapyPackage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*domain.TherapyPackage
	for _, p := range m.packages {
		if p.ClinicID == clinicID && p.IsActive {
			if publishedOnly && !p.IsPublished {
				continue
			}
			pCopy := *p
			pCopy.Steps = m.steps[p.ID]
			res = append(res, &pCopy)
		}
	}
	return res, nil
}

func (m *inMemoryClinicRepo) UpdatePackage(ctx context.Context, p *domain.TherapyPackage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.packages[p.ID] = p
	return nil
}

func (m *inMemoryClinicRepo) PublishPackage(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.packages[id]; ok {
		p.IsPublished = true
	}
	return nil
}

func (m *inMemoryClinicRepo) DeactivatePackage(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.packages[id]; ok {
		p.IsActive = false
	}
	return nil
}

func (m *inMemoryClinicRepo) SavePackageSteps(ctx context.Context, packageID string, steps []domain.PackageStep) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.steps[packageID] = steps
	return nil
}

func (m *inMemoryClinicRepo) GetPackageSteps(ctx context.Context, packageID string) ([]domain.PackageStep, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.steps[packageID], nil
}

func (m *inMemoryClinicRepo) SaveCareChecklist(ctx context.Context, cl *domain.CareChecklist) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checklists[cl.PackageID] = append(m.checklists[cl.PackageID], *cl)
	return nil
}

func (m *inMemoryClinicRepo) GetCareChecklists(ctx context.Context, packageID string) ([]domain.CareChecklist, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.checklists[packageID], nil
}

func setupTestClinicService() (service.ClinicService, *inMemoryClinicRepo) {
	repo := newInMemoryClinicRepo()
	svc := service.NewClinicService(repo)
	return svc, repo
}

func TestClinicService_ClinicManagement(t *testing.T) {
	svc, _ := setupTestClinicService()
	ctx := context.Background()

	t.Run("CreateClinic_Success", func(t *testing.T) {
		req := &service.CreateClinicRequest{
			Name:               "Panchakarma Ayurveda Center",
			RegistrationNumber: "AYUR-REG-2026-001",
			AddressStreet:      "108 Spiritual Path",
			AddressCity:        "Rishikesh",
			AddressState:       "Uttarakhand",
			AddressPostalCode:  "249201",
			ContactPhone:       "+919876543210",
			ContactEmail:       "care@rishikeshayur.com",
			Timezone:           "Asia/Kolkata",
		}

		clinic, err := svc.CreateClinic(ctx, req, "super_admin")
		if err != nil {
			t.Fatalf("unexpected error creating clinic: %v", err)
		}
		if clinic.ID == "" || clinic.Timezone != "Asia/Kolkata" {
			t.Errorf("invalid clinic record: %+v", clinic)
		}
	})

	t.Run("CreateClinic_InvalidTimezone_Rejected", func(t *testing.T) {
		req := &service.CreateClinicRequest{
			Name:               "Invalid Zone Clinic",
			RegistrationNumber: "AYUR-REG-2026-002",
			AddressStreet:      "123 Street",
			AddressCity:        "City",
			Timezone:           "Invalid/NonExistent_Zone",
		}

		_, err := svc.CreateClinic(ctx, req, "super_admin")
		if err == nil {
			t.Fatalf("expected error for invalid IANA time zone, got nil")
		}
	})

	t.Run("CreateClinic_NonSuperAdmin_Rejected", func(t *testing.T) {
		req := &service.CreateClinicRequest{
			Name:               "Unpermitted Clinic",
			RegistrationNumber: "AYUR-REG-2026-003",
			Timezone:           "Asia/Kolkata",
		}

		_, err := svc.CreateClinic(ctx, req, "clinic_admin")
		if err == nil {
			t.Fatalf("expected error when clinic_admin creates clinic, got nil")
		}
	})
}

func TestClinicService_CrossClinicIsolation(t *testing.T) {
	svc, repo := setupTestClinicService()
	ctx := context.Background()

	clinicA_ID := uuid.New().String()
	clinicB_ID := uuid.New().String()

	_ = repo.CreateClinic(ctx, &domain.Clinic{ID: clinicA_ID, Name: "Clinic A", Timezone: "Asia/Kolkata", IsActive: true})
	_ = repo.CreateClinic(ctx, &domain.Clinic{ID: clinicB_ID, Name: "Clinic B", Timezone: "Asia/Kolkata", IsActive: true})

	t.Run("ClinicAdminA_CannotModify_ClinicB", func(t *testing.T) {
		req := &service.UpdateClinicRequest{
			Name: "Hacked Clinic B",
		}

		// Clinic Admin of Clinic A attempting to update Clinic B
		_, err := svc.UpdateClinic(ctx, clinicB_ID, req, "clinic_admin", clinicA_ID)
		if err == nil {
			t.Fatalf("expected cross-clinic access violation error, got nil")
		}
	})

	t.Run("ClinicAdminA_CannotCreateRoomIn_ClinicB", func(t *testing.T) {
		roomReq := &service.CreateRoomRequest{
			Name:     "Unauthorized Room",
			RoomType: "standard",
			Capacity: 1,
		}

		_, err := svc.CreateRoom(ctx, clinicB_ID, roomReq, "clinic_admin", clinicA_ID)
		if err == nil {
			t.Fatalf("expected cross-clinic room creation error, got nil")
		}
	})

	t.Run("SuperAdmin_CanManage_AnyClinic", func(t *testing.T) {
		req := &service.UpdateClinicRequest{
			Name: "Admin Updated Clinic B",
		}

		updated, err := svc.UpdateClinic(ctx, clinicB_ID, req, "super_admin", "")
		if err != nil {
			t.Fatalf("super_admin should have access: %v", err)
		}
		if updated.Name != "Admin Updated Clinic B" {
			t.Errorf("name not updated")
		}
	})
}

func TestClinicService_DoctorSchedulesAndOverlaps(t *testing.T) {
	svc, repo := setupTestClinicService()
	ctx := context.Background()

	clinicID := uuid.New().String()
	docID := uuid.New().String()

	_ = repo.CreateClinic(ctx, &domain.Clinic{ID: clinicID, Name: "Schedule Clinic", Timezone: "Asia/Kolkata", IsActive: true})
	_ = repo.CreateDoctor(ctx, &domain.Doctor{ID: docID, UserID: uuid.New().String(), ClinicID: clinicID, MaxSessionsPerDay: 8, IsActive: true})

	t.Run("CreateValidSchedule", func(t *testing.T) {
		req := &service.CreateScheduleRequest{
			DayOfWeek:           1, // Monday
			StartTime:           "09:00",
			EndTime:             "13:00",
			SlotDurationMinutes: 60,
		}

		sch, err := svc.CreateSchedule(ctx, clinicID, docID, req, "clinic_admin", clinicID)
		if err != nil {
			t.Fatalf("unexpected error creating schedule: %v", err)
		}
		if sch.StartTime != "09:00" || sch.EndTime != "13:00" {
			t.Errorf("schedule mismatch: %+v", sch)
		}
	})

	t.Run("InvalidTimeRange_Rejection", func(t *testing.T) {
		req := &service.CreateScheduleRequest{
			DayOfWeek:           1,
			StartTime:           "14:00",
			EndTime:             "11:00", // Invalid: end before start!
			SlotDurationMinutes: 60,
		}

		_, err := svc.CreateSchedule(ctx, clinicID, docID, req, "clinic_admin", clinicID)
		if err == nil {
			t.Fatalf("expected error for end_time <= start_time, got nil")
		}
	})

	t.Run("ScheduleOverlap_Rejection", func(t *testing.T) {
		// Existing is Monday 09:00-13:00
		// Attempt to add Monday 12:00-16:00 (overlaps from 12:00 to 13:00!)
		overlappingReq := &service.CreateScheduleRequest{
			DayOfWeek:           1,
			StartTime:           "12:00",
			EndTime:             "16:00",
			SlotDurationMinutes: 60,
		}

		_, err := svc.CreateSchedule(ctx, clinicID, docID, overlappingReq, "clinic_admin", clinicID)
		if err == nil {
			t.Fatalf("expected schedule overlap error, got nil")
		}
	})

	t.Run("NonOverlappingSchedule_Success", func(t *testing.T) {
		// Monday afternoon shift: 14:00-18:00 (distinct from 09:00-13:00)
		afternoonReq := &service.CreateScheduleRequest{
			DayOfWeek:           1,
			StartTime:           "14:00",
			EndTime:             "18:00",
			SlotDurationMinutes: 60,
		}

		sch, err := svc.CreateSchedule(ctx, clinicID, docID, afternoonReq, "clinic_admin", clinicID)
		if err != nil {
			t.Fatalf("unexpected error for non-overlapping schedule: %v", err)
		}
		if sch.StartTime != "14:00" {
			t.Errorf("schedule mismatch")
		}
	})
}

func TestClinicService_DoctorLeavesAndAvailability(t *testing.T) {
	svc, repo := setupTestClinicService()
	ctx := context.Background()

	clinicID := uuid.New().String()
	docID := uuid.New().String()

	_ = repo.CreateClinic(ctx, &domain.Clinic{ID: clinicID, Name: "Availability Clinic", Timezone: "Asia/Kolkata", IsActive: true})
	_ = repo.CreateDoctor(ctx, &domain.Doctor{ID: docID, UserID: uuid.New().String(), ClinicID: clinicID, MaxSessionsPerDay: 8, IsActive: true})

	// Add Monday schedule: 09:00 - 13:00 (60-min slots -> 09:00, 10:00, 11:00, 12:00)
	_ = repo.CreateSchedule(ctx, &domain.DoctorSchedule{
		ID:                  uuid.New().String(),
		DoctorID:            docID,
		DayOfWeek:           1, // Monday
		StartTime:           "09:00",
		EndTime:             "13:00",
		SlotDurationMinutes: 60,
		IsActive:            true,
	})

	t.Run("ComputeAvailability_ActiveWorkingDay", func(t *testing.T) {
		// Date: 2026-11-02 (Monday)
		avail, err := svc.GetDoctorAvailability(ctx, docID, "2026-11-02")
		if err != nil {
			t.Fatalf("failed to get availability: %v", err)
		}
		if avail.IsOnLeave {
			t.Fatalf("expected doctor not to be on leave")
		}
		if len(avail.AvailableTimeSlots) != 4 {
			t.Errorf("expected 4 hourly slots, got %v", avail.AvailableTimeSlots)
		}
	})

	t.Run("CreateLeaveAndVerifyAvailabilityZeroed", func(t *testing.T) {
		leaveStart, _ := time.Parse(time.RFC3339, "2026-11-09T00:00:00Z")
		leaveEnd, _ := time.Parse(time.RFC3339, "2026-11-10T00:00:00Z")

		leaveReq := &service.CreateLeaveRequest{
			StartTime: leaveStart,
			EndTime:   leaveEnd,
			Reason:    "Ayurvedic Botanical Field Study",
		}

		leave, err := svc.CreateDoctorLeave(ctx, clinicID, docID, leaveReq, "clinic_admin", clinicID)
		if err != nil {
			t.Fatalf("failed to create leave: %v", err)
		}
		if leave.Status != "APPROVED" {
			t.Errorf("leave status mismatch: %s", leave.Status)
		}

		// Query availability for Monday 2026-11-09 during leave
		avail, err := svc.GetDoctorAvailability(ctx, docID, "2026-11-09")
		if err != nil {
			t.Fatalf("failed to check availability: %v", err)
		}
		if !avail.IsOnLeave {
			t.Errorf("expected doctor to be marked on leave")
		}
		if len(avail.AvailableTimeSlots) != 0 {
			t.Errorf("expected 0 slots during leave, got %v", avail.AvailableTimeSlots)
		}
	})

	t.Run("OverlappingLeave_Rejected", func(t *testing.T) {
		// Attempt overlapping leave on same range
		overlapStart, _ := time.Parse(time.RFC3339, "2026-11-09T06:00:00Z")
		overlapEnd, _ := time.Parse(time.RFC3339, "2026-11-09T18:00:00Z")

		_, err := svc.CreateDoctorLeave(ctx, clinicID, docID, &service.CreateLeaveRequest{
			StartTime: overlapStart,
			EndTime:   overlapEnd,
			Reason:    "Duplicate Leave",
		}, "clinic_admin", clinicID)

		if err == nil {
			t.Fatalf("expected overlapping leave rejection, got nil")
		}
	})
}

func TestClinicService_TherapyPackagesAndVersioning(t *testing.T) {
	svc, _ := setupTestClinicService()
	ctx := context.Background()

	clinicID := uuid.New().String()

	t.Run("CreatePackage_Version1", func(t *testing.T) {
		req := &service.CreatePackageRequest{
			Name:                     "Virechana Detox Karma",
			Category:                 "detox",
			Description:              "Purgation therapy for Pitta-related imbalances",
			PriceCents:               2500000,
			TotalSessions:            5,
			DurationMinutes:          90,
			RecommendedIntervalsDays: 1,
		}

		pkg, err := svc.CreatePackage(ctx, clinicID, req, "clinic_admin", clinicID)
		if err != nil {
			t.Fatalf("failed to create package: %v", err)
		}
		if pkg.Version != 1 || pkg.IsPublished {
			t.Errorf("expected version 1 and unpublished, got %+v", pkg)
		}
	})

	t.Run("PublishPackage_WithoutSteps_Rejected", func(t *testing.T) {
		pkg, _ := svc.CreatePackage(ctx, clinicID, &service.CreatePackageRequest{
			Name:            "Empty Package",
			Category:        "rejuvenation",
			DurationMinutes: 60,
			TotalSessions:   1,
		}, "clinic_admin", clinicID)

		err := svc.PublishPackage(ctx, clinicID, pkg.ID, "clinic_admin", clinicID)
		if err == nil {
			t.Fatalf("expected error publishing package without steps, got nil")
		}
	})

	t.Run("ConfigureSteps_Publish_AndVerifySnapshotVersioning", func(t *testing.T) {
		pkg, _ := svc.CreatePackage(ctx, clinicID, &service.CreatePackageRequest{
			Name:            "Nasya Therapy Protocol",
			Category:        "panchakarma_head",
			DurationMinutes: 45,
			TotalSessions:   3,
			PriceCents:      1200000,
		}, "clinic_admin", clinicID)

		// 1. Configure Steps
		steps := []service.PackageStepRequest{
			{
				StepOrder:       1,
				Name:            "Facial Steam (Mukha Sweda)",
				DurationMinutes: 15,
				Instructions:    []string{"Apply Dashamoola decoction steam to face and neck"},
			},
			{
				StepOrder:       2,
				Name:            "Anu Taila Administration",
				DurationMinutes: 30,
				Instructions:    []string{"Instill 8 drops of lukewarm Anu Taila per nostril"},
			},
		}

		err := svc.SavePackageSteps(ctx, clinicID, pkg.ID, steps, "clinic_admin", clinicID)
		if err != nil {
			t.Fatalf("failed to save steps: %v", err)
		}

		// 2. Publish
		err = svc.PublishPackage(ctx, clinicID, pkg.ID, "clinic_admin", clinicID)
		if err != nil {
			t.Fatalf("failed to publish package: %v", err)
		}

		// 3. Update published package: Version MUST increment to preserve snapshots for bookings
		updateReq := &service.UpdatePackageRequest{
			Name:            "Nasya Therapy Protocol - Advanced",
			DurationMinutes: 50,
			PriceCents:      1400000,
		}

		updated, err := svc.UpdatePackage(ctx, clinicID, pkg.ID, updateReq, "clinic_admin", clinicID)
		if err != nil {
			t.Fatalf("failed to update published package: %v", err)
		}
		if updated.Version != 2 {
			t.Fatalf("expected version increment to 2, got %d", updated.Version)
		}

		// 4. Verify snapshot retrieval
		snapshot, err := svc.GetPackageSnapshot(ctx, pkg.ID)
		if err != nil {
			t.Fatalf("failed to get package snapshot: %v", err)
		}
		if snapshot.Version != 2 || len(snapshot.Steps) != 2 {
			t.Errorf("snapshot mismatch: %+v", snapshot)
		}
	})
}
