package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/audit"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
)

// RegisterPatientRequest captures public patient sign-up details.
type RegisterPatientRequest struct {
	Email                 string `json:"email"`
	Phone                 string `json:"phone"`
	Password              string `json:"password"`
	Role                  string `json:"role,omitempty"` // Strictly checked: non-patient causes 403 Forbidden
	FullName              string `json:"full_name"`
	DateOfBirth           string `json:"date_of_birth"`
	BiologicalSex         string `json:"biological_sex"`
	BloodGroup            string `json:"blood_group,omitempty"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone"`
	AddressStreet         string `json:"address_street,omitempty"`
	AddressCity           string `json:"address_city,omitempty"`
	AddressState          string `json:"address_state,omitempty"`
	AddressPostalCode     string `json:"address_postal_code,omitempty"`
}

// ProvisionStaffRequest captures authorized staff creation details.
type ProvisionStaffRequest struct {
	Email    string      `json:"email"`
	Phone    string      `json:"phone"`
	Password string      `json:"password"`
	Role     domain.Role `json:"role"`
	ClinicID *string     `json:"clinic_id,omitempty"`
	FullName string      `json:"full_name"`
}

// LoginRequest captures credentials for authentication.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshTokenRequest captures token rotation request.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutRequest captures token logout and revocation request.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

// UpdateProfileRequest captures patient demographic updates.
type UpdateProfileRequest struct {
	FullName              string `json:"full_name"`
	DateOfBirth           string `json:"date_of_birth"`
	BiologicalSex         string `json:"biological_sex"`
	BloodGroup            string `json:"blood_group,omitempty"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone"`
	AddressStreet         string `json:"address_street,omitempty"`
	AddressCity           string `json:"address_city,omitempty"`
	AddressState          string `json:"address_state,omitempty"`
	AddressPostalCode     string `json:"address_postal_code,omitempty"`
}

// AuthResponse returns the issued token pair and user information.
type AuthResponse struct {
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	TokenType    string                 `json:"token_type"`
	ExpiresIn    int64                  `json:"expires_in"`
	User         *domain.User           `json:"user"`
	Profile      *domain.PatientProfile `json:"profile,omitempty"`
}

type AuthService interface {
	RegisterPatient(ctx context.Context, req *RegisterPatientRequest, clientIP, userAgent string) (*AuthResponse, error)
	ProvisionStaff(ctx context.Context, actorID, actorRole, actorClinicID string, req *ProvisionStaffRequest, clientIP, userAgent string) (*domain.User, error)
	Login(ctx context.Context, email, password, clientIP, userAgent string) (*AuthResponse, error)
	RefreshToken(ctx context.Context, rawRefreshToken, clientIP, userAgent string) (*AuthResponse, error)
	Logout(ctx context.Context, userID, jti, rawRefreshToken, clientIP, userAgent string) error
	GetProfile(ctx context.Context, userID string) (*domain.PatientProfile, error)
	UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*domain.PatientProfile, error)
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	UpdateUserStatus(ctx context.Context, actorRole string, targetUserID string, isActive bool, clientIP, userAgent string) error
	ValidateToken(ctx context.Context, tokenStr string) (*auth.Claims, error)
	GetPatientSummary(ctx context.Context, patientID string) (*domain.PatientProfile, error)
}

