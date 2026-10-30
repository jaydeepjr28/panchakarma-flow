package service_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
)

// inMemoryUserRepo provides a fast, thread-safe mock implementation of UserRepository for business logic testing.
type inMemoryUserRepo struct {
	mu       sync.RWMutex
	users    map[string]*domain.User
	profiles map[string]*domain.PatientProfile
	sessions map[string]*domain.RefreshSession
}

func newInMemoryUserRepo() *inMemoryUserRepo {
	return &inMemoryUserRepo{
		users:    make(map[string]*domain.User),
		profiles: make(map[string]*domain.PatientProfile),
		sessions: make(map[string]*domain.RefreshSession),
	}
}

func (m *inMemoryUserRepo) CreateUser(ctx context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
	return nil
}

func (m *inMemoryUserRepo) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return nil, nil
}

func (m *inMemoryUserRepo) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.users[id], nil
}

func (m *inMemoryUserRepo) UpdateUserStatus(ctx context.Context, id string, isActive bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, exists := m.users[id]; exists {
		u.IsActive = isActive
	}
	return nil
}

func (m *inMemoryUserRepo) CreatePatientProfile(ctx context.Context, profile *domain.PatientProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[profile.UserID] = profile
	return nil
}

func (m *inMemoryUserRepo) GetPatientProfile(ctx context.Context, userID string) (*domain.PatientProfile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.profiles[userID], nil
}

func (m *inMemoryUserRepo) UpdatePatientProfile(ctx context.Context, profile *domain.PatientProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[profile.UserID] = profile
	return nil
}

func (m *inMemoryUserRepo) CreateRefreshSession(ctx context.Context, session *domain.RefreshSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.TokenHash] = session
	return nil
}

func (m *inMemoryUserRepo) GetRefreshSessionByHash(ctx context.Context, tokenHash string) (*domain.RefreshSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[tokenHash], nil
}

func (m *inMemoryUserRepo) RevokeRefreshSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.ID == sessionID {
			s.IsRevoked = true
		}
	}
	return nil
}

func (m *inMemoryUserRepo) RevokeRefreshFamily(ctx context.Context, familyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.FamilyID == familyID {
			s.IsRevoked = true
		}
	}
	return nil
}

func setupTestAuthService(t *testing.T) (service.AuthService, *auth.TokenManager, *inMemoryUserRepo) {
	repo := newInMemoryUserRepo()
	tm, err := auth.NewTokenManager("panchakarma_flow_test_super_secret_key_32_chars", 15*time.Minute, 7*24*time.Hour, nil)
	if err != nil {
		t.Fatalf("failed to init TokenManager: %v", err)
	}
	svc := service.NewAuthService(repo, tm)
	return svc, tm, repo
}

