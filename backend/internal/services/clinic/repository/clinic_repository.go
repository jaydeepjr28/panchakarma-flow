package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/domain"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
)

type ClinicRepository interface {
	// Clinics
	CreateClinic(ctx context.Context, clinic *domain.Clinic) error
	GetClinicByID(ctx context.Context, id string) (*domain.Clinic, error)
	GetClinicByRegistrationNumber(ctx context.Context, regNum string) (*domain.Clinic, error)
	ListClinics(ctx context.Context, city string, activeOnly bool) ([]*domain.Clinic, error)
	UpdateClinic(ctx context.Context, clinic *domain.Clinic) error
	DeactivateClinic(ctx context.Context, id string) error

	// Treatment Rooms & Equipment
	CreateRoom(ctx context.Context, room *domain.TreatmentRoom) error
	GetRoomByID(ctx context.Context, id string) (*domain.TreatmentRoom, error)
	ListRooms(ctx context.Context, clinicID string) ([]*domain.TreatmentRoom, error)
	UpdateRoom(ctx context.Context, room *domain.TreatmentRoom) error
	DeactivateRoom(ctx context.Context, id string) error
	AddRoomEquipment(ctx context.Context, equipment *domain.RoomEquipment) error
	ListRoomEquipment(ctx context.Context, roomID string) ([]domain.RoomEquipment, error)

	// Doctors
	CreateDoctor(ctx context.Context, doctor *domain.Doctor) error
	GetDoctorByID(ctx context.Context, id string) (*domain.Doctor, error)
	GetDoctorByUserID(ctx context.Context, userID string) (*domain.Doctor, error)
	ListDoctors(ctx context.Context, clinicID string) ([]*domain.Doctor, error)
	UpdateDoctor(ctx context.Context, doctor *domain.Doctor) error
	DeactivateDoctor(ctx context.Context, id string) error

	// Therapists
	CreateTherapist(ctx context.Context, therapist *domain.Therapist) error
	GetTherapistByID(ctx context.Context, id string) (*domain.Therapist, error)
	GetTherapistByUserID(ctx context.Context, userID string) (*domain.Therapist, error)
	ListTherapists(ctx context.Context, clinicID string) ([]*domain.Therapist, error)
	UpdateTherapist(ctx context.Context, therapist *domain.Therapist) error
	DeactivateTherapist(ctx context.Context, id string) error

	// Schedules
	CreateSchedule(ctx context.Context, schedule *domain.DoctorSchedule) error
	GetScheduleByID(ctx context.Context, id string) (*domain.DoctorSchedule, error)
	ListDoctorSchedules(ctx context.Context, doctorID string) ([]*domain.DoctorSchedule, error)
	ListDoctorSchedulesForDay(ctx context.Context, doctorID string, dayOfWeek int) ([]*domain.DoctorSchedule, error)
	DeleteSchedule(ctx context.Context, id string) error

	// Leaves & Exceptions
	CreateLeave(ctx context.Context, leave *domain.DoctorLeave) error
	GetLeaveByID(ctx context.Context, id string) (*domain.DoctorLeave, error)
	ListDoctorLeaves(ctx context.Context, doctorID string) ([]*domain.DoctorLeave, error)
	GetApprovedLeavesInRange(ctx context.Context, doctorID string, start, end time.Time) ([]*domain.DoctorLeave, error)
	UpdateLeaveStatus(ctx context.Context, id string, status string) error

	// Packages & Steps
	CreatePackage(ctx context.Context, pkg *domain.TherapyPackage) error
	GetPackageByID(ctx context.Context, id string) (*domain.TherapyPackage, error)
	ListPackages(ctx context.Context, clinicID string, publishedOnly bool) ([]*domain.TherapyPackage, error)
	UpdatePackage(ctx context.Context, pkg *domain.TherapyPackage) error
	PublishPackage(ctx context.Context, id string) error
	DeactivatePackage(ctx context.Context, id string) error
	SavePackageSteps(ctx context.Context, packageID string, steps []domain.PackageStep) error
	GetPackageSteps(ctx context.Context, packageID string) ([]domain.PackageStep, error)
	SaveCareChecklist(ctx context.Context, checklist *domain.CareChecklist) error
	GetCareChecklists(ctx context.Context, packageID string) ([]domain.CareChecklist, error)
}