type authService struct {
	repo repository.UserRepository
	tm   *auth.TokenManager
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(repo repository.UserRepository, tm *auth.TokenManager) AuthService {
	return &authService{
		repo: repo,
		tm:   tm,
	}
}

// RegisterPatient registers a new patient, strictly rejecting any privilege escalation.
func (s *authService) RegisterPatient(ctx context.Context, req *RegisterPatientRequest, clientIP, userAgent string) (*AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, errors.BadRequest("Email and password are required", nil)
	}

	// Security Rule: Public registration must never allow users to grant themselves staff or administrator roles
	if req.Role != "" && req.Role != string(domain.RolePatient) {
		audit.LogSecurityEvent(ctx, audit.SecurityAuditEvent{
			Action:      audit.ActionUnauthorizedAccess,
			IPAddress:   clientIP,
			UserAgent:   userAgent,
			Description: fmt.Sprintf("Privilege escalation rejected: public user attempted to register as %s", req.Role),
			Success:     false,
		})
		return nil, errors.Forbidden("Public registration cannot grant staff or administrator roles", nil)
	}

	if req.FullName == "" {
		return nil, errors.BadRequest("Full name is required for patient registration", nil)
	}
	if req.DateOfBirth == "" || req.BiologicalSex == "" {
		return nil, errors.BadRequest("Date of birth and biological sex are required", nil)
	}
	if req.EmergencyContactName == "" || req.EmergencyContactPhone == "" {
		return nil, errors.BadRequest("Emergency contact details are required", nil)
	}

	// Check if user already exists
	existing, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.Internal("Database query failed", err)
	}
	if existing != nil {
		return nil, errors.Conflict("USER_EXISTS", "A user with this email already exists", nil)
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, errors.Internal("Failed to hash password", err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: passwordHash,
		Role:         domain.RolePatient,
		ClinicID:     nil,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, errors.Internal("Failed to persist user", err)
	}

	profile := &domain.PatientProfile{
		ID:                    uuid.New().String(),
		UserID:                user.ID,
		FullName:              req.FullName,
		DateOfBirth:           req.DateOfBirth,
		BiologicalSex:         req.BiologicalSex,
		BloodGroup:            req.BloodGroup,
		EmergencyContactName:  req.EmergencyContactName,
		EmergencyContactPhone: req.EmergencyContactPhone,
		AddressStreet:         req.AddressStreet,
		AddressCity:           req.AddressCity,
		AddressState:          req.AddressState,
		AddressPostalCode:     req.AddressPostalCode,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := s.repo.CreatePatientProfile(ctx, profile); err != nil {
		return nil, errors.Internal("Failed to create patient profile", err)
	}

	// Generate JWT access token and opaque refresh token
	accessToken, _, _, err := s.tm.GenerateAccessToken(user.ID, user.Email, string(user.Role), "")
	if err != nil {
		return nil, errors.Internal("Failed to generate access token", err)
	}

	rawRefreshToken, tokenHash, err := s.tm.GenerateRefreshToken()
	if err != nil {
		return nil, errors.Internal("Failed to generate refresh token", err)
	}

	familyID := uuid.New().String()
	session := &domain.RefreshSession{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		FamilyID:  familyID,
		IsRevoked: false,
		UserAgent: userAgent,
		IPAddress: clientIP,
		ExpiresAt: now.Add(s.tm.GetRefreshTTL()),
		CreatedAt: now,
	}

	if err := s.repo.CreateRefreshSession(ctx, session); err != nil {
		return nil, errors.Internal("Failed to record refresh session", err)
	}

	audit.LogSecurityEvent(ctx, audit.SecurityAuditEvent{
		Action:      audit.ActionUserRegistered,
		ActorID:     user.ID,
		TargetID:    user.ID,
		Role:        string(user.Role),
		IPAddress:   clientIP,
		UserAgent:   userAgent,
		Description: "Patient registered successfully",
		Success:     true,
	})

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.tm.GetAccessTTL().Seconds()),
		User:         user,
		Profile:      profile,
	}, nil
}

