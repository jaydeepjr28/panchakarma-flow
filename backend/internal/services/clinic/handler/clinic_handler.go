package handler

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/clinic/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/auth"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/errors"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/middleware"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/response"
	clinicv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/clinic/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ClinicHTTPHandler struct {
	svc service.ClinicService
	tm  *auth.TokenManager
}

func NewClinicHTTPHandler(svc service.ClinicService, tm *auth.TokenManager) *ClinicHTTPHandler {
	return &ClinicHTTPHandler{
		svc: svc,
		tm:  tm,
	}
}

func (h *ClinicHTTPHandler) RegisterRoutes(rg *gin.RouterGroup) {
	clinics := rg.Group("/clinics")
	{
		// Public browsing
		clinics.GET("", h.ListClinics)
		clinics.GET("/:id", h.GetClinic)
		clinics.GET("/:id/packages", h.ListPackages)
		clinics.GET("/:id/packages/:packageId", h.GetPackage)
		clinics.GET("/:id/doctors", h.ListDoctors)
		clinics.GET("/:id/doctors/:doctorId", h.GetDoctor)
		clinics.GET("/:id/doctors/:doctorId/availability", h.GetDoctorAvailability)

		// Platform Administrator (SuperAdmin)
		clinics.POST("", middleware.AuthRequired(h.tm), middleware.RequireRoles("super_admin"), h.CreateClinic)
		clinics.DELETE("/:id", middleware.AuthRequired(h.tm), middleware.RequireRoles("super_admin"), h.DeactivateClinic)

		// Clinic Management (SuperAdmin or ClinicAdmin scoped to clinic)
		staffAuth := []gin.HandlerFunc{
			middleware.AuthRequired(h.tm),
			middleware.RequireRoles("super_admin", "clinic_admin", "doctor"),
		}

		clinics.PUT("/:id", middleware.AuthRequired(h.tm), middleware.RequireRoles("super_admin", "clinic_admin"), h.UpdateClinic)

		// Treatment Rooms
		clinics.POST("/:id/rooms", append(staffAuth, h.CreateRoom)...)
		clinics.GET("/:id/rooms", append(staffAuth, h.ListRooms)...)
		clinics.GET("/:id/rooms/:roomId", append(staffAuth, h.GetRoom)...)
		clinics.POST("/:id/rooms/:roomId/equipment", append(staffAuth, h.AddRoomEquipment)...)

		// Staff profiles
		clinics.POST("/:id/doctors", middleware.AuthRequired(h.tm), middleware.RequireRoles("super_admin", "clinic_admin"), h.CreateDoctorProfile)
		clinics.POST("/:id/therapists", middleware.AuthRequired(h.tm), middleware.RequireRoles("super_admin", "clinic_admin"), h.CreateTherapistProfile)
		clinics.GET("/:id/therapists", append(staffAuth, h.ListTherapists)...)

		// Schedules & Leaves
		clinics.POST("/:id/doctors/:doctorId/schedules", append(staffAuth, h.CreateSchedule)...)
		clinics.GET("/:id/doctors/:doctorId/schedules", append(staffAuth, h.ListDoctorSchedules)...)
		clinics.DELETE("/:id/doctors/:doctorId/schedules/:scheduleId", append(staffAuth, h.DeleteSchedule)...)
		clinics.POST("/:id/doctors/:doctorId/leaves", append(staffAuth, h.CreateDoctorLeave)...)
		clinics.GET("/:id/doctors/:doctorId/leaves", append(staffAuth, h.ListDoctorLeaves)...)

		// Therapy Packages
		clinics.POST("/:id/packages", append(staffAuth, h.CreatePackage)...)
		clinics.PUT("/:id/packages/:packageId", append(staffAuth, h.UpdatePackage)...)
		clinics.POST("/:id/packages/:packageId/publish", append(staffAuth, h.PublishPackage)...)
		clinics.DELETE("/:id/packages/:packageId", append(staffAuth, h.DeactivatePackage)...)
		clinics.POST("/:id/packages/:packageId/steps", append(staffAuth, h.SavePackageSteps)...)
		clinics.POST("/:id/packages/:packageId/checklists", append(staffAuth, h.SaveCareChecklist)...)

		// Slot & Room validation REST endpoint
		clinics.POST("/:id/availability/validate", h.ValidateSlotAndRoom)
	}
}

