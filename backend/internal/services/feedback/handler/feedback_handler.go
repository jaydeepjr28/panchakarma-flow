package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/feedback/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/feedback/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/middleware"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/response"
)

type FeedbackHTTPHandler struct {
	svc          service.FeedbackService
	tokenManager *auth.TokenManager
}

func NewFeedbackHTTPHandler(svc service.FeedbackService, tokenManager *auth.TokenManager) *FeedbackHTTPHandler {
	return &FeedbackHTTPHandler{
		svc:          svc,
		tokenManager: tokenManager,
	}
}

func (h *FeedbackHTTPHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Group with JWT Authentication
	fb := rg.Group("/feedback")
	fb.Use(middleware.AuthRequired(h.tokenManager))
	{
		fb.POST("", h.SubmitFeedback)
		fb.GET("", h.ListFeedback)
		fb.GET("/:id", h.GetFeedback)
		fb.POST("/:id/acknowledge", middleware.RequireRoles("clinic_admin", "doctor", "super_admin"), h.AcknowledgeFeedback)

		// Adverse Events Subgroup
		fb.POST("/adverse-events", h.ReportAdverseEvent)
		fb.GET("/adverse-events", h.ListAdverseEvents)
		fb.GET("/adverse-events/:id", h.GetAdverseEvent)
		fb.POST("/adverse-events/:id/acknowledge", middleware.RequireRoles("clinic_admin", "doctor", "super_admin"), h.AcknowledgeAdverseEvent)
	}

	analytics := rg.Group("/analytics")
	analytics.Use(middleware.AuthRequired(h.tokenManager))
	{
		analytics.GET("/clinics/:id/dashboard", middleware.RequireRoles("clinic_admin", "doctor", "super_admin"), h.GetClinicDashboard)
		analytics.GET("/clinics/:id/daily-metrics", middleware.RequireRoles("clinic_admin", "doctor", "super_admin"), h.GetClinicMetrics)
		analytics.GET("/clinics/:id/export", middleware.RequireRoles("clinic_admin", "doctor", "super_admin"), h.ExportCSV)
		analytics.POST("/reconcile", middleware.RequireRoles("clinic_admin", "super_admin"), h.ReconcileData)
	}
}

func mapAppError(err error) *errors.AppError {
	if err == nil {
		return nil
	}
	if appErr, ok := err.(*errors.AppError); ok {
		return appErr
	}
	return errors.Internal("Internal server error", err)
}

// -----------------------------------------------------------------------------
// PATIENT FEEDBACK HANDLERS
// -----------------------------------------------------------------------------

func (h *FeedbackHTTPHandler) SubmitFeedback(c *gin.Context) {
	actorID, _ := c.Get(middleware.ContextUserIDKey)
	actorRole, _ := c.Get(middleware.ContextUserRoleKey)

	var req domain.SubmitFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid feedback payload", err))
		return
	}

	fb, err := h.svc.SubmitFeedback(c.Request.Context(), actorID.(string), actorRole.(string), req)
	if err != nil {
		response.Error(c, mapAppError(err))
		return
	}

	response.Created(c, fb)
}

func (h *FeedbackHTTPHandler) GetFeedback(c *gin.Context) {
	actorID, _ := c.Get(middleware.ContextUserIDKey)
	actorRole, _ := c.Get(middleware.ContextUserRoleKey)
	actorClinicID, _ := c.Get(middleware.ContextClinicIDKey)
	id := c.Param("id")

	actorClinicStr := ""
	if actorClinicID != nil {
		actorClinicStr = actorClinicID.(string)
	}

	fb, err := h.svc.GetFeedbackByID(c.Request.Context(), actorID.(string), actorRole.(string), actorClinicStr, id)
	if err != nil {
		response.Error(c, mapAppError(err))
		return
	}

	response.OK(c, fb)
}

