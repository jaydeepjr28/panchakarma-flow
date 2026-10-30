package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/domain"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
)

type UserRepository interface {
	// User operations
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	UpdateUserStatus(ctx context.Context, id string, isActive bool) error

	// Patient Profile operations
	CreatePatientProfile(ctx context.Context, profile *domain.PatientProfile) error
	GetPatientProfile(ctx context.Context, userID string) (*domain.PatientProfile, error)
	UpdatePatientProfile(ctx context.Context, profile *domain.PatientProfile) error

	// Refresh Session operations
	CreateRefreshSession(ctx context.Context, session *domain.RefreshSession) error
	GetRefreshSessionByHash(ctx context.Context, tokenHash string) (*domain.RefreshSession, error)
	RevokeRefreshSession(ctx context.Context, sessionID string) error
	RevokeRefreshFamily(ctx context.Context, familyID string) error
}

type PostgresUserRepository struct {
	db *database.DB
}

func NewPostgresUserRepository(db *database.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// CreateUser inserts a new user record.
func (r *PostgresUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	if r.db == nil || r.db.DB == nil {
		return fmt.Errorf("database connection is unavailable")
	}
	query := `
		INSERT INTO users (id, email, phone, password_hash, role, clinic_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.Phone, user.PasswordHash,
		user.Role, user.ClinicID, user.IsActive, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

// GetUserByEmail queries a user by email case-insensitively.
func (r *PostgresUserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if r.db == nil || r.db.DB == nil {
		return nil, fmt.Errorf("database connection is unavailable")
	}
	query := `
		SELECT id, email, phone, password_hash, role, clinic_id, is_active, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`
	var u domain.User
	var clinicID sql.NullString

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.Role,
		&clinicID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}

	if clinicID.Valid {
		u.ClinicID = &clinicID.String
	}

	return &u, nil
}

// GetUserByID queries a user by their UUID primary key.
func (r *PostgresUserRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	if r.db == nil || r.db.DB == nil {
		return nil, fmt.Errorf("database connection is unavailable")
	}
	query := `
		SELECT id, email, phone, password_hash, role, clinic_id, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u domain.User
	var clinicID sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.Role,
		&clinicID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query user by id: %w", err)
	}

	if clinicID.Valid {
		u.ClinicID = &clinicID.String
	}

	return &u, nil
}

// UpdateUserStatus enables or disables a user account.
func (r *PostgresUserRepository) UpdateUserStatus(ctx context.Context, id string, isActive bool) error {
	if r.db == nil || r.db.DB == nil {
		return fmt.Errorf("database connection is unavailable")
	}
	query := `
		UPDATE users
		SET is_active = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`
	res, err := r.db.ExecContext(ctx, query, isActive, id)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found with id %s", id)
	}
	return nil
}

