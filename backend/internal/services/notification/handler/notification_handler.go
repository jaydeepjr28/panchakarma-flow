package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	pkgErrors "github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/middleware"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/response"
)

// NotificationHTTPHandler manages REST APIs for user notifications and settings.
type NotificationHTTPHandler struct {
	svc          service.NotificationService
	tokenManager *auth.TokenManager
}

// NewNotificationHTTPHandler initializes the HTTP handler.
func NewNotificationHTTPHandler(svc service.NotificationService, tm *auth.TokenManager) *NotificationHTTPHandler {
	return &NotificationHTTPHandler{
		svc:          svc,
		tokenManager: tm,
	}
}

// RegisterRoutes registers endpoints on the Gin router.
func (h *NotificationHTTPHandler) RegisterRoutes(rg *gin.RouterGroup) {
	notifs := rg.Group("/notifications")
	if h.tokenManager != nil {
		notifs.Use(middleware.AuthRequired(h.tokenManager))
	}

	{
		notifs.GET("", h.ListNotifications)
		notifs.PUT("/:id/read", h.MarkRead)
		notifs.PUT("/read-all", h.MarkAllRead)
		notifs.GET("/preferences", h.GetPreferences)
		notifs.PUT("/preferences", h.UpdatePreferences)
		notifs.GET("/delivery-logs", h.ListDeliveryLogs)
	}
}

func (h *NotificationHTTPHandler) getUserID(c *gin.Context) string {
	if val, exists := c.Get(middleware.ContextUserIDKey); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	// Fallback to query param for administrative/test queries
	if q := c.Query("user_id"); q != "" {
		return q
	}
	return "anonymous"
}

// ListNotifications handles GET /api/v1/notifications
func (h *NotificationHTTPHandler) ListNotifications(c *gin.Context) {
	userID := h.getUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	list, unreadCount, err := h.svc.ListUserNotifications(c.Request.Context(), userID, limit, offset)
	if err != nil {
		response.Error(c, pkgErrors.Internal("Failed to list notifications", err))
		return
	}

	response.OK(c, gin.H{
		"notifications": list,
		"unread_count":  unreadCount,
		"limit":         limit,
		"offset":        offset,
	})
}

// MarkRead handles PUT /api/v1/notifications/:id/read
func (h *NotificationHTTPHandler) MarkRead(c *gin.Context) {
	id := c.Param("id")
	userID := h.getUserID(c)

	if err := h.svc.MarkNotificationRead(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, domain.ErrNotificationNotFound) {
			response.Error(c, pkgErrors.NotFound("Notification not found", err))
			return
		}
		response.Error(c, pkgErrors.Internal("Failed to mark notification as read", err))
		return
	}

	response.OK(c, gin.H{"status": "READ", "id": id})
}

// MarkAllRead handles PUT /api/v1/notifications/read-all
func (h *NotificationHTTPHandler) MarkAllRead(c *gin.Context) {
	userID := h.getUserID(c)

	if err := h.svc.MarkAllNotificationsRead(c.Request.Context(), userID); err != nil {
		response.Error(c, pkgErrors.Internal("Failed to mark all as read", err))
		return
	}

	response.OK(c, gin.H{"status": "ALL_READ"})
}

// GetPreferences handles GET /api/v1/notifications/preferences
func (h *NotificationHTTPHandler) GetPreferences(c *gin.Context) {
	userID := h.getUserID(c)

	pref, err := h.svc.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, pkgErrors.Internal("Failed to get preferences", err))
		return
	}

	response.OK(c, pref)
}

// UpdatePreferences handles PUT /api/v1/notifications/preferences
func (h *NotificationHTTPHandler) UpdatePreferences(c *gin.Context) {
	userID := h.getUserID(c)

	var body struct {
		EmailEnabled    bool `json:"email_enabled"`
		SMSEnabled      bool `json:"sms_enabled"`
		WhatsAppEnabled bool `json:"whatsapp_enabled"`
		InAppEnabled    bool `json:"in_app_enabled"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, pkgErrors.BadRequest("Invalid preferences payload", err))
		return
	}

	pref := &domain.NotificationPreference{
		UserID:          userID,
		EmailEnabled:    body.EmailEnabled,
		SMSEnabled:      body.SMSEnabled,
		WhatsAppEnabled: body.WhatsAppEnabled,
		InAppEnabled:    body.InAppEnabled,
	}

	if err := h.svc.UpdatePreferences(c.Request.Context(), pref); err != nil {
		response.Error(c, pkgErrors.Internal("Failed to update preferences", err))
		return
	}

	response.OK(c, pref)
}

// ListDeliveryLogs handles GET /api/v1/notifications/delivery-logs
func (h *NotificationHTTPHandler) ListDeliveryLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	attempts, err := h.svc.ListDeliveryAttempts(c.Request.Context(), limit)
	if err != nil {
		response.Error(c, pkgErrors.Internal("Failed to list delivery logs", err))
		return
	}

	response.OK(c, gin.H{
		"attempts": attempts,
		"count":    len(attempts),
	})
}