type PostgresClinicRepository struct {
	db *database.DB
}

func NewPostgresClinicRepository(db *database.DB) *PostgresClinicRepository {
	return &PostgresClinicRepository{db: db}
}

// Helper functions for PostgreSQL text[] arrays
func formatPGArray(arr []string) string {
	if len(arr) == 0 {
		return "{}"
	}
	escaped := make([]string, len(arr))
	for i, v := range arr {
		escaped[i] = fmt.Sprintf(`"%s"`, strings.ReplaceAll(v, `"`, `\"`))
	}
	return "{" + strings.Join(escaped, ",") + "}"
}

func parsePGArray(src string) []string {
	src = strings.Trim(src, "{}")
	if src == "" {
		return []string{}
	}
	rawItems := strings.Split(src, ",")
	results := make([]string, 0, len(rawItems))
	for _, item := range rawItems {
		trimmed := strings.Trim(item, ` "`)
		if trimmed != "" {
			results = append(results, trimmed)
		}
	}
	return results
}

// --- Clinics ---

func (r *PostgresClinicRepository) CreateClinic(ctx context.Context, c *domain.Clinic) error {
	query := `
		INSERT INTO clinics (
			id, name, registration_number, address_street, address_city, address_state, address_postal_code,
			contact_phone, contact_email, description, supported_languages, operating_hours, timezone, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	_, err := r.db.ExecContext(ctx, query,
		c.ID, c.Name, c.RegistrationNumber, c.AddressStreet, c.AddressCity, c.AddressState, c.AddressPostalCode,
		c.ContactPhone, c.ContactEmail, c.Description, formatPGArray(c.SupportedLanguages), c.OperatingHours,
		c.Timezone, c.IsActive, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert clinic: %w", err)
	}
	return nil
}

func (r *PostgresClinicRepository) GetClinicByID(ctx context.Context, id string) (*domain.Clinic, error) {
	query := `
		SELECT id, name, registration_number, address_street, address_city, address_state, address_postal_code,
		       contact_phone, contact_email, description, supported_languages, operating_hours, timezone, is_active, created_at, updated_at
		FROM clinics
		WHERE id = $1
	`
	var c domain.Clinic
	var desc sql.NullString
	var langRaw string
	var opHours []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.RegistrationNumber, &c.AddressStreet, &c.AddressCity, &c.AddressState, &c.AddressPostalCode,
		&c.ContactPhone, &c.ContactEmail, &desc, &langRaw, &opHours, &c.Timezone, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query clinic by id: %w", err)
	}

	if desc.Valid {
		c.Description = desc.String
	}
	c.SupportedLanguages = parsePGArray(langRaw)
	c.OperatingHours = opHours
	return &c, nil
}

func (r *PostgresClinicRepository) GetClinicByRegistrationNumber(ctx context.Context, regNum string) (*domain.Clinic, error) {
	query := `
		SELECT id, name, registration_number, address_street, address_city, address_state, address_postal_code,
		       contact_phone, contact_email, description, supported_languages, operating_hours, timezone, is_active, created_at, updated_at
		FROM clinics
		WHERE LOWER(registration_number) = LOWER($1)
	`
	var c domain.Clinic
	var desc sql.NullString
	var langRaw string
	var opHours []byte

	err := r.db.QueryRowContext(ctx, query, regNum).Scan(
		&c.ID, &c.Name, &c.RegistrationNumber, &c.AddressStreet, &c.AddressCity, &c.AddressState, &c.AddressPostalCode,
		&c.ContactPhone, &c.ContactEmail, &desc, &langRaw, &opHours, &c.Timezone, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query clinic by registration number: %w", err)
	}

	if desc.Valid {
		c.Description = desc.String
	}
	c.SupportedLanguages = parsePGArray(langRaw)
	c.OperatingHours = opHours
	return &c, nil
}

func (r *PostgresClinicRepository) ListClinics(ctx context.Context, city string, activeOnly bool) ([]*domain.Clinic, error) {
	query := `
		SELECT id, name, registration_number, address_street, address_city, address_state, address_postal_code,
		       contact_phone, contact_email, description, supported_languages, operating_hours, timezone, is_active, created_at, updated_at
		FROM clinics
		WHERE ($1 = '' OR LOWER(address_city) = LOWER($1))
		  AND ($2 = FALSE OR is_active = TRUE)
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query, city, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to list clinics: %w", err)
	}
	defer rows.Close()

	var clinics []*domain.Clinic
	for rows.Next() {
		var c domain.Clinic
		var desc sql.NullString
		var langRaw string
		var opHours []byte

		if err := rows.Scan(
			&c.ID, &c.Name, &c.RegistrationNumber, &c.AddressStreet, &c.AddressCity, &c.AddressState, &c.AddressPostalCode,
			&c.ContactPhone, &c.ContactEmail, &desc, &langRaw, &opHours, &c.Timezone, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan clinic row: %w", err)
		}
		if desc.Valid {
			c.Description = desc.String
		}
		c.SupportedLanguages = parsePGArray(langRaw)
		c.OperatingHours = opHours
		clinics = append(clinics, &c)
	}
	return clinics, nil
}