// ProvisionStaff securely provisions doctors, therapists, or administrators.
func (s *authService) ProvisionStaff(ctx context.Context, actorID, actorRole, actorClinicID string, req *ProvisionStaffRequest, clientIP, userAgent string) (*domain.User, error) {
	if actorRole != string(domain.RoleSuperAdmin) && actorRole != string(domain.RoleClinicAdmin) {
		audit.LogSecurityEvent(ctx, audit.SecurityAuditEvent{
			Action:      audit.ActionUnauthorizedAccess,
			ActorID:     actorID,
			Role:        actorRole,
			IPAddress:   clientIP,
			UserAgent:   userAgent,
			Description: "Unauthorized attempt to provision staff",
			Success:     false,
		})
		return nil, errors.Forbidden("Only clinic administrators and platform super administrators can provision staff", nil)
	}

	if req.Email == "" || req.Password == "" || req.Role == "" {
		return nil, errors.BadRequest("Email, password, and role are required", nil)
	}

	// RBAC hierarchy rules
	if actorRole == string(domain.RoleClinicAdmin) {
		if req.Role == domain.RoleSuperAdmin || req.Role == domain.RoleClinicAdmin {
			return nil, errors.Forbidden("Clinic administrators can only provision therapists and doctors", nil)
		}
		if actorClinicID == "" {
			return nil, errors.Forbidden("Clinic administrator must be associated with a clinic", nil)
		}
		// Force staff to actor's clinic
		req.ClinicID = &actorClinicID
	} else if actorRole == string(domain.RoleSuperAdmin) {
		if req.Role != domain.RoleSuperAdmin && req.ClinicID == nil {
			return nil, errors.BadRequest("Clinic ID is required for clinic staff provisioning", nil)
		}
	}

	existing, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.Internal("Database lookup failed", err)
	}
	if existing != nil {
		return nil, errors.Conflict("USER_EXISTS", "A user with this email already exists", nil)
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, errors.Internal("Failed to hash password", err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: passwordHash,
		Role:         req.Role,
		ClinicID:     req.ClinicID,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, errors.Internal("Failed to persist staff user", err)
	}

	clinicIDStr := ""
	if user.ClinicID != nil {
		clinicIDStr = *user.ClinicID
	}

	audit.LogSecurityEvent(ctx, audit.SecurityAuditEvent{
		Action:      audit.ActionStaffProvisioned,
		ActorID:     actorID,
		TargetID:    user.ID,
		Role:        string(user.Role),
		ClinicID:    clinicIDStr,
		IPAddress:   clientIP,
		UserAgent:   userAgent,
		Description: fmt.Sprintf("Staff member provisioned with role %s", user.Role),
		Success:     true,
	})

	return user, nil
}

// Login authenticates a user with email and password, returning tokens.
func (s *authService) Login(ctx context.Context, email, password, clientIP, userAgent string) (*AuthResponse, error) {
	if email == "" || password == "" {
		return nil, errors.BadRequest("Email and password are required", nil)
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.Internal("Database query failed", err)
	}

	if user == nil || !auth.CheckPasswordHash(password, user.PasswordHash) {
		audit.LogSecurityEvent(ctx, audit.SecurityAuditEvent{
			Action:      audit.ActionLoginFailed,
			IPAddress:   clientIP,
			UserAgent:   userAgent,
			Description: fmt.Sprintf("Failed login attempt for email: %s", email),
			Success:     false,
		})
		return nil, errors.Unauthorized("Invalid email or password", nil)
	}

	if !user.IsActive {
		audit.LogSecurityEvent(ctx, audit.SecurityAuditEvent{
			Action:      audit.ActionLoginFailed,
			ActorID:     user.ID,
			Role:        string(user.Role),
			IPAddress:   clientIP,
			UserAgent:   userAgent,
			Description: "Login rejected: account is disabled",
			Success:     false,
		})
		return nil, errors.Forbidden("Account has been disabled. Please contact clinic administration.", nil)
	}

	clinicIDStr := ""
	if user.ClinicID != nil {
		clinicIDStr = *user.ClinicID
	}

	accessToken, _, _, err := s.tm.GenerateAccessToken(user.ID, user.Email, string(user.Role), clinicIDStr)
	if err != nil {
		return nil, errors.Internal("Failed to generate access token", err)
	}

	rawRefreshToken, tokenHash, err := s.tm.GenerateRefreshToken()
	if err != nil {
		return nil, errors.Internal("Failed to generate refresh token", err)
	}

	familyID := uuid.New().String()
	now := time.Now().UTC()
	session := &domain.RefreshSession{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		FamilyID:  familyID,
		IsRevoked: false,
		UserAgent: userAgent,
		IPAddress: clientIP,
		ExpiresAt: now.Add(s.tm.GetRefreshTTL()),
		CreatedAt: now,
	}

	if err := s.repo.CreateRefreshSession(ctx, session); err != nil {
		return nil, errors.Internal("Failed to record refresh session", err)
	}

	var profile *domain.PatientProfile
	if user.Role == domain.RolePatient {
		profile, _ = s.repo.GetPatientProfile(ctx, user.ID)
	}

	audit.LogSecurityEvent(ctx, audit.SecurityAuditEvent{
		Action:      audit.ActionLoginSuccess,
		ActorID:     user.ID,
		TargetID:    user.ID,
		Role:        string(user.Role),
		ClinicID:    clinicIDStr,
		IPAddress:   clientIP,
		UserAgent:   userAgent,
		Description: "User logged in successfully",
		Success:     true,
	})

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.tm.GetAccessTTL().Seconds()),
		User:         user,
		Profile:      profile,
	}, nil
}

