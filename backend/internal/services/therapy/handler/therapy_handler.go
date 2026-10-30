package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	pkgErrors "github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/middleware"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/response"
)

// TherapyHTTPHandler handles REST operations for the Therapy Service.
type TherapyHTTPHandler struct {
	svc          service.TherapyService
	tokenManager *auth.TokenManager
}

// NewTherapyHTTPHandler initializes the Therapy HTTP Handler.
func NewTherapyHTTPHandler(svc service.TherapyService, tm *auth.TokenManager) *TherapyHTTPHandler {
	return &TherapyHTTPHandler{
		svc:          svc,
		tokenManager: tm,
	}
}

// RegisterRoutes mounts therapy execution and monitoring endpoints onto Gin router group.
func (h *TherapyHTTPHandler) RegisterRoutes(rg *gin.RouterGroup) {
	sessions := rg.Group("/sessions")
	if h.tokenManager != nil {
		sessions.Use(middleware.AuthRequired(h.tokenManager))
	}
	{
		// Check-in & readiness verification
		sessions.POST("/check-in", middleware.RequireRoles("therapist", "clinic_admin", "super_admin"), h.CheckIn)

		// Therapist assigned sessions & patient sessions
		sessions.GET("/therapist/assigned", middleware.RequireRoles("therapist", "super_admin"), h.ListAssignedSessions)
		sessions.GET("/patient/me", middleware.RequireRoles("patient", "super_admin"), h.ListPatientSessions)

		// Session details
		sessions.GET("/:id", h.GetSession)

		// Session execution lifecycle
		sessions.POST("/:id/start", middleware.RequireRoles("therapist", "super_admin"), h.StartSession)
		sessions.POST("/:id/complete", middleware.RequireRoles("therapist", "super_admin"), h.CompleteSession)

		// Ordered steps execution
		sessions.POST("/:id/steps/:stepOrder/start", middleware.RequireRoles("therapist", "super_admin"), h.StartStep)
		sessions.POST("/:id/steps/:stepOrder/pause", middleware.RequireRoles("therapist", "super_admin"), h.PauseStep)
		sessions.POST("/:id/steps/:stepOrder/resume", middleware.RequireRoles("therapist", "super_admin"), h.ResumeStep)
		sessions.POST("/:id/steps/:stepOrder/complete", middleware.RequireRoles("therapist", "super_admin"), h.CompleteStep)

		// Materials used
		sessions.POST("/:id/materials", middleware.RequireRoles("therapist", "clinic_admin", "super_admin"), h.RecordMaterial)

		// Complications
		sessions.POST("/:id/complications", middleware.RequireRoles("therapist", "patient", "doctor", "clinic_admin", "super_admin"), h.FlagComplication)
		sessions.POST("/:id/complications/:compId/resolve", middleware.RequireRoles("doctor", "super_admin"), h.ResolveComplication)

		// Follow-ups
		sessions.POST("/:id/follow-ups", middleware.RequireRoles("therapist", "doctor", "super_admin"), h.CreateFollowUp)
		sessions.POST("/:id/follow-ups/:followUpId/link", middleware.RequireRoles("therapist", "doctor", "clinic_admin", "super_admin"), h.LinkFollowUpAppointment)

		// Clinical corrections with audit log
		sessions.PUT("/:id/notes", middleware.RequireRoles("therapist", "clinic_admin", "super_admin"), h.CorrectTherapistNotes)
	}
}

// Check-in Request Payload
type checkInDTO struct {
	AppointmentID        string                `json:"appointment_id" binding:"required"`
	ClinicID             string                `json:"clinic_id"`
	TherapistID          string                `json:"therapist_id"`
	BaselineVitals       domain.BaselineVitals `json:"baseline_vitals"`
	PreChecklistVerified bool                  `json:"pre_checklist_verified"`
	PreChecklistData     json.RawMessage       `json:"pre_checklist_data"`
}