func (r *PostgresClinicRepository) UpdateClinic(ctx context.Context, c *domain.Clinic) error {
	query := `
		UPDATE clinics
		SET name = $1, address_street = $2, address_city = $3, address_state = $4, address_postal_code = $5,
		    contact_phone = $6, contact_email = $7, description = $8, supported_languages = $9,
		    operating_hours = $10, timezone = $11, updated_at = CURRENT_TIMESTAMP
		WHERE id = $12
	`
	_, err := r.db.ExecContext(ctx, query,
		c.Name, c.AddressStreet, c.AddressCity, c.AddressState, c.AddressPostalCode,
		c.ContactPhone, c.ContactEmail, c.Description, formatPGArray(c.SupportedLanguages),
		c.OperatingHours, c.Timezone, c.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update clinic: %w", err)
	}
	return nil
}

func (r *PostgresClinicRepository) DeactivateClinic(ctx context.Context, id string) error {
	query := `UPDATE clinics SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// --- Treatment Rooms & Equipment ---

func (r *PostgresClinicRepository) CreateRoom(ctx context.Context, room *domain.TreatmentRoom) error {
	query := `
		INSERT INTO treatment_rooms (id, clinic_id, name, room_type, capacity, sanitation_buffer_minutes, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		room.ID, room.ClinicID, room.Name, room.RoomType, room.Capacity,
		room.SanitationBufferMinutes, room.IsActive, room.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert treatment room: %w", err)
	}
	return nil
}

func (r *PostgresClinicRepository) GetRoomByID(ctx context.Context, id string) (*domain.TreatmentRoom, error) {
	query := `
		SELECT id, clinic_id, name, room_type, capacity, sanitation_buffer_minutes, is_active, created_at
		FROM treatment_rooms
		WHERE id = $1
	`
	var rm domain.TreatmentRoom
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rm.ID, &rm.ClinicID, &rm.Name, &rm.RoomType, &rm.Capacity,
		&rm.SanitationBufferMinutes, &rm.IsActive, &rm.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query room by id: %w", err)
	}

	equip, _ := r.ListRoomEquipment(ctx, id)
	rm.Equipment = equip
	return &rm, nil
}

func (r *PostgresClinicRepository) ListRooms(ctx context.Context, clinicID string) ([]*domain.TreatmentRoom, error) {
	query := `
		SELECT id, clinic_id, name, room_type, capacity, sanitation_buffer_minutes, is_active, created_at
		FROM treatment_rooms
		WHERE clinic_id = $1
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query, clinicID)
	if err != nil {
		return nil, fmt.Errorf("failed to list treatment rooms: %w", err)
	}
	defer rows.Close()

	var rooms []*domain.TreatmentRoom
	for rows.Next() {
		var rm domain.TreatmentRoom
		if err := rows.Scan(
			&rm.ID, &rm.ClinicID, &rm.Name, &rm.RoomType, &rm.Capacity,
			&rm.SanitationBufferMinutes, &rm.IsActive, &rm.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan room row: %w", err)
		}
		equip, _ := r.ListRoomEquipment(ctx, rm.ID)
		rm.Equipment = equip
		rooms = append(rooms, &rm)
	}
	return rooms, nil
}

func (r *PostgresClinicRepository) UpdateRoom(ctx context.Context, room *domain.TreatmentRoom) error {
	query := `
		UPDATE treatment_rooms
		SET name = $1, room_type = $2, capacity = $3, sanitation_buffer_minutes = $4, is_active = $5
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query, room.Name, room.RoomType, room.Capacity, room.SanitationBufferMinutes, room.IsActive, room.ID)
	return err
}

