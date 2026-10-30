package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/feedback/domain"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
)

type FeedbackRepository interface {
	// Patient Feedback
	CreateFeedbackWithTx(ctx context.Context, fb *domain.PatientFeedback, outbox *domain.OutboxEvent) error
	GetFeedbackByID(ctx context.Context, id string) (*domain.PatientFeedback, error)
	GetFeedbackByAppointmentID(ctx context.Context, appointmentID string) (*domain.PatientFeedback, error)
	ListFeedbackByClinic(ctx context.Context, clinicID string, startDate, endDate *time.Time, limit, offset int) ([]*domain.PatientFeedback, int, error)
	ListFeedbackByPatient(ctx context.Context, patientID string, limit, offset int) ([]*domain.PatientFeedback, int, error)
	AcknowledgeFeedback(ctx context.Context, id string, ackBy string, notes string, status string) error

	// Adverse Events
	CreateAdverseEvent(ctx context.Context, ae *domain.AdverseEvent) error
	GetAdverseEventByID(ctx context.Context, id string) (*domain.AdverseEvent, error)
	ListAdverseEventsByClinic(ctx context.Context, clinicID string, severity, status *string, startDate, endDate *time.Time, limit, offset int) ([]*domain.AdverseEvent, int, error)
	ListAdverseEventsByPatient(ctx context.Context, patientID string, limit, offset int) ([]*domain.AdverseEvent, int, error)
	AcknowledgeAdverseEvent(ctx context.Context, id string, ackBy string, notes string, status string, resolved bool) error

	// Analytics Operational Projections
	UpsertAppointmentProjection(ctx context.Context, apt *domain.AnalyticsAppointment) error
	UpsertSessionProjection(ctx context.Context, sess *domain.AnalyticsSession) error
	GetAppointmentProjection(ctx context.Context, appointmentID string) (*domain.AnalyticsAppointment, error)
	GetSessionProjection(ctx context.Context, sessionID string) (*domain.AnalyticsSession, error)

	// Metrics
	CalculateClinicMetrics(ctx context.Context, clinicID string, startDate, endDate time.Time, loc *time.Location) (*domain.ClinicDashboardMetrics, error)

	// Kafka Deduplication Ledger
	IsEventProcessed(ctx context.Context, eventID string) (bool, error)
	RecordConsumedEvent(ctx context.Context, eventID, eventType, consumerGroup, payloadDigest string) error

	// Transactional Outbox
	GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
	MarkOutboxEventPublished(ctx context.Context, id string) error
	MarkOutboxEventFailed(ctx context.Context, id string) error
}

type feedbackRepository struct {
	db *database.DB
}

func NewFeedbackRepository(db *database.DB) FeedbackRepository {
	return &feedbackRepository{db: db}
}

// sanitizeUUID ensures input string is a valid UUID, otherwise generating a deterministic v5 or new v4
func sanitizeUUID(val string) string {
	if val == "" {
		return uuid.New().String()
	}
	parsed, err := uuid.Parse(val)
	if err != nil {
		return uuid.NewSHA1(uuid.NameSpaceOID, []byte(val)).String()
	}
	return parsed.String()
}

// -----------------------------------------------------------------------------
// PATIENT FEEDBACK
// -----------------------------------------------------------------------------

