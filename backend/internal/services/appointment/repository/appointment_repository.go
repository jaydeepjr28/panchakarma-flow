package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/appointment/domain"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
)

// ListFilter provides query parameters for listing appointments.
type ListFilter struct {
	ClinicID  string
	PatientID *string
	DoctorID  *string
	Status    *domain.AppointmentStatus
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
}

// TimeRange represents a busy interval.
type TimeRange struct {
	StartTime time.Time
	EndTime   time.Time
	DoctorID  string
	RoomID    string
}

// AppointmentRepository defines data access for the appointment service.
type AppointmentRepository interface {
	// Holds
	CreateHold(ctx context.Context, hold *domain.AppointmentHold) error
	GetHoldByToken(ctx context.Context, token string) (*domain.AppointmentHold, error)
	GetHoldByID(ctx context.Context, holdID string) (*domain.AppointmentHold, error)
	UpdateHoldStatus(ctx context.Context, holdID string, status domain.HoldStatus) error
	ExpireStaleHolds(ctx context.Context) (int64, error)
	CheckActiveHoldConflict(ctx context.Context, doctorID, roomID string, start, end time.Time) (bool, error)

	// Appointments
	CreateAppointmentWithTx(ctx context.Context, appt *domain.Appointment, holdID *string, outbox *domain.OutboxEvent) error
	GetAppointmentByID(ctx context.Context, id string) (*domain.Appointment, error)
	GetAppointmentByIdempotencyKey(ctx context.Context, key string) (*domain.Appointment, error)
	ListAppointments(ctx context.Context, filter ListFilter) ([]*domain.Appointment, int, error)
	UpdateAppointmentStatusWithTx(ctx context.Context, apptID string, fromStatus, toStatus domain.AppointmentStatus, reason string, changedBy *string, outbox *domain.OutboxEvent) error
	RescheduleAppointmentWithTx(ctx context.Context, oldApptID string, newAppt *domain.Appointment, outbox *domain.OutboxEvent) error

	// Waitlist
	CreateWaitlistEntry(ctx context.Context, entry *domain.AppointmentWaitlist) error
	GetWaitlistEntry(ctx context.Context, id string) (*domain.AppointmentWaitlist, error)
	ListWaitlistEntries(ctx context.Context, clinicID string, patientID *string, status *domain.WaitlistStatus) ([]*domain.AppointmentWaitlist, error)
	FindNextWaitlistMatch(ctx context.Context, clinicID, packageID string, doctorID *string, dateStr string) (*domain.AppointmentWaitlist, error)
	OfferWaitlistSlot(ctx context.Context, waitlistID string, start, end, expiresAt time.Time, outbox *domain.OutboxEvent) error
	AcceptWaitlistOffer(ctx context.Context, waitlistID string) error

	// Outbox & Audits
	CreateOutboxEvent(ctx context.Context, event *domain.OutboxEvent) error
	GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
	MarkOutboxEventProcessed(ctx context.Context, id string) error
	MarkOutboxEventFailed(ctx context.Context, id string, reason string) error
	GetAppointmentHistory(ctx context.Context, appointmentID string) ([]*domain.AppointmentHistory, error)

	// Availability Booked Ranges
	GetBookedRanges(ctx context.Context, clinicID string, doctorID, roomID *string, start, end time.Time) ([]TimeRange, error)
}

type postgresAppointmentRepository struct {
	db *database.DB
}

// NewPostgresAppointmentRepository initializes a PostgreSQL appointment repository.
func NewPostgresAppointmentRepository(db *database.DB) AppointmentRepository {
	return &postgresAppointmentRepository{db: db}
}

