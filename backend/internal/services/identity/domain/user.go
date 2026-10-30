package domain

import (
	"time"
)

type Role string

const (
	RolePatient     Role = "patient"
	RoleDoctor      Role = "doctor"
	RoleTherapist   Role = "therapist"
	RoleClinicAdmin Role = "clinic_admin"
	RoleSuperAdmin  Role = "super_admin"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	ClinicID     *string   `json:"clinic_id,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RefreshSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"`
	FamilyID  string    `json:"family_id"`
	IsRevoked bool      `json:"is_revoked"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type PatientProfile struct {
	ID                    string    `json:"id"`
	UserID                string    `json:"user_id"`
	FullName              string    `json:"full_name"`
	DateOfBirth           string    `json:"date_of_birth"`
	BiologicalSex         string    `json:"biological_sex"`
	BloodGroup            string    `json:"blood_group,omitempty"`
	EmergencyContactName  string    `json:"emergency_contact_name"`
	EmergencyContactPhone string    `json:"emergency_contact_phone"`
	AddressStreet         string    `json:"address_street,omitempty"`
	AddressCity           string    `json:"address_city,omitempty"`
	AddressState          string    `json:"address_state,omitempty"`
	AddressPostalCode     string    `json:"address_postal_code,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}