func (r *feedbackRepository) CreateFeedbackWithTx(ctx context.Context, fb *domain.PatientFeedback, outbox *domain.OutboxEvent) error {
	fb.ID = sanitizeUUID(fb.ID)
	fb.AppointmentID = sanitizeUUID(fb.AppointmentID)
	fb.PatientID = sanitizeUUID(fb.PatientID)
	fb.ClinicID = sanitizeUUID(fb.ClinicID)
	fb.DoctorID = sanitizeUUID(fb.DoctorID)
	fb.TherapistID = sanitizeUUID(fb.TherapistID)

	var sessID sql.NullString
	if fb.SessionID != nil && *fb.SessionID != "" {
		sessID = sql.NullString{String: sanitizeUUID(*fb.SessionID), Valid: true}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	query := `
		INSERT INTO patient_feedback (
			id, appointment_id, session_id, patient_id, clinic_id, doctor_id, therapist_id,
			overall_rating, therapist_rating, facility_cleanliness_rating, comments,
			is_anonymous, status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14
		) RETURNING created_at`

	now := time.Now().UTC()
	err = tx.QueryRowContext(ctx, query,
		fb.ID, fb.AppointmentID, sessID, fb.PatientID, fb.ClinicID, fb.DoctorID, fb.TherapistID,
		fb.OverallRating, fb.TherapistRating, fb.FacilityCleanlinessRating, fb.Comments,
		fb.IsAnonymous, "SUBMITTED", now,
	).Scan(&fb.CreatedAt)
	if err != nil {
		return err
	}

	if outbox != nil {
		outbox.ID = sanitizeUUID(outbox.ID)
		outboxQuery := `
			INSERT INTO outbox_events (
				id, aggregate_type, aggregate_id, event_type, payload, trace_id, status, retry_count, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

		_, err = tx.ExecContext(ctx, outboxQuery,
			outbox.ID, outbox.AggregateType, outbox.AggregateID, outbox.EventType,
			outbox.Payload, outbox.TraceID, "PENDING", 0, now,
		)
		if err != nil {
			return fmt.Errorf("failed to write outbox event: %w", err)
		}
	}

	return tx.Commit()
}

func (r *feedbackRepository) GetFeedbackByID(ctx context.Context, id string) (*domain.PatientFeedback, error) {
	id = sanitizeUUID(id)
	query := `
		SELECT id, appointment_id, session_id, patient_id, clinic_id, doctor_id, therapist_id,
		       overall_rating, therapist_rating, facility_cleanliness_rating, comments,
		       is_anonymous, status, acknowledged_at, acknowledged_by, staff_notes, created_at
		FROM patient_feedback
		WHERE id = $1`

	return r.scanFeedbackRow(r.db.QueryRowContext(ctx, query, id))
}

func (r *feedbackRepository) GetFeedbackByAppointmentID(ctx context.Context, appointmentID string) (*domain.PatientFeedback, error) {
	appointmentID = sanitizeUUID(appointmentID)
	query := `
		SELECT id, appointment_id, session_id, patient_id, clinic_id, doctor_id, therapist_id,
		       overall_rating, therapist_rating, facility_cleanliness_rating, comments,
		       is_anonymous, status, acknowledged_at, acknowledged_by, staff_notes, created_at
		FROM patient_feedback
		WHERE appointment_id = $1`

	return r.scanFeedbackRow(r.db.QueryRowContext(ctx, query, appointmentID))
}

func (r *feedbackRepository) ListFeedbackByClinic(ctx context.Context, clinicID string, startDate, endDate *time.Time, limit, offset int) ([]*domain.PatientFeedback, int, error) {
	clinicID = sanitizeUUID(clinicID)
	where := "WHERE clinic_id = $1"
	args := []interface{}{clinicID}
	idx := 2

	if startDate != nil {
		where += fmt.Sprintf(" AND created_at >= $%d", idx)
		args = append(args, *startDate)
		idx++
	}
	if endDate != nil {
		where += fmt.Sprintf(" AND created_at <= $%d", idx)
		args = append(args, *endDate)
		idx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM patient_feedback %s", where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, appointment_id, session_id, patient_id, clinic_id, doctor_id, therapist_id,
		       overall_rating, therapist_rating, facility_cleanliness_rating, comments,
		       is_anonymous, status, acknowledged_at, acknowledged_by, staff_notes, created_at
		FROM patient_feedback
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*domain.PatientFeedback
	for rows.Next() {
		fb, err := r.scanFeedback(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, fb)
	}
	return result, total, nil
}

func (r *feedbackRepository) ListFeedbackByPatient(ctx context.Context, patientID string, limit, offset int) ([]*domain.PatientFeedback, int, error) {
	patientID = sanitizeUUID(patientID)
	countQuery := "SELECT COUNT(*) FROM patient_feedback WHERE patient_id = $1"
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, patientID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, appointment_id, session_id, patient_id, clinic_id, doctor_id, therapist_id,
		       overall_rating, therapist_rating, facility_cleanliness_rating, comments,
		       is_anonymous, status, acknowledged_at, acknowledged_by, staff_notes, created_at
		FROM patient_feedback
		WHERE patient_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, patientID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*domain.PatientFeedback
	for rows.Next() {
		fb, err := r.scanFeedback(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, fb)
	}
	return result, total, nil
}

func (r *feedbackRepository) AcknowledgeFeedback(ctx context.Context, id string, ackBy string, notes string, status string) error {
	id = sanitizeUUID(id)
	ackBy = sanitizeUUID(ackBy)
	if status == "" {
		status = "ACKNOWLEDGED"
	}
	now := time.Now().UTC()

	query := `
		UPDATE patient_feedback
		SET status = $1, acknowledged_at = $2, acknowledged_by = $3, staff_notes = $4
		WHERE id = $5`

	res, err := r.db.ExecContext(ctx, query, status, now, ackBy, notes, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrFeedbackNotFound
	}
	return nil
}

func (r *feedbackRepository) scanFeedbackRow(row *sql.Row) (*domain.PatientFeedback, error) {
	var fb domain.PatientFeedback
	var sessID, ackBy, staffNotes, comments sql.NullString
	var ackAt sql.NullTime

	err := row.Scan(
		&fb.ID, &fb.AppointmentID, &sessID, &fb.PatientID, &fb.ClinicID, &fb.DoctorID, &fb.TherapistID,
		&fb.OverallRating, &fb.TherapistRating, &fb.FacilityCleanlinessRating, &comments,
		&fb.IsAnonymous, &fb.Status, &ackAt, &ackBy, &staffNotes, &fb.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrFeedbackNotFound
		}
		return nil, err
	}

	if sessID.Valid {
		fb.SessionID = &sessID.String
	}
	if comments.Valid {
		fb.Comments = &comments.String
	}
	if ackAt.Valid {
		fb.AcknowledgedAt = &ackAt.Time
	}
	if ackBy.Valid {
		fb.AcknowledgedBy = &ackBy.String
	}
	if staffNotes.Valid {
		fb.StaffNotes = &staffNotes.String
	}
	return &fb, nil
}

func (r *feedbackRepository) scanFeedback(rows *sql.Rows) (*domain.PatientFeedback, error) {
	var fb domain.PatientFeedback
	var sessID, ackBy, staffNotes, comments sql.NullString
	var ackAt sql.NullTime

	err := rows.Scan(
		&fb.ID, &fb.AppointmentID, &sessID, &fb.PatientID, &fb.ClinicID, &fb.DoctorID, &fb.TherapistID,
		&fb.OverallRating, &fb.TherapistRating, &fb.FacilityCleanlinessRating, &comments,
		&fb.IsAnonymous, &fb.Status, &ackAt, &ackBy, &staffNotes, &fb.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if sessID.Valid {
		fb.SessionID = &sessID.String
	}
	if comments.Valid {
		fb.Comments = &comments.String
	}
	if ackAt.Valid {
		fb.AcknowledgedAt = &ackAt.Time
	}
	if ackBy.Valid {
		fb.AcknowledgedBy = &ackBy.String
	}
	if staffNotes.Valid {
		fb.StaffNotes = &staffNotes.String
	}
	return &fb, nil
}

// -----------------------------------------------------------------------------
// ADVERSE EVENTS
// -----------------------------------------------------------------------------

func (r *feedbackRepository) CreateAdverseEvent(ctx context.Context, ae *domain.AdverseEvent) error {
	ae.ID = sanitizeUUID(ae.ID)
	ae.PatientID = sanitizeUUID(ae.PatientID)
	ae.ClinicID = sanitizeUUID(ae.ClinicID)
	ae.ReportedBy = sanitizeUUID(ae.ReportedBy)

	var sessID, aptID, docID, immActions sql.NullString
	if ae.SessionID != nil && *ae.SessionID != "" {
		sessID = sql.NullString{String: sanitizeUUID(*ae.SessionID), Valid: true}
	}
	if ae.AppointmentID != nil && *ae.AppointmentID != "" {
		aptID = sql.NullString{String: sanitizeUUID(*ae.AppointmentID), Valid: true}
	}
	if ae.AttendingDoctorID != nil && *ae.AttendingDoctorID != "" {
		docID = sql.NullString{String: sanitizeUUID(*ae.AttendingDoctorID), Valid: true}
	}
	if ae.ImmediateActionsTaken != nil {
		immActions = sql.NullString{String: *ae.ImmediateActionsTaken, Valid: true}
	}

	vitalsJSON, _ := json.Marshal(ae.VitalsAtIncident)
	now := time.Now().UTC()

	query := `
		INSERT INTO adverse_events (
			id, session_id, appointment_id, patient_id, clinic_id,
			severity, symptoms, vitals_at_incident, immediate_actions_taken,
			status, reported_by, reporter_role, attending_doctor_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15
		) RETURNING created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		ae.ID, sessID, aptID, ae.PatientID, ae.ClinicID,
		ae.Severity, ae.Symptoms, vitalsJSON, immActions,
		"REPORTED", ae.ReportedBy, ae.ReporterRole, docID,
		now, now,
	).Scan(&ae.CreatedAt, &ae.UpdatedAt)
	return err
}

func (r *feedbackRepository) GetAdverseEventByID(ctx context.Context, id string) (*domain.AdverseEvent, error) {
	id = sanitizeUUID(id)
	query := `
		SELECT id, session_id, appointment_id, patient_id, clinic_id,
		       severity, symptoms, vitals_at_incident, immediate_actions_taken,
		       status, reported_by, reporter_role, attending_doctor_id,
		       acknowledged_at, acknowledged_by, staff_notes, resolved_at,
		       created_at, updated_at
		FROM adverse_events
		WHERE id = $1`

	return r.scanAdverseEventRow(r.db.QueryRowContext(ctx, query, id))
}

func (r *feedbackRepository) ListAdverseEventsByClinic(ctx context.Context, clinicID string, severity, status *string, startDate, endDate *time.Time, limit, offset int) ([]*domain.AdverseEvent, int, error) {
	clinicID = sanitizeUUID(clinicID)
	where := "WHERE clinic_id = $1"
	args := []interface{}{clinicID}
	idx := 2

	if severity != nil && *severity != "" {
		where += fmt.Sprintf(" AND severity = $%d", idx)
		args = append(args, *severity)
		idx++
	}
	if status != nil && *status != "" {
		where += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, *status)
		idx++
	}
	if startDate != nil {
		where += fmt.Sprintf(" AND created_at >= $%d", idx)
		args = append(args, *startDate)
		idx++
	}
	if endDate != nil {
		where += fmt.Sprintf(" AND created_at <= $%d", idx)
		args = append(args, *endDate)
		idx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM adverse_events %s", where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, session_id, appointment_id, patient_id, clinic_id,
		       severity, symptoms, vitals_at_incident, immediate_actions_taken,
		       status, reported_by, reporter_role, attending_doctor_id,
		       acknowledged_at, acknowledged_by, staff_notes, resolved_at,
		       created_at, updated_at
		FROM adverse_events
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*domain.AdverseEvent
	for rows.Next() {
		ae, err := r.scanAdverseEvent(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, ae)
	}
	return result, total, nil
}

func (r *feedbackRepository) ListAdverseEventsByPatient(ctx context.Context, patientID string, limit, offset int) ([]*domain.AdverseEvent, int, error) {
	patientID = sanitizeUUID(patientID)
	countQuery := "SELECT COUNT(*) FROM adverse_events WHERE patient_id = $1"
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, patientID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, session_id, appointment_id, patient_id, clinic_id,
		       severity, symptoms, vitals_at_incident, immediate_actions_taken,
		       status, reported_by, reporter_role, attending_doctor_id,
		       acknowledged_at, acknowledged_by, staff_notes, resolved_at,
		       created_at, updated_at
		FROM adverse_events
		WHERE patient_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, patientID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*domain.AdverseEvent
	for rows.Next() {
		ae, err := r.scanAdverseEvent(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, ae)
	}
	return result, total, nil
}

func (r *feedbackRepository) AcknowledgeAdverseEvent(ctx context.Context, id string, ackBy string, notes string, status string, resolved bool) error {
	id = sanitizeUUID(id)
	ackBy = sanitizeUUID(ackBy)
	now := time.Now().UTC()

	var query string
	if resolved {
		query = `
			UPDATE adverse_events
			SET status = $1, acknowledged_at = $2, acknowledged_by = $3, staff_notes = $4, resolved_at = $2, updated_at = $2
			WHERE id = $5`
	} else {
		query = `
			UPDATE adverse_events
			SET status = $1, acknowledged_at = $2, acknowledged_by = $3, staff_notes = $4, updated_at = $2
			WHERE id = $5`
	}

	res, err := r.db.ExecContext(ctx, query, status, now, ackBy, notes, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrAdverseEventNotFound
	}
	return nil
}

func (r *feedbackRepository) scanAdverseEventRow(row *sql.Row) (*domain.AdverseEvent, error) {
	var ae domain.AdverseEvent
	var sessID, aptID, docID, immActions, ackBy, staffNotes sql.NullString
	var ackAt, resAt sql.NullTime
	var vitalsBytes []byte

	err := row.Scan(
		&ae.ID, &sessID, &aptID, &ae.PatientID, &ae.ClinicID,
		&ae.Severity, &ae.Symptoms, &vitalsBytes, &immActions,
		&ae.Status, &ae.ReportedBy, &ae.ReporterRole, &docID,
		&ackAt, &ackBy, &staffNotes, &resAt,
		&ae.CreatedAt, &ae.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrAdverseEventNotFound
		}
		return nil, err
	}

	if sessID.Valid {
		ae.SessionID = &sessID.String
	}
	if aptID.Valid {
		ae.AppointmentID = &aptID.String
	}
	if docID.Valid {
		ae.AttendingDoctorID = &docID.String
	}
	if immActions.Valid {
		ae.ImmediateActionsTaken = &immActions.String
	}
	if ackAt.Valid {
		ae.AcknowledgedAt = &ackAt.Time
	}
	if ackBy.Valid {
		ae.AcknowledgedBy = &ackBy.String
	}
	if staffNotes.Valid {
		ae.StaffNotes = &staffNotes.String
	}
	if resAt.Valid {
		ae.ResolvedAt = &resAt.Time
	}
	if len(vitalsBytes) > 0 {
		_ = json.Unmarshal(vitalsBytes, &ae.VitalsAtIncident)
	}
	return &ae, nil
}