// CreateHold inserts a provisional hold row.
func (r *postgresAppointmentRepository) CreateHold(ctx context.Context, hold *domain.AppointmentHold) error {
	if hold.ID == "" {
		hold.ID = uuid.New().String()
	}
	if hold.CreatedAt.IsZero() {
		hold.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO appointment_holds (
			id, hold_token, patient_id, clinic_id, doctor_id, room_id, package_id,
			slot_range, session_number, total_sessions, package_booking_id,
			expires_at, status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			tstzrange($8, $9, '[)'), $10, $11, $12,
			$13, $14, $15
		)`

	_, err := r.db.ExecContext(ctx, query,
		hold.ID, hold.HoldToken, hold.PatientID, hold.ClinicID, hold.DoctorID, hold.RoomID, hold.PackageID,
		hold.StartTime, hold.EndTime, hold.SessionNumber, hold.TotalSessions, hold.PackageBookingID,
		hold.ExpiresAt, string(hold.Status), hold.CreatedAt,
	)
	if err != nil {
		return r.mapError(err)
	}
	return nil
}

// GetHoldByToken fetches an appointment hold by token.
func (r *postgresAppointmentRepository) GetHoldByToken(ctx context.Context, token string) (*domain.AppointmentHold, error) {
	query := `
		SELECT id, hold_token, patient_id, clinic_id, doctor_id, room_id, package_id,
		       lower(slot_range) AS start_time, upper(slot_range) AS end_time,
		       session_number, total_sessions, package_booking_id,
		       expires_at, status, created_at
		FROM appointment_holds
		WHERE hold_token = $1`

	var h domain.AppointmentHold
	var statusStr string
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&h.ID, &h.HoldToken, &h.PatientID, &h.ClinicID, &h.DoctorID, &h.RoomID, &h.PackageID,
		&h.StartTime, &h.EndTime,
		&h.SessionNumber, &h.TotalSessions, &h.PackageBookingID,
		&h.ExpiresAt, &statusStr, &h.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrHoldNotFound
		}
		return nil, err
	}
	h.Status = domain.HoldStatus(statusStr)
	return &h, nil
}

// GetHoldByID fetches an appointment hold by ID.
func (r *postgresAppointmentRepository) GetHoldByID(ctx context.Context, holdID string) (*domain.AppointmentHold, error) {
	query := `
		SELECT id, hold_token, patient_id, clinic_id, doctor_id, room_id, package_id,
		       lower(slot_range) AS start_time, upper(slot_range) AS end_time,
		       session_number, total_sessions, package_booking_id,
		       expires_at, status, created_at
		FROM appointment_holds
		WHERE id = $1`

	var h domain.AppointmentHold
	var statusStr string
	err := r.db.QueryRowContext(ctx, query, holdID).Scan(
		&h.ID, &h.HoldToken, &h.PatientID, &h.ClinicID, &h.DoctorID, &h.RoomID, &h.PackageID,
		&h.StartTime, &h.EndTime,
		&h.SessionNumber, &h.TotalSessions, &h.PackageBookingID,
		&h.ExpiresAt, &statusStr, &h.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrHoldNotFound
		}
		return nil, err
	}
	h.Status = domain.HoldStatus(statusStr)
	return &h, nil
}

// UpdateHoldStatus updates the hold status.
func (r *postgresAppointmentRepository) UpdateHoldStatus(ctx context.Context, holdID string, status domain.HoldStatus) error {
	query := `UPDATE appointment_holds SET status = $1 WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, string(status), holdID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrHoldNotFound
	}
	return nil
}