func TestAuthService_PatientRegistration(t *testing.T) {
	svc, _, _ := setupTestAuthService(t)
	ctx := context.Background()

	t.Run("Success_PatientRegistration", func(t *testing.T) {
		req := &service.RegisterPatientRequest{
			Email:                 "patient1@example.com",
			Phone:                 "+919876543210",
			Password:              "StrongPassword123!",
			FullName:              "Rajesh Kumar",
			DateOfBirth:           "1985-04-12",
			BiologicalSex:         "male",
			BloodGroup:            "O+",
			EmergencyContactName:  "Sunita Kumar",
			EmergencyContactPhone: "+919876543211",
		}

		res, err := svc.RegisterPatient(ctx, req, "127.0.0.1", "UnitTestAgent")
		if err != nil {
			t.Fatalf("unexpected registration failure: %v", err)
		}
		if res.AccessToken == "" || res.RefreshToken == "" {
			t.Errorf("expected access and refresh tokens, got empty")
		}
		if res.User.Role != domain.RolePatient {
			t.Errorf("expected role 'patient', got %s", res.User.Role)
		}
		if res.Profile == nil || res.Profile.FullName != "Rajesh Kumar" {
			t.Errorf("profile not populated properly: %+v", res.Profile)
		}
	})

	t.Run("DuplicateRegistration_Rejected", func(t *testing.T) {
		req := &service.RegisterPatientRequest{
			Email:                 "duplicate@example.com",
			Phone:                 "+919876543210",
			Password:              "StrongPassword123!",
			FullName:              "Same Person",
			DateOfBirth:           "1990-01-01",
			BiologicalSex:         "female",
			EmergencyContactName:  "Contact",
			EmergencyContactPhone: "+919876543211",
		}

		_, err := svc.RegisterPatient(ctx, req, "127.0.0.1", "UnitTestAgent")
		if err != nil {
			t.Fatalf("first registration failed: %v", err)
		}

		// Second registration with same email
		_, err = svc.RegisterPatient(ctx, req, "127.0.0.1", "UnitTestAgent")
		if err == nil {
			t.Fatalf("expected duplicate registration error, got nil")
		}
	})

	t.Run("PrivilegeEscalation_StrictlyRejected", func(t *testing.T) {
		// Attempting public registration with staff/admin role
		escalationRoles := []string{"doctor", "therapist", "clinic_admin", "super_admin"}
		for _, targetRole := range escalationRoles {
			req := &service.RegisterPatientRequest{
				Email:                 targetRole + "@example.com",
				Phone:                 "+919876543210",
				Password:              "StrongPassword123!",
				Role:                  targetRole,
				FullName:              "Hacker Man",
				DateOfBirth:           "1990-01-01",
				BiologicalSex:         "male",
				EmergencyContactName:  "Nobody",
				EmergencyContactPhone: "+919876543211",
			}

			_, err := svc.RegisterPatient(ctx, req, "192.168.1.100", "MaliciousBrowser")
			if err == nil {
				t.Fatalf("expected privilege escalation rejection for role %s, got nil", targetRole)
			}
		}
	})
}

func TestAuthService_LoginAndAuthentication(t *testing.T) {
	svc, _, repo := setupTestAuthService(t)
	ctx := context.Background()

	// Seed registered patient
	pwd := "MySecurePass2026!"
	req := &service.RegisterPatientRequest{
		Email:                 "login.test@example.com",
		Phone:                 "+919876543210",
		Password:              pwd,
		FullName:              "Login Tester",
		DateOfBirth:           "1992-06-20",
		BiologicalSex:         "female",
		EmergencyContactName:  "Kin",
		EmergencyContactPhone: "+919876543211",
	}
	registered, err := svc.RegisterPatient(ctx, req, "127.0.0.1", "UnitTestAgent")
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	t.Run("ValidLogin", func(t *testing.T) {
		res, err := svc.Login(ctx, "login.test@example.com", pwd, "127.0.0.1", "UnitTestAgent")
		if err != nil {
			t.Fatalf("unexpected login failure: %v", err)
		}
		if res.AccessToken == "" || res.RefreshToken == "" {
			t.Fatalf("expected non-empty tokens")
		}
		if res.User.ID != registered.User.ID {
			t.Errorf("user ID mismatch")
		}
	})

	t.Run("InvalidPassword", func(t *testing.T) {
		_, err := svc.Login(ctx, "login.test@example.com", "WrongPassword!", "127.0.0.1", "UnitTestAgent")
		if err == nil {
			t.Fatalf("expected login failure for wrong password, got nil")
		}
	})

	t.Run("NonExistentEmail", func(t *testing.T) {
		_, err := svc.Login(ctx, "nobody@example.com", pwd, "127.0.0.1", "UnitTestAgent")
		if err == nil {
			t.Fatalf("expected login failure for non-existent user, got nil")
		}
	})

	t.Run("DisabledAccount_Rejection", func(t *testing.T) {
		// Disable user account
		_ = repo.UpdateUserStatus(ctx, registered.User.ID, false)

		_, err := svc.Login(ctx, "login.test@example.com", pwd, "127.0.0.1", "UnitTestAgent")
		if err == nil {
			t.Fatalf("expected login rejection for disabled account, got nil")
		}

		// Re-enable account
		_ = repo.UpdateUserStatus(ctx, registered.User.ID, true)
	})
}