func (h *FeedbackHTTPHandler) ListFeedback(c *gin.Context) {
	actorID, _ := c.Get(middleware.ContextUserIDKey)
	actorRole, _ := c.Get(middleware.ContextUserRoleKey)
	actorClinicID, _ := c.Get(middleware.ContextClinicIDKey)

	actorClinicStr := ""
	if actorClinicID != nil {
		actorClinicStr = actorClinicID.(string)
	}

	clinicID := c.Query("clinic_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startDate = &t
		}
	}
	if e := c.Query("end_date"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			endDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
			endDate = &endDay
		}
	}

	list, total, err := h.svc.ListFeedback(c.Request.Context(), actorID.(string), actorRole.(string), actorClinicStr, clinicID, startDate, endDate, limit, offset)
	if err != nil {
		response.Error(c, mapAppError(err))
		return
	}

	response.OK(c, gin.H{
		"feedbacks": list,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

func (h *FeedbackHTTPHandler) AcknowledgeFeedback(c *gin.Context) {
	actorID, _ := c.Get(middleware.ContextUserIDKey)
	actorRole, _ := c.Get(middleware.ContextUserRoleKey)
	actorClinicID, _ := c.Get(middleware.ContextClinicIDKey)
	id := c.Param("id")

	actorClinicStr := ""
	if actorClinicID != nil {
		actorClinicStr = actorClinicID.(string)
	}

	var req domain.AcknowledgeFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid acknowledgment payload: staff_notes required", err))
		return
	}

	fb, err := h.svc.AcknowledgeFeedback(c.Request.Context(), actorID.(string), actorRole.(string), actorClinicStr, id, req)
	if err != nil {
		response.Error(c, mapAppError(err))
		return
	}

	response.OK(c, fb)
}

// -----------------------------------------------------------------------------
// ADVERSE EVENTS HANDLERS
// -----------------------------------------------------------------------------

func (h *FeedbackHTTPHandler) ReportAdverseEvent(c *gin.Context) {
	actorID, _ := c.Get(middleware.ContextUserIDKey)
	actorRole, _ := c.Get(middleware.ContextUserRoleKey)
	actorClinicID, _ := c.Get(middleware.ContextClinicIDKey)

	actorClinicStr := ""
	if actorClinicID != nil {
		actorClinicStr = actorClinicID.(string)
	}

	var req domain.ReportAdverseEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid adverse event payload", err))
		return
	}

	ae, err := h.svc.ReportAdverseEvent(c.Request.Context(), actorID.(string), actorRole.(string), actorClinicStr, req)
	if err != nil {
		response.Error(c, mapAppError(err))
		return
	}

	response.Created(c, ae)
}

func (h *FeedbackHTTPHandler) GetAdverseEvent(c *gin.Context) {
	actorID, _ := c.Get(middleware.ContextUserIDKey)
	actorRole, _ := c.Get(middleware.ContextUserRoleKey)
	actorClinicID, _ := c.Get(middleware.ContextClinicIDKey)
	id := c.Param("id")

	actorClinicStr := ""
	if actorClinicID != nil {
		actorClinicStr = actorClinicID.(string)
	}

	ae, err := h.svc.GetAdverseEventByID(c.Request.Context(), actorID.(string), actorRole.(string), actorClinicStr, id)
	if err != nil {
		response.Error(c, mapAppError(err))
		return
	}

	response.OK(c, ae)
}

func (h *FeedbackHTTPHandler) ListAdverseEvents(c *gin.Context) {
	actorID, _ := c.Get(middleware.ContextUserIDKey)
	actorRole, _ := c.Get(middleware.ContextUserRoleKey)
	actorClinicID, _ := c.Get(middleware.ContextClinicIDKey)

	actorClinicStr := ""
	if actorClinicID != nil {
		actorClinicStr = actorClinicID.(string)
	}

	clinicID := c.Query("clinic_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	var severity, status *string
	if s := c.Query("severity"); s != "" {
		severity = &s
	}
	if st := c.Query("status"); st != "" {
		status = &st
	}

	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startDate = &t
		}
	}
	if e := c.Query("end_date"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			endDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
			endDate = &endDay
		}
	}

	list, total, err := h.svc.ListAdverseEvents(c.Request.Context(), actorID.(string), actorRole.(string), actorClinicStr, clinicID, severity, status, startDate, endDate, limit, offset)
	if err != nil {
		response.Error(c, mapAppError(err))
		return
	}

	response.OK(c, gin.H{
		"adverse_events": list,
		"total":          total,
		"limit":          limit,
		"offset":         offset,
	})
}