func (r *feedbackRepository) scanAdverseEvent(rows *sql.Rows) (*domain.AdverseEvent, error) {
	var ae domain.AdverseEvent
	var sessID, aptID, docID, immActions, ackBy, staffNotes sql.NullString
	var ackAt, resAt sql.NullTime
	var vitalsBytes []byte

	err := rows.Scan(
		&ae.ID, &sessID, &aptID, &ae.PatientID, &ae.ClinicID,
		&ae.Severity, &ae.Symptoms, &vitalsBytes, &immActions,
		&ae.Status, &ae.ReportedBy, &ae.ReporterRole, &docID,
		&ackAt, &ackBy, &staffNotes, &resAt,
		&ae.CreatedAt, &ae.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if sessID.Valid {
		ae.SessionID = &sessID.String
	}
	if aptID.Valid {
		ae.AppointmentID = &aptID.String
	}
	if docID.Valid {
		ae.AttendingDoctorID = &docID.String
	}
	if immActions.Valid {
		ae.ImmediateActionsTaken = &immActions.String
	}
	if ackAt.Valid {
		ae.AcknowledgedAt = &ackAt.Time
	}
	if ackBy.Valid {
		ae.AcknowledgedBy = &ackBy.String
	}
	if staffNotes.Valid {
		ae.StaffNotes = &staffNotes.String
	}
	if resAt.Valid {
		ae.ResolvedAt = &resAt.Time
	}
	if len(vitalsBytes) > 0 {
		_ = json.Unmarshal(vitalsBytes, &ae.VitalsAtIncident)
	}
	return &ae, nil
}

// -----------------------------------------------------------------------------
// OPERATIONAL PROJECTIONS & RECONCILIATION
// -----------------------------------------------------------------------------

func (r *feedbackRepository) UpsertAppointmentProjection(ctx context.Context, apt *domain.AnalyticsAppointment) error {
	apt.ID = sanitizeUUID(apt.ID)
	apt.ClinicID = sanitizeUUID(apt.ClinicID)
	apt.PatientID = sanitizeUUID(apt.PatientID)

	var docID, roomID, pkgID, pkgName, parentID sql.NullString
	if apt.DoctorID != nil && *apt.DoctorID != "" {
		docID = sql.NullString{String: sanitizeUUID(*apt.DoctorID), Valid: true}
	}
	if apt.RoomID != nil && *apt.RoomID != "" {
		roomID = sql.NullString{String: sanitizeUUID(*apt.RoomID), Valid: true}
	}
	if apt.PackageID != nil && *apt.PackageID != "" {
		pkgID = sql.NullString{String: sanitizeUUID(*apt.PackageID), Valid: true}
	}
	if apt.PackageName != nil {
		pkgName = sql.NullString{String: *apt.PackageName, Valid: true}
	}
	if apt.ParentAppointmentID != nil && *apt.ParentAppointmentID != "" {
		parentID = sql.NullString{String: sanitizeUUID(*apt.ParentAppointmentID), Valid: true}
	}

	query := `
		INSERT INTO analytics_appointments (
			id, clinic_id, patient_id, doctor_id, room_id, package_id, package_name,
			status, scheduled_start, scheduled_end, is_follow_up, parent_appointment_id,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			scheduled_start = EXCLUDED.scheduled_start,
			scheduled_end = EXCLUDED.scheduled_end,
			is_follow_up = EXCLUDED.is_follow_up,
			updated_at = EXCLUDED.updated_at`

	now := time.Now().UTC()
	if apt.CreatedAt.IsZero() {
		apt.CreatedAt = now
	}
	apt.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		apt.ID, apt.ClinicID, apt.PatientID, docID, roomID, pkgID, pkgName,
		apt.Status, apt.ScheduledStart, apt.ScheduledEnd, apt.IsFollowUp, parentID,
		apt.CreatedAt, apt.UpdatedAt,
	)
	return err
}