// ExpireStaleHolds transitions active holds past expiration to 'EXPIRED'.
func (r *postgresAppointmentRepository) ExpireStaleHolds(ctx context.Context) (int64, error) {
	query := `
		UPDATE appointment_holds
		SET status = 'EXPIRED'
		WHERE status = 'ACTIVE' AND expires_at <= CURRENT_TIMESTAMP`

	res, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// CheckActiveHoldConflict verifies whether another active hold or confirmed appointment covers doctor or room during slot.
func (r *postgresAppointmentRepository) CheckActiveHoldConflict(ctx context.Context, doctorID, roomID string, start, end time.Time) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM appointment_holds
			WHERE status = 'ACTIVE'
			  AND expires_at > CURRENT_TIMESTAMP
			  AND (doctor_id = $1 OR room_id = $2)
			  AND slot_range && tstzrange($3, $4, '[)')
			UNION ALL
			SELECT 1 FROM appointments
			WHERE status IN ('CONFIRMED', 'CHECKED_IN', 'IN_PROGRESS')
			  AND (doctor_id = $1 OR room_id = $2)
			  AND slot_range && tstzrange($3, $4, '[)')
		)`
	var conflict bool
	err := r.db.QueryRowContext(ctx, query, doctorID, roomID, start, end).Scan(&conflict)
	if err != nil {
		return false, err
	}
	return conflict, nil
}

// CreateAppointmentWithTx inserts an appointment inside an authoritative PostgreSQL transaction.
func (r *postgresAppointmentRepository) CreateAppointmentWithTx(ctx context.Context, appt *domain.Appointment, holdID *string, outbox *domain.OutboxEvent) error {
	if appt.ID == "" {
		appt.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if appt.CreatedAt.IsZero() {
		appt.CreatedAt = now
	}
	appt.UpdatedAt = now

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		// 1. If holdID provided, transition hold to CONFIRMED
		if holdID != nil && *holdID != "" {
			res, err := tx.ExecContext(ctx, `UPDATE appointment_holds SET status = 'CONFIRMED' WHERE id = $1 AND status = 'ACTIVE'`, *holdID)
			if err != nil {
				return err
			}
			rows, _ := res.RowsAffected()
			if rows == 0 {
				return domain.ErrHoldExpired
			}
		}

		// 2. Authoritative PostgreSQL Insert - GiST Exclusion constraint enforces zero double booking
		insertApptQuery := `
			INSERT INTO appointments (
				id, patient_id, clinic_id, doctor_id, room_id, package_id,
				slot_range, status, session_number, total_sessions, package_booking_id,
				idempotency_key, notes, cancellation_reason, approved_by, approved_at,
				rejection_reason, rescheduled_to_id, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				tstzrange($7, $8, '[)'), $9, $10, $11, $12,
				$13, $14, $15, $16, $17,
				$18, $19, $20, $21
			)`

		_, err := tx.ExecContext(ctx, insertApptQuery,
			appt.ID, appt.PatientID, appt.ClinicID, appt.DoctorID, appt.RoomID, appt.PackageID,
			appt.StartTime, appt.EndTime, string(appt.Status), appt.SessionNumber, appt.TotalSessions, appt.PackageBookingID,
			appt.IdempotencyKey, appt.Notes, appt.CancellationReason, appt.ApprovedBy, appt.ApprovedAt,
			appt.RejectionReason, appt.RescheduledToID, appt.CreatedAt, appt.UpdatedAt,
		)
		if err != nil {
			return r.mapError(err)
		}

		// 3. Record Initial History
		historyQuery := `
			INSERT INTO appointment_history (
				id, appointment_id, from_status, to_status, changed_by, reason, created_at
			) VALUES ($1, $2, NULL, $3, $4, $5, $6)`
		_, err = tx.ExecContext(ctx, historyQuery,
			uuid.New().String(), appt.ID, string(appt.Status), appt.ApprovedBy, "Initial creation", now,
		)
		if err != nil {
			return err
		}

		// 4. Record Transactional Outbox Event
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
				outbox.Payload, outbox.TraceID, string(outbox.Status), outbox.RetryCount, outbox.CreatedAt,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// GetAppointmentByID retrieves an appointment by its ID.
func (r *postgresAppointmentRepository) GetAppointmentByID(ctx context.Context, id string) (*domain.Appointment, error) {
	query := `
		SELECT id, patient_id, clinic_id, doctor_id, room_id, package_id,
		       lower(slot_range) AS start_time, upper(slot_range) AS end_time,
		       status, session_number, total_sessions, package_booking_id,
		       idempotency_key, notes, cancellation_reason, approved_by, approved_at,
		       rejection_reason, rescheduled_to_id, created_at, updated_at
		FROM appointments
		WHERE id = $1`

	return r.scanAppointment(r.db.QueryRowContext(ctx, query, id))
}

// GetAppointmentByIdempotencyKey retrieves an appointment by idempotency key.
func (r *postgresAppointmentRepository) GetAppointmentByIdempotencyKey(ctx context.Context, key string) (*domain.Appointment, error) {
	query := `
		SELECT id, patient_id, clinic_id, doctor_id, room_id, package_id,
		       lower(slot_range) AS start_time, upper(slot_range) AS end_time,
		       status, session_number, total_sessions, package_booking_id,
		       idempotency_key, notes, cancellation_reason, approved_by, approved_at,
		       rejection_reason, rescheduled_to_id, created_at, updated_at
		FROM appointments
		WHERE idempotency_key = $1`

	return r.scanAppointment(r.db.QueryRowContext(ctx, query, key))
}

// ListAppointments retrieves appointments matching filters with count.
func (r *postgresAppointmentRepository) ListAppointments(ctx context.Context, filter ListFilter) ([]*domain.Appointment, int, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	if filter.ClinicID != "" {
		conditions = append(conditions, "clinic_id = $"+string(rune('0'+idx)))
		args = append(args, filter.ClinicID)
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
	if filter.StartDate != nil {
		conditions = append(conditions, "lower(slot_range) >= $"+string(rune('0'+idx)))
		args = append(args, *filter.StartDate)
		idx++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, "upper(slot_range) <= $"+string(rune('0'+idx)))
		args = append(args, *filter.EndDate)
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM appointments " + whereClause
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
		SELECT id, patient_id, clinic_id, doctor_id, room_id, package_id,
		       lower(slot_range) AS start_time, upper(slot_range) AS end_time,
		       status, session_number, total_sessions, package_booking_id,
		       idempotency_key, notes, cancellation_reason, approved_by, approved_at,
		       rejection_reason, rescheduled_to_id, created_at, updated_at
		FROM appointments ` + whereClause + `
		ORDER BY lower(slot_range) ASC
		LIMIT $` + string(rune('0'+idx)) + ` OFFSET $` + string(rune('0'+idx+1))

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*domain.Appointment
	for rows.Next() {
		var a domain.Appointment
		var statusStr string
		err := rows.Scan(
			&a.ID, &a.PatientID, &a.ClinicID, &a.DoctorID, &a.RoomID, &a.PackageID,
			&a.StartTime, &a.EndTime,
			&statusStr, &a.SessionNumber, &a.TotalSessions, &a.PackageBookingID,
			&a.IdempotencyKey, &a.Notes, &a.CancellationReason, &a.ApprovedBy, &a.ApprovedAt,
			&a.RejectionReason, &a.RescheduledToID, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		a.Status = domain.AppointmentStatus(statusStr)
		result = append(result, &a)
	}

	return result, total, nil
}