func (h *TherapyHTTPHandler) CheckIn(c *gin.Context) {
	var dto checkInDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Error(c, pkgErrors.BadRequest("Invalid check-in request payload", err))
		return
	}

	clinicID := dto.ClinicID
	if clinicID == "" {
		clinicID = h.getClinicID(c)
	}
	therapistID := dto.TherapistID
	if therapistID == "" {
		therapistID = h.getUserID(c)
	}

	req := service.CheckInRequest{
		AppointmentID:        dto.AppointmentID,
		ClinicID:             clinicID,
		TherapistID:          therapistID,
		BaselineVitals:       dto.BaselineVitals,
		PreChecklistVerified: dto.PreChecklistVerified,
		PreChecklistData:     dto.PreChecklistData,
	}

	session, err := h.svc.CheckInAppointment(c.Request.Context(), req)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.Created(c, session)
}

func (h *TherapyHTTPHandler) StartSession(c *gin.Context) {
	sessionID := c.Param("id")
	therapistID := h.getUserID(c)

	session, err := h.svc.StartSession(c.Request.Context(), sessionID, therapistID)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.OK(c, session)
}

func (h *TherapyHTTPHandler) StartStep(c *gin.Context) {
	sessionID := c.Param("id")
	stepOrder, err := strconv.Atoi(c.Param("stepOrder"))
	if err != nil || stepOrder <= 0 {
		response.Error(c, pkgErrors.BadRequest("Valid stepOrder integer is required", err))
		return
	}
	therapistID := h.getUserID(c)

	if err := h.svc.StartStep(c.Request.Context(), sessionID, stepOrder, therapistID); err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.OK(c, gin.H{"status": "IN_PROGRESS", "step_order": stepOrder})
}

func (h *TherapyHTTPHandler) PauseStep(c *gin.Context) {
	sessionID := c.Param("id")
	stepOrder, err := strconv.Atoi(c.Param("stepOrder"))
	if err != nil || stepOrder <= 0 {
		response.Error(c, pkgErrors.BadRequest("Valid stepOrder integer is required", err))
		return
	}
	therapistID := h.getUserID(c)

	if err := h.svc.PauseStep(c.Request.Context(), sessionID, stepOrder, therapistID); err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.OK(c, gin.H{"status": "PAUSED", "step_order": stepOrder})
}

func (h *TherapyHTTPHandler) ResumeStep(c *gin.Context) {
	sessionID := c.Param("id")
	stepOrder, err := strconv.Atoi(c.Param("stepOrder"))
	if err != nil || stepOrder <= 0 {
		response.Error(c, pkgErrors.BadRequest("Valid stepOrder integer is required", err))
		return
	}
	therapistID := h.getUserID(c)

	if err := h.svc.ResumeStep(c.Request.Context(), sessionID, stepOrder, therapistID); err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.OK(c, gin.H{"status": "IN_PROGRESS", "step_order": stepOrder})
}

type completeStepDTO struct {
	ActualDurationSeconds int    `json:"actual_duration_seconds"`
	TherapistNotes        string `json:"therapist_notes"`
	Observations          string `json:"observations"`
}

func (h *TherapyHTTPHandler) CompleteStep(c *gin.Context) {
	sessionID := c.Param("id")
	stepOrder, err := strconv.Atoi(c.Param("stepOrder"))
	if err != nil || stepOrder <= 0 {
		response.Error(c, pkgErrors.BadRequest("Valid stepOrder integer is required", err))
		return
	}

	var dto completeStepDTO
	_ = c.ShouldBindJSON(&dto)
	therapistID := h.getUserID(c)

	if err := h.svc.CompleteStep(c.Request.Context(), sessionID, stepOrder, therapistID, dto.ActualDurationSeconds, dto.TherapistNotes, dto.Observations); err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.OK(c, gin.H{"status": "COMPLETED", "step_order": stepOrder})
}

type recordMaterialDTO struct {
	MaterialName string  `json:"material_name" binding:"required"`
	Quantity     float64 `json:"quantity" binding:"required"`
	Unit         string  `json:"unit" binding:"required"`
	BatchNumber  string  `json:"batch_number"`
	Notes        string  `json:"notes"`
}

func (h *TherapyHTTPHandler) RecordMaterial(c *gin.Context) {
	sessionID := c.Param("id")
	var dto recordMaterialDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Error(c, pkgErrors.BadRequest("Invalid material record payload", err))
		return
	}

	therapistID := h.getUserID(c)
	req := service.RecordMaterialRequest{
		MaterialName: dto.MaterialName,
		Quantity:     dto.Quantity,
		Unit:         dto.Unit,
		BatchNumber:  dto.BatchNumber,
		Notes:        dto.Notes,
	}

	mat, err := h.svc.RecordMaterial(c.Request.Context(), sessionID, therapistID, req)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.Created(c, mat)
}