func (h *FeedbackHTTPHandler) AcknowledgeAdverseEvent(c *gin.Context) {
	actorID, _ := c.Get(middleware.ContextUserIDKey)
	actorRole, _ := c.Get(middleware.ContextUserRoleKey)
	actorClinicID, _ := c.Get(middleware.ContextClinicIDKey)
	id := c.Param("id")

	actorClinicStr := ""
	if actorClinicID != nil {
		actorClinicStr = actorClinicID.(string)
	}

	var req domain.AcknowledgeAdverseEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid acknowledgment payload: staff_notes and status required", err))
		return
	}

	ae, err := h.svc.AcknowledgeAdverseEvent(c.Request.Context(), actorID.(string), actorRole.(string), actorClinicStr, id, req)
	if err != nil {
		response.Error(c, mapAppError(err))
		return
	}

	response.OK(c, ae)
}

// -----------------------------------------------------------------------------
// ANALYTICS & DASHBOARD HANDLERS
// -----------------------------------------------------------------------------

func (h *FeedbackHTTPHandler) GetClinicDashboard(c *gin.Context) {
	actorID, _ := c.Get(middleware.ContextUserIDKey)
	actorRole, _ := c.Get(middleware.ContextUserRoleKey)
	actorClinicID, _ := c.Get(middleware.ContextClinicIDKey)
	clinicID := c.Param("id")

	actorClinicStr := ""
	if actorClinicID != nil {
		actorClinicStr = actorClinicID.(string)
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	tz := c.Query("timezone")

	dash, err := h.svc.GetClinicDashboard(c.Request.Context(), actorID.(string), actorRole.(string), actorClinicStr, clinicID, startDate, endDate, tz)
	if err != nil {
		response.Error(c, mapAppError(err))
		return
	}

	response.OK(c, dash)
}

func (h *FeedbackHTTPHandler) GetClinicMetrics(c *gin.Context) {
	// Maintain backwards compatibility with legacy endpoint
	h.GetClinicDashboard(c)
}

func (h *FeedbackHTTPHandler) ExportCSV(c *gin.Context) {
	actorID, _ := c.Get(middleware.ContextUserIDKey)
	actorRole, _ := c.Get(middleware.ContextUserRoleKey)
	actorClinicID, _ := c.Get(middleware.ContextClinicIDKey)
	clinicID := c.Param("id")

	actorClinicStr := ""
	if actorClinicID != nil {
		actorClinicStr = actorClinicID.(string)
	}

	exportType := c.DefaultQuery("type", "metrics")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	tz := c.Query("timezone")

	filename, csvBytes, err := h.svc.ExportCSV(c.Request.Context(), actorID.(string), actorRole.(string), actorClinicStr, clinicID, exportType, startDate, endDate, tz)
	if err != nil {
		response.Error(c, mapAppError(err))
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", csvBytes)
}

func (h *FeedbackHTTPHandler) ReconcileData(c *gin.Context) {
	var req struct {
		Appointments []*domain.AnalyticsAppointment `json:"appointments"`
		Sessions     []*domain.AnalyticsSession     `json:"sessions"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid reconciliation payload", err))
		return
	}

	if err := h.svc.ReconcileOperationalData(c.Request.Context(), req.Appointments, req.Sessions); err != nil {
		response.Error(c, mapAppError(err))
		return
	}

	response.OK(c, gin.H{
		"status":                 "RECONCILED",
		"appointments_processed": len(req.Appointments),
		"sessions_processed":     len(req.Sessions),
	})
}