func TestAuthService_TokenRotationAndReplayDetection(t *testing.T) {
	svc, _, _ := setupTestAuthService(t)
	ctx := context.Background()

	// Register a patient to obtain initial tokens
	req := &service.RegisterPatientRequest{
		Email:                 "rotation@example.com",
		Phone:                 "+919876543210",
		Password:              "Password123!",
		FullName:              "Rotation Tester",
		DateOfBirth:           "1990-01-01",
		BiologicalSex:         "male",
		EmergencyContactName:  "Kin",
		EmergencyContactPhone: "+919876543211",
	}
	authRes, err := svc.RegisterPatient(ctx, req, "127.0.0.1", "UnitTestAgent")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	initialRefreshToken := authRes.RefreshToken

	t.Run("SuccessfulTokenRotation", func(t *testing.T) {
		rotRes, err := svc.RefreshToken(ctx, initialRefreshToken, "127.0.0.1", "UnitTestAgent")
		if err != nil {
			t.Fatalf("token rotation failed: %v", err)
		}
		if rotRes.RefreshToken == initialRefreshToken {
			t.Errorf("expected new refresh token after rotation")
		}
		if rotRes.AccessToken == "" {
			t.Errorf("expected new access token")
		}

		// Replay Attack Test: Try to reuse initialRefreshToken which has now been rotated!
		_, err = svc.RefreshToken(ctx, initialRefreshToken, "127.0.0.1", "AttackerAgent")
		if err == nil {
			t.Fatalf("expected replay detection error when using rotated token, got nil")
		}

		// Verify that entire family was invalidated: rotRes.RefreshToken should also now be revoked!
		_, err = svc.RefreshToken(ctx, rotRes.RefreshToken, "127.0.0.1", "UnitTestAgent")
		if err == nil {
			t.Fatalf("expected family revocation to reject subsequent valid token in chain, got nil")
		}
	})
}

func TestAuthService_Logout(t *testing.T) {
	svc, tm, _ := setupTestAuthService(t)
	ctx := context.Background()

	// Register and login
	req := &service.RegisterPatientRequest{
		Email:                 "logout.test@example.com",
		Phone:                 "+919876543210",
		Password:              "Password123!",
		FullName:              "Logout Tester",
		DateOfBirth:           "1990-01-01",
		BiologicalSex:         "female",
		EmergencyContactName:  "Kin",
		EmergencyContactPhone: "+919876543211",
	}
	res, err := svc.RegisterPatient(ctx, req, "127.0.0.1", "UnitTestAgent")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	claims, err := tm.VerifyAccessToken(ctx, res.AccessToken)
	if err != nil {
		t.Fatalf("token verification failed: %v", err)
	}

	// Logout
	err = svc.Logout(ctx, res.User.ID, claims.ID, res.RefreshToken, "127.0.0.1", "UnitTestAgent")
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	// Verify JWT is now blacklisted
	_, err = tm.VerifyAccessToken(ctx, res.AccessToken)
	if err == nil {
		t.Fatalf("expected access token to be blacklisted after logout, got nil")
	}

	// Verify refresh token is revoked
	_, err = svc.RefreshToken(ctx, res.RefreshToken, "127.0.0.1", "UnitTestAgent")
	if err == nil {
		t.Fatalf("expected refresh token to fail after logout, got nil")
	}
}

