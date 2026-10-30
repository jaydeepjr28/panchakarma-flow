package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/handler"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	clinicv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/clinic/v1"
)

type mockClinicRepo struct {
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

func newMockClinicRepo() *mockClinicRepo {
	return &mockClinicRepo{
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

func (m *mockClinicRepo) CreateClinic(ctx context.Context, c *domain.Clinic) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clinics[c.ID] = c
	return nil
}

func (m *mockClinicRepo) GetClinicByID(ctx context.Context, id string) (*domain.Clinic, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clinics[id], nil
}

func (m *mockClinicRepo) GetClinicByRegistrationNumber(ctx context.Context, regNum string) (*domain.Clinic, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.clinics {
		if strings.EqualFold(c.RegistrationNumber, regNum) {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockClinicRepo) ListClinics(ctx context.Context, city string, activeOnly bool) ([]*domain.Clinic, error) {
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

func (m *mockClinicRepo) UpdateClinic(ctx context.Context, c *domain.Clinic) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clinics[c.ID] = c
	return nil
}

func (m *mockClinicRepo) DeactivateClinic(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.clinics[id]; ok {
		c.IsActive = false
	}
	return nil
}

func (m *mockClinicRepo) CreateRoom(ctx context.Context, r *domain.TreatmentRoom) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rooms[r.ID] = r
	return nil
}

func (m *mockClinicRepo) GetRoomByID(ctx context.Context, id string) (*domain.TreatmentRoom, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rooms[id], nil
}

func (m *mockClinicRepo) ListRooms(ctx context.Context, clinicID string) ([]*domain.TreatmentRoom, error) {
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

func (m *mockClinicRepo) UpdateRoom(ctx context.Context, room *domain.TreatmentRoom) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rooms[room.ID] = room
	return nil
}

func (m *mockClinicRepo) DeactivateRoom(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.rooms[id]; ok {
		r.IsActive = false
	}
	return nil
}

func (m *mockClinicRepo) AddRoomEquipment(ctx context.Context, eq *domain.RoomEquipment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.equipment[eq.RoomID] = append(m.equipment[eq.RoomID], *eq)
	return nil
}

func (m *mockClinicRepo) ListRoomEquipment(ctx context.Context, roomID string) ([]domain.RoomEquipment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.equipment[roomID], nil
}

func (m *mockClinicRepo) CreateDoctor(ctx context.Context, d *domain.Doctor) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.doctors[d.ID] = d
	return nil
}

func (m *mockClinicRepo) GetDoctorByID(ctx context.Context, id string) (*domain.Doctor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.doctors[id], nil
}

func (m *mockClinicRepo) GetDoctorByUserID(ctx context.Context, userID string) (*domain.Doctor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, d := range m.doctors {
		if d.UserID == userID {
			return d, nil
		}
	}
	return nil, nil
}

func (m *mockClinicRepo) ListDoctors(ctx context.Context, clinicID string) ([]*domain.Doctor, error) {
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

func (m *mockClinicRepo) UpdateDoctor(ctx context.Context, d *domain.Doctor) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.doctors[d.ID] = d
	return nil
}

func (m *mockClinicRepo) DeactivateDoctor(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.doctors[id]; ok {
		d.IsActive = false
	}
	return nil
}

func (m *mockClinicRepo) CreateTherapist(ctx context.Context, t *domain.Therapist) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.therapists[t.ID] = t
	return nil
}

func (m *mockClinicRepo) GetTherapistByID(ctx context.Context, id string) (*domain.Therapist, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.therapists[id], nil
}

func (m *mockClinicRepo) GetTherapistByUserID(ctx context.Context, userID string) (*domain.Therapist, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.therapists {
		if t.UserID == userID {
			return t, nil
		}
	}
	return nil, nil
}

func (m *mockClinicRepo) ListTherapists(ctx context.Context, clinicID string) ([]*domain.Therapist, error) {
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

func (m *mockClinicRepo) UpdateTherapist(ctx context.Context, t *domain.Therapist) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.therapists[t.ID] = t
	return nil
}

func (m *mockClinicRepo) DeactivateTherapist(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.therapists[id]; ok {
		t.IsActive = false
	}
	return nil
}

func (m *mockClinicRepo) CreateSchedule(ctx context.Context, s *domain.DoctorSchedule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.schedules[s.ID] = s
	return nil
}

func (m *mockClinicRepo) GetScheduleByID(ctx context.Context, id string) (*domain.DoctorSchedule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.schedules[id], nil
}

func (m *mockClinicRepo) ListDoctorSchedules(ctx context.Context, doctorID string) ([]*domain.DoctorSchedule, error) {
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

func (m *mockClinicRepo) ListDoctorSchedulesForDay(ctx context.Context, doctorID string, dayOfWeek int) ([]*domain.DoctorSchedule, error) {
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

func (m *mockClinicRepo) DeleteSchedule(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.schedules, id)
	return nil
}

func (m *mockClinicRepo) CreateLeave(ctx context.Context, l *domain.DoctorLeave) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leaves[l.ID] = l
	return nil
}

func (m *mockClinicRepo) GetLeaveByID(ctx context.Context, id string) (*domain.DoctorLeave, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.leaves[id], nil
}

func (m *mockClinicRepo) ListDoctorLeaves(ctx context.Context, doctorID string) ([]*domain.DoctorLeave, error) {
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

func (m *mockClinicRepo) GetApprovedLeavesInRange(ctx context.Context, doctorID string, start, end time.Time) ([]*domain.DoctorLeave, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*domain.DoctorLeave
	for _, l := range m.leaves {
		if l.DoctorID == doctorID && l.Status == "APPROVED" {
			if l.StartTime.Before(end) && l.EndTime.After(start) {
				res = append(res, l)
			}
		}
	}
	return res, nil
}

func (m *mockClinicRepo) UpdateLeaveStatus(ctx context.Context, id string, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.leaves[id]; ok {
		l.Status = status
	}
	return nil
}

func (m *mockClinicRepo) CreatePackage(ctx context.Context, p *domain.TherapyPackage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.packages[p.ID] = p
	return nil
}

func (m *mockClinicRepo) GetPackageByID(ctx context.Context, id string) (*domain.TherapyPackage, error) {
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

func (m *mockClinicRepo) ListPackages(ctx context.Context, clinicID string, publishedOnly bool) ([]*domain.TherapyPackage, error) {
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

func (m *mockClinicRepo) UpdatePackage(ctx context.Context, p *domain.TherapyPackage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.packages[p.ID] = p
	return nil
}

func (m *mockClinicRepo) PublishPackage(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.packages[id]; ok {
		p.IsPublished = true
	}
	return nil
}

func (m *mockClinicRepo) DeactivatePackage(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.packages[id]; ok {
		p.IsActive = false
	}
	return nil
}

func (m *mockClinicRepo) SavePackageSteps(ctx context.Context, packageID string, steps []domain.PackageStep) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.steps[packageID] = steps
	return nil
}

func (m *mockClinicRepo) GetPackageSteps(ctx context.Context, packageID string) ([]domain.PackageStep, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.steps[packageID], nil
}

func (m *mockClinicRepo) SaveCareChecklist(ctx context.Context, cl *domain.CareChecklist) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checklists[cl.PackageID] = append(m.checklists[cl.PackageID], *cl)
	return nil
}

func (m *mockClinicRepo) GetCareChecklists(ctx context.Context, packageID string) ([]domain.CareChecklist, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.checklists[packageID], nil
}

func setupTestRouter(t *testing.T) (*gin.Engine, *auth.TokenManager, service.ClinicService, *mockClinicRepo) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	repo := newMockClinicRepo()
	tm, err := auth.NewTokenManager("panchakarma_clinic_test_secret_at_least_32_bytes_long", 15*time.Minute, 7*24*time.Hour, nil)
	if err != nil {
		t.Fatalf("failed to init TokenManager: %v", err)
	}

	svc := service.NewClinicService(repo)
	httpHandler := handler.NewClinicHTTPHandler(svc, tm)

	apiV1 := router.Group("/api/v1")
	httpHandler.RegisterRoutes(apiV1)

	return router, tm, svc, repo
}

func TestHTTP_ClinicManagement(t *testing.T) {
	router, tm, _, _ := setupTestRouter(t)

	superAdminToken, _, _, _ := tm.GenerateAccessToken(uuid.New().String(), "admin@panchakarma.org", "super_admin", "")
	patientToken, _, _, _ := tm.GenerateAccessToken(uuid.New().String(), "patient@example.com", "patient", "")

	t.Run("CreateClinic_SuperAdmin_201Created", func(t *testing.T) {
		body := map[string]any{
			"name":                "Dhanwanthari Ayur Bhavan",
			"registration_number": "AYUR-KL-2026-888",
			"address_street":      "Vaidya Nagar",
			"address_city":        "Thrissur",
			"address_state":       "Kerala",
			"address_postal_code": "680001",
			"contact_phone":       "+914872345678",
			"contact_email":       "contact@dhanwanthari.in",
			"timezone":            "Asia/Kolkata",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/clinics", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("CreateClinic_InvalidTimezone_400BadRequest", func(t *testing.T) {
		body := map[string]any{
			"name":                "Bogus Zone Clinic",
			"registration_number": "AYUR-ERR-001",
			"address_street":      "Street",
			"address_city":        "City",
			"timezone":            "Mars/Olympus_Mons", // Invalid IANA time zone
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/clinics", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for invalid time zone, got %d", w.Code)
		}
	})

	t.Run("CreateClinic_Patient_403Forbidden", func(t *testing.T) {
		body := map[string]any{
			"name":                "Unauthorized Clinic",
			"registration_number": "AYUR-UNAUTH-001",
			"timezone":            "Asia/Kolkata",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/clinics", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Authorization", "Bearer "+patientToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for patient, got %d", w.Code)
		}
	})
}

func TestHTTP_CrossClinicIsolation(t *testing.T) {
	router, tm, _, repo := setupTestRouter(t)

	clinicA_ID := uuid.New().String()
	clinicB_ID := uuid.New().String()

	_ = repo.CreateClinic(context.Background(), &domain.Clinic{ID: clinicA_ID, Name: "Clinic Alpha", Timezone: "Asia/Kolkata", IsActive: true})
	_ = repo.CreateClinic(context.Background(), &domain.Clinic{ID: clinicB_ID, Name: "Clinic Beta", Timezone: "Asia/Kolkata", IsActive: true})

	adminA_Token, _, _, _ := tm.GenerateAccessToken(uuid.New().String(), "adminA@alpha.com", "clinic_admin", clinicA_ID)
	adminB_Token, _, _, _ := tm.GenerateAccessToken(uuid.New().String(), "adminB@beta.com", "clinic_admin", clinicB_ID)

	t.Run("AdminA_CannotUpdate_ClinicB", func(t *testing.T) {
		body := map[string]any{"name": "Attempted Cross-Clinic Renaming"}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/clinics/"+clinicB_ID, bytes.NewBuffer(jsonBytes))
		req.Header.Set("Authorization", "Bearer "+adminA_Token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden on cross-clinic modification, got %d", w.Code)
		}
	})

	t.Run("AdminB_CanUpdate_OwnClinicB", func(t *testing.T) {
		body := map[string]any{"name": "Authorized Beta Update"}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/clinics/"+clinicB_ID, bytes.NewBuffer(jsonBytes))
		req.Header.Set("Authorization", "Bearer "+adminB_Token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for own clinic update, got %d", w.Code)
		}
	})
}

func TestHTTP_RoomsAndDoctorsManagement(t *testing.T) {
	router, tm, _, repo := setupTestRouter(t)

	clinicID := uuid.New().String()
	_ = repo.CreateClinic(context.Background(), &domain.Clinic{ID: clinicID, Name: "Full Suite Clinic", Timezone: "Asia/Kolkata", IsActive: true})
	adminToken, _, _, _ := tm.GenerateAccessToken(uuid.New().String(), "admin@clinic.com", "clinic_admin", clinicID)

	t.Run("CreateTreatmentRoom", func(t *testing.T) {
		body := map[string]any{
			"name":                      "Swedana Chamber 1",
			"room_type":                 "herbal_steam",
			"capacity":                  2,
			"sanitation_buffer_minutes": 15,
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/clinics/"+clinicID+"/rooms", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for room, got %d", w.Code)
		}
	})

	t.Run("CreateDoctorProfile", func(t *testing.T) {
		body := map[string]any{
			"user_id":              uuid.New().String(),
			"qualifications":       "BAMS, MD Panchakarma",
			"specialties":          []string{"Vamana", "Virechana"},
			"experience_years":     10,
			"max_sessions_per_day": 8,
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/clinics/"+clinicID+"/doctors", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for doctor, got %d", w.Code)
		}
	})
}

