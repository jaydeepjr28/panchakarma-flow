package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
)

func TestPostgresUserRepository(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	testDB := database.NewTestDB(db)
	repo := repository.NewPostgresUserRepository(testDB)
	ctx := context.Background()

	t.Run("CreateUser_Success", func(t *testing.T) {
		user := &domain.User{
			ID:           uuid.New().String(),
			Email:        "user@example.com",
			Phone:        "+919876543210",
			PasswordHash: "hashed_pass",
			Role:         domain.RolePatient,
			ClinicID:     nil,
			IsActive:     true,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}

		mock.ExpectExec("INSERT INTO users").
			WithArgs(user.ID, user.Email, user.Phone, user.PasswordHash, user.Role, user.ClinicID, user.IsActive, user.CreatedAt, user.UpdatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("GetUserByEmail_Found", func(t *testing.T) {
		id := uuid.New().String()
		email := "find@example.com"
		now := time.Now().UTC()

		rows := sqlmock.NewRows([]string{"id", "email", "phone", "password_hash", "role", "clinic_id", "is_active", "created_at", "updated_at"}).
			AddRow(id, email, "+919876543210", "hashed_pass", "patient", nil, true, now, now)

		mock.ExpectQuery("SELECT (.+) FROM users WHERE LOWER\\(email\\) = LOWER\\(\\$1\\)").
			WithArgs(email).
			WillReturnRows(rows)

		user, err := repo.GetUserByEmail(ctx, email)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user == nil || user.ID != id || user.Email != email {
			t.Fatalf("user mismatch: got %+v", user)
		}
	})

	t.Run("GetUserByEmail_NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM users WHERE LOWER\\(email\\) = LOWER\\(\\$1\\)").
			WithArgs("notfound@example.com").
			WillReturnError(sql.ErrNoRows)

		user, err := repo.GetUserByEmail(ctx, "notfound@example.com")
		if err != nil {
			t.Fatalf("expected nil error on not found, got %v", err)
		}
		if user != nil {
			t.Fatalf("expected nil user, got %+v", user)
		}
	})

	t.Run("GetUserByID_Found", func(t *testing.T) {
		id := uuid.New().String()
		now := time.Now().UTC()

		rows := sqlmock.NewRows([]string{"id", "email", "phone", "password_hash", "role", "clinic_id", "is_active", "created_at", "updated_at"}).
			AddRow(id, "test@example.com", "+919876543210", "hashed_pass", "doctor", "clinic-123", true, now, now)

		mock.ExpectQuery("SELECT (.+) FROM users WHERE id = \\$1").
			WithArgs(id).
			WillReturnRows(rows)

		user, err := repo.GetUserByID(ctx, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user == nil || user.ID != id || *user.ClinicID != "clinic-123" {
			t.Fatalf("user mismatch: got %+v", user)
		}
	})

	t.Run("UpdateUserStatus", func(t *testing.T) {
		id := uuid.New().String()

		mock.ExpectExec("UPDATE users SET is_active = \\$1, updated_at = CURRENT_TIMESTAMP WHERE id = \\$2").
			WithArgs(false, id).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateUserStatus(ctx, id, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("PatientProfileOperations", func(t *testing.T) {
		profile := &domain.PatientProfile{
			ID:                    uuid.New().String(),
			UserID:                uuid.New().String(),
			FullName:              "Ayurveda Seeker",
			DateOfBirth:           "1990-05-15",
			BiologicalSex:         "female",
			BloodGroup:            "B+",
			EmergencyContactName:  "Jane Doe",
			EmergencyContactPhone: "+919876543211",
			AddressStreet:         "123 Herbal Way",
			AddressCity:           "Varanasi",
			AddressState:          "Uttar Pradesh",
			AddressPostalCode:     "221001",
			CreatedAt:             time.Now().UTC(),
			UpdatedAt:             time.Now().UTC(),
		}

		mock.ExpectExec("INSERT INTO patient_profiles").
			WithArgs(
				profile.ID, profile.UserID, profile.FullName, profile.DateOfBirth,
				profile.BiologicalSex, profile.BloodGroup,
				profile.EmergencyContactName, profile.EmergencyContactPhone,
				profile.AddressStreet, profile.AddressCity, profile.AddressState, profile.AddressPostalCode,
				profile.CreatedAt, profile.UpdatedAt,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.CreatePatientProfile(ctx, profile)
		if err != nil {
			t.Fatalf("unexpected error inserting profile: %v", err)
		}

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "full_name", "date_of_birth", "biological_sex", "blood_group",
			"emergency_contact_name", "emergency_contact_phone",
			"address_street", "address_city", "address_state", "address_postal_code",
			"created_at", "updated_at",
		}).AddRow(
			profile.ID, profile.UserID, profile.FullName, profile.DateOfBirth, profile.BiologicalSex, profile.BloodGroup,
			profile.EmergencyContactName, profile.EmergencyContactPhone,
			profile.AddressStreet, profile.AddressCity, profile.AddressState, profile.AddressPostalCode,
			profile.CreatedAt, profile.UpdatedAt,
		)

		mock.ExpectQuery("SELECT (.+) FROM patient_profiles WHERE user_id = \\$1").
			WithArgs(profile.UserID).
			WillReturnRows(rows)

		fetched, err := repo.GetPatientProfile(ctx, profile.UserID)
		if err != nil {
			t.Fatalf("unexpected error fetching profile: %v", err)
		}
		if fetched.FullName != profile.FullName || fetched.BloodGroup != "B+" {
			t.Fatalf("profile mismatch: got %+v", fetched)
		}

		mock.ExpectExec("UPDATE patient_profiles").
			WithArgs(
				profile.FullName, profile.DateOfBirth, profile.BiologicalSex, profile.BloodGroup,
				profile.EmergencyContactName, profile.EmergencyContactPhone,
				profile.AddressStreet, profile.AddressCity, profile.AddressState, profile.AddressPostalCode,
				profile.UserID,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.UpdatePatientProfile(ctx, profile)
		if err != nil {
			t.Fatalf("unexpected error updating profile: %v", err)
		}
	})

	t.Run("RefreshSessionOperations", func(t *testing.T) {
		session := &domain.RefreshSession{
			ID:        uuid.New().String(),
			UserID:    uuid.New().String(),
			TokenHash: "dummy_sha256_hash",
			FamilyID:  uuid.New().String(),
			IsRevoked: false,
			UserAgent: "Mozilla/5.0",
			IPAddress: "127.0.0.1",
			ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
			CreatedAt: time.Now().UTC(),
		}

		mock.ExpectExec("INSERT INTO refresh_sessions").
			WithArgs(session.ID, session.UserID, session.TokenHash, session.FamilyID,
				session.IsRevoked, session.UserAgent, session.IPAddress, session.ExpiresAt, session.CreatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.CreateRefreshSession(ctx, session)
		if err != nil {
			t.Fatalf("unexpected error creating session: %v", err)
		}

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "token_hash", "family_id", "is_revoked", "user_agent", "ip_address", "expires_at", "created_at",
		}).AddRow(
			session.ID, session.UserID, session.TokenHash, session.FamilyID, session.IsRevoked, session.UserAgent, session.IPAddress, session.ExpiresAt, session.CreatedAt,
		)

		mock.ExpectQuery("SELECT (.+) FROM refresh_sessions WHERE token_hash = \\$1").
			WithArgs(session.TokenHash).
			WillReturnRows(rows)

		fetchedSession, err := repo.GetRefreshSessionByHash(ctx, session.TokenHash)
		if err != nil {
			t.Fatalf("unexpected error getting session: %v", err)
		}
		if fetchedSession.ID != session.ID || fetchedSession.FamilyID != session.FamilyID {
			t.Fatalf("session mismatch: got %+v", fetchedSession)
		}

		mock.ExpectExec("UPDATE refresh_sessions SET is_revoked = TRUE WHERE id = \\$1").
			WithArgs(session.ID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.RevokeRefreshSession(ctx, session.ID)
		if err != nil {
			t.Fatalf("unexpected error revoking session: %v", err)
		}

		mock.ExpectExec("UPDATE refresh_sessions SET is_revoked = TRUE WHERE family_id = \\$1").
			WithArgs(session.FamilyID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.RevokeRefreshFamily(ctx, session.FamilyID)
		if err != nil {
			t.Fatalf("unexpected error revoking family: %v", err)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