func (r *feedbackRepository) UpsertSessionProjection(ctx context.Context, sess *domain.AnalyticsSession) error {
	sess.ID = sanitizeUUID(sess.ID)
	sess.AppointmentID = sanitizeUUID(sess.AppointmentID)
	sess.ClinicID = sanitizeUUID(sess.ClinicID)
	sess.PatientID = sanitizeUUID(sess.PatientID)

	var thID sql.NullString
	if sess.TherapistID != nil && *sess.TherapistID != "" {
		thID = sql.NullString{String: sanitizeUUID(*sess.TherapistID), Valid: true}
	}

	var dur sql.NullFloat64
	if sess.DurationMinutes != nil {
		dur = sql.NullFloat64{Float64: *sess.DurationMinutes, Valid: true}
	} else if sess.StartedAt != nil && sess.CompletedAt != nil && sess.CompletedAt.After(*sess.StartedAt) {
		minutes := sess.CompletedAt.Sub(*sess.StartedAt).Minutes()
		dur = sql.NullFloat64{Float64: minutes, Valid: true}
		sess.DurationMinutes = &minutes
	}

	query := `
		INSERT INTO analytics_sessions (
			id, appointment_id, clinic_id, patient_id, therapist_id,
			status, all_steps_completed, total_steps, completed_steps,
			started_at, completed_at, duration_minutes, follow_up_recommended,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			all_steps_completed = EXCLUDED.all_steps_completed,
			total_steps = EXCLUDED.total_steps,
			completed_steps = EXCLUDED.completed_steps,
			started_at = COALESCE(EXCLUDED.started_at, analytics_sessions.started_at),
			completed_at = COALESCE(EXCLUDED.completed_at, analytics_sessions.completed_at),
			duration_minutes = COALESCE(EXCLUDED.duration_minutes, analytics_sessions.duration_minutes),
			follow_up_recommended = EXCLUDED.follow_up_recommended,
			updated_at = EXCLUDED.updated_at`

	now := time.Now().UTC()
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = now
	}
	sess.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		sess.ID, sess.AppointmentID, sess.ClinicID, sess.PatientID, thID,
		sess.Status, sess.AllStepsCompleted, sess.TotalSteps, sess.CompletedSteps,
		sess.StartedAt, sess.CompletedAt, dur, sess.FollowUpRecommended,
		sess.CreatedAt, sess.UpdatedAt,
	)
	return err
}