// CreatePatientProfile creates the associated demographic profile.
func (r *PostgresUserRepository) CreatePatientProfile(ctx context.Context, profile *domain.PatientProfile) error {
	if r.db == nil || r.db.DB == nil {
		return fmt.Errorf("database connection is unavailable")
	}
	query := `
		INSERT INTO patient_profiles (
			id, user_id, full_name, date_of_birth, biological_sex, blood_group,
			emergency_contact_name, emergency_contact_phone,
			address_street, address_city, address_state, address_postal_code,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.ExecContext(ctx, query,
		profile.ID, profile.UserID, profile.FullName, profile.DateOfBirth,
		profile.BiologicalSex, profile.BloodGroup,
		profile.EmergencyContactName, profile.EmergencyContactPhone,
		profile.AddressStreet, profile.AddressCity, profile.AddressState, profile.AddressPostalCode,
		profile.CreatedAt, profile.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert patient profile: %w", err)
	}
	return nil
}

// GetPatientProfile fetches a patient profile by user ID.
func (r *PostgresUserRepository) GetPatientProfile(ctx context.Context, userID string) (*domain.PatientProfile, error) {
	query := `
		SELECT id, user_id, full_name, date_of_birth, biological_sex, blood_group,
		       emergency_contact_name, emergency_contact_phone,
		       address_street, address_city, address_state, address_postal_code,
		       created_at, updated_at
		FROM patient_profiles
		WHERE user_id = $1
	`
	var p domain.PatientProfile
	var bloodGroup, street, city, state, postal sql.NullString

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&p.ID, &p.UserID, &p.FullName, &p.DateOfBirth, &p.BiologicalSex,
		&bloodGroup, &p.EmergencyContactName, &p.EmergencyContactPhone,
		&street, &city, &state, &postal,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query patient profile: %w", err)
	}

	if bloodGroup.Valid {
		p.BloodGroup = bloodGroup.String
	}
	if street.Valid {
		p.AddressStreet = street.String
	}
	if city.Valid {
		p.AddressCity = city.String
	}
	if state.Valid {
		p.AddressState = state.String
	}
	if postal.Valid {
		p.AddressPostalCode = postal.String
	}

	return &p, nil
}

// UpdatePatientProfile updates an existing profile.
func (r *PostgresUserRepository) UpdatePatientProfile(ctx context.Context, profile *domain.PatientProfile) error {
	query := `
		UPDATE patient_profiles
		SET full_name = $1, date_of_birth = $2, biological_sex = $3, blood_group = $4,
		    emergency_contact_name = $5, emergency_contact_phone = $6,
		    address_street = $7, address_city = $8, address_state = $9, address_postal_code = $10,
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $11
	`
	_, err := r.db.ExecContext(ctx, query,
		profile.FullName, profile.DateOfBirth, profile.BiologicalSex, profile.BloodGroup,
		profile.EmergencyContactName, profile.EmergencyContactPhone,
		profile.AddressStreet, profile.AddressCity, profile.AddressState, profile.AddressPostalCode,
		profile.UserID,
	)
	if err != nil {
		return fmt.Errorf("failed to update patient profile: %w", err)
	}
	return nil
}

// CreateRefreshSession records a new active refresh token session.
func (r *PostgresUserRepository) CreateRefreshSession(ctx context.Context, session *domain.RefreshSession) error {
	query := `
		INSERT INTO refresh_sessions (id, user_id, token_hash, family_id, is_revoked, user_agent, ip_address, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.UserID, session.TokenHash, session.FamilyID,
		session.IsRevoked, session.UserAgent, session.IPAddress, session.ExpiresAt, session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert refresh session: %w", err)
	}
	return nil
}

// GetRefreshSessionByHash looks up a refresh session by its SHA-256 token hash.
func (r *PostgresUserRepository) GetRefreshSessionByHash(ctx context.Context, tokenHash string) (*domain.RefreshSession, error) {
	query := `
		SELECT id, user_id, token_hash, family_id, is_revoked, user_agent, ip_address, expires_at, created_at
		FROM refresh_sessions
		WHERE token_hash = $1
	`
	var s domain.RefreshSession
	var userAgent sql.NullString

	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&s.ID, &s.UserID, &s.TokenHash, &s.FamilyID, &s.IsRevoked,
		&userAgent, &s.IPAddress, &s.ExpiresAt, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query refresh session: %w", err)
	}

	if userAgent.Valid {
		s.UserAgent = userAgent.String
	}

	return &s, nil
}

// RevokeRefreshSession marks an individual refresh token as used/revoked.
func (r *PostgresUserRepository) RevokeRefreshSession(ctx context.Context, sessionID string) error {
	query := `
		UPDATE refresh_sessions
		SET is_revoked = TRUE
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh session: %w", err)
	}
	return nil
}

// RevokeRefreshFamily revokes all tokens in a family (triggered upon replay attack).
func (r *PostgresUserRepository) RevokeRefreshFamily(ctx context.Context, familyID string) error {
	query := `
		UPDATE refresh_sessions
		SET is_revoked = TRUE
		WHERE family_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, familyID)
	if err != nil {
		return fmt.Errorf("failed to revoke token family: %w", err)
	}
	return nil
}