// RefreshToken validates the refresh token, performs rotation, and detects replay attacks.
func (s *authService) RefreshToken(ctx context.Context, rawRefreshToken, clientIP, userAgent string) (*AuthResponse, error) {
	if rawRefreshToken == "" {
		return nil, errors.BadRequest("Refresh token is required", nil)
	}

	tokenHash := auth.HashToken(rawRefreshToken)
	session, err := s.repo.GetRefreshSessionByHash(ctx, tokenHash)
	if err != nil {
		return nil, errors.Internal("Database query failed", err)
	}
	if session == nil {
		return nil, errors.Unauthorized("Invalid refresh token", nil)
	}

	// Replay Detection: If session is already revoked, revoke the ENTIRE token family!
	if session.IsRevoked {
		_ = s.repo.RevokeRefreshFamily(ctx, session.FamilyID)
		audit.LogSecurityEvent(ctx, audit.SecurityAuditEvent{
			Action:      audit.ActionTokenReplayDetected,
			ActorID:     session.UserID,
			TargetID:    session.UserID,
			IPAddress:   clientIP,
			UserAgent:   userAgent,
			Description: fmt.Sprintf("Refresh token reuse detected for family %s; all tokens revoked", session.FamilyID),
			Success:     false,
		})
		return nil, errors.Unauthorized("Refresh token reuse detected. All sessions in this chain have been revoked for your security.", nil)
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		return nil, errors.Unauthorized("Refresh token has expired", nil)
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, errors.Internal("Failed to verify user account", err)
	}
	if user == nil || !user.IsActive {
		return nil, errors.Forbidden("User account is inactive or not found", nil)
	}

	// Revoke old session (Rotation)
	if err := s.repo.RevokeRefreshSession(ctx, session.ID); err != nil {
		return nil, errors.Internal("Failed to revoke previous session", err)
	}

	// Issue new refresh token within SAME family
	newRawRefreshToken, newTokenHash, err := s.tm.GenerateRefreshToken()
	if err != nil {
		return nil, errors.Internal("Failed to generate rotated refresh token", err)
	}

	now := time.Now().UTC()
	newSession := &domain.RefreshSession{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TokenHash: newTokenHash,
		FamilyID:  session.FamilyID,
		IsRevoked: false,
		UserAgent: userAgent,
		IPAddress: clientIP,
		ExpiresAt: now.Add(s.tm.GetRefreshTTL()),
		CreatedAt: now,
	}

	if err := s.repo.CreateRefreshSession(ctx, newSession); err != nil {
		return nil, errors.Internal("Failed to record rotated session", err)
	}

	clinicIDStr := ""
	if user.ClinicID != nil {
		clinicIDStr = *user.ClinicID
	}

	newAccessToken, _, _, err := s.tm.GenerateAccessToken(user.ID, user.Email, string(user.Role), clinicIDStr)
	if err != nil {
		return nil, errors.Internal("Failed to generate rotated access token", err)
	}

	audit.LogSecurityEvent(ctx, audit.SecurityAuditEvent{
		Action:      audit.ActionTokenRotated,
		ActorID:     user.ID,
		TargetID:    user.ID,
		Role:        string(user.Role),
		IPAddress:   clientIP,
		UserAgent:   userAgent,
		Description: "Refresh token successfully rotated",
		Success:     true,
	})

	return &AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.tm.GetAccessTTL().Seconds()),
		User:         user,
	}, nil
}

