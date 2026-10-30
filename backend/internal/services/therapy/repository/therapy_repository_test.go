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
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
)

func TestPostgresTherapyRepository_CreateSession(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	sessionID := uuid.New().String()
	apptID := uuid.New().String()
	vitalsJSON, _ := json.Marshal(domain.BaselineVitals{
		BPSystolic:   120,
		BPDiastolic:  80,
		Pulse:        72,
		TemperatureC: 36.6,
	})

	session := &domain.TherapySession{
		ID:                   sessionID,
		AppointmentID:        apptID,
		PatientID:            uuid.New().String(),
		ClinicID:             uuid.New().String(),
		DoctorID:             uuid.New().String(),
		TherapistID:          uuid.New().String(),
		RoomID:               uuid.New().String(),
		PackageID:            uuid.New().String(),
		PackageVersion:       1,
		Status:               domain.SessionStatusCheckedIn,
		BaselineVitals:       vitalsJSON,
		PreChecklistVerified: true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	steps := []domain.StepExecution{
		{
			ID:                     uuid.New().String(),
			StepID:                 uuid.New().String(),
			StepOrder:              1,
			Name:                   "Abhyanga",
			PlannedDurationMinutes: 45,
			Status:                 domain.StepStatusNotStarted,
		},
	}

	outbox := &domain.OutboxEvent{
		AggregateType: "SESSION",
		AggregateID:   sessionID,
		EventType:     "panchakarma.session.checked_in.v1",
		Payload:       []byte(`{"session_id":"` + sessionID + `"}`),
		TraceID:       "trace-1",
		Status:        "PENDING",
	}

	t.Run("CreateSession_Success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()
		repo := repository.NewPostgresTherapyRepository(database.NewTestDB(db))

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO therapy_sessions").
			WithArgs(
				session.ID, session.AppointmentID, session.PatientID, session.ClinicID, session.DoctorID, session.TherapistID,
				session.RoomID, session.PackageID, session.PackageVersion, string(session.Status), session.BaselineVitals,
				session.PreChecklistVerified, session.PreChecklistData, session.TherapistNotes, session.Observations,
				session.PostCareInstructions, session.PostCareItems, session.PostCareDispatched,
				session.StartedAt, session.CompletedAt, sqlmock.AnyArg(), sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec("INSERT INTO session_step_executions").
			WithArgs(
				steps[0].ID, session.ID, steps[0].StepID, steps[0].StepOrder, steps[0].Name, steps[0].PlannedDurationMinutes,
				steps[0].ActualDurationSeconds, string(steps[0].Status), sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec("INSERT INTO outbox_events").
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		err = repo.CreateSession(ctx, session, steps, outbox)
		if err != nil {
			t.Fatalf("unexpected error creating session: %v", err)
		}
	})

	t.Run("CreateSession_DuplicateActiveSession", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()
		repo := repository.NewPostgresTherapyRepository(database.NewTestDB(db))

		mock.ExpectBegin()
		duplicateErr := &pgconn.PgError{
			Code:           "23505",
			Message:        "duplicate key value violates unique constraint \"therapy_sessions_appointment_id_key\"",
			ConstraintName: "therapy_sessions_appointment_id_key",
		}
		mock.ExpectExec("INSERT INTO therapy_sessions").
			WillReturnError(duplicateErr)
		mock.ExpectRollback()

		err = repo.CreateSession(ctx, session, steps, outbox)
		if !errors.Is(err, domain.ErrDuplicateActiveSession) {
			t.Fatalf("expected ErrDuplicateActiveSession, got %v", err)
		}
	})
}

func TestPostgresTherapyRepository_UpdateStepAndAudit(t *testing.T) {
	ctx := context.Background()
	sessionID := uuid.New().String()

	t.Run("UpdateStepStatus_Success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()
		repo := repository.NewPostgresTherapyRepository(database.NewTestDB(db))

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT status FROM session_step_executions WHERE session_id = \\$1 AND step_order = \\$2 FOR UPDATE").
			WithArgs(sessionID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("NOT_STARTED"))

		startedAt := time.Now().UTC()
		mock.ExpectExec("UPDATE session_step_executions").
			WithArgs("IN_PROGRESS", 0, "", "", &startedAt, nil, nil, sessionID, 1).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err = repo.UpdateStepStatus(ctx, sessionID, 1, domain.StepStatusNotStarted, domain.StepStatusInProgress, 0, "", "", &startedAt, nil, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error updating step status: %v", err)
		}
	})

	t.Run("UpdateTherapistNotesWithAudit_Success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create sqlmock: %v", err)
		}
		defer db.Close()
		repo := repository.NewPostgresTherapyRepository(database.NewTestDB(db))

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT COALESCE\\(therapist_notes, ''\\) FROM therapy_sessions WHERE id = \\$1 FOR UPDATE").
			WithArgs(sessionID).
			WillReturnRows(sqlmock.NewRows([]string{"notes"}).AddRow("Initial observation"))

		mock.ExpectExec("INSERT INTO session_audit_logs").
			WithArgs(sqlmock.AnyArg(), sessionID, "therapist_notes", "Initial observation", "Corrected notes", "Typo corrected", "therapist-1", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec("UPDATE therapy_sessions SET therapist_notes = \\$1, updated_at = \\$2 WHERE id = \\$3").
			WithArgs("Corrected notes", sqlmock.AnyArg(), sessionID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		err = repo.UpdateTherapistNotesWithAudit(ctx, sessionID, "Corrected notes", "Typo corrected", "therapist-1")
		if err != nil {
			t.Fatalf("unexpected error in note update with audit: %v", err)
		}
	})
}