// UpdateAppointmentStatusWithTx transitions appointment status with FOR UPDATE row lock.
func (r *postgresAppointmentRepository) UpdateAppointmentStatusWithTx(
	ctx context.Context,
	apptID string,
	fromStatus, toStatus domain.AppointmentStatus,
	reason string,
	changedBy *string,
	outbox *domain.OutboxEvent,
) error {
	now := time.Now().UTC()

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		// 1. SELECT ... FOR UPDATE ensures no concurrent state race (e.g. Cancel vs Check-in)
		var currentStatusStr string
		err := tx.QueryRowContext(ctx, `SELECT status FROM appointments WHERE id = $1 FOR UPDATE`, apptID).Scan(&currentStatusStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrAppointmentNotFound
			}
			return err
		}

		currentStatus := domain.AppointmentStatus(currentStatusStr)
		if fromStatus != "" && currentStatus != fromStatus {
			return domain.ErrInvalidStateTransition
		}

		if err := domain.ValidateTransition(currentStatus, toStatus); err != nil {
			return err
		}

		// 2. Perform Update
		updateQuery := `
			UPDATE appointments
			SET status = $1, cancellation_reason = CASE WHEN $1 = 'CANCELLED' THEN $2 ELSE cancellation_reason END,
			    updated_at = $3
			WHERE id = $4`
		_, err = tx.ExecContext(ctx, updateQuery, string(toStatus), reason, now, apptID)
		if err != nil {
			return err
		}

		// 3. Insert History Record
		historyQuery := `
			INSERT INTO appointment_history (
				id, appointment_id, from_status, to_status, changed_by, reason, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err = tx.ExecContext(ctx, historyQuery,
			uuid.New().String(), apptID, string(currentStatus), string(toStatus), changedBy, reason, now,
		)
		if err != nil {
			return err
		}

		// 4. Insert Outbox Event
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
				outbox.Payload, outbox.TraceID, string(outbox.Status), outbox.RetryCount, outbox.CreatedAt,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// RescheduleAppointmentWithTx atomically swaps old appointment for new appointment in a single ACID transaction.
func (r *postgresAppointmentRepository) RescheduleAppointmentWithTx(
	ctx context.Context,
	oldApptID string,
	newAppt *domain.Appointment,
	outbox *domain.OutboxEvent,
) error {
	if newAppt.ID == "" {
		newAppt.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if newAppt.CreatedAt.IsZero() {
		newAppt.CreatedAt = now
	}
	newAppt.UpdatedAt = now

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		// 1. Lock and verify old appointment
		var oldStatusStr string
		err := tx.QueryRowContext(ctx, `SELECT status FROM appointments WHERE id = $1 FOR UPDATE`, oldApptID).Scan(&oldStatusStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrAppointmentNotFound
			}
			return err
		}

		oldStatus := domain.AppointmentStatus(oldStatusStr)
		if err := domain.ValidateTransition(oldStatus, domain.StatusRescheduled); err != nil {
			return err
		}

		// 2. Insert new appointment (GiST exclusion constraint validates doctor & room availability)
		insertQuery := `
			INSERT INTO appointments (
				id, patient_id, clinic_id, doctor_id, room_id, package_id,
				slot_range, status, session_number, total_sessions, package_booking_id,
				idempotency_key, notes, cancellation_reason, approved_by, approved_at,
				rejection_reason, rescheduled_to_id, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				tstzrange($7, $8, '[)'), $9, $10, $11, $12,
				$13, $14, $15, $16, $17,
				$18, $19, $20, $21
			)`

		_, err = tx.ExecContext(ctx, insertQuery,
			newAppt.ID, newAppt.PatientID, newAppt.ClinicID, newAppt.DoctorID, newAppt.RoomID, newAppt.PackageID,
			newAppt.StartTime, newAppt.EndTime, string(newAppt.Status), newAppt.SessionNumber, newAppt.TotalSessions, newAppt.PackageBookingID,
			newAppt.IdempotencyKey, newAppt.Notes, newAppt.CancellationReason, newAppt.ApprovedBy, newAppt.ApprovedAt,
			newAppt.RejectionReason, newAppt.RescheduledToID, newAppt.CreatedAt, newAppt.UpdatedAt,
		)
		if err != nil {
			return r.mapError(err)
		}

		// 3. Mark old appointment as RESCHEDULED pointing to newAppt.ID
		updateOldQuery := `
			UPDATE appointments
			SET status = 'RESCHEDULED', rescheduled_to_id = $1, updated_at = $2
			WHERE id = $3`
		_, err = tx.ExecContext(ctx, updateOldQuery, newAppt.ID, now, oldApptID)
		if err != nil {
			return err
		}

		// 4. Insert histories
		histOldQuery := `
			INSERT INTO appointment_history (
				id, appointment_id, from_status, to_status, changed_by, reason, created_at
			) VALUES ($1, $2, $3, 'RESCHEDULED', NULL, $4, $5)`
		_, err = tx.ExecContext(ctx, histOldQuery,
			uuid.New().String(), oldApptID, string(oldStatus), "Rescheduled to appointment "+newAppt.ID, now,
		)
		if err != nil {
			return err
		}

		histNewQuery := `
			INSERT INTO appointment_history (
				id, appointment_id, from_status, to_status, changed_by, reason, created_at
			) VALUES ($1, $2, NULL, $3, NULL, $4, $5)`
		_, err = tx.ExecContext(ctx, histNewQuery,
			uuid.New().String(), newAppt.ID, string(newAppt.Status), "Rescheduled from appointment "+oldApptID, now,
		)
		if err != nil {
			return err
		}

		// 5. Insert Outbox Event
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
				outbox.Payload, outbox.TraceID, string(outbox.Status), outbox.RetryCount, outbox.CreatedAt,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// CreateWaitlistEntry registers a patient's waitlist request.
func (r *postgresAppointmentRepository) CreateWaitlistEntry(ctx context.Context, entry *domain.AppointmentWaitlist) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now

	query := `
		INSERT INTO appointment_waitlists (
			id, clinic_id, patient_id, package_id, preferred_doctor_id,
			preferred_date_start, preferred_date_end, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID, entry.ClinicID, entry.PatientID, entry.PackageID, entry.PreferredDoctorID,
		entry.PreferredDateStart, entry.PreferredDateEnd, string(entry.Status), entry.CreatedAt, entry.UpdatedAt,
	)
	return err
}