type completeSessionDTO struct {
	TherapistNotes        string          `json:"therapist_notes"`
	Observations          string          `json:"observations"`
	PostCareInstructions string          `json:"post_care_instructions"`
	PostCareItems         json.RawMessage `json:"post_care_items"`
}

func (h *TherapyHTTPHandler) CompleteSession(c *gin.Context) {
	sessionID := c.Param("id")
	var dto completeSessionDTO
	_ = c.ShouldBindJSON(&dto)

	therapistID := h.getUserID(c)
	session, err := h.svc.CompleteSession(c.Request.Context(), sessionID, therapistID, dto.TherapistNotes, dto.Observations, dto.PostCareInstructions, dto.PostCareItems)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.OK(c, session)
}

type flagComplicationDTO struct {
	Severity              domain.ComplicationSeverity `json:"severity" binding:"required"`
	Symptoms              string                      `json:"symptoms" binding:"required"`
	VitalsAtIncident      domain.BaselineVitals       `json:"vitals_at_incident"`
	ImmediateActionsTaken string                      `json:"immediate_actions_taken"`
	AttendingDoctorID     string                      `json:"attending_doctor_id"`
}

func (h *TherapyHTTPHandler) FlagComplication(c *gin.Context) {
	sessionID := c.Param("id")
	var dto flagComplicationDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Error(c, pkgErrors.BadRequest("Invalid complication report payload", err))
		return
	}

	reporterID := h.getUserID(c)
	role := h.getUserRole(c)

	req := service.FlagComplicationRequest{
		Severity:              dto.Severity,
		Symptoms:              dto.Symptoms,
		VitalsAtIncident:      dto.VitalsAtIncident,
		ImmediateActionsTaken: dto.ImmediateActionsTaken,
		AttendingDoctorID:     dto.AttendingDoctorID,
	}

	comp, err := h.svc.FlagComplication(c.Request.Context(), sessionID, reporterID, role, req)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.Created(c, comp)
}

type resolveComplicationDTO struct {
	DoctorNotes  string               `json:"doctor_notes" binding:"required"`
	ResumeStatus domain.SessionStatus `json:"resume_status"`
}

func (h *TherapyHTTPHandler) ResolveComplication(c *gin.Context) {
	compID := c.Param("compId")
	var dto resolveComplicationDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Error(c, pkgErrors.BadRequest("Invalid resolve complication payload", err))
		return
	}

	doctorID := h.getUserID(c)
	if err := h.svc.ResolveComplication(c.Request.Context(), compID, doctorID, dto.DoctorNotes, dto.ResumeStatus); err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.OK(c, gin.H{"status": "RESOLVED", "complication_id": compID})
}

type createFollowUpDTO struct {
	RecommendedPackageID   *string `json:"recommended_package_id"`
	RecommendedIntervalDay *int    `json:"recommended_interval_days"`
	RecommendedDate        *string `json:"recommended_date"`
	Instructions           string  `json:"instructions" binding:"required"`
}

func (h *TherapyHTTPHandler) CreateFollowUp(c *gin.Context) {
	sessionID := c.Param("id")
	var dto createFollowUpDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Error(c, pkgErrors.BadRequest("Invalid follow-up payload", err))
		return
	}

	authorID := h.getUserID(c)
	role := h.getUserRole(c)

	req := service.CreateFollowUpRequest{
		RecommendedPackageID:   dto.RecommendedPackageID,
		RecommendedIntervalDay: dto.RecommendedIntervalDay,
		RecommendedDate:        dto.RecommendedDate,
		Instructions:           dto.Instructions,
	}

	followUp, err := h.svc.CreateFollowUp(c.Request.Context(), sessionID, authorID, role, req)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.Created(c, followUp)
}

type linkFollowUpDTO struct {
	AppointmentID string `json:"appointment_id" binding:"required"`
}