func (r *PostgresClinicRepository) DeactivateRoom(ctx context.Context, id string) error {
	query := `UPDATE treatment_rooms SET is_active = FALSE WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresClinicRepository) AddRoomEquipment(ctx context.Context, equip *domain.RoomEquipment) error {
	query := `
		INSERT INTO room_equipment (id, room_id, equipment_type, serial_number, is_operational)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query, equip.ID, equip.RoomID, equip.EquipmentType, equip.SerialNumber, equip.IsOperational)
	return err
}

func (r *PostgresClinicRepository) ListRoomEquipment(ctx context.Context, roomID string) ([]domain.RoomEquipment, error) {
	query := `
		SELECT id, room_id, equipment_type, serial_number, is_operational
		FROM room_equipment
		WHERE room_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.RoomEquipment
	for rows.Next() {
		var eq domain.RoomEquipment
		var sn sql.NullString
		if err := rows.Scan(&eq.ID, &eq.RoomID, &eq.EquipmentType, &sn, &eq.IsOperational); err != nil {
			return nil, err
		}
		if sn.Valid {
			eq.SerialNumber = sn.String
		}
		result = append(result, eq)
	}
	return result, nil
}

// --- Doctors ---

func (r *PostgresClinicRepository) CreateDoctor(ctx context.Context, doc *domain.Doctor) error {
	query := `
		INSERT INTO doctors (id, user_id, clinic_id, qualifications, specialties, experience_years, max_sessions_per_day, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		doc.ID, doc.UserID, doc.ClinicID, doc.Qualifications, formatPGArray(doc.Specialties),
		doc.ExperienceYears, doc.MaxSessionsPerDay, doc.IsActive, doc.CreatedAt, doc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert doctor: %w", err)
	}
	return nil
}

func (r *PostgresClinicRepository) GetDoctorByID(ctx context.Context, id string) (*domain.Doctor, error) {
	query := `
		SELECT id, user_id, clinic_id, qualifications, specialties, experience_years, max_sessions_per_day, is_active, created_at, updated_at
		FROM doctors
		WHERE id = $1
	`
	var d domain.Doctor
	var specRaw string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &d.UserID, &d.ClinicID, &d.Qualifications, &specRaw,
		&d.ExperienceYears, &d.MaxSessionsPerDay, &d.IsActive, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query doctor by id: %w", err)
	}
	d.Specialties = parsePGArray(specRaw)
	return &d, nil
}

func (r *PostgresClinicRepository) GetDoctorByUserID(ctx context.Context, userID string) (*domain.Doctor, error) {
	query := `
		SELECT id, user_id, clinic_id, qualifications, specialties, experience_years, max_sessions_per_day, is_active, created_at, updated_at
		FROM doctors
		WHERE user_id = $1
	`
	var d domain.Doctor
	var specRaw string
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&d.ID, &d.UserID, &d.ClinicID, &d.Qualifications, &specRaw,
		&d.ExperienceYears, &d.MaxSessionsPerDay, &d.IsActive, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query doctor by user id: %w", err)
	}
	d.Specialties = parsePGArray(specRaw)
	return &d, nil
}

func (r *PostgresClinicRepository) ListDoctors(ctx context.Context, clinicID string) ([]*domain.Doctor, error) {
	query := `
		SELECT id, user_id, clinic_id, qualifications, specialties, experience_years, max_sessions_per_day, is_active, created_at, updated_at
		FROM doctors
		WHERE clinic_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, clinicID)
	if err != nil {
		return nil, fmt.Errorf("failed to list doctors: %w", err)
	}
	defer rows.Close()

	var docs []*domain.Doctor
	for rows.Next() {
		var d domain.Doctor
		var specRaw string
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.ClinicID, &d.Qualifications, &specRaw,
			&d.ExperienceYears, &d.MaxSessionsPerDay, &d.IsActive, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan doctor row: %w", err)
		}
		d.Specialties = parsePGArray(specRaw)
		docs = append(docs, &d)
	}
	return docs, nil
}

func (r *PostgresClinicRepository) UpdateDoctor(ctx context.Context, d *domain.Doctor) error {
	query := `
		UPDATE doctors
		SET qualifications = $1, specialties = $2, experience_years = $3, max_sessions_per_day = $4,
		    is_active = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query,
		d.Qualifications, formatPGArray(d.Specialties), d.ExperienceYears, d.MaxSessionsPerDay, d.IsActive, d.ID,
	)
	return err
}

