package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
)

// Request payloads

type CreateClinicRequest struct {
	Name               string          `json:"name"`
	RegistrationNumber string          `json:"registration_number"`
	AddressStreet      string          `json:"address_street"`
	AddressCity        string          `json:"address_city"`
	AddressState       string          `json:"address_state"`
	AddressPostalCode  string          `json:"address_postal_code"`
	ContactPhone       string          `json:"contact_phone"`
	ContactEmail       string          `json:"contact_email"`
	Description        string          `json:"description,omitempty"`
	SupportedLanguages []string        `json:"supported_languages"`
	OperatingHours     json.RawMessage `json:"operating_hours"`
	Timezone           string          `json:"timezone"` // IANA time zone e.g. "Asia/Kolkata"
}

type UpdateClinicRequest struct {
	Name               string          `json:"name"`
	AddressStreet      string          `json:"address_street"`
	AddressCity        string          `json:"address_city"`
	AddressState       string          `json:"address_state"`
	AddressPostalCode  string          `json:"address_postal_code"`
	ContactPhone       string          `json:"contact_phone"`
	ContactEmail       string          `json:"contact_email"`
	Description        string          `json:"description,omitempty"`
	SupportedLanguages []string        `json:"supported_languages"`
	OperatingHours     json.RawMessage `json:"operating_hours"`
	Timezone           string          `json:"timezone"`
}

type CreateRoomRequest struct {
	Name                    string `json:"name"`
	RoomType                string `json:"room_type"`
	Capacity                int    `json:"capacity"`
	SanitationBufferMinutes int    `json:"sanitation_buffer_minutes"`
}

type AddEquipmentRequest struct {
	EquipmentType string `json:"equipment_type"`
	SerialNumber  string `json:"serial_number,omitempty"`
	IsOperational bool   `json:"is_operational"`
}

type CreateDoctorRequest struct {
	UserID            string   `json:"user_id"`
	Qualifications    string   `json:"qualifications"`
	Specialties       []string `json:"specialties"`
	ExperienceYears   int      `json:"experience_years"`
	MaxSessionsPerDay int      `json:"max_sessions_per_day"`
}

type CreateTherapistRequest struct {
	UserID          string   `json:"user_id"`
	Specializations []string `json:"specializations"`
	ExperienceYears int      `json:"experience_years"`
	Gender          string   `json:"gender,omitempty"`
}

type CreateScheduleRequest struct {
	DayOfWeek           int    `json:"day_of_week"` // 0=Sunday..6=Saturday
	StartTime           string `json:"start_time"`  // "09:00"
	EndTime             string `json:"end_time"`    // "17:00"
	SlotDurationMinutes int    `json:"slot_duration_minutes"`
}

type CreateLeaveRequest struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Reason    string    `json:"reason"`
}

type CreatePackageRequest struct {
	Name                     string `json:"name"`
	Category                 string `json:"category"`
	Description              string `json:"description"`
	PriceCents               int    `json:"price_cents"`
	TotalSessions            int    `json:"total_sessions"`
	DurationMinutes          int    `json:"duration_minutes"`
	RecommendedIntervalsDays int    `json:"recommended_intervals_days"`
}

type UpdatePackageRequest struct {
	Name                     string `json:"name"`
	Category                 string `json:"category"`
	Description              string `json:"description"`
	PriceCents               int    `json:"price_cents"`
	TotalSessions            int    `json:"total_sessions"`
	DurationMinutes          int    `json:"duration_minutes"`
	RecommendedIntervalsDays int    `json:"recommended_intervals_days"`
}