func TestHTTP_PackagePublishingAndVersioning(t *testing.T) {
	router, tm, _, repo := setupTestRouter(t)

	clinicID := uuid.New().String()
	_ = repo.CreateClinic(context.Background(), &domain.Clinic{ID: clinicID, Name: "Package Clinic", Timezone: "Asia/Kolkata", IsActive: true})
	adminToken, _, _, _ := tm.GenerateAccessToken(uuid.New().String(), "admin@clinic.com", "clinic_admin", clinicID)

	var packageID string

	t.Run("CreatePackage", func(t *testing.T) {
		body := map[string]any{
			"name":                     "Basti Detox Protocol",
			"category":                 "detox",
			"description":              "Herbal enema detox therapy",
			"price_cents":              1800000,
			"total_sessions":           8,
			"duration_minutes":         60,
			"recommended_intervals_days": 1,
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/clinics/"+clinicID+"/packages", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", w.Code)
		}

		var res struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		packageID = res.Data.ID
	})

	t.Run("PublishWithoutSteps_Rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/clinics/"+clinicID+"/packages/"+packageID+"/publish", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request when publishing without steps, got %d", w.Code)
		}
	})

	t.Run("AddSteps_Publish_AndVerifyVersionBumpOnUpdate", func(t *testing.T) {
		// 1. Add steps
		stepsBody := []map[string]any{
			{
				"step_order":       1,
				"name":             "Abhyanga Snehana",
				"duration_minutes": 30,
				"instructions":     []string{"Warm Dhanwantharam thailam"},
			},
			{
				"step_order":       2,
				"name":             "Kashaya Basti Administration",
				"duration_minutes": 30,
				"instructions":     []string{"Maintain temperature at 37C"},
			},
		}
		stepsBytes, _ := json.Marshal(stepsBody)
		stepsReq := httptest.NewRequest(http.MethodPost, "/api/v1/clinics/"+clinicID+"/packages/"+packageID+"/steps", bytes.NewBuffer(stepsBytes))
		stepsReq.Header.Set("Authorization", "Bearer "+adminToken)
		stepsReq.Header.Set("Content-Type", "application/json")
		wSteps := httptest.NewRecorder()
		router.ServeHTTP(wSteps, stepsReq)
		if wSteps.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for steps, got %d", wSteps.Code)
		}

		// 2. Publish
		pubReq := httptest.NewRequest(http.MethodPost, "/api/v1/clinics/"+clinicID+"/packages/"+packageID+"/publish", nil)
		pubReq.Header.Set("Authorization", "Bearer "+adminToken)
		wPub := httptest.NewRecorder()
		router.ServeHTTP(wPub, pubReq)
		if wPub.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for publish, got %d", wPub.Code)
		}

		// 3. Update published package: Version MUST increment
		updateBody := map[string]any{
			"name":        "Basti Detox Protocol - Enhanced",
			"price_cents": 2100000,
		}
		updateBytes, _ := json.Marshal(updateBody)
		updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/clinics/"+clinicID+"/packages/"+packageID, bytes.NewBuffer(updateBytes))
		updateReq.Header.Set("Authorization", "Bearer "+adminToken)
		updateReq.Header.Set("Content-Type", "application/json")
		wUpdate := httptest.NewRecorder()
		router.ServeHTTP(wUpdate, updateReq)
		if wUpdate.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for update, got %d", wUpdate.Code)
		}

		var updateRes struct {
			Data struct {
				Version int `json:"version"`
			} `json:"data"`
		}
		_ = json.Unmarshal(wUpdate.Body.Bytes(), &updateRes)

		if updateRes.Data.Version != 2 {
			t.Errorf("expected version to be bumped to 2 for published package update, got %d", updateRes.Data.Version)
		}
	})
}