func (r *PostgresClinicRepository) DeactivateDoctor(ctx context.Context, id string) error {
	query := `UPDATE doctors SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// --- Therapists ---

func (r *PostgresClinicRepository) CreateTherapist(ctx context.Context, t *domain.Therapist) error {
	query := `
		INSERT INTO therapists (id, user_id, clinic_id, specializations, experience_years, gender, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		t.ID, t.UserID, t.ClinicID, formatPGArray(t.Specializations),
		t.ExperienceYears, t.Gender, t.IsActive, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert therapist: %w", err)
	}
	return nil
}

func (r *PostgresClinicRepository) GetTherapistByID(ctx context.Context, id string) (*domain.Therapist, error) {
	query := `
		SELECT id, user_id, clinic_id, specializations, experience_years, gender, is_active, created_at, updated_at
		FROM therapists
		WHERE id = $1
	`
	var t domain.Therapist
	var specRaw string
	var gender sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.UserID, &t.ClinicID, &specRaw, &t.ExperienceYears, &gender,
		&t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query therapist by id: %w", err)
	}
	t.Specializations = parsePGArray(specRaw)
	if gender.Valid {
		t.Gender = gender.String
	}
	return &t, nil
}

func (r *PostgresClinicRepository) GetTherapistByUserID(ctx context.Context, userID string) (*domain.Therapist, error) {
	query := `
		SELECT id, user_id, clinic_id, specializations, experience_years, gender, is_active, created_at, updated_at
		FROM therapists
		WHERE user_id = $1
	`
	var t domain.Therapist
	var specRaw string
	var gender sql.NullString
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&t.ID, &t.UserID, &t.ClinicID, &specRaw, &t.ExperienceYears, &gender,
		&t.IsActive, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query therapist by user id: %w", err)
	}
	t.Specializations = parsePGArray(specRaw)
	if gender.Valid {
		t.Gender = gender.String
	}
	return &t, nil
}