// GetWaitlistEntry fetches a single waitlist entry.
func (r *postgresAppointmentRepository) GetWaitlistEntry(ctx context.Context, id string) (*domain.AppointmentWaitlist, error) {
	query := `
		SELECT id, clinic_id, patient_id, package_id, preferred_doctor_id,
		       preferred_date_start, preferred_date_end, status,
		       offered_slot_start, offered_slot_end, offer_expires_at,
		       created_at, updated_at
		FROM appointment_waitlists
		WHERE id = $1`

	var w domain.AppointmentWaitlist
	var statusStr string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&w.ID, &w.ClinicID, &w.PatientID, &w.PackageID, &w.PreferredDoctorID,
		&w.PreferredDateStart, &w.PreferredDateEnd, &statusStr,
		&w.OfferedSlotStart, &w.OfferedSlotEnd, &w.OfferExpiresAt,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWaitlistNotFound
		}
		return nil, err
	}
	w.Status = domain.WaitlistStatus(statusStr)
	return &w, nil
}

// ListWaitlistEntries retrieves waitlist entries filtered by clinic, patient, or status.
func (r *postgresAppointmentRepository) ListWaitlistEntries(ctx context.Context, clinicID string, patientID *string, status *domain.WaitlistStatus) ([]*domain.AppointmentWaitlist, error) {
	query := `
		SELECT id, clinic_id, patient_id, package_id, preferred_doctor_id,
		       preferred_date_start, preferred_date_end, status,
		       offered_slot_start, offered_slot_end, offer_expires_at,
		       created_at, updated_at
		FROM appointment_waitlists
		WHERE clinic_id = $1`

	args := []interface{}{clinicID}
	idx := 2

	if patientID != nil && *patientID != "" {
		query += ` AND patient_id = $` + string(rune('0'+idx))
		args = append(args, *patientID)
		idx++
	}
	if status != nil && *status != "" {
		query += ` AND status = $` + string(rune('0'+idx))
		args = append(args, string(*status))
		idx++
	}

	query += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.AppointmentWaitlist
	for rows.Next() {
		var w domain.AppointmentWaitlist
		var statusStr string
		err := rows.Scan(
			&w.ID, &w.ClinicID, &w.PatientID, &w.PackageID, &w.PreferredDoctorID,
			&w.PreferredDateStart, &w.PreferredDateEnd, &statusStr,
			&w.OfferedSlotStart, &w.OfferedSlotEnd, &w.OfferExpiresAt,
			&w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		w.Status = domain.WaitlistStatus(statusStr)
		list = append(list, &w)
	}

	return list, nil
}

// FindNextWaitlistMatch locates the earliest queued patient matching the freed appointment slot.
func (r *postgresAppointmentRepository) FindNextWaitlistMatch(ctx context.Context, clinicID, packageID string, doctorID *string, dateStr string) (*domain.AppointmentWaitlist, error) {
	query := `
		SELECT id, clinic_id, patient_id, package_id, preferred_doctor_id,
		       preferred_date_start, preferred_date_end, status,
		       offered_slot_start, offered_slot_end, offer_expires_at,
		       created_at, updated_at
		FROM appointment_waitlists
		WHERE clinic_id = $1
		  AND package_id = $2
		  AND status = 'WAITING'
		  AND preferred_date_start <= $3::date
		  AND preferred_date_end >= $3::date
		  AND (preferred_doctor_id IS NULL OR preferred_doctor_id = $4 OR $4 IS NULL)
		ORDER BY created_at ASC
		LIMIT 1`

	var w domain.AppointmentWaitlist
	var statusStr string
	err := r.db.QueryRowContext(ctx, query, clinicID, packageID, dateStr, doctorID).Scan(
		&w.ID, &w.ClinicID, &w.PatientID, &w.PackageID, &w.PreferredDoctorID,
		&w.PreferredDateStart, &w.PreferredDateEnd, &statusStr,
		&w.OfferedSlotStart, &w.OfferedSlotEnd, &w.OfferExpiresAt,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrWaitlistNotFound
		}
		return nil, err
	}
	w.Status = domain.WaitlistStatus(statusStr)
	return &w, nil
}