func getActorContext(c *gin.Context) (role, clinicID string) {
	if rVal, exists := c.Get(middleware.ContextUserRoleKey); exists {
		role, _ = rVal.(string)
	}
	if cVal, exists := c.Get(middleware.ContextClinicIDKey); exists {
		clinicID, _ = cVal.(string)
	}
	return role, clinicID
}

// --- Clinics ---

func (h *ClinicHTTPHandler) CreateClinic(c *gin.Context) {
	var req service.CreateClinicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	role, _ := getActorContext(c)
	clinic, err := h.svc.CreateClinic(c.Request.Context(), &req, role)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to create clinic", err))
		return
	}

	response.Created(c, clinic)
}

func (h *ClinicHTTPHandler) GetClinic(c *gin.Context) {
	id := c.Param("id")
	clinic, err := h.svc.GetClinic(c.Request.Context(), id)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.NotFound("Clinic not found", err))
		return
	}
	response.OK(c, clinic)
}

func (h *ClinicHTTPHandler) ListClinics(c *gin.Context) {
	city := c.Query("city")
	activeOnly := c.DefaultQuery("active_only", "true") == "true"

	clinics, err := h.svc.ListClinics(c.Request.Context(), city, activeOnly)
	if err != nil {
		response.Error(c, errors.Internal("Failed to list clinics", err))
		return
	}
	response.OK(c, gin.H{"clinics": clinics})
}

func (h *ClinicHTTPHandler) UpdateClinic(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateClinicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	role, clinicID := getActorContext(c)
	clinic, err := h.svc.UpdateClinic(c.Request.Context(), id, &req, role, clinicID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to update clinic", err))
		return
	}
	response.OK(c, clinic)
}

func (h *ClinicHTTPHandler) DeactivateClinic(c *gin.Context) {
	id := c.Param("id")
	role, _ := getActorContext(c)
	if err := h.svc.DeactivateClinic(c.Request.Context(), id, role); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to deactivate clinic", err))
		return
	}
	response.OK(c, gin.H{"message": "Clinic deactivated successfully"})
}

// --- Treatment Rooms ---

func (h *ClinicHTTPHandler) CreateRoom(c *gin.Context) {
	clinicID := c.Param("id")
	var req service.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	role, actorClinicID := getActorContext(c)
	room, err := h.svc.CreateRoom(c.Request.Context(), clinicID, &req, role, actorClinicID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to create room", err))
		return
	}
	response.Created(c, room)
}

func (h *ClinicHTTPHandler) GetRoom(c *gin.Context) {
	roomID := c.Param("roomId")
	room, err := h.svc.GetRoom(c.Request.Context(), roomID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.NotFound("Room not found", err))
		return
	}
	response.OK(c, room)
}

func (h *ClinicHTTPHandler) ListRooms(c *gin.Context) {
	clinicID := c.Param("id")
	rooms, err := h.svc.ListRooms(c.Request.Context(), clinicID)
	if err != nil {
		response.Error(c, errors.Internal("Failed to list rooms", err))
		return
	}
	response.OK(c, gin.H{"rooms": rooms})
}

func (h *ClinicHTTPHandler) AddRoomEquipment(c *gin.Context) {
	clinicID := c.Param("id")
	roomID := c.Param("roomId")
	var req service.AddEquipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	role, actorClinicID := getActorContext(c)
	equip, err := h.svc.AddRoomEquipment(c.Request.Context(), clinicID, roomID, &req, role, actorClinicID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to add room equipment", err))
		return
	}
	response.Created(c, equip)
}

// --- Doctors & Therapists ---

