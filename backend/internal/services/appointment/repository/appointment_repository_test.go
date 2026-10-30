package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
)

func TestPostgresAppointmentRepository_Holds(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	testDB := database.NewTestDB(db)
	repo := repository.NewPostgresAppointmentRepository(testDB)
	ctx := context.Background()

	now := time.Now().UTC()
	start := now.Add(24 * time.Hour)
	end := start.Add(90 * time.Minute)
	expiresAt := now.Add(10 * time.Minute)

	t.Run("CreateHold_Success", func(t *testing.T) {
		hold := &domain.AppointmentHold{
			ID:            uuid.New().String(),
			HoldToken:     "hld_tok_123",
			PatientID:     uuid.New().String(),
			ClinicID:      uuid.New().String(),
			DoctorID:      uuid.New().String(),
			RoomID:        uuid.New().String(),
			PackageID:     uuid.New().String(),
			StartTime:     start,
			EndTime:       end,
			SessionNumber: 1,
			TotalSessions: 1,
			ExpiresAt:     expiresAt,
			Status:        domain.HoldStatusActive,
			CreatedAt:     now,
		}

		mock.ExpectExec("INSERT INTO appointment_holds").
			WithArgs(
				hold.ID, hold.HoldToken, hold.PatientID, hold.ClinicID, hold.DoctorID, hold.RoomID, hold.PackageID,
				hold.StartTime, hold.EndTime, hold.SessionNumber, hold.TotalSessions, hold.PackageBookingID,
				hold.ExpiresAt, string(hold.Status), hold.CreatedAt,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.CreateHold(ctx, hold)
		if err != nil {
			t.Fatalf("unexpected error creating hold: %v", err)
		}
	})

	t.Run("GetHoldByToken_Success", func(t *testing.T) {
		token := "hld_tok_123"
		rows := sqlmock.NewRows([]string{
			"id", "hold_token", "patient_id", "clinic_id", "doctor_id", "room_id", "package_id",
			"start_time", "end_time", "session_number", "total_sessions", "package_booking_id",
			"expires_at", "status", "created_at",
		}).AddRow(
			uuid.New().String(), token, uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String(),
			start, end, 1, 1, nil, expiresAt, "ACTIVE", now,
		)

		mock.ExpectQuery("SELECT id, hold_token").
			WithArgs(token).
			WillReturnRows(rows)

		hold, err := repo.GetHoldByToken(ctx, token)
		if err != nil {
			t.Fatalf("unexpected error fetching hold by token: %v", err)
		}
		if hold.HoldToken != token {
			t.Errorf("expected token %s, got %s", token, hold.HoldToken)
		}
	})
}

func TestPostgresAppointmentRepository_CreateAppointmentWithTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	testDB := database.NewTestDB(db)
	repo := repository.NewPostgresAppointmentRepository(testDB)
	ctx := context.Background()

	now := time.Now().UTC()
	start := now.Add(24 * time.Hour)
	end := start.Add(90 * time.Minute)
	apptID := uuid.New().String()
	holdID := uuid.New().String()

	appt := &domain.Appointment{
		ID:            apptID,
		PatientID:     uuid.New().String(),
		ClinicID:      uuid.New().String(),
		DoctorID:      uuid.New().String(),
		RoomID:        uuid.New().String(),
		PackageID:     uuid.New().String(),
		StartTime:     start,
		EndTime:       end,
		Status:        domain.StatusConfirmed,
		SessionNumber: 1,
		TotalSessions: 1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	payload, _ := json.Marshal(map[string]string{"appointment_id": apptID})
	outbox := &domain.OutboxEvent{
		AggregateType: "APPOINTMENT",
		AggregateID:   apptID,
		EventType:     "panchakarma.appointment.confirmed.v1",
		Payload:       payload,
		TraceID:       "trace-123",
		Status:        domain.OutboxStatusPending,
	}

	t.Run("SuccessWithHoldAndOutbox", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE appointment_holds").
			WithArgs(holdID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO appointments").
			WithArgs(
				appt.ID, appt.PatientID, appt.ClinicID, appt.DoctorID, appt.RoomID, appt.PackageID,
				appt.StartTime, appt.EndTime, string(appt.Status), appt.SessionNumber, appt.TotalSessions, appt.PackageBookingID,
				appt.IdempotencyKey, appt.Notes, appt.CancellationReason, appt.ApprovedBy, appt.ApprovedAt,
				appt.RejectionReason, appt.RescheduledToID, appt.CreatedAt, appt.UpdatedAt,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO appointment_history").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO outbox_events").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.CreateAppointmentWithTx(ctx, appt, &holdID, outbox)
		if err != nil {
			t.Fatalf("unexpected error creating appointment with tx: %v", err)
		}
	})

	t.Run("ExclusionConstraintViolation_ReturnsDoubleBookedError", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE appointment_holds").
			WithArgs(holdID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		
		// Simulate PostgreSQL GiST exclusion constraint error (23P01)
		pgExclusionErr := &pgconn.PgError{
			Code:           "23P01",
			Message:        "conflicting key value violates exclusion constraint \"no_overlapping_doctor_appointments\"",
			ConstraintName: "no_overlapping_doctor_appointments",
		}
		mock.ExpectExec("INSERT INTO appointments").
			WillReturnError(pgExclusionErr)
		mock.ExpectRollback()

		err := repo.CreateAppointmentWithTx(ctx, appt, &holdID, outbox)
		if err == nil {
			t.Fatalf("expected error on exclusion violation, got nil")
		}
		if !errors.Is(err, domain.ErrSlotDoubleBooked) {
			t.Fatalf("expected domain.ErrSlotDoubleBooked, got %v", err)
		}
	})
}

func TestPostgresAppointmentRepository_UpdateAppointmentStatusWithTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	testDB := database.NewTestDB(db)
	repo := repository.NewPostgresAppointmentRepository(testDB)
	ctx := context.Background()

	apptID := uuid.New().String()

	t.Run("CancelAppointment_Success", func(t *testing.T) {
		mock.ExpectBegin()
		// Lock FOR UPDATE
		mock.ExpectQuery("SELECT status FROM appointments WHERE id = \\$1 FOR UPDATE").
			WithArgs(apptID).
			WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("CONFIRMED"))
		mock.ExpectExec("UPDATE appointments").
			WithArgs("CANCELLED", "Emergency", sqlmock.AnyArg(), apptID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO appointment_history").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO outbox_events").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		outbox := &domain.OutboxEvent{
			AggregateType: "APPOINTMENT",
			AggregateID:   apptID,
			EventType:     "panchakarma.appointment.cancelled.v1",
			Payload:       []byte(`{}`),
			TraceID:       "trc",
			Status:        domain.OutboxStatusPending,
		}

		err := repo.UpdateAppointmentStatusWithTx(ctx, apptID, domain.StatusConfirmed, domain.StatusCancelled, "Emergency", nil, outbox)
		if err != nil {
			t.Fatalf("unexpected error updating appointment status: %v", err)
		}
	})

	t.Run("InvalidTransition_Rejected", func(t *testing.T) {
		mock.ExpectBegin()
		// Return already CANCELLED
		mock.ExpectQuery("SELECT status FROM appointments WHERE id = \\$1 FOR UPDATE").
			WithArgs(apptID).
			WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("CANCELLED"))
		mock.ExpectRollback()

		err := repo.UpdateAppointmentStatusWithTx(ctx, apptID, domain.StatusConfirmed, domain.StatusCancelled, "Emergency", nil, nil)
		if !errors.Is(err, domain.ErrInvalidStateTransition) {
			t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
		}
	})
}
