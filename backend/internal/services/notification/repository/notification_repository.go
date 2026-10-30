package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/domain"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/database"
)

// NotificationRepository defines database persistence operations for Notification Service.
type NotificationRepository interface {
	// Notifications
	CreateNotification(ctx context.Context, notif *domain.Notification, outbox *domain.OutboxEvent) error
	GetNotificationByID(ctx context.Context, id string) (*domain.Notification, error)
	ListNotificationsByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error)
	MarkNotificationRead(ctx context.Context, id, userID string) error
	MarkAllNotificationsRead(ctx context.Context, userID string) error

	// Preferences
	GetPreferences(ctx context.Context, userID string) (*domain.NotificationPreference, error)
	SavePreferences(ctx context.Context, pref *domain.NotificationPreference) error

	// Idempotency (Deduplication)
	IsEventProcessed(ctx context.Context, eventID, consumerGroup string) (bool, error)
	RecordConsumedEvent(ctx context.Context, eventID, consumerGroup, eventType string) error

	// Delivery Auditing
	RecordDeliveryAttempt(ctx context.Context, attempt *domain.DeliveryAttempt) error
	GetDeliveryAttempts(ctx context.Context, notifID string) ([]*domain.DeliveryAttempt, error)
	ListDeliveryAttempts(ctx context.Context, limit int) ([]*domain.DeliveryAttempt, error)

	// Scheduled Reminders
	CreateScheduledReminder(ctx context.Context, reminder *domain.ScheduledReminder) error
	GetDueReminders(ctx context.Context, now time.Time, limit int) ([]*domain.ScheduledReminder, error)
	UpdateReminderStatus(ctx context.Context, id string, status domain.ReminderStatus) error
	CancelRemindersForAppointment(ctx context.Context, appointmentID string) error
	UpdateRemindersTargetTime(ctx context.Context, appointmentID string, newTargetTime, newScheduledTime time.Time) error

	// Transactional Outbox
	GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
	MarkOutboxEventProcessed(ctx context.Context, id string) error
	MarkOutboxEventFailed(ctx context.Context, id string, errStr string) error
}

type postgresNotificationRepository struct {
	db *database.DB
}

// NewPostgresNotificationRepository initializes repository with a PostgreSQL DB connection.
func NewPostgresNotificationRepository(db *database.DB) NotificationRepository {
	return &postgresNotificationRepository{db: db}
}

func parseOrDeterministicUUID(val string) string {
	if _, err := uuid.Parse(val); err == nil {
		return val
	}
	if val == "" {
		return "00000000-0000-0000-0000-000000000000"
	}
	return uuid.NewMD5(uuid.NameSpaceDNS, []byte(val)).String()
}

// CreateNotification inserts a notification and optional outbox event atomically.
func (r *postgresNotificationRepository) CreateNotification(ctx context.Context, notif *domain.Notification, outbox *domain.OutboxEvent) error {
	if notif.ID == "" {
		notif.ID = uuid.New().String()
	}
	notif.UserID = parseOrDeterministicUUID(notif.UserID)
	if notif.ReferenceID != nil {
		if *notif.ReferenceID == "" {
			notif.ReferenceID = nil
		} else {
			cleaned := parseOrDeterministicUUID(*notif.ReferenceID)
			notif.ReferenceID = &cleaned
		}
	}
	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = time.Now().UTC()
	}
	if len(notif.ChannelDispatches) == 0 {
		notif.ChannelDispatches = []byte("{}")
	}

	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		query := `
			INSERT INTO notifications (
				id, user_id, title, message, notification_type, priority,
				reference_id, reference_type, is_read, channel_dispatches, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

		_, err := tx.ExecContext(ctx, query,
			notif.ID, notif.UserID, notif.Title, notif.Message, notif.NotificationType,
			string(notif.Priority), notif.ReferenceID, notif.ReferenceType, notif.IsRead,
			notif.ChannelDispatches, notif.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert notification: %w", err)
		}

		if outbox != nil {
			if outbox.ID == "" {
				outbox.ID = uuid.New().String()
			}
			outbox.AggregateID = parseOrDeterministicUUID(outbox.AggregateID)
			if outbox.CreatedAt.IsZero() {
				outbox.CreatedAt = time.Now().UTC()
			}
			outboxQuery := `
				INSERT INTO outbox_events (
					id, aggregate_type, aggregate_id, event_type, payload, trace_id, status, retry_count, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

			_, err = tx.ExecContext(ctx, outboxQuery,
				outbox.ID, outbox.AggregateType, outbox.AggregateID, outbox.EventType,
				outbox.Payload, outbox.TraceID, outbox.Status, outbox.RetryCount, outbox.CreatedAt,
			)
			if err != nil {
				return fmt.Errorf("failed to insert outbox event: %w", err)
			}
		}

		return nil
	})
}

