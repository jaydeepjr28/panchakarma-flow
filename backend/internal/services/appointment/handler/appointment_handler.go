package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/repository"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	pkgErrors "github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/middleware"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/response"
)

// AppointmentHTTPHandler routes HTTP requests to the appointment service.
type AppointmentHTTPHandler struct {
	svc          service.AppointmentService
	tokenManager *auth.TokenManager
}

// NewAppointmentHTTPHandler initializes the HTTP handler.
func NewAppointmentHTTPHandler(svc service.AppointmentService, tm *auth.TokenManager) *AppointmentHTTPHandler {
	return &AppointmentHTTPHandler{
		svc:          svc,
		tokenManager: tm,
	}
}

// RegisterRoutes mounts all appointment REST endpoints onto the Gin router group.
func (h *AppointmentHTTPHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Public or Authenticated Availability Search
	rg.GET("/availability", h.SearchAvailability)

	// Protected Bookings and Holds
	bookings := rg.Group("/bookings")
	if h.tokenManager != nil {
		bookings.Use(middleware.AuthRequired(h.tokenManager))
	}
	{
		bookings.POST("/hold", middleware.RequireRoles("patient", "clinic_admin", "super_admin"), h.CreateHold)
		bookings.POST("/confirm", middleware.RequireRoles("patient", "clinic_admin", "super_admin"), h.ConfirmAppointment)
		bookings.POST("/direct", middleware.RequireRoles("patient", "clinic_admin", "super_admin"), h.BookDirect)
		bookings.POST("/package", middleware.RequireRoles("patient", "clinic_admin", "super_admin"), h.BookMultiSessionPackage)
	}

	// Protected Appointments Management
	appts := rg.Group("/appointments")
	if h.tokenManager != nil {
		appts.Use(middleware.AuthRequired(h.tokenManager))
	}
	{
		appts.GET("", h.ListAppointments)
		appts.GET("/:id", h.GetAppointment)
		appts.POST("/:id/cancel", middleware.RequireRoles("patient", "doctor", "clinic_admin", "super_admin"), h.CancelAppointment)
		appts.POST("/:id/reschedule", middleware.RequireRoles("patient", "clinic_admin", "super_admin"), h.RescheduleAppointment)
	}

	// Protected Waitlist
	waitlists := rg.Group("/waitlists")
	if h.tokenManager != nil {
		waitlists.Use(middleware.AuthRequired(h.tokenManager))
	}
	{
		waitlists.POST("", middleware.RequireRoles("patient", "clinic_admin", "super_admin"), h.JoinWaitlist)
		waitlists.GET("/me", middleware.RequireRoles("patient"), h.GetMyWaitlists)
		waitlists.POST("/:id/accept", middleware.RequireRoles("patient"), h.AcceptWaitlistOffer)
	}
}

func (h *AppointmentHTTPHandler) SearchAvailability(c *gin.Context) {
	clinicID := c.Query("clinic_id")
	if clinicID == "" {
		response.Error(c, pkgErrors.BadRequest("clinic_id query parameter is required", nil))
		return
	}

	req := service.SearchSlotsRequest{
		ClinicID:  clinicID,
		PackageID: c.Query("package_id"),
		DoctorID:  c.Query("doctor_id"),
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
	}

	result, err := h.svc.SearchAvailableSlots(c.Request.Context(), req)
	if err != nil {
		response.Error(c, pkgErrors.Internal("Failed searching slot availability", err))
		return
	}

	response.OK(c, result)
}