func (h *ClinicHTTPHandler) CreateDoctorProfile(c *gin.Context) {
	clinicID := c.Param("id")
	var req service.CreateDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	role, actorClinicID := getActorContext(c)
	doc, err := h.svc.CreateDoctorProfile(c.Request.Context(), clinicID, &req, role, actorClinicID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to create doctor profile", err))
		return
	}
	response.Created(c, doc)
}

func (h *ClinicHTTPHandler) GetDoctor(c *gin.Context) {
	doctorID := c.Param("doctorId")
	doc, err := h.svc.GetDoctor(c.Request.Context(), doctorID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.NotFound("Doctor not found", err))
		return
	}
	response.OK(c, doc)
}

func (h *ClinicHTTPHandler) ListDoctors(c *gin.Context) {
	clinicID := c.Param("id")
	docs, err := h.svc.ListDoctors(c.Request.Context(), clinicID)
	if err != nil {
		response.Error(c, errors.Internal("Failed to list doctors", err))
		return
	}
	response.OK(c, gin.H{"doctors": docs})
}

func (h *ClinicHTTPHandler) CreateTherapistProfile(c *gin.Context) {
	clinicID := c.Param("id")
	var req service.CreateTherapistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	role, actorClinicID := getActorContext(c)
	therapist, err := h.svc.CreateTherapistProfile(c.Request.Context(), clinicID, &req, role, actorClinicID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to create therapist profile", err))
		return
	}
	response.Created(c, therapist)
}

func (h *ClinicHTTPHandler) ListTherapists(c *gin.Context) {
	clinicID := c.Param("id")
	therapists, err := h.svc.ListTherapists(c.Request.Context(), clinicID)
	if err != nil {
		response.Error(c, errors.Internal("Failed to list therapists", err))
		return
	}
	response.OK(c, gin.H{"therapists": therapists})
}

// --- Schedules & Leaves ---

func (h *ClinicHTTPHandler) CreateSchedule(c *gin.Context) {
	clinicID := c.Param("id")
	doctorID := c.Param("doctorId")
	var req service.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	role, actorClinicID := getActorContext(c)
	sch, err := h.svc.CreateSchedule(c.Request.Context(), clinicID, doctorID, &req, role, actorClinicID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to create schedule", err))
		return
	}
	response.Created(c, sch)
}

func (h *ClinicHTTPHandler) ListDoctorSchedules(c *gin.Context) {
	doctorID := c.Param("doctorId")
	schedules, err := h.svc.ListDoctorSchedules(c.Request.Context(), doctorID)
	if err != nil {
		response.Error(c, errors.Internal("Failed to list doctor schedules", err))
		return
	}
	response.OK(c, gin.H{"schedules": schedules})
}

func (h *ClinicHTTPHandler) DeleteSchedule(c *gin.Context) {
	clinicID := c.Param("id")
	scheduleID := c.Param("scheduleId")
	role, actorClinicID := getActorContext(c)
	if err := h.svc.DeleteSchedule(c.Request.Context(), clinicID, scheduleID, role, actorClinicID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to delete schedule", err))
		return
	}
	response.OK(c, gin.H{"message": "Schedule deleted successfully"})
}

func (h *ClinicHTTPHandler) CreateDoctorLeave(c *gin.Context) {
	clinicID := c.Param("id")
	doctorID := c.Param("doctorId")
	var req service.CreateLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	role, actorClinicID := getActorContext(c)
	leave, err := h.svc.CreateDoctorLeave(c.Request.Context(), clinicID, doctorID, &req, role, actorClinicID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to create leave", err))
		return
	}
	response.Created(c, leave)
}

func (h *ClinicHTTPHandler) ListDoctorLeaves(c *gin.Context) {
	doctorID := c.Param("doctorId")
	leaves, err := h.svc.ListDoctorLeaves(c.Request.Context(), doctorID)
	if err != nil {
		response.Error(c, errors.Internal("Failed to list leaves", err))
		return
	}
	response.OK(c, gin.H{"leaves": leaves})
}

func (h *ClinicHTTPHandler) GetDoctorAvailability(c *gin.Context) {
	doctorID := c.Param("doctorId")
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	avail, err := h.svc.GetDoctorAvailability(c.Request.Context(), doctorID, dateStr)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to compute doctor availability", err))
		return
	}
	response.OK(c, avail)
}