// Logout blacklists the access token JTI and revokes the refresh token.
func (s *authService) Logout(ctx context.Context, userID, jti, rawRefreshToken, clientIP, userAgent string) error {
	// Blacklist access token JTI for remaining TTL
	if jti != "" {
		_ = s.tm.BlacklistToken(ctx, jti, s.tm.GetAccessTTL())
	}

	// Revoke refresh token if provided
	if rawRefreshToken != "" {
		tokenHash := auth.HashToken(rawRefreshToken)
		session, err := s.repo.GetRefreshSessionByHash(ctx, tokenHash)
		if err == nil && session != nil {
			_ = s.repo.RevokeRefreshSession(ctx, session.ID)
		}
	}

	audit.LogSecurityEvent(ctx, audit.SecurityAuditEvent{
		Action:      audit.ActionLogout,
		ActorID:     userID,
		TargetID:    userID,
		IPAddress:   clientIP,
		UserAgent:   userAgent,
		Description: "User logged out successfully",
		Success:     true,
	})

	return nil
}

// GetProfile retrieves the patient profile.
func (s *authService) GetProfile(ctx context.Context, userID string) (*domain.PatientProfile, error) {
	profile, err := s.repo.GetPatientProfile(ctx, userID)
	if err != nil {
		return nil, errors.Internal("Database query failed", err)
	}
	if profile == nil {
		return nil, errors.NotFound("Patient profile not found", nil)
	}
	return profile, nil
}

// UpdateProfile updates demographic details for a patient.
func (s *authService) UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*domain.PatientProfile, error) {
	existing, err := s.repo.GetPatientProfile(ctx, userID)
	if err != nil {
		return nil, errors.Internal("Database query failed", err)
	}
	if existing == nil {
		return nil, errors.NotFound("Patient profile not found", nil)
	}

	if req.FullName != "" {
		existing.FullName = req.FullName
	}
	if req.DateOfBirth != "" {
		existing.DateOfBirth = req.DateOfBirth
	}
	if req.BiologicalSex != "" {
		existing.BiologicalSex = req.BiologicalSex
	}
	if req.BloodGroup != "" {
		existing.BloodGroup = req.BloodGroup
	}
	if req.EmergencyContactName != "" {
		existing.EmergencyContactName = req.EmergencyContactName
	}
	if req.EmergencyContactPhone != "" {
		existing.EmergencyContactPhone = req.EmergencyContactPhone
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
	existing.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdatePatientProfile(ctx, existing); err != nil {
		return nil, errors.Internal("Failed to update patient profile", err)
	}

	return existing, nil
}

// GetUserByID looks up a user account by UUID.
func (s *authService) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, errors.Internal("Database lookup failed", err)
	}
	if user == nil {
		return nil, errors.NotFound("User not found", nil)
	}
	return user, nil
}

// UpdateUserStatus enables or disables an account.
func (s *authService) UpdateUserStatus(ctx context.Context, actorRole string, targetUserID string, isActive bool, clientIP, userAgent string) error {
	if actorRole != string(domain.RoleSuperAdmin) && actorRole != string(domain.RoleClinicAdmin) {
		return errors.Forbidden("Insufficient permissions to update user status", nil)
	}

	if err := s.repo.UpdateUserStatus(ctx, targetUserID, isActive); err != nil {
		return errors.Internal("Failed to update user status", err)
	}

	action := "ACCOUNT_ACTIVATED"
	if !isActive {
		action = audit.ActionAccountDisabled
	}

	audit.LogSecurityEvent(ctx, audit.SecurityAuditEvent{
		Action:      action,
		TargetID:    targetUserID,
		Role:        actorRole,
		IPAddress:   clientIP,
		UserAgent:   userAgent,
		Description: fmt.Sprintf("User status updated to isActive=%v", isActive),
		Success:     true,
	})

	return nil
}

// ValidateToken verifies token claims and revocation.
func (s *authService) ValidateToken(ctx context.Context, tokenStr string) (*auth.Claims, error) {
	return s.tm.VerifyAccessToken(ctx, tokenStr)
}

// GetPatientSummary fetches minimal patient summary for inter-service gRPC.
func (s *authService) GetPatientSummary(ctx context.Context, patientID string) (*domain.PatientProfile, error) {
	profile, err := s.repo.GetPatientProfile(ctx, patientID)
	if err != nil {
		return nil, errors.Internal("Database lookup failed", err)
	}
	if profile == nil {
		return nil, errors.NotFound("Patient profile not found", nil)
	}
	return profile, nil
}