func (h *AppointmentHTTPHandler) CreateHold(c *gin.Context) {
	var body struct {
		ClinicID         string    `json:"clinic_id" binding:"required"`
		PackageID        string    `json:"package_id" binding:"required"`
		DoctorID         string    `json:"doctor_id"`
		RoomID           string    `json:"room_id"`
		AutoAssignDoctor bool      `json:"auto_assign_doctor"`
		StartTime        time.Time `json:"start_time" binding:"required"`
		DurationMinutes  int       `json:"duration_minutes"`
		SessionNumber    int       `json:"session_number"`
		TotalSessions    int       `json:"total_sessions"`
		PackageBookingID *string   `json:"package_booking_id"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, pkgErrors.BadRequest("Invalid request payload", err))
		return
	}

	patientID := h.getUserID(c)
	req := service.CreateHoldRequest{
		PatientID:        patientID,
		ClinicID:         body.ClinicID,
		DoctorID:         body.DoctorID,
		RoomID:           body.RoomID,
		PackageID:        body.PackageID,
		AutoAssignDoctor: body.AutoAssignDoctor,
		StartTime:        body.StartTime,
		DurationMinutes:  body.DurationMinutes,
		SessionNumber:    body.SessionNumber,
		TotalSessions:    body.TotalSessions,
		PackageBookingID: body.PackageBookingID,
	}

	resp, err := h.svc.CreateHold(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrSlotDoubleBooked) {
			response.Error(c, pkgErrors.Conflict(pkgErrors.CodeSlotAlreadyHeld, "This time slot is already held or booked", err))
			return
		}
		if strings.Contains(err.Error(), "slot rejected by clinic validation:") {
			response.Error(c, pkgErrors.BadRequest(err.Error(), err))
			return
		}
		response.Error(c, pkgErrors.Internal("Failed to acquire hold", err))
		return
	}

	response.Created(c, resp)
}

func (h *AppointmentHTTPHandler) ConfirmAppointment(c *gin.Context) {
	var body struct {
		HoldToken       string `json:"hold_token" binding:"required"`
		ConsentRecordID string `json:"consent_record_id"`
		Notes           string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, pkgErrors.BadRequest("Invalid confirmation payload", err))
		return
	}

	idempKey := c.GetHeader("X-Idempotency-Key")
	var idempPtr *string
	if idempKey != "" {
		idempPtr = &idempKey
	}

	patientID := h.getUserID(c)
	req := service.ConfirmAppointmentRequest{
		HoldToken:       body.HoldToken,
		PatientID:       patientID,
		ConsentRecordID: body.ConsentRecordID,
		Notes:           body.Notes,
		IdempotencyKey:  idempPtr,
	}

	appt, err := h.svc.ConfirmAppointment(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrHoldExpired) {
			response.Error(c, pkgErrors.Conflict(pkgErrors.CodeConflict, "Hold has expired, please choose a slot again", err))
			return
		}
		if errors.Is(err, domain.ErrHoldNotFound) {
			response.Error(c, pkgErrors.NotFound("Hold not found or already consumed", err))
			return
		}
		if errors.Is(err, domain.ErrSlotDoubleBooked) {
			response.Error(c, pkgErrors.Conflict(pkgErrors.CodeDoubleBooking, "Doctor or room is already booked for this slot", err))
			return
		}
		if errors.Is(err, domain.ErrUnauthorizedAccess) {
			response.Error(c, pkgErrors.Forbidden("Unauthorized to confirm this hold", err))
			return
		}
		response.Error(c, pkgErrors.Internal("Failed to confirm appointment", err))
		return
	}

	response.OK(c, appt)
}

func (h *AppointmentHTTPHandler) BookDirect(c *gin.Context) {
	var body struct {
		ClinicID         string    `json:"clinic_id" binding:"required"`
		DoctorID         string    `json:"doctor_id"`
		RoomID           string    `json:"room_id"`
		PackageID        string    `json:"package_id" binding:"required"`
		AutoAssignDoctor bool      `json:"auto_assign_doctor"`
		StartTime        time.Time `json:"start_time" binding:"required"`
		DurationMinutes  int       `json:"duration_minutes"`
		Notes            string    `json:"notes"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, pkgErrors.BadRequest("Invalid direct booking payload", err))
		return
	}

	idempKey := c.GetHeader("X-Idempotency-Key")
	var idempPtr *string
	if idempKey != "" {
		idempPtr = &idempKey
	}

	patientID := h.getUserID(c)
	req := service.DirectBookingRequest{
		PatientID:        patientID,
		ClinicID:         body.ClinicID,
		DoctorID:         body.DoctorID,
		RoomID:           body.RoomID,
		PackageID:        body.PackageID,
		AutoAssignDoctor: body.AutoAssignDoctor,
		StartTime:        body.StartTime,
		DurationMinutes:  body.DurationMinutes,
		Notes:            body.Notes,
		IdempotencyKey:   idempPtr,
	}

	appt, err := h.svc.BookDirectAppointment(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrSlotDoubleBooked) {
			response.Error(c, pkgErrors.Conflict(pkgErrors.CodeDoubleBooking, "Doctor or room is already booked for this slot", err))
			return
		}
		if strings.Contains(err.Error(), "slot rejected by clinic validation:") {
			response.Error(c, pkgErrors.BadRequest(err.Error(), err))
			return
		}
		response.Error(c, pkgErrors.Internal(err.Error(), err))
		return
	}

	response.Created(c, appt)
}