func (r *feedbackRepository) GetAppointmentProjection(ctx context.Context, appointmentID string) (*domain.AnalyticsAppointment, error) {
	appointmentID = sanitizeUUID(appointmentID)
	query := `
		SELECT id, clinic_id, patient_id, doctor_id, room_id, package_id, package_name,
		       status, scheduled_start, scheduled_end, is_follow_up, parent_appointment_id,
		       created_at, updated_at
		FROM analytics_appointments
		WHERE id = $1`

	var apt domain.AnalyticsAppointment
	var docID, roomID, pkgID, pkgName, parentID sql.NullString
	err := r.db.QueryRowContext(ctx, query, appointmentID).Scan(
		&apt.ID, &apt.ClinicID, &apt.PatientID, &docID, &roomID, &pkgID, &pkgName,
		&apt.Status, &apt.ScheduledStart, &apt.ScheduledEnd, &apt.IsFollowUp, &parentID,
		&apt.CreatedAt, &apt.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if docID.Valid {
		apt.DoctorID = &docID.String
	}
	if roomID.Valid {
		apt.RoomID = &roomID.String
	}
	if pkgID.Valid {
		apt.PackageID = &pkgID.String
	}
	if pkgName.Valid {
		apt.PackageName = &pkgName.String
	}
	if parentID.Valid {
		apt.ParentAppointmentID = &parentID.String
	}
	return &apt, nil
}

func (r *feedbackRepository) GetSessionProjection(ctx context.Context, sessionID string) (*domain.AnalyticsSession, error) {
	sessionID = sanitizeUUID(sessionID)
	query := `
		SELECT id, appointment_id, clinic_id, patient_id, therapist_id,
		       status, all_steps_completed, total_steps, completed_steps,
		       started_at, completed_at, duration_minutes, follow_up_recommended,
		       created_at, updated_at
		FROM analytics_sessions
		WHERE id = $1`

	var sess domain.AnalyticsSession
	var thID sql.NullString
	var dur sql.NullFloat64
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&sess.ID, &sess.AppointmentID, &sess.ClinicID, &sess.PatientID, &thID,
		&sess.Status, &sess.AllStepsCompleted, &sess.TotalSteps, &sess.CompletedSteps,
		&sess.StartedAt, &sess.CompletedAt, &dur, &sess.FollowUpRecommended,
		&sess.CreatedAt, &sess.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if thID.Valid {
		sess.TherapistID = &thID.String
	}
	if dur.Valid {
		sess.DurationMinutes = &dur.Float64
	}
	return &sess, nil
}

// -----------------------------------------------------------------------------
// METRICS CALCULATION (PARAMETERIZED SQL WITH ZERO-DENOMINATOR PROTECTION)
// -----------------------------------------------------------------------------

func (r *feedbackRepository) CalculateClinicMetrics(ctx context.Context, clinicID string, startDate, endDate time.Time, loc *time.Location) (*domain.ClinicDashboardMetrics, error) {
	clinicID = sanitizeUUID(clinicID)

	dash := &domain.ClinicDashboardMetrics{
		ClinicID:  clinicID,
		StartDate: startDate.Format("2006-01-02"),
		EndDate:   endDate.Format("2006-01-02"),
		Timezone:  loc.String(),
		Complications: domain.ComplicationBreakdown{
			BySeverity: make(map[string]int64),
			ByStatus:   make(map[string]int64),
		},
		DailyTrends: []domain.DailyMetricRollup{},
	}

	// 1. Booking Conversion Metric
	// Numerator: Confirmed, Checked-in, In-progress, or Completed appointments
	// Denominator: Total created appointments scheduled in window
	convQuery := `
		SELECT 
			COALESCE(COUNT(*) FILTER (WHERE status IN ('CONFIRMED', 'CHECKED_IN', 'IN_PROGRESS', 'COMPLETED')), 0),
			COALESCE(COUNT(*), 0)
		FROM analytics_appointments
		WHERE clinic_id = $1 AND scheduled_start >= $2 AND scheduled_start <= $3`

	var convNum, convDen int64
	if err := r.db.QueryRowContext(ctx, convQuery, clinicID, startDate, endDate).Scan(&convNum, &convDen); err != nil {
		return nil, fmt.Errorf("booking conversion error: %w", err)
	}
	dash.BookingConversion = domain.MetricResult{
		Numerator:   float64(convNum),
		Denominator: float64(convDen),
		Unit:        "%",
		Description: "Percentage of booked appointments that reached confirmed or completed status",
	}
	if convDen > 0 {
		dash.BookingConversion.Value = math.Round((float64(convNum)/float64(convDen))*10000) / 100
	}

	// 2. Cancellation and No-Show Rates
	// Total scheduled pool: Confirmed, Cancelled, No-Show, Completed, Checked-in, In-progress
	cancelQuery := `
		SELECT 
			COALESCE(COUNT(*) FILTER (WHERE status = 'CANCELLED'), 0),
			COALESCE(COUNT(*) FILTER (WHERE status = 'NO_SHOW'), 0),
			COALESCE(COUNT(*) FILTER (WHERE status IN ('CONFIRMED', 'CANCELLED', 'NO_SHOW', 'COMPLETED', 'CHECKED_IN', 'IN_PROGRESS')), 0)
		FROM analytics_appointments
		WHERE clinic_id = $1 AND scheduled_start >= $2 AND scheduled_start <= $3`

	var cancNum, noShowNum, schedTotal int64
	if err := r.db.QueryRowContext(ctx, cancelQuery, clinicID, startDate, endDate).Scan(&cancNum, &noShowNum, &schedTotal); err != nil {
		return nil, fmt.Errorf("cancellation rate error: %w", err)
	}

	dash.CancellationRate = domain.MetricResult{
		Numerator:   float64(cancNum),
		Denominator: float64(schedTotal),
		Unit:        "%",
		Description: "Proportion of scheduled appointments cancelled",
	}
	if schedTotal > 0 {
		dash.CancellationRate.Value = math.Round((float64(cancNum)/float64(schedTotal))*10000) / 100
	}

	dash.NoShowRate = domain.MetricResult{
		Numerator:   float64(noShowNum),
		Denominator: float64(schedTotal),
		Unit:        "%",
		Description: "Proportion of scheduled appointments resulting in no-show",
	}
	if schedTotal > 0 {
		dash.NoShowRate.Value = math.Round((float64(noShowNum)/float64(schedTotal))*10000) / 100
	}

	// 3. Checklist Adherence
	// Numerator: Completed sessions with all checklist steps completed
	// Denominator: Total completed sessions
	adhQuery := `
		SELECT 
			COALESCE(COUNT(*) FILTER (WHERE all_steps_completed = true), 0),
			COALESCE(COUNT(*), 0)
		FROM analytics_sessions
		WHERE clinic_id = $1 AND status = 'COMPLETED' AND completed_at >= $2 AND completed_at <= $3`

	var adhNum, adhDen int64
	if err := r.db.QueryRowContext(ctx, adhQuery, clinicID, startDate, endDate).Scan(&adhNum, &adhDen); err != nil {
		return nil, fmt.Errorf("checklist adherence error: %w", err)
	}
	dash.ChecklistAdherence = domain.MetricResult{
		Numerator:   float64(adhNum),
		Denominator: float64(adhDen),
		Unit:        "%",
		Description: "Percentage of completed therapy sessions strictly adhering to step protocols without skipped steps",
	}
	if adhDen > 0 {
		dash.ChecklistAdherence.Value = math.Round((float64(adhNum)/float64(adhDen))*10000) / 100
	} else {
		dash.ChecklistAdherence.Value = 100.0 // Default 100% adherence if zero completed
	}

	// 4. Average Session Duration
	// Numerator: Sum of session duration in minutes
	// Denominator: Count of completed sessions with duration recorded
	durQuery := `
		SELECT 
			COALESCE(SUM(duration_minutes), 0.0),
			COALESCE(COUNT(*), 0)
		FROM analytics_sessions
		WHERE clinic_id = $1 AND status = 'COMPLETED' AND completed_at >= $2 AND completed_at <= $3
		  AND duration_minutes IS NOT NULL AND duration_minutes > 0`

	var durSum float64
	var durCount int64
	if err := r.db.QueryRowContext(ctx, durQuery, clinicID, startDate, endDate).Scan(&durSum, &durCount); err != nil {
		return nil, fmt.Errorf("session duration error: %w", err)
	}
	dash.AverageSessionDuration = domain.MetricResult{
		Numerator:   math.Round(durSum*100) / 100,
		Denominator: float64(durCount),
		Unit:        "minutes",
		Description: "Mean duration of procedural therapy sessions in minutes",
	}
	if durCount > 0 {
		dash.AverageSessionDuration.Value = math.Round((durSum/float64(durCount))*10) / 10
	}

	// 5. Average Feedback Rating
	// Numerator: Sum of overall ratings
	// Denominator: Total submitted reviews
	ratingQuery := `
		SELECT 
			COALESCE(SUM(overall_rating), 0),
			COALESCE(COUNT(*), 0)
		FROM patient_feedback
		WHERE clinic_id = $1 AND created_at >= $2 AND created_at <= $3`

	var ratingSum, ratingCount int64
	if err := r.db.QueryRowContext(ctx, ratingQuery, clinicID, startDate, endDate).Scan(&ratingSum, &ratingCount); err != nil {
		return nil, fmt.Errorf("feedback rating error: %w", err)
	}
	dash.AverageFeedbackRating = domain.MetricResult{
		Numerator:   float64(ratingSum),
		Denominator: float64(ratingCount),
		Unit:        "stars",
		Description: "Average patient satisfaction rating (1 to 5 stars)",
	}
	if ratingCount > 0 {
		dash.AverageFeedbackRating.Value = math.Round((float64(ratingSum)/float64(ratingCount))*100) / 100
	}

	// 6. Therapist Utilization
	// Numerator: Total procedural minutes delivered by clinic therapists
	// Denominator: Total working capacity (Distinct active therapists * days in range * 8h * 60m)
	thCountQuery := `
		SELECT COUNT(DISTINCT therapist_id)
		FROM analytics_sessions
		WHERE clinic_id = $1 AND therapist_id IS NOT NULL`

	var activeTherapists int64
	_ = r.db.QueryRowContext(ctx, thCountQuery, clinicID).Scan(&activeTherapists)
	if activeTherapists == 0 {
		activeTherapists = 3 // Minimum baseline therapist pool for multi-room clinic
	}

	days := math.Ceil(endDate.Sub(startDate).Hours()/24.0) + 1
	if days <= 0 {
		days = 1
	}
	capacityMinutes := float64(activeTherapists) * days * 8.0 * 60.0

	dash.TherapistUtilization = domain.MetricResult{
		Numerator:   math.Round(durSum*100) / 100,
		Denominator: capacityMinutes,
		Unit:        "%",
		Description: "Proportion of therapist operational working hours spent in direct procedural delivery",
	}
	if capacityMinutes > 0 {
		dash.TherapistUtilization.Value = math.Min(100.0, math.Round((durSum/capacityMinutes)*10000)/100)
	}

	// 7. Follow-Up Completion Rate
	// Numerator: Completed follow-up consultations in window
	// Denominator: Completed sessions recommending follow-up
	fuQuery := `
		SELECT 
			(SELECT COALESCE(COUNT(*), 0) FROM analytics_appointments 
			 WHERE clinic_id = $1 AND is_follow_up = true AND scheduled_start >= $2 AND scheduled_start <= $3
			   AND status IN ('CONFIRMED', 'CHECKED_IN', 'IN_PROGRESS', 'COMPLETED')),
			(SELECT COALESCE(COUNT(*), 0) FROM analytics_sessions 
			 WHERE clinic_id = $1 AND follow_up_recommended = true AND completed_at >= $2 AND completed_at <= $3)`

	var fuNum, fuDen int64
	if err := r.db.QueryRowContext(ctx, fuQuery, clinicID, startDate, endDate).Scan(&fuNum, &fuDen); err != nil {
		return nil, fmt.Errorf("follow-up completion error: %w", err)
	}
	dash.FollowUpCompletion = domain.MetricResult{
		Numerator:   float64(fuNum),
		Denominator: float64(fuDen),
		Unit:        "%",
		Description: "Percentage of discharged patients who completed their recommended follow-up consultation",
	}
	if fuDen > 0 {
		dash.FollowUpCompletion.Value = math.Round((float64(fuNum)/float64(fuDen))*10000) / 100
	}

	// 8. Reported Complications Counts & Severity Breakdown
	compQuery := `
		SELECT 
			COALESCE(COUNT(*), 0),
			COALESCE(COUNT(*) FILTER (WHERE status = 'RESOLVED'), 0),
			COALESCE(COUNT(*) FILTER (WHERE severity = 'LOW'), 0),
			COALESCE(COUNT(*) FILTER (WHERE severity = 'MEDIUM'), 0),
			COALESCE(COUNT(*) FILTER (WHERE severity = 'HIGH'), 0),
			COALESCE(COUNT(*) FILTER (WHERE severity = 'CRITICAL'), 0),
			COALESCE(COUNT(*) FILTER (WHERE status = 'REPORTED'), 0),
			COALESCE(COUNT(*) FILTER (WHERE status = 'UNDER_REVIEW'), 0),
			COALESCE(COUNT(*) FILTER (WHERE status = 'ACKNOWLEDGED'), 0)
		FROM adverse_events
		WHERE clinic_id = $1 AND created_at >= $2 AND created_at <= $3`

	var totComp, resComp, lowC, medC, highC, critC, repS, revS, ackS int64
	if err := r.db.QueryRowContext(ctx, compQuery, clinicID, startDate, endDate).Scan(
		&totComp, &resComp, &lowC, &medC, &highC, &critC, &repS, &revS, &ackS,
	); err != nil {
		return nil, fmt.Errorf("complications breakdown error: %w", err)
	}

	dash.Complications.TotalCount = totComp
	if totComp > 0 {
		dash.Complications.ResolvedRate = math.Round((float64(resComp)/float64(totComp))*10000) / 100
	}
	dash.Complications.BySeverity["LOW"] = lowC
	dash.Complications.BySeverity["MEDIUM"] = medC
	dash.Complications.BySeverity["HIGH"] = highC
	dash.Complications.BySeverity["CRITICAL"] = critC

	dash.Complications.ByStatus["REPORTED"] = repS
	dash.Complications.ByStatus["UNDER_REVIEW"] = revS
	dash.Complications.ByStatus["ACKNOWLEDGED"] = ackS
	dash.Complications.ByStatus["RESOLVED"] = resComp

	// 9. Daily Trends Rollup
	trendsQuery := `
		SELECT 
			d.metric_date::text,
			COALESCE(a.total_apt, 0),
			COALESCE(s.completed_sess, 0),
			COALESCE(a.cancelled_apt, 0),
			COALESCE(a.noshow_apt, 0),
			COALESCE(s.avg_dur, 0.0),
			f.avg_rating,
			COALESCE(c.comp_count, 0)
		FROM (
			SELECT generate_series($2::date, $3::date, '1 day'::interval)::date AS metric_date
		) d
		LEFT JOIN (
			SELECT 
				scheduled_start::date AS d_date,
				COUNT(*) AS total_apt,
				COUNT(*) FILTER (WHERE status = 'CANCELLED') AS cancelled_apt,
				COUNT(*) FILTER (WHERE status = 'NO_SHOW') AS noshow_apt
			FROM analytics_appointments
			WHERE clinic_id = $1
			GROUP BY scheduled_start::date
		) a ON d.metric_date = a.d_date
		LEFT JOIN (
			SELECT 
				completed_at::date AS s_date,
				COUNT(*) AS completed_sess,
				AVG(duration_minutes) AS avg_dur
			FROM analytics_sessions
			WHERE clinic_id = $1 AND status = 'COMPLETED'
			GROUP BY completed_at::date
		) s ON d.metric_date = s.s_date
		LEFT JOIN (
			SELECT 
				created_at::date AS f_date,
				AVG(overall_rating) AS avg_rating
			FROM patient_feedback
			WHERE clinic_id = $1
			GROUP BY created_at::date
		) f ON d.metric_date = f.f_date
		LEFT JOIN (
			SELECT 
				created_at::date AS c_date,
				COUNT(*) AS comp_count
			FROM adverse_events
			WHERE clinic_id = $1
			GROUP BY created_at::date
		) c ON d.metric_date = c.c_date
		ORDER BY d.metric_date ASC`

	tRows, err := r.db.QueryContext(ctx, trendsQuery, clinicID, startDate, endDate)
	if err == nil {
		defer tRows.Close()
		for tRows.Next() {
			var dt domain.DailyMetricRollup
			var avgRating sql.NullFloat64
			if err := tRows.Scan(
				&dt.Date, &dt.TotalAppointments, &dt.CompletedSessions,
				&dt.CancelledCount, &dt.NoShowCount, &dt.AvgSessionMinutes,
				&avgRating, &dt.ComplicationsReported,
			); err == nil {
				if avgRating.Valid {
					rounded := math.Round(avgRating.Float64*100) / 100
					dt.AverageRating = &rounded
				}
				dt.AvgSessionMinutes = math.Round(dt.AvgSessionMinutes*10) / 10
				dash.DailyTrends = append(dash.DailyTrends, dt)
			}
		}
	}

	return dash, nil
}

// -----------------------------------------------------------------------------
// KAFKA DEDUPLICATION LEDGER
// -----------------------------------------------------------------------------

func (r *feedbackRepository) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM consumed_events WHERE event_id = $1)"
	err := r.db.QueryRowContext(ctx, query, eventID).Scan(&exists)
	return exists, err
}

