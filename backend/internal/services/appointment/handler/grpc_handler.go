package handler

import (
	"context"
	"errors"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/repository"
	appointmentv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/appointment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AppointmentGRPCHandler implements appointmentv1.AppointmentServiceServer.
type AppointmentGRPCHandler struct {
	appointmentv1.UnimplementedAppointmentServiceServer
	repo repository.AppointmentRepository
}

// NewAppointmentGRPCHandler initializes the appointment gRPC server handler.
func NewAppointmentGRPCHandler(repo repository.AppointmentRepository) *AppointmentGRPCHandler {
	return &AppointmentGRPCHandler{repo: repo}
}

// VerifyAppointmentForSession verifies appointment eligibility before therapist check-in.
func (g *AppointmentGRPCHandler) VerifyAppointmentForSession(ctx context.Context, req *appointmentv1.VerifyAppointmentRequest) (*appointmentv1.VerifyAppointmentResponse, error) {
	if req.AppointmentId == "" {
		return nil, status.Error(codes.InvalidArgument, "appointment_id is required")
	}

	appt, err := g.repo.GetAppointmentByID(ctx, req.AppointmentId)
	if err != nil {
		if errors.Is(err, domain.ErrAppointmentNotFound) {
			return &appointmentv1.VerifyAppointmentResponse{
				IsEligible:      false,
				RejectionReason: "APPOINTMENT_NOT_FOUND",
			}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Verify clinic matches
	if req.ClinicId != "" && appt.ClinicID != req.ClinicId {
		return &appointmentv1.VerifyAppointmentResponse{
			IsEligible:      false,
			RejectionReason: "CLINIC_MISMATCH",
		}, nil
	}

	// Check status: can only check in if CONFIRMED or already CHECKED_IN
	if appt.Status != domain.StatusConfirmed && appt.Status != domain.StatusCheckedIn {
		return &appointmentv1.VerifyAppointmentResponse{
			IsEligible:      false,
			RejectionReason: string(appt.Status),
			PatientId:       appt.PatientID,
			DoctorId:        appt.DoctorID,
			RoomId:          appt.RoomID,
			PackageId:       appt.PackageID,
		}, nil
	}

	return &appointmentv1.VerifyAppointmentResponse{
		IsEligible:         true,
		PatientId:          appt.PatientID,
		DoctorId:           appt.DoctorID,
		RoomId:             appt.RoomID,
		PackageId:          appt.PackageID,
		ScheduledStartUnix: appt.StartTime.Unix(),
		ScheduledEndUnix:   appt.EndTime.Unix(),
	}, nil
}

// GetAppointmentSummary returns high-level summary details for downstream services.
func (g *AppointmentGRPCHandler) GetAppointmentSummary(ctx context.Context, req *appointmentv1.GetAppointmentSummaryRequest) (*appointmentv1.GetAppointmentSummaryResponse, error) {
	if req.AppointmentId == "" {
		return nil, status.Error(codes.InvalidArgument, "appointment_id is required")
	}

	appt, err := g.repo.GetAppointmentByID(ctx, req.AppointmentId)
	if err != nil {
		if errors.Is(err, domain.ErrAppointmentNotFound) {
			return nil, status.Error(codes.NotFound, "appointment not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &appointmentv1.GetAppointmentSummaryResponse{
		AppointmentId:      appt.ID,
		PatientId:          appt.PatientID,
		DoctorId:           appt.DoctorID,
		ClinicId:           appt.ClinicID,
		Status:             string(appt.Status),
		ScheduledStartUnix: appt.StartTime.Unix(),
		ScheduledEndUnix:   appt.EndTime.Unix(),
	}, nil
}