func (h *AppointmentHTTPHandler) BookMultiSessionPackage(c *gin.Context) {
	var body struct {
		ClinicID        string      `json:"clinic_id" binding:"required"`
		PackageID       string      `json:"package_id" binding:"required"`
		PreferredDoctor string      `json:"preferred_doctor_id"`
		SessionStarts   []time.Time `json:"session_starts" binding:"required"`
		DurationMinutes int         `json:"duration_minutes"`
		Notes           string      `json:"notes"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, pkgErrors.BadRequest("Invalid multi-session booking payload", err))
		return
	}

	patientID := h.getUserID(c)
	req := service.MultiSessionPackageRequest{
		PatientID:       patientID,
		ClinicID:        body.ClinicID,
		PackageID:       body.PackageID,
		PreferredDoctor: body.PreferredDoctor,
		SessionStarts:   body.SessionStarts,
		DurationMinutes: body.DurationMinutes,
		Notes:           body.Notes,
	}

	appts, err := h.svc.BookMultiSessionPackage(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrSlotDoubleBooked) {
			response.Error(c, pkgErrors.Conflict(pkgErrors.CodeDoubleBooking, "One or more requested session slots are already booked", err))
			return
		}
		response.Error(c, pkgErrors.Internal("Failed booking multi-session package", err))
		return
	}

	response.Created(c, gin.H{
		"package_booking_id": appts[0].PackageBookingID,
		"total_sessions":     len(appts),
		"appointments":       appts,
	})
}

func (h *AppointmentHTTPHandler) ListAppointments(c *gin.Context) {
	userID := h.getUserID(c)
	role := h.getUserRole(c)
	clinicID := h.getClinicID(c)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := repository.ListFilter{
		ClinicID: c.Query("clinic_id"),
		Limit:    limit,
		Offset:   offset,
	}

	if statusStr := c.Query("status"); statusStr != "" {
		st := domain.AppointmentStatus(statusStr)
		filter.Status = &st
	}

	appts, total, err := h.svc.ListAppointments(c.Request.Context(), filter, userID, role, clinicID)
	if err != nil {
		response.Error(c, pkgErrors.Internal("Failed listing appointments", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    appts,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

func (h *AppointmentHTTPHandler) GetAppointment(c *gin.Context) {
	apptID := c.Param("id")
	userID := h.getUserID(c)
	role := h.getUserRole(c)
	clinicID := h.getClinicID(c)

	appt, err := h.svc.GetAppointment(c.Request.Context(), apptID, userID, role, clinicID)
	if err != nil {
		if errors.Is(err, domain.ErrAppointmentNotFound) {
			response.Error(c, pkgErrors.NotFound("Appointment not found", err))
			return
		}
		if errors.Is(err, domain.ErrUnauthorizedAccess) {
			response.Error(c, pkgErrors.Forbidden("Unauthorized to view this appointment", err))
			return
		}
		response.Error(c, pkgErrors.Internal("Failed to retrieve appointment", err))
		return
	}

	response.OK(c, appt)
}

func (h *AppointmentHTTPHandler) CancelAppointment(c *gin.Context) {
	apptID := c.Param("id")
	userID := h.getUserID(c)
	role := h.getUserRole(c)
	clinicID := h.getClinicID(c)

	var body struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, pkgErrors.BadRequest("Cancellation reason is required", err))
		return
	}

	err := h.svc.CancelAppointment(c.Request.Context(), apptID, userID, role, clinicID, body.Reason)
	if err != nil {
		if errors.Is(err, domain.ErrAppointmentNotFound) {
			response.Error(c, pkgErrors.NotFound("Appointment not found", err))
			return
		}
		if errors.Is(err, domain.ErrUnauthorizedAccess) {
			response.Error(c, pkgErrors.Forbidden("Unauthorized to cancel this appointment", err))
			return
		}
		if errors.Is(err, domain.ErrInvalidStateTransition) {
			response.Error(c, pkgErrors.New(http.StatusUnprocessableEntity, "INVALID_STATE", "Cannot cancel appointment in its current state", err))
			return
		}
		response.Error(c, pkgErrors.Internal("Failed to cancel appointment", err))
		return
	}

	response.OK(c, gin.H{"message": "Appointment cancelled successfully"})
}

func (h *AppointmentHTTPHandler) RescheduleAppointment(c *gin.Context) {
	apptID := c.Param("id")
	userID := h.getUserID(c)
	role := h.getUserRole(c)
	clinicID := h.getClinicID(c)

	var body struct {
		NewStartTime    time.Time `json:"new_start_time" binding:"required"`
		DurationMinutes int       `json:"duration_minutes"`
		NewDoctorID     string    `json:"new_doctor_id"`
		NewRoomID       string    `json:"new_room_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, pkgErrors.BadRequest("new_start_time is required", err))
		return
	}

	req := service.RescheduleRequest{
		AppointmentID:   apptID,
		RequesterID:     userID,
		RequesterRole:   role,
		ClinicID:        clinicID,
		NewStartTime:    body.NewStartTime,
		DurationMinutes: body.DurationMinutes,
		NewDoctorID:     body.NewDoctorID,
		NewRoomID:       body.NewRoomID,
	}

	newAppt, err := h.svc.RescheduleAppointment(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrSlotDoubleBooked) {
			response.Error(c, pkgErrors.Conflict(pkgErrors.CodeDoubleBooking, "Target slot is already booked, original appointment remains intact", err))
			return
		}
		if errors.Is(err, domain.ErrUnauthorizedAccess) {
			response.Error(c, pkgErrors.Forbidden("Unauthorized to reschedule this appointment", err))
			return
		}
		response.Error(c, pkgErrors.Internal("Failed to reschedule appointment", err))
		return
	}

	response.OK(c, newAppt)
}