// OfferWaitlistSlot records an offered slot for a waitlisted patient with expiration.
func (r *postgresAppointmentRepository) OfferWaitlistSlot(ctx context.Context, waitlistID string, start, end, expiresAt time.Time, outbox *domain.OutboxEvent) error {
	now := time.Now().UTC()

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		query := `
			UPDATE appointment_waitlists
			SET status = 'OFFERED', offered_slot_start = $1, offered_slot_end = $2,
			    offer_expires_at = $3, updated_at = $4
			WHERE id = $5 AND status = 'WAITING'`

		res, err := tx.ExecContext(ctx, query, start, end, expiresAt, now, waitlistID)
		if err != nil {
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return domain.ErrWaitlistNotFound
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
				outbox.Payload, outbox.TraceID, string(outbox.Status), outbox.RetryCount, outbox.CreatedAt,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// AcceptWaitlistOffer transitions an offered waitlist entry to fulfilled.
func (r *postgresAppointmentRepository) AcceptWaitlistOffer(ctx context.Context, waitlistID string) error {
	now := time.Now().UTC()
	query := `
		UPDATE appointment_waitlists
		SET status = 'FULFILLED', updated_at = $1
		WHERE id = $2 AND status = 'OFFERED' AND offer_expires_at > $1`

	res, err := r.db.ExecContext(ctx, query, now, waitlistID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrWaitlistOfferExpired
	}
	return nil
}

// CreateOutboxEvent persists an asynchronous domain event record.
func (r *postgresAppointmentRepository) CreateOutboxEvent(ctx context.Context, event *domain.OutboxEvent) error {
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
		event.Payload, event.TraceID, string(event.Status), event.RetryCount, event.CreatedAt,
	)
	return err
}

// GetPendingOutboxEvents fetches uncommitted outbox events for Kafka dispatch.
func (r *postgresAppointmentRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
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
		var statusStr string
		err := rows.Scan(
			&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload, &e.TraceID,
			&statusStr, &e.RetryCount, &e.CreatedAt, &e.ProcessedAt,
		)
		if err != nil {
			return nil, err
		}
		e.Status = domain.OutboxStatus(statusStr)
		events = append(events, &e)
	}
	return events, nil
}

