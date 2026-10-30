package domain

import (
	"encoding/json"
	"time"
)

// Clinic represents a physical Panchakarma wellness center.
type Clinic struct {
	ID                 string          `json:"id"`
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
	OperatingHours     json.RawMessage `json:"operating_hours"` // JSON structure per day
	Timezone           string          `json:"timezone"`        // IANA time zone (e.g., "Asia/Kolkata")
	IsActive           bool            `json:"is_active"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

// TreatmentRoom represents a therapy room (Chikitsa Kaksha).
type TreatmentRoom struct {
	ID                      string          `json:"id"`
	ClinicID                string          `json:"clinic_id"`
	Name                    string          `json:"name"`
	RoomType                string          `json:"room_type"`
	Capacity                int             `json:"capacity"`
	SanitationBufferMinutes int             `json:"sanitation_buffer_minutes"`
	IsActive                bool            `json:"is_active"`
	CreatedAt               time.Time       `json:"created_at"`
	Equipment               []RoomEquipment `json:"equipment,omitempty"`
}

// RoomEquipment represents specialized Panchakarma equipment.
type RoomEquipment struct {
	ID            string `json:"id"`
	RoomID        string `json:"room_id"`
	EquipmentType string `json:"equipment_type"` // 'droni_table', 'steam_chamber', 'shirodhara_stand'
	SerialNumber  string `json:"serial_number,omitempty"`
	IsOperational bool   `json:"is_operational"`
}

// Doctor represents an Ayurvedic physician associated with a clinic.
type Doctor struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"` // References Identity service user
	ClinicID          string    `json:"clinic_id"`
	Qualifications    string    `json:"qualifications"` // 'BAMS, MD (Ayurveda)'
	Specialties       []string  `json:"specialties"`
	ExperienceYears   int       `json:"experience_years"`
	MaxSessionsPerDay int       `json:"max_sessions_per_day"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Therapist represents a Panchakarma therapist assigned to a clinic.
type Therapist struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"` // References Identity service user
	ClinicID        string    `json:"clinic_id"`
	Specializations []string  `json:"specializations"`
	ExperienceYears int       `json:"experience_years"`
	Gender          string    `json:"gender,omitempty"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// DoctorSchedule defines weekly recurring operating hours for a physician.
type DoctorSchedule struct {
	ID                  string `json:"id"`
	DoctorID            string `json:"doctor_id"`
	DayOfWeek           int    `json:"day_of_week"` // 0=Sunday, 1=Monday ... 6=Saturday
	StartTime           string `json:"start_time"`  // "HH:MM", e.g. "09:00"
	EndTime             string `json:"end_time"`    // "HH:MM", e.g. "17:00"
	SlotDurationMinutes int    `json:"slot_duration_minutes"`
	IsActive            bool   `json:"is_active"`
}

// DoctorLeave represents an approved leave or schedule exception.
type DoctorLeave struct {
	ID        string    `json:"id"`
	DoctorID  string    `json:"doctor_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Reason    string    `json:"reason"`
	Status    string    `json:"status"` // 'PENDING', 'APPROVED', 'CANCELLED'
	CreatedAt time.Time `json:"created_at"`
}

// TherapyPackage defines a standardized Panchakarma therapy offering.
type TherapyPackage struct {
	ID                       string          `json:"id"`
	ClinicID                 string          `json:"clinic_id"`
	Name                     string          `json:"name"`
	Category                 string          `json:"category"` // 'detox', 'rejuvenation', 'stress_relief'
	Description              string          `json:"description"`
	PriceCents               int             `json:"price_cents"`
	TotalSessions            int             `json:"total_sessions"`
	DurationMinutes          int             `json:"duration_minutes"`
	RecommendedIntervalsDays int             `json:"recommended_intervals_days"`
	Version                  int             `json:"version"`
	IsPublished              bool            `json:"is_published"`
	IsActive                 bool            `json:"is_active"`
	CreatedAt                time.Time       `json:"created_at"`
	UpdatedAt                time.Time       `json:"updated_at"`
	Steps                    []PackageStep   `json:"steps,omitempty"`
	Checklists               []CareChecklist `json:"checklists,omitempty"`
}

// PackageStep defines an ordered procedure step within a therapy package.
type PackageStep struct {
	ID                string   `json:"id"`
	PackageID         string   `json:"package_id"`
	StepOrder         int      `json:"step_order"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	DurationMinutes   int      `json:"duration_minutes"`
	Instructions      []string `json:"instructions"`
	RequiredEquipment []string `json:"required_equipment"`
}

// CareChecklist defines pre- and post-procedure clinical instructions.
type CareChecklist struct {
	ID            string          `json:"id"`
	PackageID     string          `json:"package_id"`
	ChecklistType string          `json:"checklist_type"` // 'PRE_PROCEDURE', 'POST_PROCEDURE'
	Version       int             `json:"version"`
	Items         json.RawMessage `json:"items"` // JSON array of checklist items
	IsActive      bool            `json:"is_active"`
	CreatedAt     time.Time       `json:"created_at"`
}