func (h *AppointmentHTTPHandler) JoinWaitlist(c *gin.Context) {
	var body struct {
		ClinicID           string  `json:"clinic_id" binding:"required"`
		PackageID          string  `json:"package_id" binding:"required"`
		PreferredDoctorID  *string `json:"preferred_doctor_id"`
		PreferredDateStart string  `json:"preferred_date_start" binding:"required"`
		PreferredDateEnd   string  `json:"preferred_date_end" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, pkgErrors.BadRequest("Invalid waitlist payload", err))
		return
	}

	patientID := h.getUserID(c)
	req := service.JoinWaitlistRequest{
		ClinicID:           body.ClinicID,
		PatientID:          patientID,
		PackageID:          body.PackageID,
		PreferredDoctorID:  body.PreferredDoctorID,
		PreferredDateStart: body.PreferredDateStart,
		PreferredDateEnd:   body.PreferredDateEnd,
	}

	entry, err := h.svc.JoinWaitlist(c.Request.Context(), req)
	if err != nil {
		response.Error(c, pkgErrors.Internal("Failed to join waitlist", err))
		return
	}

	response.Created(c, entry)
}

func (h *AppointmentHTTPHandler) GetMyWaitlists(c *gin.Context) {
	patientID := h.getUserID(c)
	clinicID := c.Query("clinic_id")

	entries, err := h.svc.GetWaitlistEntries(c.Request.Context(), clinicID, &patientID, nil)
	if err != nil {
		response.Error(c, pkgErrors.Internal("Failed retrieving waitlists", err))
		return
	}

	response.OK(c, entries)
}

func (h *AppointmentHTTPHandler) AcceptWaitlistOffer(c *gin.Context) {
	waitlistID := c.Param("id")
	patientID := h.getUserID(c)

	appt, err := h.svc.AcceptWaitlistOffer(c.Request.Context(), waitlistID, patientID)
	if err != nil {
		if errors.Is(err, domain.ErrWaitlistOfferExpired) {
			response.Error(c, pkgErrors.Conflict(pkgErrors.CodeConflict, "Waitlist slot offer has expired", err))
			return
		}
		if errors.Is(err, domain.ErrUnauthorizedAccess) {
			response.Error(c, pkgErrors.Forbidden("Unauthorized to accept this offer", err))
			return
		}
		response.Error(c, pkgErrors.Internal("Failed to accept waitlist offer", err))
		return
	}

	response.OK(c, appt)
}

func (h *AppointmentHTTPHandler) getUserID(c *gin.Context) string {
	if val, exists := c.Get(middleware.ContextUserIDKey); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return "anonymous"
}

func (h *AppointmentHTTPHandler) getUserRole(c *gin.Context) string {
	if val, exists := c.Get(middleware.ContextUserRoleKey); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return "patient"
}

func (h *AppointmentHTTPHandler) getClinicID(c *gin.Context) string {
	if val, exists := c.Get(middleware.ContextClinicIDKey); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}