func TestAuthService_StaffProvisioning(t *testing.T) {
	svc, _, _ := setupTestAuthService(t)
	ctx := context.Background()

	clinicID := uuid.New().String()

	t.Run("SuperAdmin_CanProvisionClinicAdmin", func(t *testing.T) {
		req := &service.ProvisionStaffRequest{
			Email:    "newadmin@clinic.com",
			Phone:    "+919876543210",
			Password: "Password123!",
			Role:     domain.RoleClinicAdmin,
			ClinicID: &clinicID,
			FullName: "Clinic Admin One",
		}

		user, err := svc.ProvisionStaff(ctx, "super-admin-id", string(domain.RoleSuperAdmin), "", req, "127.0.0.1", "SuperAdminBrowser")
		if err != nil {
			t.Fatalf("failed to provision clinic admin: %v", err)
		}
		if user.Role != domain.RoleClinicAdmin || *user.ClinicID != clinicID {
			t.Errorf("staff mismatch: %+v", user)
		}
	})

	t.Run("ClinicAdmin_CanProvisionTherapist", func(t *testing.T) {
		req := &service.ProvisionStaffRequest{
			Email:    "therapist@clinic.com",
			Phone:    "+919876543210",
			Password: "Password123!",
			Role:     domain.RoleTherapist,
			FullName: "Acupuncture Therapist",
		}

		user, err := svc.ProvisionStaff(ctx, "admin-id", string(domain.RoleClinicAdmin), clinicID, req, "127.0.0.1", "ClinicAdminBrowser")
		if err != nil {
			t.Fatalf("failed to provision therapist: %v", err)
		}
		if user.Role != domain.RoleTherapist || *user.ClinicID != clinicID {
			t.Errorf("staff mismatch: %+v", user)
		}
	})

	t.Run("ClinicAdmin_CannotProvisionSuperAdmin", func(t *testing.T) {
		req := &service.ProvisionStaffRequest{
			Email:    "rogueadmin@clinic.com",
			Phone:    "+919876543210",
			Password: "Password123!",
			Role:     domain.RoleSuperAdmin,
			FullName: "Rogue Super Admin",
		}

		_, err := svc.ProvisionStaff(ctx, "admin-id", string(domain.RoleClinicAdmin), clinicID, req, "127.0.0.1", "ClinicAdminBrowser")
		if err == nil {
			t.Fatalf("expected error when clinic_admin tries to provision super_admin, got nil")
		}
	})

	t.Run("UnauthorizedActor_CannotProvisionStaff", func(t *testing.T) {
		req := &service.ProvisionStaffRequest{
			Email:    "therapist2@clinic.com",
			Phone:    "+919876543210",
			Password: "Password123!",
			Role:     domain.RoleTherapist,
			FullName: "Another Therapist",
		}

		_, err := svc.ProvisionStaff(ctx, "patient-id", string(domain.RolePatient), "", req, "127.0.0.1", "PatientBrowser")
		if err == nil {
			t.Fatalf("expected error when patient tries to provision staff, got nil")
		}
	})
}

func TestAuthService_ProfileManagement(t *testing.T) {
	svc, _, _ := setupTestAuthService(t)
	ctx := context.Background()

	// Register patient
	req := &service.RegisterPatientRequest{
		Email:                 "profile.tester@example.com",
		Phone:                 "+919876543210",
		Password:              "Password123!",
		FullName:              "Initial Name",
		DateOfBirth:           "1990-01-01",
		BiologicalSex:         "male",
		EmergencyContactName:  "Kin",
		EmergencyContactPhone: "+919876543211",
	}
	res, err := svc.RegisterPatient(ctx, req, "127.0.0.1", "UnitTestAgent")
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	t.Run("GetProfile", func(t *testing.T) {
		profile, err := svc.GetProfile(ctx, res.User.ID)
		if err != nil {
			t.Fatalf("get profile failed: %v", err)
		}
		if profile.FullName != "Initial Name" {
			t.Errorf("expected 'Initial Name', got %s", profile.FullName)
		}
	})

	t.Run("UpdateProfile", func(t *testing.T) {
		updateReq := &service.UpdateProfileRequest{
			FullName:          "Updated Full Name",
			AddressCity:       "Haridwar",
			AddressState:      "Uttarakhand",
			AddressPostalCode: "249401",
		}

		updated, err := svc.UpdateProfile(ctx, res.User.ID, updateReq)
		if err != nil {
			t.Fatalf("update profile failed: %v", err)
		}
		if updated.FullName != "Updated Full Name" || updated.AddressCity != "Haridwar" {
			t.Errorf("updated fields mismatch: %+v", updated)
		}
	})
}