func (r *PostgresClinicRepository) ListTherapists(ctx context.Context, clinicID string) ([]*domain.Therapist, error) {
	query := `
		SELECT id, user_id, clinic_id, specializations, experience_years, gender, is_active, created_at, updated_at
		FROM therapists
		WHERE clinic_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, clinicID)
	if err != nil {
		return nil, fmt.Errorf("failed to list therapists: %w", err)
	}
	defer rows.Close()

	var therapists []*domain.Therapist
	for rows.Next() {
		var t domain.Therapist
		var specRaw string
		var gender sql.NullString
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.ClinicID, &specRaw, &t.ExperienceYears, &gender,
			&t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan therapist row: %w", err)
		}
		t.Specializations = parsePGArray(specRaw)
		if gender.Valid {
			t.Gender = gender.String
		}
		therapists = append(therapists, &t)
	}
	return therapists, nil
}

func (r *PostgresClinicRepository) UpdateTherapist(ctx context.Context, t *domain.Therapist) error {
	query := `
		UPDATE therapists
		SET specializations = $1, experience_years = $2, gender = $3, is_active = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`
	_, err := r.db.ExecContext(ctx, query, formatPGArray(t.Specializations), t.ExperienceYears, t.Gender, t.IsActive, t.ID)
	return err
}

func (r *PostgresClinicRepository) DeactivateTherapist(ctx context.Context, id string) error {
	query := `UPDATE therapists SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// --- Schedules ---

func (r *PostgresClinicRepository) CreateSchedule(ctx context.Context, s *domain.DoctorSchedule) error {
	query := `
		INSERT INTO doctor_schedules (id, doctor_id, day_of_week, start_time, end_time, slot_duration_minutes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query, s.ID, s.DoctorID, s.DayOfWeek, s.StartTime, s.EndTime, s.SlotDurationMinutes, s.IsActive)
	if err != nil {
		return fmt.Errorf("failed to insert doctor schedule: %w", err)
	}
	return nil
}

func (r *PostgresClinicRepository) GetScheduleByID(ctx context.Context, id string) (*domain.DoctorSchedule, error) {
	query := `
		SELECT id, doctor_id, day_of_week, start_time, end_time, slot_duration_minutes, is_active
		FROM doctor_schedules
		WHERE id = $1
	`
	var s domain.DoctorSchedule
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.DoctorID, &s.DayOfWeek, &s.StartTime, &s.EndTime, &s.SlotDurationMinutes, &s.IsActive,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query schedule by id: %w", err)
	}
	return &s, nil
}

func (r *PostgresClinicRepository) ListDoctorSchedules(ctx context.Context, doctorID string) ([]*domain.DoctorSchedule, error) {
	query := `
		SELECT id, doctor_id, day_of_week, start_time, end_time, slot_duration_minutes, is_active
		FROM doctor_schedules
		WHERE doctor_id = $1 AND is_active = TRUE
		ORDER BY day_of_week ASC, start_time ASC
	`
	rows, err := r.db.QueryContext(ctx, query, doctorID)
	if err != nil {
		return nil, fmt.Errorf("failed to list doctor schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*domain.DoctorSchedule
	for rows.Next() {
		var s domain.DoctorSchedule
		if err := rows.Scan(&s.ID, &s.DoctorID, &s.DayOfWeek, &s.StartTime, &s.EndTime, &s.SlotDurationMinutes, &s.IsActive); err != nil {
			return nil, err
		}
		schedules = append(schedules, &s)
	}
	return schedules, nil
}

func (r *PostgresClinicRepository) ListDoctorSchedulesForDay(ctx context.Context, doctorID string, dayOfWeek int) ([]*domain.DoctorSchedule, error) {
	query := `
		SELECT id, doctor_id, day_of_week, start_time, end_time, slot_duration_minutes, is_active
		FROM doctor_schedules
		WHERE doctor_id = $1 AND day_of_week = $2 AND is_active = TRUE
		ORDER BY start_time ASC
	`
	rows, err := r.db.QueryContext(ctx, query, doctorID, dayOfWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to list doctor schedules for day: %w", err)
	}
	defer rows.Close()

	var schedules []*domain.DoctorSchedule
	for rows.Next() {
		var s domain.DoctorSchedule
		if err := rows.Scan(&s.ID, &s.DoctorID, &s.DayOfWeek, &s.StartTime, &s.EndTime, &s.SlotDurationMinutes, &s.IsActive); err != nil {
			return nil, err
		}
		schedules = append(schedules, &s)
	}
	return schedules, nil
}

func (r *PostgresClinicRepository) DeleteSchedule(ctx context.Context, id string) error {
	query := `DELETE FROM doctor_schedules WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// --- Leaves ---

func (r *PostgresClinicRepository) CreateLeave(ctx context.Context, l *domain.DoctorLeave) error {
	query := `
		INSERT INTO doctor_leaves (id, doctor_id, start_time, end_time, reason, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query, l.ID, l.DoctorID, l.StartTime, l.EndTime, l.Reason, l.Status, l.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert doctor leave: %w", err)
	}
	return nil
}

func (r *PostgresClinicRepository) GetLeaveByID(ctx context.Context, id string) (*domain.DoctorLeave, error) {
	query := `
		SELECT id, doctor_id, start_time, end_time, reason, status, created_at
		FROM doctor_leaves
		WHERE id = $1
	`
	var l domain.DoctorLeave
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&l.ID, &l.DoctorID, &l.StartTime, &l.EndTime, &l.Reason, &l.Status, &l.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query leave by id: %w", err)
	}
	return &l, nil
}

func (r *PostgresClinicRepository) ListDoctorLeaves(ctx context.Context, doctorID string) ([]*domain.DoctorLeave, error) {
	query := `
		SELECT id, doctor_id, start_time, end_time, reason, status, created_at
		FROM doctor_leaves
		WHERE doctor_id = $1
		ORDER BY start_time DESC
	`
	rows, err := r.db.QueryContext(ctx, query, doctorID)
	if err != nil {
		return nil, fmt.Errorf("failed to list doctor leaves: %w", err)
	}
	defer rows.Close()

	var leaves []*domain.DoctorLeave
	for rows.Next() {
		var l domain.DoctorLeave
		if err := rows.Scan(&l.ID, &l.DoctorID, &l.StartTime, &l.EndTime, &l.Reason, &l.Status, &l.CreatedAt); err != nil {
			return nil, err
		}
		leaves = append(leaves, &l)
	}
	return leaves, nil
}

func (r *PostgresClinicRepository) GetApprovedLeavesInRange(ctx context.Context, doctorID string, start, end time.Time) ([]*domain.DoctorLeave, error) {
	query := `
		SELECT id, doctor_id, start_time, end_time, reason, status, created_at
		FROM doctor_leaves
		WHERE doctor_id = $1
		  AND status = 'APPROVED'
		  AND start_time < $3
		  AND end_time > $2
	`
	rows, err := r.db.QueryContext(ctx, query, doctorID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query approved leaves: %w", err)
	}
	defer rows.Close()

	var leaves []*domain.DoctorLeave
	for rows.Next() {
		var l domain.DoctorLeave
		if err := rows.Scan(&l.ID, &l.DoctorID, &l.StartTime, &l.EndTime, &l.Reason, &l.Status, &l.CreatedAt); err != nil {
			return nil, err
		}
		leaves = append(leaves, &l)
	}
	return leaves, nil
}

func (r *PostgresClinicRepository) UpdateLeaveStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE doctor_leaves SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// --- Packages, Steps & Checklists ---

func (r *PostgresClinicRepository) CreatePackage(ctx context.Context, p *domain.TherapyPackage) error {
	query := `
		INSERT INTO therapy_packages (
			id, clinic_id, name, category, description, price_cents, total_sessions, duration_minutes,
			recommended_intervals_days, version, is_published, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.ClinicID, p.Name, p.Category, p.Description, p.PriceCents, p.TotalSessions,
		p.DurationMinutes, p.RecommendedIntervalsDays, p.Version, p.IsPublished, p.IsActive, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert therapy package: %w", err)
	}
	return nil
}

func (r *PostgresClinicRepository) GetPackageByID(ctx context.Context, id string) (*domain.TherapyPackage, error) {
	query := `
		SELECT id, clinic_id, name, category, description, price_cents, total_sessions, duration_minutes,
		       recommended_intervals_days, version, is_published, is_active, created_at, updated_at
		FROM therapy_packages
		WHERE id = $1
	`
	var p domain.TherapyPackage
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.ClinicID, &p.Name, &p.Category, &p.Description, &p.PriceCents, &p.TotalSessions,
		&p.DurationMinutes, &p.RecommendedIntervalsDays, &p.Version, &p.IsPublished, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query package by id: %w", err)
	}

	steps, _ := r.GetPackageSteps(ctx, p.ID)
	p.Steps = steps
	checklists, _ := r.GetCareChecklists(ctx, p.ID)
	p.Checklists = checklists
	return &p, nil
}

func (r *PostgresClinicRepository) ListPackages(ctx context.Context, clinicID string, publishedOnly bool) ([]*domain.TherapyPackage, error) {
	query := `
		SELECT id, clinic_id, name, category, description, price_cents, total_sessions, duration_minutes,
		       recommended_intervals_days, version, is_published, is_active, created_at, updated_at
		FROM therapy_packages
		WHERE clinic_id = $1
		  AND is_active = TRUE
		  AND ($2 = FALSE OR is_published = TRUE)
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query, clinicID, publishedOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}
	defer rows.Close()

	var packages []*domain.TherapyPackage
	for rows.Next() {
		var p domain.TherapyPackage
		if err := rows.Scan(
			&p.ID, &p.ClinicID, &p.Name, &p.Category, &p.Description, &p.PriceCents, &p.TotalSessions,
			&p.DurationMinutes, &p.RecommendedIntervalsDays, &p.Version, &p.IsPublished, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		steps, _ := r.GetPackageSteps(ctx, p.ID)
		p.Steps = steps
		packages = append(packages, &p)
	}
	return packages, nil
}

func (r *PostgresClinicRepository) UpdatePackage(ctx context.Context, p *domain.TherapyPackage) error {
	query := `
		UPDATE therapy_packages
		SET name = $1, category = $2, description = $3, price_cents = $4, total_sessions = $5,
		    duration_minutes = $6, recommended_intervals_days = $7, version = $8,
		    is_published = $9, is_active = $10, updated_at = CURRENT_TIMESTAMP
		WHERE id = $11
	`
	_, err := r.db.ExecContext(ctx, query,
		p.Name, p.Category, p.Description, p.PriceCents, p.TotalSessions,
		p.DurationMinutes, p.RecommendedIntervalsDays, p.Version, p.IsPublished, p.IsActive, p.ID,
	)
	return err
}

func (r *PostgresClinicRepository) PublishPackage(ctx context.Context, id string) error {
	query := `UPDATE therapy_packages SET is_published = TRUE, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresClinicRepository) DeactivatePackage(ctx context.Context, id string) error {
	query := `UPDATE therapy_packages SET is_active = FALSE, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *PostgresClinicRepository) SavePackageSteps(ctx context.Context, packageID string, steps []domain.PackageStep) error {
	// Delete existing steps
	_, err := r.db.ExecContext(ctx, `DELETE FROM therapy_package_steps WHERE package_id = $1`, packageID)
	if err != nil {
		return err
	}

	insertQuery := `
		INSERT INTO therapy_package_steps (id, package_id, step_order, name, description, duration_minutes, instructions, required_equipment)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	for _, s := range steps {
		_, err := r.db.ExecContext(ctx, insertQuery,
			s.ID, packageID, s.StepOrder, s.Name, s.Description, s.DurationMinutes,
			formatPGArray(s.Instructions), formatPGArray(s.RequiredEquipment),
		)
		if err != nil {
			return fmt.Errorf("failed to insert step %d: %w", s.StepOrder, err)
		}
	}
	return nil
}

func (r *PostgresClinicRepository) GetPackageSteps(ctx context.Context, packageID string) ([]domain.PackageStep, error) {
	query := `
		SELECT id, package_id, step_order, name, description, duration_minutes, instructions, required_equipment
		FROM therapy_package_steps
		WHERE package_id = $1
		ORDER BY step_order ASC
	`
	rows, err := r.db.QueryContext(ctx, query, packageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []domain.PackageStep
	for rows.Next() {
		var s domain.PackageStep
		var instRaw, equipRaw string
		if err := rows.Scan(&s.ID, &s.PackageID, &s.StepOrder, &s.Name, &s.Description, &s.DurationMinutes, &instRaw, &equipRaw); err != nil {
			return nil, err
		}
		s.Instructions = parsePGArray(instRaw)
		s.RequiredEquipment = parsePGArray(equipRaw)
		steps = append(steps, s)
	}
	return steps, nil
}

func (r *PostgresClinicRepository) SaveCareChecklist(ctx context.Context, cl *domain.CareChecklist) error {
	query := `
		INSERT INTO care_checklists (id, package_id, checklist_type, version, items, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		cl.ID, cl.PackageID, cl.ChecklistType, cl.Version, json.RawMessage(cl.Items), cl.IsActive, cl.CreatedAt,
	)
	return err
}

func (r *PostgresClinicRepository) GetCareChecklists(ctx context.Context, packageID string) ([]domain.CareChecklist, error) {
	query := `
		SELECT id, package_id, checklist_type, version, items, is_active, created_at
		FROM care_checklists
		WHERE package_id = $1 AND is_active = TRUE
		ORDER BY checklist_type ASC, version DESC
	`
	rows, err := r.db.QueryContext(ctx, query, packageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checklists []domain.CareChecklist
	for rows.Next() {
		var cl domain.CareChecklist
		var items []byte
		if err := rows.Scan(&cl.ID, &cl.PackageID, &cl.ChecklistType, &cl.Version, &items, &cl.IsActive, &cl.CreatedAt); err != nil {
			return nil, err
		}
		cl.Items = items
		checklists = append(checklists, cl)
	}
	return checklists, nil
}