// --- Therapy Packages ---

func (h *ClinicHTTPHandler) CreatePackage(c *gin.Context) {
	clinicID := c.Param("id")
	var req service.CreatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	role, actorClinicID := getActorContext(c)
	pkg, err := h.svc.CreatePackage(c.Request.Context(), clinicID, &req, role, actorClinicID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to create package", err))
		return
	}
	response.Created(c, pkg)
}

func (h *ClinicHTTPHandler) GetPackage(c *gin.Context) {
	packageID := c.Param("packageId")
	pkg, err := h.svc.GetPackage(c.Request.Context(), packageID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.NotFound("Package not found", err))
		return
	}
	response.OK(c, pkg)
}

func (h *ClinicHTTPHandler) ListPackages(c *gin.Context) {
	clinicID := c.Param("id")
	role, actorClinicID := getActorContext(c)
	publishedOnly := true
	if (role == "super_admin") || (role == "clinic_admin" && actorClinicID == clinicID) {
		publishedOnly = false
	}

	packages, err := h.svc.ListPackages(c.Request.Context(), clinicID, publishedOnly)
	if err != nil {
		response.Error(c, errors.Internal("Failed to list packages", err))
		return
	}
	response.OK(c, gin.H{"packages": packages})
}

func (h *ClinicHTTPHandler) UpdatePackage(c *gin.Context) {
	clinicID := c.Param("id")
	packageID := c.Param("packageId")
	var req service.UpdatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid request body", err))
		return
	}

	role, actorClinicID := getActorContext(c)
	pkg, err := h.svc.UpdatePackage(c.Request.Context(), clinicID, packageID, &req, role, actorClinicID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to update package", err))
		return
	}
	response.OK(c, pkg)
}

func (h *ClinicHTTPHandler) PublishPackage(c *gin.Context) {
	clinicID := c.Param("id")
	packageID := c.Param("packageId")
	role, actorClinicID := getActorContext(c)

	if err := h.svc.PublishPackage(c.Request.Context(), clinicID, packageID, role, actorClinicID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to publish package", err))
		return
	}
	response.OK(c, gin.H{"message": "Package published successfully"})
}

func (h *ClinicHTTPHandler) DeactivatePackage(c *gin.Context) {
	clinicID := c.Param("id")
	packageID := c.Param("packageId")
	role, actorClinicID := getActorContext(c)

	if err := h.svc.DeactivatePackage(c.Request.Context(), clinicID, packageID, role, actorClinicID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to deactivate package", err))
		return
	}
	response.OK(c, gin.H{"message": "Package deactivated successfully"})
}

func (h *ClinicHTTPHandler) SavePackageSteps(c *gin.Context) {
	clinicID := c.Param("id")
	packageID := c.Param("packageId")
	var req []service.PackageStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid steps payload, expected array", err))
		return
	}

	role, actorClinicID := getActorContext(c)
	if err := h.svc.SavePackageSteps(c.Request.Context(), clinicID, packageID, req, role, actorClinicID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to configure package steps", err))
		return
	}
	response.OK(c, gin.H{"message": "Package steps saved successfully"})
}

func (h *ClinicHTTPHandler) SaveCareChecklist(c *gin.Context) {
	clinicID := c.Param("id")
	packageID := c.Param("packageId")
	var req service.CareChecklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid checklist payload", err))
		return
	}

	role, actorClinicID := getActorContext(c)
	if err := h.svc.SaveCareChecklist(c.Request.Context(), clinicID, packageID, &req, role, actorClinicID); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
			return
		}
		response.Error(c, errors.Internal("Failed to save care checklist", err))
		return
	}
	response.OK(c, gin.H{"message": "Care checklist saved successfully"})
}