func (r *feedbackRepository) RecordConsumedEvent(ctx context.Context, eventID, eventType, consumerGroup, payloadDigest string) error {
	query := `
		INSERT INTO consumed_events (event_id, event_type, consumer_group, processed_at, payload_digest)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, eventID, eventType, consumerGroup, time.Now().UTC(), payloadDigest)
	return err
}

// -----------------------------------------------------------------------------
// TRANSACTIONAL OUTBOX
// -----------------------------------------------------------------------------

func (r *feedbackRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, trace_id, status, retry_count, created_at, processed_at
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
		var ev domain.OutboxEvent
		var procAt sql.NullTime
		err := rows.Scan(
			&ev.ID, &ev.AggregateType, &ev.AggregateID, &ev.EventType,
			&ev.Payload, &ev.TraceID, &ev.Status, &ev.RetryCount,
			&ev.CreatedAt, &procAt,
		)
		if err != nil {
			return nil, err
		}
		if procAt.Valid {
			ev.ProcessedAt = &procAt.Time
		}
		events = append(events, &ev)
	}
	return events, nil
}

func (r *feedbackRepository) MarkOutboxEventPublished(ctx context.Context, id string) error {
	id = sanitizeUUID(id)
	now := time.Now().UTC()
	query := "UPDATE outbox_events SET status = 'PUBLISHED', processed_at = $1 WHERE id = $2"
	_, err := r.db.ExecContext(ctx, query, now, id)
	return err
}

func (r *feedbackRepository) MarkOutboxEventFailed(ctx context.Context, id string) error {
	id = sanitizeUUID(id)
	query := "UPDATE outbox_events SET retry_count = retry_count + 1 WHERE id = $1"
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
