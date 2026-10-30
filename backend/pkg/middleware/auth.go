package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/response"
)

const (
	ContextUserIDKey   = "user_id"
	ContextUserRoleKey = "role"
	ContextEmailKey    = "email"
	ContextClinicIDKey = "clinic_id"
	ContextJTIKey      = "jti"
)

// AuthRequired validates the Bearer access token, checks revocation, and sets user identity claims.
func AuthRequired(tm *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, errors.Unauthorized("Authorization header is required", nil))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, errors.Unauthorized("Invalid authorization format, expected 'Bearer <token>'", nil))
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims, err := tm.VerifyAccessToken(c.Request.Context(), tokenStr)
		if err != nil {
			response.Error(c, errors.Unauthorized(err.Error(), err))
			c.Abort()
			return
		}

		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextUserRoleKey, claims.Role)
		c.Set(ContextEmailKey, claims.Email)
		c.Set(ContextClinicIDKey, claims.ClinicID)
		c.Set(ContextJTIKey, claims.ID)

		c.Next()
	}
}

// RequireRoles restricts access to specific user roles.
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(ContextUserRoleKey)
		if !exists {
			response.Error(c, errors.Unauthorized("Authentication required", nil))
			c.Abort()
			return
		}

		role, ok := roleVal.(string)
		if !ok {
			response.Error(c, errors.Forbidden("Invalid role context", nil))
			c.Abort()
			return
		}

		isAllowed := false
		for _, r := range allowedRoles {
			if r == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			response.Error(c, errors.Forbidden("Insufficient permissions for this operation", nil))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireResourceOwnerOrRoles ensures that only the resource owner or users with permitted roles can access.
func RequireResourceOwnerOrRoles(paramName string, allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actorIDVal, exists := c.Get(ContextUserIDKey)
		if !exists {
			response.Error(c, errors.Unauthorized("Authentication required", nil))
			c.Abort()
			return
		}
		actorID, _ := actorIDVal.(string)

		roleVal, _ := c.Get(ContextUserRoleKey)
		role, _ := roleVal.(string)

		targetID := c.Param(paramName)

		// 1. Is owner?
		if targetID == actorID || targetID == "me" {
			c.Next()
			return
		}

		// 2. Has role permission?
		for _, r := range allowedRoles {
			if r == role {
				c.Next()
				return
			}
		}

		response.Error(c, errors.Forbidden("You do not have permission to access or modify this resource", nil))
		c.Abort()
	}
}