// GetNotificationByID fetches a single notification record.
func (r *postgresNotificationRepository) GetNotificationByID(ctx context.Context, id string) (*domain.Notification, error) {
	cleanID := parseOrDeterministicUUID(id)
	query := `
		SELECT id, user_id, title, message, notification_type, priority,
		       reference_id, reference_type, is_read, read_at, channel_dispatches, created_at
		FROM notifications
		WHERE id = $1`

	var n domain.Notification
	var prioStr string
	err := r.db.QueryRowContext(ctx, query, cleanID).Scan(
		&n.ID, &n.UserID, &n.Title, &n.Message, &n.NotificationType, &prioStr,
		&n.ReferenceID, &n.ReferenceType, &n.IsRead, &n.ReadAt, &n.ChannelDispatches, &n.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotificationNotFound
		}
		return nil, err
	}
	n.Priority = domain.Priority(prioStr)
	return &n, nil
}

// ListNotificationsByUserID queries user notifications with unread count and pagination.
func (r *postgresNotificationRepository) ListNotificationsByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error) {
	if limit <= 0 {
		limit = 20
	}
	cleanUID := parseOrDeterministicUUID(userID)

	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`
	var unreadCount int
	if err := r.db.QueryRowContext(ctx, countQuery, cleanUID).Scan(&unreadCount); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, title, message, notification_type, priority,
		       reference_id, reference_type, is_read, read_at, channel_dispatches, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, cleanUID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*domain.Notification
	for rows.Next() {
		var n domain.Notification
		var prioStr string
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Title, &n.Message, &n.NotificationType, &prioStr,
			&n.ReferenceID, &n.ReferenceType, &n.IsRead, &n.ReadAt, &n.ChannelDispatches, &n.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		n.Priority = domain.Priority(prioStr)
		list = append(list, &n)
	}

	return list, unreadCount, nil
}

// MarkNotificationRead updates read status and timestamp.
func (r *postgresNotificationRepository) MarkNotificationRead(ctx context.Context, id, userID string) error {
	cleanID := parseOrDeterministicUUID(id)
	cleanUID := parseOrDeterministicUUID(userID)

	query := `
		UPDATE notifications
		SET is_read = TRUE, read_at = $1
		WHERE id = $2 AND user_id = $3`

	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, query, now, cleanID, cleanUID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}

// MarkAllNotificationsRead marks all user notifications as read.
func (r *postgresNotificationRepository) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	cleanUID := parseOrDeterministicUUID(userID)
	query := `
		UPDATE notifications
		SET is_read = TRUE, read_at = $1
		WHERE user_id = $2 AND is_read = FALSE`

	_, err := r.db.ExecContext(ctx, query, time.Now().UTC(), cleanUID)
	return err
}

// GetPreferences fetches delivery preferences for a user, defaulting to all enabled if not configured.
func (r *postgresNotificationRepository) GetPreferences(ctx context.Context, userID string) (*domain.NotificationPreference, error) {
	cleanUID := parseOrDeterministicUUID(userID)
	query := `
		SELECT user_id, email_enabled, sms_enabled, whatsapp_enabled, in_app_enabled, updated_at
		FROM notification_preferences
		WHERE user_id = $1`

	var p domain.NotificationPreference
	err := r.db.QueryRowContext(ctx, query, cleanUID).Scan(
		&p.UserID, &p.EmailEnabled, &p.SMSEnabled, &p.WhatsAppEnabled, &p.InAppEnabled, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &domain.NotificationPreference{
				UserID:          cleanUID,
				EmailEnabled:    true,
				SMSEnabled:      true,
				WhatsAppEnabled: true,
				InAppEnabled:    true,
				UpdatedAt:       time.Now().UTC(),
			}, nil
		}
		return nil, err
	}
	return &p, nil
}

// SavePreferences creates or updates user notification preferences.
func (r *postgresNotificationRepository) SavePreferences(ctx context.Context, pref *domain.NotificationPreference) error {
	pref.UserID = parseOrDeterministicUUID(pref.UserID)
	query := `
		INSERT INTO notification_preferences (user_id, email_enabled, sms_enabled, whatsapp_enabled, in_app_enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			email_enabled = EXCLUDED.email_enabled,
			sms_enabled = EXCLUDED.sms_enabled,
			whatsapp_enabled = EXCLUDED.whatsapp_enabled,
			in_app_enabled = EXCLUDED.in_app_enabled,
			updated_at = EXCLUDED.updated_at`

	now := time.Now().UTC()
	pref.UpdatedAt = now
	_, err := r.db.ExecContext(ctx, query,
		pref.UserID, pref.EmailEnabled, pref.SMSEnabled, pref.WhatsAppEnabled, pref.InAppEnabled, now,
	)
	return err
}

// IsEventProcessed checks if an event ID has already been consumed by this consumer group.
func (r *postgresNotificationRepository) IsEventProcessed(ctx context.Context, eventID, consumerGroup string) (bool, error) {
	parsedID := parseOrDeterministicUUID(eventID)
	query := `SELECT 1 FROM consumed_events WHERE event_id = $1 AND consumer_group = $2`
	var dummy int
	err := r.db.QueryRowContext(ctx, query, parsedID, consumerGroup).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// RecordConsumedEvent inserts an idempotency record for a processed Kafka message.
func (r *postgresNotificationRepository) RecordConsumedEvent(ctx context.Context, eventID, consumerGroup, eventType string) error {
	parsedID := parseOrDeterministicUUID(eventID)
	query := `
		INSERT INTO consumed_events (event_id, consumer_group, event_type, processed_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_id, consumer_group) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, parsedID, consumerGroup, eventType, time.Now().UTC())
	return err
}

