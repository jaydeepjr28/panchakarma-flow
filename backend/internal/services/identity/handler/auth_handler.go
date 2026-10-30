package handler

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/identity/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/middleware"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/response"
	identityv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/identity/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	svc service.AuthService
	tm  *auth.TokenManager
}

func NewAuthHandler(svc service.AuthService, tm *auth.TokenManager) *AuthHandler {
	return &AuthHandler{
		svc: svc,
		tm:  tm,
	}
}

func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Authentication routes
	authGroup := rg.Group("/auth")
	{
		// Public endpoints with rate limiting
		authGroup.POST("/register", middleware.RateLimiter(10, 1*time.Minute), h.Register)
		authGroup.POST("/login", middleware.RateLimiter(5, 1*time.Minute), h.Login)
		authGroup.POST("/refresh", middleware.RateLimiter(20, 1*time.Minute), h.Refresh)

		// Authenticated endpoints
		authGroup.POST("/logout", middleware.AuthRequired(h.tm), h.Logout)
		authGroup.GET("/me", middleware.AuthRequired(h.tm), h.GetMe)
		authGroup.POST("/provision-staff", middleware.AuthRequired(h.tm), middleware.RequireRoles("super_admin", "clinic_admin"), h.ProvisionStaff)
	}

	// User management routes
	usersGroup := rg.Group("/users")
	{
		usersGroup.PATCH("/:id/status", middleware.AuthRequired(h.tm), middleware.RequireRoles("super_admin", "clinic_admin"), h.UpdateUserStatus)
	}

	// Patient profile routes
	patients := rg.Group("/patients")
	{
		patients.GET("/:id/profile", middleware.AuthRequired(h.tm), middleware.RequireResourceOwnerOrRoles("id", "super_admin", "clinic_admin", "doctor"), h.GetProfile)
		patients.PUT("/:id/profile", middleware.AuthRequired(h.tm), middleware.RequireResourceOwnerOrRoles("id", "super_admin", "clinic_admin"), h.UpdateProfile)
		patients.GET("/:id/summary", middleware.AuthRequired(h.tm), middleware.RequireResourceOwnerOrRoles("id", "super_admin", "clinic_admin", "doctor", "therapist"), h.GetPatientSummary)
	}
}

// Register handles public patient sign-up.
func (h *AuthHandler) Register(c *gin.Context) {
	var req service.RegisterPatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	res, err := h.svc.RegisterPatient(c.Request.Context(), &req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Registration failed", err))
		return
	}

	response.Created(c, res)
}

// Login authenticates a user and returns short-lived JWT + refresh token.
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	res, err := h.svc.Login(c.Request.Context(), req.Email, req.Password, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Login failed", err))
		return
	}

	response.OK(c, res)
}

// Refresh rotates the refresh token and returns a new access/refresh pair.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req service.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	res, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Token refresh failed", err))
		return
	}

	response.OK(c, res)
}

// Logout blacklists the current JWT and revokes the associated refresh session.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req service.LogoutRequest
	_ = c.ShouldBindJSON(&req)

	userID, _ := c.Get(middleware.ContextUserIDKey)
	jti, _ := c.Get(middleware.ContextJTIKey)

	userIDStr, _ := userID.(string)
	jtiStr, _ := jti.(string)

	err := h.svc.Logout(c.Request.Context(), userIDStr, jtiStr, req.RefreshToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Logout failed", err))
		return
	}

	response.OK(c, gin.H{
		"message": "Successfully logged out",
	})
}

// GetMe returns current authenticated user information.
func (h *AuthHandler) GetMe(c *gin.Context) {
	userIDVal, _ := c.Get(middleware.ContextUserIDKey)
	userID, _ := userIDVal.(string)

	user, err := h.svc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.NotFound("User not found", err))
		return
	}

	profile, _ := h.svc.GetProfile(c.Request.Context(), userID)

	response.OK(c, gin.H{
		"user":    user,
		"profile": profile,
	})
}