type PackageStepRequest struct {
	StepOrder         int      `json:"step_order"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	DurationMinutes   int      `json:"duration_minutes"`
	Instructions      []string `json:"instructions"`
	RequiredEquipment []string `json:"required_equipment"`
}

type CareChecklistRequest struct {
	ChecklistType string          `json:"checklist_type"` // 'PRE_PROCEDURE', 'POST_PROCEDURE'
	Items         json.RawMessage `json:"items"`
}

type DoctorAvailability struct {
	DoctorID           string   `json:"doctor_id"`
	Date               string   `json:"date"`
	MaxSessionsPerDay  int      `json:"max_sessions_per_day"`
	IsOnLeave          bool     `json:"is_on_leave"`
	AvailableTimeSlots []string `json:"available_time_slots"`
}

type SlotValidationResult struct {
	IsValid                 bool   `json:"is_valid"`
	RejectionReason         string `json:"rejection_reason,omitempty"`
	SanitationBufferMinutes int    `json:"sanitation_buffer_minutes"`
}

type ClinicService interface {
	// Clinic management
	CreateClinic(ctx context.Context, req *CreateClinicRequest, actorRole string) (*domain.Clinic, error)
	GetClinic(ctx context.Context, id string) (*domain.Clinic, error)
	ListClinics(ctx context.Context, city string, activeOnly bool) ([]*domain.Clinic, error)
	UpdateClinic(ctx context.Context, id string, req *UpdateClinicRequest, actorRole, actorClinicID string) (*domain.Clinic, error)
	DeactivateClinic(ctx context.Context, id string, actorRole string) error

	// Rooms & Equipment
	CreateRoom(ctx context.Context, clinicID string, req *CreateRoomRequest, actorRole, actorClinicID string) (*domain.TreatmentRoom, error)
	GetRoom(ctx context.Context, roomID string) (*domain.TreatmentRoom, error)
	ListRooms(ctx context.Context, clinicID string) ([]*domain.TreatmentRoom, error)
	AddRoomEquipment(ctx context.Context, clinicID, roomID string, req *AddEquipmentRequest, actorRole, actorClinicID string) (*domain.RoomEquipment, error)

	// Staff
	CreateDoctorProfile(ctx context.Context, clinicID string, req *CreateDoctorRequest, actorRole, actorClinicID string) (*domain.Doctor, error)
	GetDoctor(ctx context.Context, doctorID string) (*domain.Doctor, error)
	ListDoctors(ctx context.Context, clinicID string) ([]*domain.Doctor, error)
	CreateTherapistProfile(ctx context.Context, clinicID string, req *CreateTherapistRequest, actorRole, actorClinicID string) (*domain.Therapist, error)
	GetTherapist(ctx context.Context, therapistID string) (*domain.Therapist, error)
	ListTherapists(ctx context.Context, clinicID string) ([]*domain.Therapist, error)

	// Schedules & Leaves
	CreateSchedule(ctx context.Context, clinicID, doctorID string, req *CreateScheduleRequest, actorRole, actorClinicID string) (*domain.DoctorSchedule, error)
	ListDoctorSchedules(ctx context.Context, doctorID string) ([]*domain.DoctorSchedule, error)
	DeleteSchedule(ctx context.Context, clinicID, scheduleID string, actorRole, actorClinicID string) error
	CreateDoctorLeave(ctx context.Context, clinicID, doctorID string, req *CreateLeaveRequest, actorRole, actorClinicID string) (*domain.DoctorLeave, error)
	ListDoctorLeaves(ctx context.Context, doctorID string) ([]*domain.DoctorLeave, error)

	// Therapy Packages
	CreatePackage(ctx context.Context, clinicID string, req *CreatePackageRequest, actorRole, actorClinicID string) (*domain.TherapyPackage, error)
	GetPackage(ctx context.Context, packageID string) (*domain.TherapyPackage, error)
	ListPackages(ctx context.Context, clinicID string, publishedOnly bool) ([]*domain.TherapyPackage, error)
	UpdatePackage(ctx context.Context, clinicID, packageID string, req *UpdatePackageRequest, actorRole, actorClinicID string) (*domain.TherapyPackage, error)
	PublishPackage(ctx context.Context, clinicID, packageID string, actorRole, actorClinicID string) error
	DeactivatePackage(ctx context.Context, clinicID, packageID string, actorRole, actorClinicID string) error
	SavePackageSteps(ctx context.Context, clinicID, packageID string, steps []PackageStepRequest, actorRole, actorClinicID string) error
	SaveCareChecklist(ctx context.Context, clinicID, packageID string, req *CareChecklistRequest, actorRole, actorClinicID string) error

	// Availability & Internal gRPC checks
	GetDoctorAvailability(ctx context.Context, doctorID string, dateStr string) (*DoctorAvailability, error)
	ValidateSlotAndRoom(ctx context.Context, clinicID, doctorID, roomID string, startUnix, endUnix int64, packageID string) (*SlotValidationResult, error)
	GetPackageSnapshot(ctx context.Context, packageID string) (*domain.TherapyPackage, error)
}

type clinicService struct {
	repo repository.ClinicRepository
}

func NewClinicService(repo repository.ClinicRepository) ClinicService {
	return &clinicService{repo: repo}
}

// AuthorizeClinicAccess verifies that the actor has permission to manage resources for the specified clinic.
func (s *clinicService) AuthorizeClinicAccess(actorRole, actorClinicID, targetClinicID string) error {
	if actorRole == "super_admin" {
		return nil
	}
	if actorRole == "clinic_admin" || actorRole == "doctor" || actorRole == "therapist" {
		if actorClinicID != targetClinicID {
			return errors.Forbidden("Cross-clinic access denied: you do not have permission for this clinic", nil)
		}
		return nil
	}
	return errors.Forbidden("Insufficient permissions for clinic operations", nil)
}

// ValidateTimezone validates that the provided string is a valid IANA time zone.
func ValidateTimezone(tz string) error {
	if tz == "" {
		return fmt.Errorf("timezone cannot be empty")
	}
	_, err := time.LoadLocation(tz)
	if err != nil {
		return fmt.Errorf("invalid IANA time zone: %w", err)
	}
	return nil
}

// --- Clinic Management ---

func (s *clinicService) CreateClinic(ctx context.Context, req *CreateClinicRequest, actorRole string) (*domain.Clinic, error) {
	if actorRole != "super_admin" {
		return nil, errors.Forbidden("Only platform super administrators can create clinics", nil)
	}

	if req.Name == "" || req.RegistrationNumber == "" || req.AddressStreet == "" || req.AddressCity == "" {
		return nil, errors.BadRequest("Clinic name, registration number, and address are required", nil)
	}

	if err := ValidateTimezone(req.Timezone); err != nil {
		return nil, errors.BadRequest(err.Error(), err)
	}

	existing, err := s.repo.GetClinicByRegistrationNumber(ctx, req.RegistrationNumber)
	if err != nil {
		return nil, errors.Internal("Database lookup failed", err)
	}
	if existing != nil {
		return nil, errors.Conflict("CLINIC_EXISTS", "A clinic with this registration number already exists", nil)
	}

	if len(req.OperatingHours) == 0 {
		req.OperatingHours = json.RawMessage(`{"monday":[{"open":"08:00","close":"18:00"}],"tuesday":[{"open":"08:00","close":"18:00"}],"wednesday":[{"open":"08:00","close":"18:00"}],"thursday":[{"open":"08:00","close":"18:00"}],"friday":[{"open":"08:00","close":"18:00"}],"saturday":[{"open":"08:00","close":"14:00"}]}`)
	}
	if len(req.SupportedLanguages) == 0 {
		req.SupportedLanguages = []string{"English"}
	}

	now := time.Now().UTC()
	clinic := &domain.Clinic{
		ID:                 uuid.New().String(),
		Name:               req.Name,
		RegistrationNumber: req.RegistrationNumber,
		AddressStreet:      req.AddressStreet,
		AddressCity:        req.AddressCity,
		AddressState:       req.AddressState,
		AddressPostalCode:  req.AddressPostalCode,
		ContactPhone:       req.ContactPhone,
		ContactEmail:       req.ContactEmail,
		Description:        req.Description,
		SupportedLanguages: req.SupportedLanguages,
		OperatingHours:     req.OperatingHours,
		Timezone:           req.Timezone,
		IsActive:           true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.repo.CreateClinic(ctx, clinic); err != nil {
		return nil, errors.Internal("Failed to persist clinic", err)
	}

	return clinic, nil
}

func (s *clinicService) GetClinic(ctx context.Context, id string) (*domain.Clinic, error) {
	c, err := s.repo.GetClinicByID(ctx, id)
	if err != nil {
		return nil, errors.Internal("Database lookup failed", err)
	}
	if c == nil {
		return nil, errors.NotFound("Clinic not found", nil)
	}
	return c, nil
}

func (s *clinicService) ListClinics(ctx context.Context, city string, activeOnly bool) ([]*domain.Clinic, error) {
	return s.repo.ListClinics(ctx, city, activeOnly)
}

func (s *clinicService) UpdateClinic(ctx context.Context, id string, req *UpdateClinicRequest, actorRole, actorClinicID string) (*domain.Clinic, error) {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, id); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetClinicByID(ctx, id)
	if err != nil {
		return nil, errors.Internal("Database lookup failed", err)
	}
	if existing == nil {
		return nil, errors.NotFound("Clinic not found", nil)
	}

	if req.Timezone != "" {
		if err := ValidateTimezone(req.Timezone); err != nil {
			return nil, errors.BadRequest(err.Error(), err)
		}
		existing.Timezone = req.Timezone
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.AddressStreet != "" {
		existing.AddressStreet = req.AddressStreet
	}
	if req.AddressCity != "" {
		existing.AddressCity = req.AddressCity
	}
	if req.AddressState != "" {
		existing.AddressState = req.AddressState
	}
	if req.AddressPostalCode != "" {
		existing.AddressPostalCode = req.AddressPostalCode
	}
	if req.ContactPhone != "" {
		existing.ContactPhone = req.ContactPhone
	}
	if req.ContactEmail != "" {
		existing.ContactEmail = req.ContactEmail
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if len(req.SupportedLanguages) > 0 {
		existing.SupportedLanguages = req.SupportedLanguages
	}
	if len(req.OperatingHours) > 0 {
		existing.OperatingHours = req.OperatingHours
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateClinic(ctx, existing); err != nil {
		return nil, errors.Internal("Failed to update clinic", err)
	}

	return existing, nil
}

func (s *clinicService) DeactivateClinic(ctx context.Context, id string, actorRole string) error {
	if actorRole != "super_admin" {
		return errors.Forbidden("Only platform super administrators can deactivate clinics", nil)
	}
	return s.repo.DeactivateClinic(ctx, id)
}

// --- Treatment Rooms & Equipment ---

func (s *clinicService) CreateRoom(ctx context.Context, clinicID string, req *CreateRoomRequest, actorRole, actorClinicID string) (*domain.TreatmentRoom, error) {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, errors.BadRequest("Room name is required", nil)
	}
	if req.Capacity <= 0 {
		req.Capacity = 1
	}
	if req.SanitationBufferMinutes <= 0 {
		req.SanitationBufferMinutes = 15
	}
	if req.RoomType == "" {
		req.RoomType = "standard_panchakarma"
	}

	room := &domain.TreatmentRoom{
		ID:                      uuid.New().String(),
		ClinicID:                clinicID,
		Name:                    req.Name,
		RoomType:                req.RoomType,
		Capacity:                req.Capacity,
		SanitationBufferMinutes: req.SanitationBufferMinutes,
		IsActive:                true,
		CreatedAt:               time.Now().UTC(),
	}

	if err := s.repo.CreateRoom(ctx, room); err != nil {
		return nil, errors.Internal("Failed to create treatment room", err)
	}

	return room, nil
}

func (s *clinicService) GetRoom(ctx context.Context, roomID string) (*domain.TreatmentRoom, error) {
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, errors.Internal("Database lookup failed", err)
	}
	if room == nil {
		return nil, errors.NotFound("Treatment room not found", nil)
	}
	return room, nil
}

func (s *clinicService) ListRooms(ctx context.Context, clinicID string) ([]*domain.TreatmentRoom, error) {
	return s.repo.ListRooms(ctx, clinicID)
}

func (s *clinicService) AddRoomEquipment(ctx context.Context, clinicID, roomID string, req *AddEquipmentRequest, actorRole, actorClinicID string) (*domain.RoomEquipment, error) {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return nil, err
	}

	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil || room == nil || room.ClinicID != clinicID {
		return nil, errors.NotFound("Treatment room not found in this clinic", nil)
	}

	equip := &domain.RoomEquipment{
		ID:            uuid.New().String(),
		RoomID:        roomID,
		EquipmentType: req.EquipmentType,
		SerialNumber:  req.SerialNumber,
		IsOperational: req.IsOperational,
	}

	if err := s.repo.AddRoomEquipment(ctx, equip); err != nil {
		return nil, errors.Internal("Failed to add room equipment", err)
	}

	return equip, nil
}

// --- Doctors & Therapists ---

func (s *clinicService) CreateDoctorProfile(ctx context.Context, clinicID string, req *CreateDoctorRequest, actorRole, actorClinicID string) (*domain.Doctor, error) {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return nil, err
	}

	if req.UserID == "" || req.Qualifications == "" {
		return nil, errors.BadRequest("User ID and qualifications are required", nil)
	}
	if req.MaxSessionsPerDay <= 0 || req.MaxSessionsPerDay > 16 {
		req.MaxSessionsPerDay = 8
	}

	existing, _ := s.repo.GetDoctorByUserID(ctx, req.UserID)
	if existing != nil {
		return nil, errors.Conflict("DOCTOR_EXISTS", "A doctor profile already exists for this user", nil)
	}

	now := time.Now().UTC()
	doc := &domain.Doctor{
		ID:                uuid.New().String(),
		UserID:            req.UserID,
		ClinicID:          clinicID,
		Qualifications:    req.Qualifications,
		Specialties:       req.Specialties,
		ExperienceYears:   req.ExperienceYears,
		MaxSessionsPerDay: req.MaxSessionsPerDay,
		IsActive:          true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.CreateDoctor(ctx, doc); err != nil {
		return nil, errors.Internal("Failed to create doctor profile", err)
	}

	return doc, nil
}

func (s *clinicService) GetDoctor(ctx context.Context, doctorID string) (*domain.Doctor, error) {
	doc, err := s.repo.GetDoctorByID(ctx, doctorID)
	if err != nil {
		return nil, errors.Internal("Database lookup failed", err)
	}
	if doc == nil {
		return nil, errors.NotFound("Doctor not found", nil)
	}
	return doc, nil
}

func (s *clinicService) ListDoctors(ctx context.Context, clinicID string) ([]*domain.Doctor, error) {
	return s.repo.ListDoctors(ctx, clinicID)
}

func (s *clinicService) CreateTherapistProfile(ctx context.Context, clinicID string, req *CreateTherapistRequest, actorRole, actorClinicID string) (*domain.Therapist, error) {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return nil, err
	}

	if req.UserID == "" {
		return nil, errors.BadRequest("User ID is required", nil)
	}

	existing, _ := s.repo.GetTherapistByUserID(ctx, req.UserID)
	if existing != nil {
		return nil, errors.Conflict("THERAPIST_EXISTS", "A therapist profile already exists for this user", nil)
	}

	now := time.Now().UTC()
	therapist := &domain.Therapist{
		ID:              uuid.New().String(),
		UserID:          req.UserID,
		ClinicID:        clinicID,
		Specializations: req.Specializations,
		ExperienceYears: req.ExperienceYears,
		Gender:          req.Gender,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateTherapist(ctx, therapist); err != nil {
		return nil, errors.Internal("Failed to create therapist profile", err)
	}

	return therapist, nil
}

func (s *clinicService) GetTherapist(ctx context.Context, therapistID string) (*domain.Therapist, error) {
	t, err := s.repo.GetTherapistByID(ctx, therapistID)
	if err != nil {
		return nil, errors.Internal("Database lookup failed", err)
	}
	if t == nil {
		return nil, errors.NotFound("Therapist not found", nil)
	}
	return t, nil
}

func (s *clinicService) ListTherapists(ctx context.Context, clinicID string) ([]*domain.Therapist, error) {
	return s.repo.ListTherapists(ctx, clinicID)
}

// --- Schedules & Leaves ---

func parseTimeToMinutes(tStr string) (int, error) {
	tStr = strings.TrimSpace(tStr)
	if strings.Count(tStr, ":") >= 2 {
		parts := strings.Split(tStr, ":")
		tStr = parts[0] + ":" + parts[1]
	}
	parsed, err := time.Parse("15:04", tStr)
	if err != nil {
		return 0, fmt.Errorf("invalid time format, expected HH:MM (e.g. 09:00): %w", err)
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func (s *clinicService) CreateSchedule(ctx context.Context, clinicID, doctorID string, req *CreateScheduleRequest, actorRole, actorClinicID string) (*domain.DoctorSchedule, error) {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return nil, err
	}

	doc, err := s.repo.GetDoctorByID(ctx, doctorID)
	if err != nil || doc == nil || doc.ClinicID != clinicID {
		return nil, errors.NotFound("Doctor not found in this clinic", nil)
	}

	if req.DayOfWeek < 0 || req.DayOfWeek > 6 {
		return nil, errors.BadRequest("Day of week must be between 0 (Sunday) and 6 (Saturday)", nil)
	}

	startMin, err := parseTimeToMinutes(req.StartTime)
	if err != nil {
		return nil, errors.BadRequest(err.Error(), err)
	}
	endMin, err := parseTimeToMinutes(req.EndTime)
	if err != nil {
		return nil, errors.BadRequest(err.Error(), err)
	}
	if endMin <= startMin {
		return nil, errors.BadRequest("End time must be after start time", nil)
	}

	if req.SlotDurationMinutes <= 0 {
		req.SlotDurationMinutes = 60
	}

	// Schedule overlap check
	existingSchedules, err := s.repo.ListDoctorSchedulesForDay(ctx, doctorID, req.DayOfWeek)
	if err != nil {
		return nil, errors.Internal("Failed to check existing schedules", err)
	}

	for _, exist := range existingSchedules {
		eStart, _ := parseTimeToMinutes(exist.StartTime)
		eEnd, _ := parseTimeToMinutes(exist.EndTime)

		// Overlap condition: NOT (newEnd <= existStart OR newStart >= existEnd)
		if !(endMin <= eStart || startMin >= eEnd) {
			return nil, errors.BadRequest(fmt.Sprintf("Schedule [%s-%s] overlaps with an existing shift [%s-%s] on day %d",
				req.StartTime, req.EndTime, exist.StartTime, exist.EndTime, req.DayOfWeek), nil)
		}
	}

	schedule := &domain.DoctorSchedule{
		ID:                  uuid.New().String(),
		DoctorID:            doctorID,
		DayOfWeek:           req.DayOfWeek,
		StartTime:           req.StartTime,
		EndTime:             req.EndTime,
		SlotDurationMinutes: req.SlotDurationMinutes,
		IsActive:            true,
	}

	if err := s.repo.CreateSchedule(ctx, schedule); err != nil {
		return nil, errors.Internal("Failed to insert schedule", err)
	}

	return schedule, nil
}

func (s *clinicService) ListDoctorSchedules(ctx context.Context, doctorID string) ([]*domain.DoctorSchedule, error) {
	return s.repo.ListDoctorSchedules(ctx, doctorID)
}

func (s *clinicService) DeleteSchedule(ctx context.Context, clinicID, scheduleID string, actorRole, actorClinicID string) error {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return err
	}
	return s.repo.DeleteSchedule(ctx, scheduleID)
}

func (s *clinicService) CreateDoctorLeave(ctx context.Context, clinicID, doctorID string, req *CreateLeaveRequest, actorRole, actorClinicID string) (*domain.DoctorLeave, error) {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return nil, err
	}

	doc, err := s.repo.GetDoctorByID(ctx, doctorID)
	if err != nil || doc == nil || doc.ClinicID != clinicID {
		return nil, errors.NotFound("Doctor not found in this clinic", nil)
	}

	if !req.EndTime.After(req.StartTime) {
		return nil, errors.BadRequest("Leave end time must be after start time", nil)
	}
	if req.Reason == "" {
		return nil, errors.BadRequest("Leave reason is required", nil)
	}

	// Check overlapping approved leaves
	overlapping, err := s.repo.GetApprovedLeavesInRange(ctx, doctorID, req.StartTime, req.EndTime)
	if err != nil {
		return nil, errors.Internal("Failed to check existing leaves", err)
	}
	if len(overlapping) > 0 {
		return nil, errors.Conflict("LEAVE_OVERLAP", "Requested leave range overlaps with an existing approved leave", nil)
	}

	leave := &domain.DoctorLeave{
		ID:        uuid.New().String(),
		DoctorID:  doctorID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Reason:    req.Reason,
		Status:    "APPROVED",
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateLeave(ctx, leave); err != nil {
		return nil, errors.Internal("Failed to record doctor leave", err)
	}

	return leave, nil
}

func (s *clinicService) ListDoctorLeaves(ctx context.Context, doctorID string) ([]*domain.DoctorLeave, error) {
	return s.repo.ListDoctorLeaves(ctx, doctorID)
}

// --- Therapy Packages ---

func (s *clinicService) CreatePackage(ctx context.Context, clinicID string, req *CreatePackageRequest, actorRole, actorClinicID string) (*domain.TherapyPackage, error) {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return nil, err
	}

	if req.Name == "" || req.Category == "" {
		return nil, errors.BadRequest("Package name and category are required", nil)
	}
	if req.PriceCents < 0 {
		return nil, errors.BadRequest("Price cannot be negative", nil)
	}
	if req.TotalSessions <= 0 {
		req.TotalSessions = 1
	}
	if req.DurationMinutes <= 0 {
		return nil, errors.BadRequest("Duration must be greater than 0", nil)
	}
	if req.RecommendedIntervalsDays <= 0 {
		req.RecommendedIntervalsDays = 1
	}

	now := time.Now().UTC()
	pkg := &domain.TherapyPackage{
		ID:                       uuid.New().String(),
		ClinicID:                 clinicID,
		Name:                     req.Name,
		Category:                 req.Category,
		Description:              req.Description,
		PriceCents:               req.PriceCents,
		TotalSessions:            req.TotalSessions,
		DurationMinutes:          req.DurationMinutes,
		RecommendedIntervalsDays: req.RecommendedIntervalsDays,
		Version:                  1,
		IsPublished:              false,
		IsActive:                 true,
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	if err := s.repo.CreatePackage(ctx, pkg); err != nil {
		return nil, errors.Internal("Failed to create therapy package", err)
	}

	return pkg, nil
}

func (s *clinicService) GetPackage(ctx context.Context, packageID string) (*domain.TherapyPackage, error) {
	pkg, err := s.repo.GetPackageByID(ctx, packageID)
	if err != nil {
		return nil, errors.Internal("Database lookup failed", err)
	}
	if pkg == nil {
		return nil, errors.NotFound("Therapy package not found", nil)
	}
	return pkg, nil
}

func (s *clinicService) ListPackages(ctx context.Context, clinicID string, publishedOnly bool) ([]*domain.TherapyPackage, error) {
	return s.repo.ListPackages(ctx, clinicID, publishedOnly)
}

func (s *clinicService) UpdatePackage(ctx context.Context, clinicID, packageID string, req *UpdatePackageRequest, actorRole, actorClinicID string) (*domain.TherapyPackage, error) {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetPackageByID(ctx, packageID)
	if err != nil || existing == nil || existing.ClinicID != clinicID {
		return nil, errors.NotFound("Therapy package not found in this clinic", nil)
	}

	// Package Versioning: If already published, bump version to protect historical bookings
	if existing.IsPublished {
		existing.Version++
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Category != "" {
		existing.Category = req.Category
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.PriceCents >= 0 {
		existing.PriceCents = req.PriceCents
	}
	if req.TotalSessions > 0 {
		existing.TotalSessions = req.TotalSessions
	}
	if req.DurationMinutes > 0 {
		existing.DurationMinutes = req.DurationMinutes
	}
	if req.RecommendedIntervalsDays > 0 {
		existing.RecommendedIntervalsDays = req.RecommendedIntervalsDays
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdatePackage(ctx, existing); err != nil {
		return nil, errors.Internal("Failed to update therapy package", err)
	}

	return existing, nil
}

func (s *clinicService) PublishPackage(ctx context.Context, clinicID, packageID string, actorRole, actorClinicID string) error {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return err
	}

	pkg, err := s.repo.GetPackageByID(ctx, packageID)
	if err != nil || pkg == nil || pkg.ClinicID != clinicID {
		return errors.NotFound("Therapy package not found in this clinic", nil)
	}

	steps, _ := s.repo.GetPackageSteps(ctx, packageID)
	if len(steps) == 0 {
		return errors.BadRequest("Cannot publish therapy package without configured steps", nil)
	}

	return s.repo.PublishPackage(ctx, packageID)
}

func (s *clinicService) DeactivatePackage(ctx context.Context, clinicID, packageID string, actorRole, actorClinicID string) error {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return err
	}
	return s.repo.DeactivatePackage(ctx, packageID)
}

func (s *clinicService) SavePackageSteps(ctx context.Context, clinicID, packageID string, stepsReq []PackageStepRequest, actorRole, actorClinicID string) error {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return err
	}

	pkg, err := s.repo.GetPackageByID(ctx, packageID)
	if err != nil || pkg == nil || pkg.ClinicID != clinicID {
		return errors.NotFound("Therapy package not found in this clinic", nil)
	}

	if len(stepsReq) == 0 {
		return errors.BadRequest("At least one step is required", nil)
	}

	steps := make([]domain.PackageStep, len(stepsReq))
	for i, sr := range stepsReq {
		if sr.StepOrder <= 0 {
			sr.StepOrder = i + 1
		}
		if sr.DurationMinutes <= 0 {
			return errors.BadRequest(fmt.Sprintf("Step %d duration must be greater than 0", sr.StepOrder), nil)
		}
		steps[i] = domain.PackageStep{
			ID:                uuid.New().String(),
			PackageID:         packageID,
			StepOrder:         sr.StepOrder,
			Name:              sr.Name,
			Description:       sr.Description,
			DurationMinutes:   sr.DurationMinutes,
			Instructions:      sr.Instructions,
			RequiredEquipment: sr.RequiredEquipment,
		}
	}

	return s.repo.SavePackageSteps(ctx, packageID, steps)
}

func (s *clinicService) SaveCareChecklist(ctx context.Context, clinicID, packageID string, req *CareChecklistRequest, actorRole, actorClinicID string) error {
	if err := s.AuthorizeClinicAccess(actorRole, actorClinicID, clinicID); err != nil {
		return err
	}

	pkg, err := s.repo.GetPackageByID(ctx, packageID)
	if err != nil || pkg == nil || pkg.ClinicID != clinicID {
		return errors.NotFound("Therapy package not found in this clinic", nil)
	}

	if req.ChecklistType != "PRE_PROCEDURE" && req.ChecklistType != "POST_PROCEDURE" {
		return errors.BadRequest("Checklist type must be PRE_PROCEDURE or POST_PROCEDURE", nil)
	}

	cl := &domain.CareChecklist{
		ID:            uuid.New().String(),
		PackageID:     packageID,
		ChecklistType: req.ChecklistType,
		Version:       pkg.Version,
		Items:         req.Items,
		IsActive:      true,
		CreatedAt:     time.Now().UTC(),
	}

	return s.repo.SaveCareChecklist(ctx, cl)
}

// --- Availability & Internal Validation ---

func (s *clinicService) GetDoctorAvailability(ctx context.Context, doctorID string, dateStr string) (*DoctorAvailability, error) {
	doc, err := s.repo.GetDoctorByID(ctx, doctorID)
	if err != nil || doc == nil {
		return nil, errors.NotFound("Doctor not found", nil)
	}

	clinic, err := s.repo.GetClinicByID(ctx, doc.ClinicID)
	if err != nil || clinic == nil {
		return nil, errors.NotFound("Associated clinic not found", nil)
	}

	loc, err := time.LoadLocation(clinic.Timezone)
	if err != nil {
		loc = time.UTC
	}

	parsedDate, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return nil, errors.BadRequest("Invalid date format, expected YYYY-MM-DD", err)
	}

	dayOfWeek := int(parsedDate.Weekday())

	// Check if on leave
	dayStart := parsedDate
	dayEnd := parsedDate.Add(24 * time.Hour)
	leaves, err := s.repo.GetApprovedLeavesInRange(ctx, doctorID, dayStart, dayEnd)
	if err != nil {
		return nil, errors.Internal("Failed to check doctor leaves", err)
	}

	if len(leaves) > 0 {
		return &DoctorAvailability{
			DoctorID:           doctorID,
			Date:               dateStr,
			MaxSessionsPerDay:  doc.MaxSessionsPerDay,
			IsOnLeave:          true,
			AvailableTimeSlots: []string{},
		}, nil
	}

	// Query doctor's weekly schedules for this day
	schedules, err := s.repo.ListDoctorSchedulesForDay(ctx, doctorID, dayOfWeek)
	if err != nil {
		return nil, errors.Internal("Failed to query doctor schedules", err)
	}

	var slots []string
	sessionCount := 0

	for _, sch := range schedules {
		startMin, _ := parseTimeToMinutes(sch.StartTime)
		endMin, _ := parseTimeToMinutes(sch.EndTime)
		step := sch.SlotDurationMinutes
		if step <= 0 {
			step = 60
		}

		for m := startMin; m+step <= endMin; m += step {
			if sessionCount >= doc.MaxSessionsPerDay {
				break
			}
			hour := m / 60
			minute := m % 60
			slots = append(slots, fmt.Sprintf("%02d:%02d", hour, minute))
			sessionCount++
		}
	}

	return &DoctorAvailability{
		DoctorID:           doctorID,
		Date:               dateStr,
		MaxSessionsPerDay:  doc.MaxSessionsPerDay,
		IsOnLeave:          false,
		AvailableTimeSlots: slots,
	}, nil
}

func (s *clinicService) ValidateSlotAndRoom(ctx context.Context, clinicID, doctorID, roomID string, startUnix, endUnix int64, packageID string) (*SlotValidationResult, error) {
	doc, err := s.repo.GetDoctorByID(ctx, doctorID)
	if err != nil || doc == nil || doc.ClinicID != clinicID || !doc.IsActive {
		return &SlotValidationResult{
			IsValid:         false,
			RejectionReason: "DOCTOR_UNAVAILABLE_OR_INACTIVE",
		}, nil
	}

	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil || room == nil || room.ClinicID != clinicID || !room.IsActive {
		return &SlotValidationResult{
			IsValid:         false,
			RejectionReason: "ROOM_UNAVAILABLE_OR_INACTIVE",
		}, nil
	}

	startTime := time.Unix(startUnix, 0).UTC()
	endTime := time.Unix(endUnix, 0).UTC()

	// Check if doctor is on approved leave during this time window
	leaves, err := s.repo.GetApprovedLeavesInRange(ctx, doctorID, startTime, endTime)
	if err != nil || len(leaves) > 0 {
		return &SlotValidationResult{
			IsValid:         false,
			RejectionReason: "DOCTOR_ON_LEAVE",
		}, nil
	}

	// Verify time against clinic timezone and doctor schedule
	clinic, _ := s.repo.GetClinicByID(ctx, clinicID)
	loc := time.UTC
	if clinic != nil {
		if l, err := time.LoadLocation(clinic.Timezone); err == nil {
			loc = l
		}
	}

	localStart := startTime.In(loc)
	localEnd := endTime.In(loc)
	dayOfWeek := int(localStart.Weekday())

	schedules, err := s.repo.ListDoctorSchedulesForDay(ctx, doctorID, dayOfWeek)
	if err != nil || len(schedules) == 0 {
		return &SlotValidationResult{
			IsValid:         false,
			RejectionReason: "DOCTOR_NOT_SCHEDULED_ON_DAY",
		}, nil
	}

	slotStartMin := localStart.Hour()*60 + localStart.Minute()
	slotEndMin := localEnd.Hour()*60 + localEnd.Minute()

	withinSchedule := false
	for _, sch := range schedules {
		schStart, _ := parseTimeToMinutes(sch.StartTime)
		schEnd, _ := parseTimeToMinutes(sch.EndTime)
		if slotStartMin >= schStart && slotEndMin <= schEnd {
			withinSchedule = true
			break
		}
	}

	if !withinSchedule {
		return &SlotValidationResult{
			IsValid:         false,
			RejectionReason: "SLOT_OUTSIDE_DOCTOR_WORKING_HOURS",
		}, nil
	}

	return &SlotValidationResult{
		IsValid:                 true,
		SanitationBufferMinutes: room.SanitationBufferMinutes,
	}, nil
}

func (s *clinicService) GetPackageSnapshot(ctx context.Context, packageID string) (*domain.TherapyPackage, error) {
	pkg, err := s.repo.GetPackageByID(ctx, packageID)
	if err != nil {
		return nil, errors.Internal("Database lookup failed", err)
	}
	if pkg == nil {
		return nil, errors.NotFound("Therapy package not found", nil)
	}
	return pkg, nil
}