func TestGRPC_ClinicService(t *testing.T) {
	_, _, svc, repo := setupTestRouter(t)
	grpcHandler := handler.NewClinicGRPCHandler(svc)
	ctx := context.Background()

	clinicID := uuid.New().String()
	docID := uuid.New().String()
	roomID := uuid.New().String()
	packageID := uuid.New().String()

	_ = repo.CreateClinic(ctx, &domain.Clinic{ID: clinicID, Name: "gRPC Clinic", Timezone: "Asia/Kolkata", IsActive: true})
	_ = repo.CreateDoctor(ctx, &domain.Doctor{ID: docID, UserID: uuid.New().String(), ClinicID: clinicID, MaxSessionsPerDay: 8, IsActive: true})
	_ = repo.CreateRoom(ctx, &domain.TreatmentRoom{ID: roomID, ClinicID: clinicID, Name: "Room 1", Capacity: 1, SanitationBufferMinutes: 20, IsActive: true})

	_ = repo.CreatePackage(ctx, &domain.TherapyPackage{
		ID:              packageID,
		ClinicID:        clinicID,
		Name:            "Abhyanga Complete",
		DurationMinutes: 60,
		TotalSessions:   1,
		PriceCents:      1500000,
		Version:         1,
		IsPublished:     true,
		IsActive:        true,
	})

	t.Run("GetPackageSnapshot", func(t *testing.T) {
		res, err := grpcHandler.GetPackageSnapshot(ctx, &clinicv1.GetPackageSnapshotRequest{
			PackageId: packageID,
		})
		if err != nil {
			t.Fatalf("gRPC error: %v", err)
		}
		if res.PackageId != packageID || res.ClinicId != clinicID || res.TotalDurationMinutes != 60 {
			t.Errorf("snapshot mismatch: %+v", res)
		}
	})

	t.Run("ValidateSlotAndRoom_InactiveDoctorOrRoom", func(t *testing.T) {
		res, err := grpcHandler.ValidateSlotAndRoom(ctx, &clinicv1.ValidateSlotAndRoomRequest{
			ClinicId:      clinicID,
			DoctorId:      "nonexistent-doc",
			RoomId:        roomID,
			StartTimeUnix: time.Now().Unix(),
			EndTimeUnix:   time.Now().Add(1 * time.Hour).Unix(),
		})
		if err != nil {
			t.Fatalf("unexpected gRPC error: %v", err)
		}
		if res.IsValid {
			t.Errorf("expected invalid result for nonexistent doctor")
		}
	})
}