// ProvisionStaff securely creates staff accounts (doctors, therapists, clinic admins).
func (h *AuthHandler) ProvisionStaff(c *gin.Context) {
	var req service.ProvisionStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	actorIDVal, _ := c.Get(middleware.ContextUserIDKey)
	roleVal, _ := c.Get(middleware.ContextUserRoleKey)
	clinicVal, _ := c.Get(middleware.ContextClinicIDKey)

	actorID, _ := actorIDVal.(string)
	actorRole, _ := roleVal.(string)
	actorClinicID, _ := clinicVal.(string)

	user, err := h.svc.ProvisionStaff(c.Request.Context(), actorID, actorRole, actorClinicID, &req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to provision staff", err))
		return
	}

	response.Created(c, gin.H{
		"user": user,
	})
}

// UpdateUserStatus enables or disables a user account.
func (h *AuthHandler) UpdateUserStatus(c *gin.Context) {
	targetID := c.Param("id")
	roleVal, _ := c.Get(middleware.ContextUserRoleKey)
	actorRole, _ := roleVal.(string)

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body, 'is_active' boolean required", err))
		return
	}

	err := h.svc.UpdateUserStatus(c.Request.Context(), actorRole, targetID, req.IsActive, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to update user status", err))
		return
	}

	response.OK(c, gin.H{
		"message":   "User status updated successfully",
		"user_id":   targetID,
		"is_active": req.IsActive,
	})
}

// GetProfile returns a patient's demographic profile.
func (h *AuthHandler) GetProfile(c *gin.Context) {
	targetID := c.Param("id")
	if targetID == "me" {
		actorIDVal, _ := c.Get(middleware.ContextUserIDKey)
		targetID, _ = actorIDVal.(string)
	}

	profile, err := h.svc.GetProfile(c.Request.Context(), targetID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.NotFound("Patient profile not found", err))
		return
	}

	response.OK(c, profile)
}

// UpdateProfile updates a patient's demographic profile.
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	targetID := c.Param("id")
	if targetID == "me" {
		actorIDVal, _ := c.Get(middleware.ContextUserIDKey)
		targetID, _ = actorIDVal.(string)
	}

	var req service.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	profile, err := h.svc.UpdateProfile(c.Request.Context(), targetID, &req)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to update profile", err))
		return
	}

	response.OK(c, profile)
}

// GetPatientSummary returns a minimal patient summary.
func (h *AuthHandler) GetPatientSummary(c *gin.Context) {
	targetID := c.Param("id")
	if targetID == "me" {
		actorIDVal, _ := c.Get(middleware.ContextUserIDKey)
		targetID, _ = actorIDVal.(string)
	}

	profile, err := h.svc.GetPatientSummary(c.Request.Context(), targetID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.NotFound("Patient profile not found", err))
		return
	}

	response.OK(c, profile)
}

// GRPCHandler implements identityv1.IdentityServiceServer.
type GRPCHandler struct {
	identityv1.UnimplementedIdentityServiceServer
	svc service.AuthService
}

func NewGRPCHandler(svc service.AuthService) *GRPCHandler {
	return &GRPCHandler{svc: svc}
}

// ValidateToken verifies token validity, signature, expiration, and revocation blacklist.
func (g *GRPCHandler) ValidateToken(ctx context.Context, req *identityv1.ValidateTokenRequest) (*identityv1.ValidateTokenResponse, error) {
	if req.AccessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "access token required")
	}

	claims, err := g.svc.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return &identityv1.ValidateTokenResponse{
			IsValid: false,
		}, nil
	}

	return &identityv1.ValidateTokenResponse{
		IsValid:  true,
		UserId:   claims.UserID,
		Role:     claims.Role,
		ClinicId: claims.ClinicID,
	}, nil
}

// GetPatientSummary retrieves minimal patient profile for clinical records.
func (g *GRPCHandler) GetPatientSummary(ctx context.Context, req *identityv1.GetPatientSummaryRequest) (*identityv1.GetPatientSummaryResponse, error) {
	if req.PatientId == "" {
		return nil, status.Error(codes.InvalidArgument, "patient id required")
	}

	profile, err := g.svc.GetPatientSummary(ctx, req.PatientId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "patient not found")
	}

	return &identityv1.GetPatientSummaryResponse{
		PatientId:      profile.UserID,
		FullName:       profile.FullName,
		EmergencyPhone: profile.EmergencyContactPhone,
		BloodGroup:     profile.BloodGroup,
	}, nil
}