// RecordDeliveryAttempt logs an outbound channel dispatch attempt.
func (r *postgresNotificationRepository) RecordDeliveryAttempt(ctx context.Context, attempt *domain.DeliveryAttempt) error {
	if attempt.ID == "" {
		attempt.ID = uuid.New().String()
	}
	attempt.NotificationID = parseOrDeterministicUUID(attempt.NotificationID)
	if attempt.CreatedAt.IsZero() {
		attempt.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO notification_delivery_attempts (
			id, notification_id, channel, recipient, status, provider,
			attempt_number, error_message, dispatched_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		attempt.ID, attempt.NotificationID, attempt.Channel, attempt.Recipient,
		string(attempt.Status), attempt.Provider, attempt.AttemptNumber,
		attempt.ErrorMessage, attempt.DispatchedAt, attempt.CreatedAt,
	)
	return err
}

// GetDeliveryAttempts fetches dispatch records for a notification.
func (r *postgresNotificationRepository) GetDeliveryAttempts(ctx context.Context, notifID string) ([]*domain.DeliveryAttempt, error) {
	cleanID := parseOrDeterministicUUID(notifID)
	query := `
		SELECT id, notification_id, channel, recipient, status, provider,
		       attempt_number, error_message, dispatched_at, created_at
		FROM notification_delivery_attempts
		WHERE notification_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, cleanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []*domain.DeliveryAttempt
	for rows.Next() {
		var a domain.DeliveryAttempt
		var statusStr string
		if err := rows.Scan(
			&a.ID, &a.NotificationID, &a.Channel, &a.Recipient, &statusStr, &a.Provider,
			&a.AttemptNumber, &a.ErrorMessage, &a.DispatchedAt, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		a.Status = domain.DeliveryStatus(statusStr)
		attempts = append(attempts, &a)
	}
	return attempts, nil
}

// ListDeliveryAttempts fetches recent audit attempts for system monitoring.
func (r *postgresNotificationRepository) ListDeliveryAttempts(ctx context.Context, limit int) ([]*domain.DeliveryAttempt, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, notification_id, channel, recipient, status, provider,
		       attempt_number, error_message, dispatched_at, created_at
		FROM notification_delivery_attempts
		ORDER BY created_at DESC
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []*domain.DeliveryAttempt
	for rows.Next() {
		var a domain.DeliveryAttempt
		var statusStr string
		if err := rows.Scan(
			&a.ID, &a.NotificationID, &a.Channel, &a.Recipient, &statusStr, &a.Provider,
			&a.AttemptNumber, &a.ErrorMessage, &a.DispatchedAt, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		a.Status = domain.DeliveryStatus(statusStr)
		attempts = append(attempts, &a)
	}
	return attempts, nil
}

// CreateScheduledReminder saves a prospective reminder.
func (r *postgresNotificationRepository) CreateScheduledReminder(ctx context.Context, reminder *domain.ScheduledReminder) error {
	if reminder.ID == "" {
		reminder.ID = uuid.New().String()
	}
	reminder.AppointmentID = parseOrDeterministicUUID(reminder.AppointmentID)
	reminder.PatientID = parseOrDeterministicUUID(reminder.PatientID)
	reminder.ClinicID = parseOrDeterministicUUID(reminder.ClinicID)
	now := time.Now().UTC()
	if reminder.CreatedAt.IsZero() {
		reminder.CreatedAt = now
	}
	reminder.UpdatedAt = now

	query := `
		INSERT INTO scheduled_reminders (
			id, appointment_id, patient_id, clinic_id, reminder_type,
			scheduled_time, target_event_time, status, idempotency_key, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (idempotency_key) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query,
		reminder.ID, reminder.AppointmentID, reminder.PatientID, reminder.ClinicID,
		reminder.ReminderType, reminder.ScheduledTime, reminder.TargetEventTime,
		string(reminder.Status), reminder.IdempotencyKey, reminder.CreatedAt, reminder.UpdatedAt,
	)
	return err
}

// GetDueReminders fetches reminders where scheduled_time <= now.
func (r *postgresNotificationRepository) GetDueReminders(ctx context.Context, now time.Time, limit int) ([]*domain.ScheduledReminder, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, appointment_id, patient_id, clinic_id, reminder_type,
		       scheduled_time, target_event_time, status, idempotency_key, created_at, updated_at
		FROM scheduled_reminders
		WHERE status = 'SCHEDULED' AND scheduled_time <= $1
		ORDER BY scheduled_time ASC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ScheduledReminder
	for rows.Next() {
		var s domain.ScheduledReminder
		var statusStr string
		if err := rows.Scan(
			&s.ID, &s.AppointmentID, &s.PatientID, &s.ClinicID, &s.ReminderType,
			&s.ScheduledTime, &s.TargetEventTime, &statusStr, &s.IdempotencyKey,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		s.Status = domain.ReminderStatus(statusStr)
		list = append(list, &s)
	}
	return list, nil
}

// UpdateReminderStatus updates status (e.g., DISPATCHED, CANCELLED).
func (r *postgresNotificationRepository) UpdateReminderStatus(ctx context.Context, id string, status domain.ReminderStatus) error {
	cleanID := parseOrDeterministicUUID(id)
	query := `UPDATE scheduled_reminders SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, string(status), time.Now().UTC(), cleanID)
	return err
}

// CancelRemindersForAppointment cancels all pending reminders when an appointment is cancelled.
func (r *postgresNotificationRepository) CancelRemindersForAppointment(ctx context.Context, appointmentID string) error {
	cleanApptID := parseOrDeterministicUUID(appointmentID)
	query := `
		UPDATE scheduled_reminders
		SET status = 'CANCELLED', updated_at = $1
		WHERE appointment_id = $2 AND status = 'SCHEDULED'`

	_, err := r.db.ExecContext(ctx, query, time.Now().UTC(), cleanApptID)
	return err
}

// UpdateRemindersTargetTime updates reminder times when an appointment is rescheduled.
func (r *postgresNotificationRepository) UpdateRemindersTargetTime(ctx context.Context, appointmentID string, newTargetTime, newScheduledTime time.Time) error {
	cleanApptID := parseOrDeterministicUUID(appointmentID)
	query := `
		UPDATE scheduled_reminders
		SET target_event_time = $1, scheduled_time = $2, updated_at = $3
		WHERE appointment_id = $4 AND status = 'SCHEDULED'`

	_, err := r.db.ExecContext(ctx, query, newTargetTime, newScheduledTime, time.Now().UTC(), cleanApptID)
	return err
}

// GetPendingOutboxEvents fetches pending outbox events for Kafka dispatch.
func (r *postgresNotificationRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
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
		var e domain.OutboxEvent
		if err := rows.Scan(
			&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload,
			&e.TraceID, &e.Status, &e.RetryCount, &e.CreatedAt, &e.ProcessedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	return events, nil
}

// MarkOutboxEventProcessed marks outbox entry as sent.
func (r *postgresNotificationRepository) MarkOutboxEventProcessed(ctx context.Context, id string) error {
	now := time.Now().UTC()
	query := `UPDATE outbox_events SET status = 'PROCESSED', processed_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, now, id)
	return err
}

// MarkOutboxEventFailed updates retry counter and status.
func (r *postgresNotificationRepository) MarkOutboxEventFailed(ctx context.Context, id string, errStr string) error {
	query := `
		UPDATE outbox_events
		SET retry_count = retry_count + 1,
		    status = CASE WHEN retry_count >= 5 THEN 'FAILED' ELSE 'PENDING' END
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
