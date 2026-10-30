package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/domain"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
)

// ListFilter provides query parameters for sessions.
type ListFilter struct {
	ClinicID    string
	TherapistID *string
	PatientID   *string
	DoctorID    *string
	Status      *domain.SessionStatus
	Limit       int
	Offset      int
}

// TherapyRepository specifies data access methods for therapy operations.
type TherapyRepository interface {
	// Sessions
	CreateSession(ctx context.Context, session *domain.TherapySession, steps []domain.StepExecution, outbox *domain.OutboxEvent) error
	GetSessionByID(ctx context.Context, id string) (*domain.TherapySession, error)
	GetSessionByAppointmentID(ctx context.Context, appointmentID string) (*domain.TherapySession, error)
	ListSessions(ctx context.Context, filter ListFilter) ([]*domain.TherapySession, int, error)
	UpdateSessionStatusWithTx(ctx context.Context, sessionID string, fromStatus, toStatus domain.SessionStatus, notes, observations string, startedAt, completedAt *time.Time, outbox *domain.OutboxEvent) error
	UpdateSessionPostCare(ctx context.Context, sessionID string, instructions string, items json.RawMessage, dispatched bool) error

	// Steps
	GetStepExecution(ctx context.Context, sessionID string, stepOrder int) (*domain.StepExecution, error)
	UpdateStepStatus(ctx context.Context, sessionID string, stepOrder int, fromStatus, toStatus domain.StepStatus, durationSeconds int, notes, obs string, startedAt, pausedAt, completedAt *time.Time, outbox *domain.OutboxEvent) error

	// Materials
	RecordMaterial(ctx context.Context, material *domain.SessionMaterial) error
	GetSessionMaterials(ctx context.Context, sessionID string) ([]domain.SessionMaterial, error)

	// Complications
	FlagComplicationWithTx(ctx context.Context, comp *domain.ComplicationEvent, outbox *domain.OutboxEvent) error
	ResolveComplicationWithTx(ctx context.Context, compID string, doctorID string, notes string, resumeStatus domain.SessionStatus, outbox *domain.OutboxEvent) error
	GetComplicationByID(ctx context.Context, id string) (*domain.ComplicationEvent, error)
	ListComplications(ctx context.Context, sessionID string) ([]*domain.ComplicationEvent, error)

	// Follow-ups
	CreateFollowUp(ctx context.Context, followUp *domain.SessionFollowUp) error
	GetSessionFollowUps(ctx context.Context, sessionID string) ([]domain.SessionFollowUp, error)
	LinkFollowUpAppointment(ctx context.Context, followUpID, appointmentID string) error

	// Audit Logs / Corrections
	RecordAuditLog(ctx context.Context, audit *domain.SessionAuditLog) error
	GetSessionAuditLogs(ctx context.Context, sessionID string) ([]domain.SessionAuditLog, error)
	UpdateTherapistNotesWithAudit(ctx context.Context, sessionID string, newNotes string, reason string, correctedBy string) error

	// Outbox
	CreateOutboxEvent(ctx context.Context, event *domain.OutboxEvent) error
	GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
	MarkOutboxEventProcessed(ctx context.Context, id string) error
	MarkOutboxEventFailed(ctx context.Context, id string, reason string) error
}

type postgresTherapyRepository struct {
	db *database.DB
}

// NewPostgresTherapyRepository instantiates the repository.
func NewPostgresTherapyRepository(db *database.DB) TherapyRepository {
	return &postgresTherapyRepository{db: db}
}