func (h *TherapyHTTPHandler) LinkFollowUpAppointment(c *gin.Context) {
	followUpID := c.Param("followUpId")
	var dto linkFollowUpDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Error(c, pkgErrors.BadRequest("Appointment ID is required to link follow-up", err))
		return
	}

	if err := h.svc.LinkFollowUpAppointment(c.Request.Context(), followUpID, dto.AppointmentID); err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.OK(c, gin.H{"status": "LINKED", "follow_up_id": followUpID, "appointment_id": dto.AppointmentID})
}

type correctNotesDTO struct {
	TherapistNotes string `json:"therapist_notes" binding:"required"`
	Reason         string `json:"reason" binding:"required"`
}

func (h *TherapyHTTPHandler) CorrectTherapistNotes(c *gin.Context) {
	sessionID := c.Param("id")
	var dto correctNotesDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.Error(c, pkgErrors.BadRequest("Clinical note and justification reason are required", err))
		return
	}

	correctedBy := h.getUserID(c)
	if err := h.svc.CorrectTherapistNotes(c.Request.Context(), sessionID, correctedBy, dto.TherapistNotes, dto.Reason); err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.OK(c, gin.H{"status": "CORRECTED", "session_id": sessionID})
}

func (h *TherapyHTTPHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("id")
	requesterID := h.getUserID(c)
	role := h.getUserRole(c)
	clinicID := h.getClinicID(c)

	session, err := h.svc.GetSession(c.Request.Context(), sessionID, requesterID, role, clinicID)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.OK(c, session)
}

func (h *TherapyHTTPHandler) ListAssignedSessions(c *gin.Context) {
	therapistID := h.getUserID(c)
	clinicID := h.getClinicID(c)

	var statusPtr *domain.SessionStatus
	if st := c.Query("status"); st != "" {
		s := domain.SessionStatus(st)
		statusPtr = &s
	}

	sessions, err := h.svc.ListAssignedSessionsForTherapist(c.Request.Context(), therapistID, clinicID, statusPtr)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.OK(c, sessions)
}

func (h *TherapyHTTPHandler) ListPatientSessions(c *gin.Context) {
	patientID := h.getUserID(c)
	sessions, err := h.svc.ListPatientSessions(c.Request.Context(), patientID)
	if err != nil {
		h.handleDomainError(c, err)
		return
	}

	response.OK(c, sessions)
}

func (h *TherapyHTTPHandler) handleDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrPreChecklistRequired):
		response.Error(c, pkgErrors.BadRequest(err.Error(), err))
	case errors.Is(err, domain.ErrDuplicateActiveSession):
		response.Error(c, pkgErrors.Conflict(pkgErrors.CodeConflict, err.Error(), err))
	case errors.Is(err, domain.ErrSessionNotFound), errors.Is(err, domain.ErrStepNotFound), errors.Is(err, domain.ErrComplicationNotFound):
		response.Error(c, pkgErrors.NotFound(err.Error(), err))
	case errors.Is(err, domain.ErrStepOutOfOrder), errors.Is(err, domain.ErrSessionCompleted), errors.Is(err, domain.ErrIncompleteSteps), errors.Is(err, domain.ErrInvalidSessionTransition), errors.Is(err, domain.ErrInvalidStepTransition):
		response.Error(c, pkgErrors.Conflict(pkgErrors.CodeConflict, err.Error(), err))
	case errors.Is(err, domain.ErrUnauthorizedAccess), errors.Is(err, domain.ErrUnauthorizedCrossClinic), errors.Is(err, domain.ErrUnauthorizedPatientAccess):
		response.Error(c, pkgErrors.Forbidden(err.Error(), err))
	case errors.Is(err, domain.ErrCorrectionReasonRequired):
		response.Error(c, pkgErrors.BadRequest(err.Error(), err))
	default:
		response.Error(c, pkgErrors.Internal(fmt.Sprintf("Internal therapy execution error: %v", err), err))
	}
}

func (h *TherapyHTTPHandler) getUserID(c *gin.Context) string {
	if val, exists := c.Get(middleware.ContextUserIDKey); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return "anonymous"
}

func (h *TherapyHTTPHandler) getUserRole(c *gin.Context) string {
	if val, exists := c.Get(middleware.ContextUserRoleKey); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return "patient"
}

func (h *TherapyHTTPHandler) getClinicID(c *gin.Context) string {
	if val, exists := c.Get(middleware.ContextClinicIDKey); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}