func (h *ClinicHTTPHandler) ValidateSlotAndRoom(c *gin.Context) {
	clinicID := c.Param("id")
	var req struct {
		DoctorID  string `json:"doctor_id"`
		RoomID    string `json:"room_id"`
		StartUnix int64  `json:"start_time_unix"`
		EndUnix   int64  `json:"end_time_unix"`
		PackageID string `json:"package_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.BadRequest("Invalid validation body", err))
		return
	}

	res, err := h.svc.ValidateSlotAndRoom(c.Request.Context(), clinicID, req.DoctorID, req.RoomID, req.StartUnix, req.EndUnix, req.PackageID)
	if err != nil {
		response.Error(c, errors.Internal("Slot validation failed", err))
		return
	}
	response.OK(c, res)
}

// ClinicGRPCHandler implements clinicv1.ClinicServiceServer.
type ClinicGRPCHandler struct {
	clinicv1.UnimplementedClinicServiceServer
	svc service.ClinicService
}

func NewClinicGRPCHandler(svc service.ClinicService) *ClinicGRPCHandler {
	return &ClinicGRPCHandler{svc: svc}
}

func (g *ClinicGRPCHandler) ValidateSlotAndRoom(ctx context.Context, req *clinicv1.ValidateSlotAndRoomRequest) (*clinicv1.ValidateSlotAndRoomResponse, error) {
	if req.ClinicId == "" || req.DoctorId == "" || req.RoomId == "" {
		return nil, status.Error(codes.InvalidArgument, "clinic_id, doctor_id, and room_id are required")
	}

	result, err := g.svc.ValidateSlotAndRoom(ctx, req.ClinicId, req.DoctorId, req.RoomId, req.StartTimeUnix, req.EndTimeUnix, req.PackageId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &clinicv1.ValidateSlotAndRoomResponse{
		IsValid:                  result.IsValid,
		RejectionReason:          result.RejectionReason,
		SanitationBufferMinutes: int32(result.SanitationBufferMinutes),
	}, nil
}

func (g *ClinicGRPCHandler) GetPackageSnapshot(ctx context.Context, req *clinicv1.GetPackageSnapshotRequest) (*clinicv1.GetPackageSnapshotResponse, error) {
	if req.PackageId == "" {
		return nil, status.Error(codes.InvalidArgument, "package_id required")
	}

	pkg, err := g.svc.GetPackageSnapshot(ctx, req.PackageId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "package not found")
	}

	var protoSteps []*clinicv1.PackageStepSnapshot
	for _, s := range pkg.Steps {
		protoSteps = append(protoSteps, &clinicv1.PackageStepSnapshot{
			StepId:            s.ID,
			StepOrder:         int32(s.StepOrder),
			Name:              s.Name,
			DurationMinutes:   int32(s.DurationMinutes),
			RequiredEquipment: s.RequiredEquipment,
		})
	}

	return &clinicv1.GetPackageSnapshotResponse{
		PackageId:            pkg.ID,
		ClinicId:             pkg.ClinicID,
		Name:                 pkg.Name,
		TotalDurationMinutes: int32(pkg.DurationMinutes),
		TotalSessions:        int32(pkg.TotalSessions),
		PriceCents:           int64(pkg.PriceCents),
		Version:              int32(pkg.Version),
		Steps:                protoSteps,
	}, nil
}

func (g *ClinicGRPCHandler) GetDoctorAvailability(ctx context.Context, req *clinicv1.GetDoctorAvailabilityRequest) (*clinicv1.GetDoctorAvailabilityResponse, error) {
	if req.DoctorId == "" {
		return nil, status.Error(codes.InvalidArgument, "doctor_id required")
	}

	dateStr := time.Unix(req.DateUnix, 0).UTC().Format("2006-01-02")
	avail, err := g.svc.GetDoctorAvailability(ctx, req.DoctorId, dateStr)
	if err != nil {
		return nil, status.Error(codes.NotFound, "doctor not found or availability unavailable")
	}

	return &clinicv1.GetDoctorAvailabilityResponse{
		DoctorId:           avail.DoctorID,
		MaxSessionsPerDay:  int32(avail.MaxSessionsPerDay),
		IsOnLeave:          avail.IsOnLeave,
		AvailableTimeSlots: avail.AvailableTimeSlots,
	}, nil
}