// CreateSession transactionally inserts therapy session and ordered step executions.
func (r *postgresTherapyRepository) CreateSession(ctx context.Context, session *domain.TherapySession, steps []domain.StepExecution, outbox *domain.OutboxEvent) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	session.UpdatedAt = now

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		// 1. Insert session
		sessionQuery := `
			INSERT INTO therapy_sessions (
				id, appointment_id, patient_id, clinic_id, doctor_id, therapist_id,
				room_id, package_id, package_version, status, baseline_vitals,
				pre_checklist_verified, pre_checklist_data, therapist_notes, observations,
				post_care_instructions, post_care_items, post_care_dispatched,
				started_at, completed_at, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11,
				$12, $13, $14, $15,
				$16, $17, $18,
				$19, $20, $21, $22
			)`

		_, err := tx.ExecContext(ctx, sessionQuery,
			session.ID, session.AppointmentID, session.PatientID, session.ClinicID, session.DoctorID, session.TherapistID,
			session.RoomID, session.PackageID, session.PackageVersion, string(session.Status), session.BaselineVitals,
			session.PreChecklistVerified, session.PreChecklistData, session.TherapistNotes, session.Observations,
			session.PostCareInstructions, session.PostCareItems, session.PostCareDispatched,
			session.StartedAt, session.CompletedAt, session.CreatedAt, session.UpdatedAt,
		)
		if err != nil {
			return r.mapError(err)
		}

		// 2. Insert steps
		stepQuery := `
			INSERT INTO session_step_executions (
				id, session_id, step_id, step_order, name, planned_duration_minutes,
				actual_duration_seconds, status, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

		for _, s := range steps {
			stepID := s.ID
			if stepID == "" {
				stepID = uuid.New().String()
			}
			_, err = tx.ExecContext(ctx, stepQuery,
				stepID, session.ID, s.StepID, s.StepOrder, s.Name, s.PlannedDurationMinutes,
				s.ActualDurationSeconds, string(s.Status), now,
			)
			if err != nil {
				return err
			}
		}

		// 3. Insert Outbox Event
		if outbox != nil {
			if outbox.ID == "" {
				outbox.ID = uuid.New().String()
			}
			if outbox.CreatedAt.IsZero() {
				outbox.CreatedAt = now
			}
			outboxQuery := `
				INSERT INTO outbox_events (
					id, aggregate_type, aggregate_id, event_type, payload, trace_id,
					status, retry_count, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

			_, err = tx.ExecContext(ctx, outboxQuery,
				outbox.ID, outbox.AggregateType, outbox.AggregateID, outbox.EventType,
				outbox.Payload, outbox.TraceID, outbox.Status, outbox.RetryCount, outbox.CreatedAt,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// GetSessionByID retrieves a session with all child steps, materials, and follow-ups.
func (r *postgresTherapyRepository) GetSessionByID(ctx context.Context, id string) (*domain.TherapySession, error) {
	query := `
		SELECT id, appointment_id, patient_id, clinic_id, doctor_id, therapist_id,
		       room_id, package_id, package_version, status, baseline_vitals,
		       pre_checklist_verified, pre_checklist_data, therapist_notes, observations,
		       post_care_instructions, post_care_items, post_care_dispatched,
		       started_at, completed_at, created_at, updated_at
		FROM therapy_sessions
		WHERE id = $1`

	sess, err := r.scanSession(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		return nil, err
	}

	// Load child steps
	steps, err := r.getStepsForSession(ctx, sess.ID)
	if err == nil {
		sess.Steps = steps
	}

	// Load materials
	materials, err := r.GetSessionMaterials(ctx, sess.ID)
	if err == nil {
		sess.Materials = materials
	}

	// Load follow-ups
	followUps, err := r.GetSessionFollowUps(ctx, sess.ID)
	if err == nil {
		sess.FollowUps = followUps
	}

	return sess, nil
}

// GetSessionByAppointmentID retrieves session for a given appointment.
func (r *postgresTherapyRepository) GetSessionByAppointmentID(ctx context.Context, appointmentID string) (*domain.TherapySession, error) {
	query := `
		SELECT id, appointment_id, patient_id, clinic_id, doctor_id, therapist_id,
		       room_id, package_id, package_version, status, baseline_vitals,
		       pre_checklist_verified, pre_checklist_data, therapist_notes, observations,
		       post_care_instructions, post_care_items, post_care_dispatched,
		       started_at, completed_at, created_at, updated_at
		FROM therapy_sessions
		WHERE appointment_id = $1`

	sess, err := r.scanSession(r.db.QueryRowContext(ctx, query, appointmentID))
	if err != nil {
		return nil, err
	}

	steps, err := r.getStepsForSession(ctx, sess.ID)
	if err == nil {
		sess.Steps = steps
	}
	return sess, nil
}

// ListSessions queries sessions matching filters with pagination.
func (r *postgresTherapyRepository) ListSessions(ctx context.Context, filter ListFilter) ([]*domain.TherapySession, int, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	if filter.ClinicID != "" {
		conditions = append(conditions, "clinic_id = $"+string(rune('0'+idx)))
		args = append(args, filter.ClinicID)
		idx++
	}
	if filter.TherapistID != nil && *filter.TherapistID != "" {
		conditions = append(conditions, "therapist_id = $"+string(rune('0'+idx)))
		args = append(args, *filter.TherapistID)
		idx++
	}
	if filter.PatientID != nil && *filter.PatientID != "" {
		conditions = append(conditions, "patient_id = $"+string(rune('0'+idx)))
		args = append(args, *filter.PatientID)
		idx++
	}
	if filter.DoctorID != nil && *filter.DoctorID != "" {
		conditions = append(conditions, "doctor_id = $"+string(rune('0'+idx)))
		args = append(args, *filter.DoctorID)
		idx++
	}
	if filter.Status != nil && *filter.Status != "" {
		conditions = append(conditions, "status = $"+string(rune('0'+idx)))
		args = append(args, string(*filter.Status))
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM therapy_sessions " + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, appointment_id, patient_id, clinic_id, doctor_id, therapist_id,
		       room_id, package_id, package_version, status, baseline_vitals,
		       pre_checklist_verified, pre_checklist_data, therapist_notes, observations,
		       post_care_instructions, post_care_items, post_care_dispatched,
		       started_at, completed_at, created_at, updated_at
		FROM therapy_sessions ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $` + string(rune('0'+idx)) + ` OFFSET $` + string(rune('0'+idx+1))

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []*domain.TherapySession
	for rows.Next() {
		var s domain.TherapySession
		var statusStr string
		var rawChecklist []byte
		var rawPostCare []byte
		err := rows.Scan(
			&s.ID, &s.AppointmentID, &s.PatientID, &s.ClinicID, &s.DoctorID, &s.TherapistID,
			&s.RoomID, &s.PackageID, &s.PackageVersion, &statusStr, &s.BaselineVitals,
			&s.PreChecklistVerified, &rawChecklist, &s.TherapistNotes, &s.Observations,
			&s.PostCareInstructions, &rawPostCare, &s.PostCareDispatched,
			&s.StartedAt, &s.CompletedAt, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		s.Status = domain.SessionStatus(statusStr)
		if len(rawChecklist) > 0 {
			s.PreChecklistData = rawChecklist
		}
		if len(rawPostCare) > 0 {
			s.PostCareItems = rawPostCare
		}
		sessions = append(sessions, &s)
	}

	return sessions, total, nil
}

// UpdateSessionStatusWithTx performs transition with FOR UPDATE row lock.
func (r *postgresTherapyRepository) UpdateSessionStatusWithTx(
	ctx context.Context,
	sessionID string,
	fromStatus, toStatus domain.SessionStatus,
	notes, observations string,
	startedAt, completedAt *time.Time,
	outbox *domain.OutboxEvent,
) error {
	now := time.Now().UTC()

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		var currentStatusStr string
		err := tx.QueryRowContext(ctx, `SELECT status FROM therapy_sessions WHERE id = $1 FOR UPDATE`, sessionID).Scan(&currentStatusStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrSessionNotFound
			}
			return err
		}

		currentStatus := domain.SessionStatus(currentStatusStr)
		if currentStatus == domain.SessionStatusCompleted || currentStatus == domain.SessionStatusAbortedClinical {
			return domain.ErrSessionCompleted
		}
		if fromStatus != "" && currentStatus != fromStatus {
			return domain.ErrInvalidSessionTransition
		}
		if err := domain.ValidateSessionTransition(currentStatus, toStatus); err != nil {
			return err
		}

		query := `
			UPDATE therapy_sessions
			SET status = $1,
			    therapist_notes = COALESCE(NULLIF($2, ''), therapist_notes),
			    observations = COALESCE(NULLIF($3, ''), observations),
			    started_at = COALESCE($4, started_at),
			    completed_at = COALESCE($5, completed_at),
			    updated_at = $6
			WHERE id = $7`

		_, err = tx.ExecContext(ctx, query, string(toStatus), notes, observations, startedAt, completedAt, now, sessionID)
		if err != nil {
			return err
		}

		if outbox != nil {
			if outbox.ID == "" {
				outbox.ID = uuid.New().String()
			}
			if outbox.CreatedAt.IsZero() {
				outbox.CreatedAt = now
			}
			outboxQuery := `
				INSERT INTO outbox_events (
					id, aggregate_type, aggregate_id, event_type, payload, trace_id,
					status, retry_count, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
			_, err = tx.ExecContext(ctx, outboxQuery,
				outbox.ID, outbox.AggregateType, outbox.AggregateID, outbox.EventType,
				outbox.Payload, outbox.TraceID, outbox.Status, outbox.RetryCount, outbox.CreatedAt,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// UpdateSessionPostCare saves instructions and marks post care as dispatched.
func (r *postgresTherapyRepository) UpdateSessionPostCare(ctx context.Context, sessionID string, instructions string, items json.RawMessage, dispatched bool) error {
	query := `
		UPDATE therapy_sessions
		SET post_care_instructions = $1, post_care_items = $2, post_care_dispatched = $3, updated_at = $4
		WHERE id = $5`

	res, err := r.db.ExecContext(ctx, query, instructions, items, dispatched, time.Now().UTC(), sessionID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

// GetStepExecution fetches an individual procedural step by order.
func (r *postgresTherapyRepository) GetStepExecution(ctx context.Context, sessionID string, stepOrder int) (*domain.StepExecution, error) {
	query := `
		SELECT id, session_id, step_id, step_order, name, planned_duration_minutes,
		       actual_duration_seconds, status, started_at, paused_at, completed_at,
		       therapist_notes, observations, created_at
		FROM session_step_executions
		WHERE session_id = $1 AND step_order = $2`

	var s domain.StepExecution
	var statusStr string
	var notes, obs *string
	err := r.db.QueryRowContext(ctx, query, sessionID, stepOrder).Scan(
		&s.ID, &s.SessionID, &s.StepID, &s.StepOrder, &s.Name, &s.PlannedDurationMinutes,
		&s.ActualDurationSeconds, &statusStr, &s.StartedAt, &s.PausedAt, &s.CompletedAt,
		&notes, &obs, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrStepNotFound
		}
		return nil, err
	}
	s.Status = domain.StepStatus(statusStr)
	if notes != nil {
		s.TherapistNotes = *notes
	}
	if obs != nil {
		s.Observations = *obs
	}
	return &s, nil
}

// UpdateStepStatus transitions a step state and records duration and observations.
func (r *postgresTherapyRepository) UpdateStepStatus(
	ctx context.Context,
	sessionID string,
	stepOrder int,
	fromStatus, toStatus domain.StepStatus,
	durationSeconds int,
	notes, obs string,
	startedAt, pausedAt, completedAt *time.Time,
	outbox *domain.OutboxEvent,
) error {
	now := time.Now().UTC()

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		// Verify step exists
		var currentStatusStr string
		err := tx.QueryRowContext(ctx, `SELECT status FROM session_step_executions WHERE session_id = $1 AND step_order = $2 FOR UPDATE`, sessionID, stepOrder).Scan(&currentStatusStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrStepNotFound
			}
			return err
		}

		currentStatus := domain.StepStatus(currentStatusStr)
		if fromStatus != "" && currentStatus != fromStatus {
			return domain.ErrInvalidStepTransition
		}
		if err := domain.ValidateStepTransition(currentStatus, toStatus); err != nil {
			return err
		}

		query := `
			UPDATE session_step_executions
			SET status = $1,
			    actual_duration_seconds = CASE WHEN $2 > 0 THEN $2 ELSE actual_duration_seconds END,
			    therapist_notes = COALESCE(NULLIF($3, ''), therapist_notes),
			    observations = COALESCE(NULLIF($4, ''), observations),
			    started_at = COALESCE($5, started_at),
			    paused_at = COALESCE($6, paused_at),
			    completed_at = COALESCE($7, completed_at)
			WHERE session_id = $8 AND step_order = $9`

		_, err = tx.ExecContext(ctx, query, string(toStatus), durationSeconds, notes, obs, startedAt, pausedAt, completedAt, sessionID, stepOrder)
		if err != nil {
			return err
		}

		if outbox != nil {
			if outbox.ID == "" {
				outbox.ID = uuid.New().String()
			}
			if outbox.CreatedAt.IsZero() {
				outbox.CreatedAt = now
			}
			outboxQuery := `
				INSERT INTO outbox_events (
					id, aggregate_type, aggregate_id, event_type, payload, trace_id,
					status, retry_count, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

			_, err = tx.ExecContext(ctx, outboxQuery,
				outbox.ID, outbox.AggregateType, outbox.AggregateID, outbox.EventType,
				outbox.Payload, outbox.TraceID, outbox.Status, outbox.RetryCount, outbox.CreatedAt,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// RecordMaterial stores medical consumables used in therapy.
func (r *postgresTherapyRepository) RecordMaterial(ctx context.Context, m *domain.SessionMaterial) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO session_materials (
			id, session_id, material_name, quantity, unit, batch_number, notes, recorded_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		m.ID, m.SessionID, m.MaterialName, m.Quantity, m.Unit, m.BatchNumber, m.Notes, m.RecordedBy, m.CreatedAt,
	)
	return err
}

// GetSessionMaterials retrieves supplies used for a session.
func (r *postgresTherapyRepository) GetSessionMaterials(ctx context.Context, sessionID string) ([]domain.SessionMaterial, error) {
	query := `
		SELECT id, session_id, material_name, quantity, unit, batch_number, notes, recorded_by, created_at
		FROM session_materials
		WHERE session_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.SessionMaterial
	for rows.Next() {
		var m domain.SessionMaterial
		var batch, notes *string
		err := rows.Scan(&m.ID, &m.SessionID, &m.MaterialName, &m.Quantity, &m.Unit, &batch, &notes, &m.RecordedBy, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		if batch != nil {
			m.BatchNumber = *batch
		}
		if notes != nil {
			m.Notes = *notes
		}
		list = append(list, m)
	}
	return list, nil
}

// FlagComplicationWithTx persists an adverse event, transitions session to COMPLICATION_ALERT, and writes emergency outbox event.
func (r *postgresTherapyRepository) FlagComplicationWithTx(ctx context.Context, comp *domain.ComplicationEvent, outbox *domain.OutboxEvent) error {
	if comp.ID == "" {
		comp.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if comp.CreatedAt.IsZero() {
		comp.CreatedAt = now
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		// 1. Insert complication event
		compQuery := `
			INSERT INTO complication_events (
				id, session_id, reporter_id, reporter_role, severity, symptoms,
				vitals_at_incident, immediate_actions_taken, status, attending_doctor_id, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

		_, err := tx.ExecContext(ctx, compQuery,
			comp.ID, comp.SessionID, comp.ReporterID, comp.ReporterRole, string(comp.Severity),
			comp.Symptoms, comp.VitalsAtIncident, comp.ImmediateActionsTaken, string(comp.Status),
			comp.AttendingDoctorID, comp.CreatedAt,
		)
		if err != nil {
			return err
		}

		// 2. Transition session to COMPLICATION_ALERT
		sessUpdateQuery := `
			UPDATE therapy_sessions
			SET status = 'COMPLICATION_ALERT', updated_at = $1
			WHERE id = $2`
		_, err = tx.ExecContext(ctx, sessUpdateQuery, now, comp.SessionID)
		if err != nil {
			return err
		}

		// 3. Write Emergency Outbox Event
		if outbox != nil {
			if outbox.ID == "" {
				outbox.ID = uuid.New().String()
			}
			if outbox.CreatedAt.IsZero() {
				outbox.CreatedAt = now
			}
			outboxQuery := `
				INSERT INTO outbox_events (
					id, aggregate_type, aggregate_id, event_type, payload, trace_id,
					status, retry_count, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

			_, err = tx.ExecContext(ctx, outboxQuery,
				outbox.ID, outbox.AggregateType, outbox.AggregateID, outbox.EventType,
				outbox.Payload, outbox.TraceID, outbox.Status, outbox.RetryCount, outbox.CreatedAt,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// ResolveComplicationWithTx records physician resolution and returns session to IN_PROGRESS or ABORTED_CLINICAL.
func (r *postgresTherapyRepository) ResolveComplicationWithTx(
	ctx context.Context,
	compID string,
	doctorID string,
	notes string,
	resumeStatus domain.SessionStatus,
	outbox *domain.OutboxEvent,
) error {
	now := time.Now().UTC()

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		var sessionID string
		err := tx.QueryRowContext(ctx, `SELECT session_id FROM complication_events WHERE id = $1 FOR UPDATE`, compID).Scan(&sessionID)
		if err != nil {
			return err
		}

		// Update complication
		compUpdateQuery := `
			UPDATE complication_events
			SET status = 'RESOLVED', doctor_notes = $1, doctor_cleared_at = $2
			WHERE id = $3`
		_, err = tx.ExecContext(ctx, compUpdateQuery, notes, now, compID)
		if err != nil {
			return err
		}

		// Update session
		sessUpdateQuery := `
			UPDATE therapy_sessions
			SET status = $1, updated_at = $2
			WHERE id = $3`
		_, err = tx.ExecContext(ctx, sessUpdateQuery, string(resumeStatus), now, sessionID)
		if err != nil {
			return err
		}

		if outbox != nil {
			if outbox.ID == "" {
				outbox.ID = uuid.New().String()
			}
			if outbox.CreatedAt.IsZero() {
				outbox.CreatedAt = now
			}
			outboxQuery := `
				INSERT INTO outbox_events (
					id, aggregate_type, aggregate_id, event_type, payload, trace_id,
					status, retry_count, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

			_, err = tx.ExecContext(ctx, outboxQuery,
				outbox.ID, outbox.AggregateType, outbox.AggregateID, outbox.EventType,
				outbox.Payload, outbox.TraceID, outbox.Status, outbox.RetryCount, outbox.CreatedAt,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// GetComplicationByID retrieves a single complication record.
func (r *postgresTherapyRepository) GetComplicationByID(ctx context.Context, id string) (*domain.ComplicationEvent, error) {
	query := `
		SELECT id, session_id, reporter_id, reporter_role, severity, symptoms,
		       vitals_at_incident, immediate_actions_taken, status, attending_doctor_id,
		       doctor_notes, doctor_cleared_at, created_at
		FROM complication_events
		WHERE id = $1`

	var c domain.ComplicationEvent
	var sevStr, statusStr string
	var docNotes *string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.SessionID, &c.ReporterID, &c.ReporterRole, &sevStr, &c.Symptoms,
		&c.VitalsAtIncident, &c.ImmediateActionsTaken, &statusStr, &c.AttendingDoctorID,
		&docNotes, &c.DoctorClearedAt, &c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("complication event not found")
		}
		return nil, err
	}
	c.Severity = domain.ComplicationSeverity(sevStr)
	c.Status = domain.ComplicationStatus(statusStr)
	if docNotes != nil {
		c.DoctorNotes = *docNotes
	}
	return &c, nil
}

// ListComplications lists all complications for a session.
func (r *postgresTherapyRepository) ListComplications(ctx context.Context, sessionID string) ([]*domain.ComplicationEvent, error) {
	query := `
		SELECT id, session_id, reporter_id, reporter_role, severity, symptoms,
		       vitals_at_incident, immediate_actions_taken, status, attending_doctor_id,
		       doctor_notes, doctor_cleared_at, created_at
		FROM complication_events
		WHERE session_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ComplicationEvent
	for rows.Next() {
		var c domain.ComplicationEvent
		var sevStr, statusStr string
		var docNotes *string
		err := rows.Scan(
			&c.ID, &c.SessionID, &c.ReporterID, &c.ReporterRole, &sevStr, &c.Symptoms,
			&c.VitalsAtIncident, &c.ImmediateActionsTaken, &statusStr, &c.AttendingDoctorID,
			&docNotes, &c.DoctorClearedAt, &c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		c.Severity = domain.ComplicationSeverity(sevStr)
		c.Status = domain.ComplicationStatus(statusStr)
		if docNotes != nil {
			c.DoctorNotes = *docNotes
		}
		list = append(list, &c)
	}
	return list, nil
}

// CreateFollowUp saves practitioner follow-up recommendation.
func (r *postgresTherapyRepository) CreateFollowUp(ctx context.Context, f *domain.SessionFollowUp) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	if f.CreatedAt.IsZero() {
		f.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO session_follow_ups (
			id, session_id, recommended_by, recommended_package_id,
			recommended_interval_days, recommended_date, instructions, linked_appointment_id, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		f.ID, f.SessionID, f.RecommendedBy, f.RecommendedPackageID,
		f.RecommendedIntervalDay, f.RecommendedDate, f.Instructions, f.LinkedAppointmentID, f.Status, f.CreatedAt,
	)
	return err
}

// GetSessionFollowUps retrieves follow-up entries for a session.
func (r *postgresTherapyRepository) GetSessionFollowUps(ctx context.Context, sessionID string) ([]domain.SessionFollowUp, error) {
	query := `
		SELECT id, session_id, recommended_by, recommended_package_id,
		       recommended_interval_days, recommended_date, instructions, linked_appointment_id, status, created_at
		FROM session_follow_ups
		WHERE session_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.SessionFollowUp
	for rows.Next() {
		var f domain.SessionFollowUp
		var recPkg *string
		var recDays *int
		var recDate *string
		var linkedAppt *string
		err := rows.Scan(
			&f.ID, &f.SessionID, &f.RecommendedBy, &recPkg,
			&recDays, &recDate, &f.Instructions, &linkedAppt, &f.Status, &f.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		f.RecommendedPackageID = recPkg
		f.RecommendedIntervalDay = recDays
		f.RecommendedDate = recDate
		f.LinkedAppointmentID = linkedAppt
		list = append(list, f)
	}
	return list, nil
}

// LinkFollowUpAppointment associates an existing follow-up recommendation with a newly booked appointment.
func (r *postgresTherapyRepository) LinkFollowUpAppointment(ctx context.Context, followUpID, appointmentID string) error {
	query := `UPDATE session_follow_ups SET linked_appointment_id = $1, status = 'SCHEDULED' WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, appointmentID, followUpID)
	return err
}

// RecordAuditLog records clinical record corrections.
func (r *postgresTherapyRepository) RecordAuditLog(ctx context.Context, a *domain.SessionAuditLog) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO session_audit_logs (
			id, session_id, field_name, old_value, new_value, reason, corrected_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, query,
		a.ID, a.SessionID, a.FieldName, a.OldValue, a.NewValue, a.Reason, a.CorrectedBy, a.CreatedAt,
	)
	return err
}

// GetSessionAuditLogs retrieves full historical changes for a session.
func (r *postgresTherapyRepository) GetSessionAuditLogs(ctx context.Context, sessionID string) ([]domain.SessionAuditLog, error) {
	query := `
		SELECT id, session_id, field_name, old_value, new_value, reason, corrected_by, created_at
		FROM session_audit_logs
		WHERE session_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.SessionAuditLog
	for rows.Next() {
		var a domain.SessionAuditLog
		var oldVal *string
		err := rows.Scan(&a.ID, &a.SessionID, &a.FieldName, &oldVal, &a.NewValue, &a.Reason, &a.CorrectedBy, &a.CreatedAt)
		if err != nil {
			return nil, err
		}
		if oldVal != nil {
			a.OldValue = *oldVal
		}
		list = append(list, a)
	}
	return list, nil
}

// UpdateTherapistNotesWithAudit transactionally modifies notes while appending an immutable audit trail entry.
func (r *postgresTherapyRepository) UpdateTherapistNotesWithAudit(ctx context.Context, sessionID string, newNotes string, reason string, correctedBy string) error {
	now := time.Now().UTC()

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		var oldNotes string
		err := tx.QueryRowContext(ctx, `SELECT COALESCE(therapist_notes, '') FROM therapy_sessions WHERE id = $1 FOR UPDATE`, sessionID).Scan(&oldNotes)
		if err != nil {
			return err
		}

		// 1. Insert audit log
		auditQuery := `
			INSERT INTO session_audit_logs (
				id, session_id, field_name, old_value, new_value, reason, corrected_by, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
		_, err = tx.ExecContext(ctx, auditQuery,
			uuid.New().String(), sessionID, "therapist_notes", oldNotes, newNotes, reason, correctedBy, now,
		)
		if err != nil {
			return err
		}

		// 2. Update session notes
		_, err = tx.ExecContext(ctx, `UPDATE therapy_sessions SET therapist_notes = $1, updated_at = $2 WHERE id = $3`, newNotes, now, sessionID)
		return err
	})
}

// CreateOutboxEvent stores an event for Kafka dispatch.
func (r *postgresTherapyRepository) CreateOutboxEvent(ctx context.Context, event *domain.OutboxEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO outbox_events (
			id, aggregate_type, aggregate_id, event_type, payload, trace_id,
			status, retry_count, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.AggregateType, event.AggregateID, event.EventType,
		event.Payload, event.TraceID, event.Status, event.RetryCount, event.CreatedAt,
	)
	return err
}

// GetPendingOutboxEvents fetches pending Kafka outbox messages.
func (r *postgresTherapyRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, trace_id,
		       status, retry_count, created_at, processed_at
		FROM outbox_events
		WHERE status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.OutboxEvent
	for rows.Next() {
		var e domain.OutboxEvent
		err := rows.Scan(
			&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload, &e.TraceID,
			&e.Status, &e.RetryCount, &e.CreatedAt, &e.ProcessedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	return events, nil
}

// MarkOutboxEventProcessed marks outbox entry processed.
func (r *postgresTherapyRepository) MarkOutboxEventProcessed(ctx context.Context, id string) error {
	query := `UPDATE outbox_events SET status = 'PROCESSED', processed_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now().UTC(), id)
	return err
}

// MarkOutboxEventFailed increments retry counter.
func (r *postgresTherapyRepository) MarkOutboxEventFailed(ctx context.Context, id string, reason string) error {
	query := `UPDATE outbox_events SET retry_count = retry_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *postgresTherapyRepository) scanSession(row *sql.Row) (*domain.TherapySession, error) {
	var s domain.TherapySession
	var statusStr string
	var rawChecklist []byte
	var rawPostCare []byte
	var notes, obs, postCare *string

	err := row.Scan(
		&s.ID, &s.AppointmentID, &s.PatientID, &s.ClinicID, &s.DoctorID, &s.TherapistID,
		&s.RoomID, &s.PackageID, &s.PackageVersion, &statusStr, &s.BaselineVitals,
		&s.PreChecklistVerified, &rawChecklist, &notes, &obs,
		&postCare, &rawPostCare, &s.PostCareDispatched,
		&s.StartedAt, &s.CompletedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}
	s.Status = domain.SessionStatus(statusStr)
	if len(rawChecklist) > 0 {
		s.PreChecklistData = rawChecklist
	}
	if len(rawPostCare) > 0 {
		s.PostCareItems = rawPostCare
	}
	if notes != nil {
		s.TherapistNotes = *notes
	}
	if obs != nil {
		s.Observations = *obs
	}
	if postCare != nil {
		s.PostCareInstructions = *postCare
	}
	return &s, nil
}

func (r *postgresTherapyRepository) getStepsForSession(ctx context.Context, sessionID string) ([]domain.StepExecution, error) {
	query := `
		SELECT id, session_id, step_id, step_order, name, planned_duration_minutes,
		       actual_duration_seconds, status, started_at, paused_at, completed_at,
		       therapist_notes, observations, created_at
		FROM session_step_executions
		WHERE session_id = $1
		ORDER BY step_order ASC`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []domain.StepExecution
	for rows.Next() {
		var s domain.StepExecution
		var statusStr string
		var notes, obs *string
		err := rows.Scan(
			&s.ID, &s.SessionID, &s.StepID, &s.StepOrder, &s.Name, &s.PlannedDurationMinutes,
			&s.ActualDurationSeconds, &statusStr, &s.StartedAt, &s.PausedAt, &s.CompletedAt,
			&notes, &obs, &s.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		s.Status = domain.StepStatus(statusStr)
		if notes != nil {
			s.TherapistNotes = *notes
		}
		if obs != nil {
			s.Observations = *obs
		}
		steps = append(steps, s)
	}
	return steps, nil
}

func (r *postgresTherapyRepository) mapError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "appointment_id") {
			return domain.ErrDuplicateActiveSession
		}
	}
	if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
		return domain.ErrDuplicateActiveSession
	}
	return err
}