// MarkOutboxEventProcessed marks outbox entry as published.
func (r *postgresAppointmentRepository) MarkOutboxEventProcessed(ctx context.Context, id string) error {
	now := time.Now().UTC()
	query := `UPDATE outbox_events SET status = 'PROCESSED', processed_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, now, id)
	return err
}

// MarkOutboxEventFailed increments retry counter and marks status.
func (r *postgresAppointmentRepository) MarkOutboxEventFailed(ctx context.Context, id string, reason string) error {
	query := `UPDATE outbox_events SET retry_count = retry_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetAppointmentHistory retrieves chronological transition logs for an appointment.
func (r *postgresAppointmentRepository) GetAppointmentHistory(ctx context.Context, appointmentID string) ([]*domain.AppointmentHistory, error) {
	query := `
		SELECT id, appointment_id, from_status, to_status, changed_by, reason, created_at
		FROM appointment_history
		WHERE appointment_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, appointmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.AppointmentHistory
	for rows.Next() {
		var h domain.AppointmentHistory
		var fromStr, toStr *string
		err := rows.Scan(&h.ID, &h.AppointmentID, &fromStr, &toStr, &h.ChangedBy, &h.Reason, &h.CreatedAt)
		if err != nil {
			return nil, err
		}
		if fromStr != nil {
			s := domain.AppointmentStatus(*fromStr)
			h.FromStatus = &s
		}
		if toStr != nil {
			h.ToStatus = domain.AppointmentStatus(*toStr)
		}
		list = append(list, &h)
	}
	return list, nil
}

// GetBookedRanges retrieves existing confirmed, checked-in, or in-progress bookings within time window.
func (r *postgresAppointmentRepository) GetBookedRanges(ctx context.Context, clinicID string, doctorID, roomID *string, start, end time.Time) ([]TimeRange, error) {
	query := `
		SELECT lower(slot_range) AS start_time, upper(slot_range) AS end_time, doctor_id, room_id
		FROM appointments
		WHERE clinic_id = $1
		  AND status IN ('CONFIRMED', 'CHECKED_IN', 'IN_PROGRESS')
		  AND slot_range && tstzrange($2, $3, '[)')`

	args := []interface{}{clinicID, start, end}
	idx := 4

	if doctorID != nil && *doctorID != "" {
		query += ` AND doctor_id = $` + string(rune('0'+idx))
		args = append(args, *doctorID)
		idx++
	}
	if roomID != nil && *roomID != "" {
		query += ` AND room_id = $` + string(rune('0'+idx))
		args = append(args, *roomID)
		idx++
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []TimeRange
	for rows.Next() {
		var tr TimeRange
		if err := rows.Scan(&tr.StartTime, &tr.EndTime, &tr.DoctorID, &tr.RoomID); err != nil {
			return nil, err
		}
		list = append(list, tr)
	}
	return list, nil
}

func (r *postgresAppointmentRepository) scanAppointment(row *sql.Row) (*domain.Appointment, error) {
	var a domain.Appointment
	var statusStr string
	err := row.Scan(
		&a.ID, &a.PatientID, &a.ClinicID, &a.DoctorID, &a.RoomID, &a.PackageID,
		&a.StartTime, &a.EndTime,
		&statusStr, &a.SessionNumber, &a.TotalSessions, &a.PackageBookingID,
		&a.IdempotencyKey, &a.Notes, &a.CancellationReason, &a.ApprovedBy, &a.ApprovedAt,
		&a.RejectionReason, &a.RescheduledToID, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAppointmentNotFound
		}
		return nil, err
	}
	a.Status = domain.AppointmentStatus(statusStr)
	return &a, nil
}

func (r *postgresAppointmentRepository) mapError(err error) error {
	if err == nil {
		return nil
	}
	// Check for PostgreSQL exclusion violation (SQLSTATE 23P01)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23P01" || strings.Contains(pgErr.ConstraintName, "no_overlapping") {
			return domain.ErrSlotDoubleBooked
		}
	}
	if strings.Contains(err.Error(), "exclusion_violation") || strings.Contains(err.Error(), "no_overlapping") {
		return domain.ErrSlotDoubleBooked
	}
	return err
}
