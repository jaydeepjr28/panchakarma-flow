package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/provider"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/kafka"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	appointmentv1 "github.com/jaydeepjr28/panchakarma-flow/proto/gen/go/appointment/v1"
	"google.golang.org/grpc"
)

// AppointmentStatusChecker allows verifying appointment state before delivering timed reminders.
type AppointmentStatusChecker interface {
	IsAppointmentActive(ctx context.Context, appointmentID string) (bool, error)
}

// gRPCAppointmentStatusChecker verifies appointment state over gRPC with appointment-service.
type gRPCAppointmentStatusChecker struct {
	client appointmentv1.AppointmentServiceClient
}

// NewGRPCAppointmentStatusChecker creates a new gRPC-backed appointment status checker.
func NewGRPCAppointmentStatusChecker(cc grpc.ClientConnInterface) AppointmentStatusChecker {
	if cc == nil {
		return nil
	}
	return &gRPCAppointmentStatusChecker{client: appointmentv1.NewAppointmentServiceClient(cc)}
}

func (c *gRPCAppointmentStatusChecker) IsAppointmentActive(ctx context.Context, appointmentID string) (bool, error) {
	if c == nil || c.client == nil {
		return true, nil
	}
	callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := c.client.GetAppointmentSummary(callCtx, &appointmentv1.GetAppointmentSummaryRequest{
		AppointmentId: appointmentID,
	})
	if err != nil {
		logger.WithContext(ctx).Warn("Failed to check appointment status via gRPC, defaulting to active",
			slog.String("appointment_id", appointmentID),
			slog.String("error", err.Error()),
		)
		return true, nil
	}

	if resp.Status == "CANCELLED" || resp.Status == "RESCHEDULED" {
		return false, nil
	}
	return true, nil
}

// NotificationService provides the notification management operations.
type NotificationService interface {
	ProcessKafkaEvent(ctx context.Context, envelope *kafka.EventEnvelope) error
	RouteToDLQ(ctx context.Context, env *kafka.EventEnvelope, reason error) error
	SendDirectNotification(ctx context.Context, userID, title, message, notifType string, prio domain.Priority, refID, refType *string) (*domain.Notification, error)
	ListUserNotifications(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error)
	MarkNotificationRead(ctx context.Context, id, userID string) error
	MarkAllNotificationsRead(ctx context.Context, userID string) error
	GetPreferences(ctx context.Context, userID string) (*domain.NotificationPreference, error)
	UpdatePreferences(ctx context.Context, pref *domain.NotificationPreference) error
	ProcessDueReminders(ctx context.Context) (int, error)
	ListDeliveryAttempts(ctx context.Context, limit int) ([]*domain.DeliveryAttempt, error)
}

type notificationService struct {
	repo          repository.NotificationRepository
	provider      provider.DeliveryProvider
	apptChecker   AppointmentStatusChecker
	consumerGroup string
	dlqProducer   *kafka.Producer
}

// NewNotificationService initializes the core Notification Service.
func NewNotificationService(
	repo repository.NotificationRepository,
	prov provider.DeliveryProvider,
	checker AppointmentStatusChecker,
	consumerGroup string,
	dlqProducer *kafka.Producer,
) NotificationService {
	if consumerGroup == "" {
		consumerGroup = "notification-service-group"
	}
	return &notificationService{
		repo:          repo,
		provider:      prov,
		apptChecker:   checker,
		consumerGroup: consumerGroup,
		dlqProducer:   dlqProducer,
	}
}

// ProcessKafkaEvent handles incoming domain events with strict idempotency and DLQ handling.
func (s *notificationService) ProcessKafkaEvent(ctx context.Context, envelope *kafka.EventEnvelope) error {
	if envelope == nil || envelope.EventID == "" {
		return fmt.Errorf("invalid nil or empty event envelope")
	}

	// 1. Idempotency Check (Deduplication)
	alreadyProcessed, err := s.repo.IsEventProcessed(ctx, envelope.EventID, s.consumerGroup)
	if err != nil {
		return fmt.Errorf("failed checking event deduplication: %w", err)
	}
	if alreadyProcessed {
		logger.WithContext(ctx).Info("Duplicate Kafka event ignored by consumer",
			slog.String("event_id", envelope.EventID),
			slog.String("event_type", envelope.EventType),
			slog.String("consumer_group", s.consumerGroup),
		)
		return nil
	}

	// 2. Dispatch based on Event Type
	var handleErr error
	switch envelope.EventType {
	case "panchakarma.appointment.confirmed.v1":
		handleErr = s.handleAppointmentConfirmed(ctx, envelope)
	case "panchakarma.appointment.cancelled.v1":
		handleErr = s.handleAppointmentCancelled(ctx, envelope)
	case "panchakarma.appointment.rescheduled.v1":
		handleErr = s.handleAppointmentRescheduled(ctx, envelope)
	case "panchakarma.session.started.v1":
		handleErr = s.handleSessionStarted(ctx, envelope)
	case "panchakarma.session.completed.v1":
		handleErr = s.handleSessionCompleted(ctx, envelope)
	case "panchakarma.complication.flagged.v1":
		handleErr = s.handleComplicationFlagged(ctx, envelope)
	case "panchakarma.complication.resolved.v1":
		handleErr = s.handleComplicationResolved(ctx, envelope)
	case "panchakarma.feedback.requested.v1":
		handleErr = s.handleFeedbackRequested(ctx, envelope)
	case "panchakarma.feedback.submitted.v1":
		handleErr = s.handleFeedbackSubmitted(ctx, envelope)
	case "panchakarma.reminder.due.v1":
		handleErr = s.handleReminderDue(ctx, envelope)
	case "panchakarma.followup.reminder.v1":
		handleErr = s.handleFollowUpReminder(ctx, envelope)
	default:
		logger.WithContext(ctx).Debug("Event type not subscribed by Notification Service, skipping",
			slog.String("event_type", envelope.EventType),
		)
	}

	if handleErr != nil {
		logger.WithContext(ctx).Error("Error executing notification event handler",
			slog.String("event_id", envelope.EventID),
			slog.String("event_type", envelope.EventType),
			slog.String("error", handleErr.Error()),
		)
		return handleErr
	}

	// 3. Record Deduplication Commit
	if err := s.repo.RecordConsumedEvent(ctx, envelope.EventID, s.consumerGroup, envelope.EventType); err != nil {
		logger.WithContext(ctx).Error("Failed to record consumed event idempotency", "error", err)
		return err
	}

	return nil
}

// -----------------------------------------------------------------------------
// EVENT HANDLERS
// -----------------------------------------------------------------------------

func (s *notificationService) handleAppointmentConfirmed(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload domain.AppointmentConfirmedPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("malformed appointment confirmed payload: %w", err)
	}

	// 1. Create Patient In-App & Multi-Channel Notification
	title := "Appointment Confirmed"
	msg := fmt.Sprintf("Your Panchakarma therapy session (%s) has been successfully scheduled for %s.",
		payload.PackageName, payload.ScheduledStart)

	notif, err := s.createAndDispatch(ctx, payload.PatientID, title, msg, "APPOINTMENT_CONFIRMED",
		domain.PriorityNormal, &payload.AppointmentID, "APPOINTMENT")
	if err != nil {
		// Log but do not fail the event (avoid failing caller or blocking partition)
		logger.WithContext(ctx).Error("Failed to dispatch appointment confirmed notification", "error", err)
	}

	// 2. Schedule Future Preparation Reminder
	// Parse scheduled start time
	startTime, err := time.Parse(time.RFC3339, payload.ScheduledStart)
	if err == nil {
		// Calculate 24h prior reminder (or immediately if within 24h)
		reminderTime := startTime.Add(-24 * time.Hour)
		if reminderTime.Before(time.Now().UTC()) {
			reminderTime = time.Now().UTC().Add(10 * time.Minute)
		}

		reminder := &domain.ScheduledReminder{
			ID:              uuid.New().String(),
			AppointmentID:   payload.AppointmentID,
			PatientID:       payload.PatientID,
			ClinicID:        payload.ClinicID,
			ReminderType:    "24h_pre_therapy_fasting",
			ScheduledTime:   reminderTime,
			TargetEventTime: startTime,
			Status:          domain.ReminderStatusScheduled,
			IdempotencyKey:  fmt.Sprintf("rem-24h-%s", payload.AppointmentID),
		}

		if err := s.repo.CreateScheduledReminder(ctx, reminder); err != nil {
			logger.WithContext(ctx).Warn("Failed to register scheduled reminder", "error", err)
		}
	}

	_ = notif
	return nil
}

func (s *notificationService) handleAppointmentCancelled(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload domain.AppointmentCancelledPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("malformed appointment cancelled payload: %w", err)
	}

	// 1. Cancel all future scheduled reminders for this appointment
	if err := s.repo.CancelRemindersForAppointment(ctx, payload.AppointmentID); err != nil {
		logger.WithContext(ctx).Warn("Failed to cancel scheduled reminders for cancelled appointment", "error", err)
	}

	// 2. Notify Patient
	title := "Appointment Cancelled"
	reason := payload.Reason
	if reason == "" {
		reason = "Requested by patient or clinic"
	}
	msg := fmt.Sprintf("Your appointment has been cancelled. Reason: %s.", reason)

	_, err := s.createAndDispatch(ctx, payload.PatientID, title, msg, "APPOINTMENT_CANCELLED",
		domain.PriorityNormal, &payload.AppointmentID, "APPOINTMENT")
	return err
}

func (s *notificationService) handleAppointmentRescheduled(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload domain.AppointmentRescheduledPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("malformed appointment rescheduled payload: %w", err)
	}

	// 1. Update Scheduled Reminders target time
	newStartTime, err := time.Parse(time.RFC3339, payload.NewStartTime)
	if err == nil {
		newReminderTime := newStartTime.Add(-24 * time.Hour)
		if newReminderTime.Before(time.Now().UTC()) {
			newReminderTime = time.Now().UTC().Add(10 * time.Minute)
		}
		_ = s.repo.UpdateRemindersTargetTime(ctx, payload.AppointmentID, newStartTime, newReminderTime)
	}

	// 2. Notify Patient
	title := "Appointment Rescheduled"
	msg := fmt.Sprintf("Your appointment has been rescheduled to %s. Reason: %s.",
		payload.NewStartTime, payload.Reason)

	_, err = s.createAndDispatch(ctx, payload.PatientID, title, msg, "APPOINTMENT_RESCHEDULED",
		domain.PriorityNormal, &payload.AppointmentID, "APPOINTMENT")
	return err
}

func (s *notificationService) handleSessionStarted(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload domain.SessionStartedPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("malformed session started payload: %w", err)
	}

	title := "Therapy Session In Progress"
	msg := "Your Panchakarma procedural session has begun. Please follow your therapist's guidance."

	_, err := s.createAndDispatch(ctx, payload.PatientID, title, msg, "SESSION_STARTED",
		domain.PriorityNormal, &payload.SessionID, "SESSION")
	return err
}

func (s *notificationService) handleSessionCompleted(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload domain.SessionCompletedPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("malformed session completed payload: %w", err)
	}

	title := "Therapy Completed — Post-Care Guidance"
	msg := "Your therapy session is complete. Please consult the mobile portal for your personalized post-therapy diet (Pathya) instructions."
	if payload.PostCareInstructions != "" {
		msg = fmt.Sprintf("Your therapy session is complete. Post-care guidance: %s", payload.PostCareInstructions)
	}

	_, err := s.createAndDispatch(ctx, payload.PatientID, title, msg, "SESSION_COMPLETED",
		domain.PriorityNormal, &payload.SessionID, "SESSION")
	return err
}

func (s *notificationService) handleComplicationFlagged(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload domain.ComplicationFlaggedPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("malformed complication flagged payload: %w", err)
	}

	// High priority medical alert for attending physician and emergency staff
	title := "EMERGENCY: Adverse Reaction / Complication Reported"
	msg := fmt.Sprintf("Urgent clinical attention required for Session %s. Severity: %s. Symptoms: %s.",
		payload.SessionID, payload.Severity, payload.Symptoms)

	// Send to doctor
	targetUser := payload.AttendingDoctorID
	if targetUser == "" {
		targetUser = "staff_on_duty"
	}

	_, err := s.createAndDispatch(ctx, targetUser, title, msg, "COMPLICATION_EMERGENCY",
		domain.PriorityEmergency, &payload.ComplicationID, "COMPLICATION")
	return err
}

func (s *notificationService) handleComplicationResolved(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload domain.ComplicationResolvedPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("malformed complication resolved payload: %w", err)
	}

	title := "Clinical Complication Resolved"
	msg := fmt.Sprintf("Complication for Session %s has been reviewed and cleared by the attending physician.",
		payload.SessionID)

	_, err := s.createAndDispatch(ctx, payload.DoctorID, title, msg, "COMPLICATION_RESOLVED",
		domain.PriorityNormal, &payload.ComplicationID, "COMPLICATION")
	return err
}

func (s *notificationService) handleFeedbackRequested(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload domain.FeedbackRequestedPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("malformed feedback requested payload: %w", err)
	}

	title := "Therapy Feedback Request"
	msg := "How was your Panchakarma treatment? Please take a moment to rate your session experience and recovery progress."

	_, err := s.createAndDispatch(ctx, payload.PatientID, title, msg, "FEEDBACK_REQUESTED",
		domain.PriorityLow, &payload.AppointmentID, "APPOINTMENT")
	return err
}

func (s *notificationService) handleFeedbackSubmitted(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload domain.FeedbackSubmittedPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("malformed feedback submitted payload: %w", err)
	}

	title := "Feedback Received"
	msg := fmt.Sprintf("Thank you for sharing your Panchakarma experience (Rating: %d/5). Your feedback helps us maintain authentic Ayurvedic care.", payload.Rating)

	_, err := s.createAndDispatch(ctx, payload.PatientID, title, msg, "FEEDBACK_SUBMITTED",
		domain.PriorityLow, &payload.AppointmentID, "APPOINTMENT")
	return err
}

func (s *notificationService) handleReminderDue(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload domain.ReminderDuePayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("malformed reminder due payload: %w", err)
	}

	title := fmt.Sprintf("Therapy Preparation Reminder (%s)", payload.ReminderType)
	msg := "Gentle reminder: You have a scheduled Panchakarma session coming up. Please adhere to your dietary and rest guidelines."

	_, err := s.createAndDispatch(ctx, payload.PatientID, title, msg, "REMINDER_DUE",
		domain.PriorityNormal, &payload.AppointmentID, "APPOINTMENT")
	return err
}

func (s *notificationService) handleFollowUpReminder(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload domain.FollowUpReminderPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("malformed follow up reminder payload: %w", err)
	}

	title := "Post-Therapy Follow-Up Care Due"
	msg := fmt.Sprintf("Your physician recommended follow-up therapy on %s. Please book your next session to continue your Ayurvedic healing.", payload.RecommendedDate)
	if payload.Instructions != "" {
		msg += " Note: " + payload.Instructions
	}

	_, err := s.createAndDispatch(ctx, payload.PatientID, title, msg, "FOLLOW_UP_REMINDER",
		domain.PriorityNormal, &payload.SessionID, "SESSION")
	return err
}

// RouteToDLQ publishes failed event envelope to the appropriate Dead Letter Queue topic.
func (s *notificationService) RouteToDLQ(ctx context.Context, env *kafka.EventEnvelope, reason error) error {
	if s.dlqProducer == nil {
		return nil
	}
	dlqTopic := s.resolveDLQTopic(env.EventType)
	logger.WithContext(ctx).Warn("Routing failed event to Dead Letter Queue (DLQ)",
		slog.String("event_id", env.EventID),
		slog.String("event_type", env.EventType),
		slog.String("dlq_topic", dlqTopic),
		slog.String("reason", reason.Error()),
	)
	return s.dlqProducer.PublishEvent(ctx, dlqTopic, env.AggregateID, env)
}

func (s *notificationService) resolveDLQTopic(eventType string) string {
	switch {
	case strings.HasPrefix(eventType, "panchakarma.appointment."):
		return "panchakarma.appointments.events.DLQ"
	case strings.HasPrefix(eventType, "panchakarma.session."):
		return "panchakarma.sessions.events.DLQ"
	case strings.HasPrefix(eventType, "panchakarma.complication."):
		return "panchakarma.complications.emergency.DLQ"
	case strings.HasPrefix(eventType, "panchakarma.feedback."):
		return "panchakarma.feedback.events.DLQ"
	default:
		return "panchakarma.notifications.events.DLQ"
	}
}

// -----------------------------------------------------------------------------
// HELPER: CREATE & DISPATCH WITH CHANNELS AUDITING
// -----------------------------------------------------------------------------

func (s *notificationService) createAndDispatch(
	ctx context.Context,
	userID, title, message, notifType string,
	priority domain.Priority,
	refID *string,
	refType string,
) (*domain.Notification, error) {
	if userID == "" {
		userID = "00000000-0000-0000-0000-000000000000"
	}
	if refID != nil && *refID == "" {
		refID = nil
	}
	now := time.Now().UTC()
	notif := &domain.Notification{
		ID:                uuid.New().String(),
		UserID:            userID,
		Title:             title,
		Message:           message,
		NotificationType:  notifType,
		Priority:          priority,
		ReferenceID:       refID,
		ReferenceType:     &refType,
		IsRead:            false,
		ChannelDispatches: []byte("{}"),
		CreatedAt:         now,
	}

	// 1. Persist notification in database
	if err := s.repo.CreateNotification(ctx, notif, nil); err != nil {
		return nil, fmt.Errorf("failed to save notification: %w", err)
	}

	// 2. Fetch User Channel Preferences
	pref, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		logger.WithContext(ctx).Warn("Failed to get preferences, applying defaults", "user_id", userID)
		pref = &domain.NotificationPreference{EmailEnabled: true, SMSEnabled: true, InAppEnabled: true}
	}

	// 3. Multi-Channel Dispatches (In-App is always recorded)
	dispatches := map[string]string{"in_app": "DELIVERED"}

	if s.provider != nil {
		// Mock recipient resolution based on userID
		userPrefix := userID
		if len(userPrefix) > 8 {
			userPrefix = userPrefix[:8]
		}
		userEmail := fmt.Sprintf("%s@panchakarma-patient.org", userPrefix)
		userPhone := "+919876543210"

		// Send Email if enabled
		if pref.EmailEnabled {
			attempt, err := s.provider.SendEmail(ctx, notif.ID, userEmail, title, message)
			if attempt != nil {
				_ = s.repo.RecordDeliveryAttempt(ctx, attempt)
				if err == nil {
					dispatches["email"] = "DELIVERED"
				} else {
					dispatches["email"] = "FAILED"
				}
			}
		}

		// Send SMS if enabled
		if pref.SMSEnabled {
			attempt, err := s.provider.SendSMS(ctx, notif.ID, userPhone, message)
			if attempt != nil {
				_ = s.repo.RecordDeliveryAttempt(ctx, attempt)
				if err == nil {
					dispatches["sms"] = "DELIVERED"
				} else {
					dispatches["sms"] = "FAILED"
				}
			}
		}

		// Emergency priority triggers push notification regardless of standard preference
		if priority == domain.PriorityEmergency {
			attempt, err := s.provider.SendPush(ctx, notif.ID, "push-device-token-emerg", title, message, map[string]string{
				"notification_id": notif.ID,
				"priority":        string(priority),
			})
			if attempt != nil {
				_ = s.repo.RecordDeliveryAttempt(ctx, attempt)
				if err == nil {
					dispatches["push"] = "DELIVERED"
				} else {
					dispatches["push"] = "FAILED"
				}
			}
		}
	}

	bytes, _ := json.Marshal(dispatches)
	notif.ChannelDispatches = bytes
	return notif, nil
}

// -----------------------------------------------------------------------------
// SCHEDULED REMINDERS PROCESSING
// -----------------------------------------------------------------------------

// ProcessDueReminders executes due reminder checks and enforces appointment state rechecks before delivery.
func (s *notificationService) ProcessDueReminders(ctx context.Context) (int, error) {
	if s.repo == nil {
		return 0, nil
	}
	now := time.Now().UTC()
	reminders, err := s.repo.GetDueReminders(ctx, now, 50)
	if err != nil {
		return 0, err
	}

	dispatchedCount := 0
	for _, rem := range reminders {
		// Recheck appointment status before sending time-sensitive reminder
		if s.apptChecker != nil {
			isActive, err := s.apptChecker.IsAppointmentActive(ctx, rem.AppointmentID)
			if err == nil && !isActive {
				// Appointment was cancelled or rescheduled: abort reminder
				logger.WithContext(ctx).Info("Scheduled reminder aborted: appointment is no longer active",
					slog.String("reminder_id", rem.ID),
					slog.String("appointment_id", rem.AppointmentID),
				)
				_ = s.repo.UpdateReminderStatus(ctx, rem.ID, domain.ReminderStatusCancelled)
				continue
			}
		}

		// Dispatch reminder notification
		title := "Upcoming Therapy Preparation Reminder"
		msg := "Reminder: Your scheduled Panchakarma session is coming up. Please maintain light meals and adequate rest."

		_, err := s.createAndDispatch(ctx, rem.PatientID, title, msg, "REMINDER_DUE",
			domain.PriorityNormal, &rem.AppointmentID, "APPOINTMENT")
		if err != nil {
			logger.WithContext(ctx).Warn("Failed to dispatch due reminder", "error", err)
			continue
		}

		_ = s.repo.UpdateReminderStatus(ctx, rem.ID, domain.ReminderStatusDispatched)
		dispatchedCount++
	}

	return dispatchedCount, nil
}

// -----------------------------------------------------------------------------
// USER API IMPLEMENTATIONS
// -----------------------------------------------------------------------------

func (s *notificationService) SendDirectNotification(ctx context.Context, userID, title, message, notifType string, prio domain.Priority, refID, refType *string) (*domain.Notification, error) {
	rType := ""
	if refType != nil {
		rType = *refType
	}
	return s.createAndDispatch(ctx, userID, title, message, notifType, prio, refID, rType)
}

func (s *notificationService) ListUserNotifications(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error) {
	return s.repo.ListNotificationsByUserID(ctx, userID, limit, offset)
}

func (s *notificationService) MarkNotificationRead(ctx context.Context, id, userID string) error {
	return s.repo.MarkNotificationRead(ctx, id, userID)
}

func (s *notificationService) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllNotificationsRead(ctx, userID)
}

func (s *notificationService) GetPreferences(ctx context.Context, userID string) (*domain.NotificationPreference, error) {
	return s.repo.GetPreferences(ctx, userID)
}

func (s *notificationService) UpdatePreferences(ctx context.Context, pref *domain.NotificationPreference) error {
	if pref == nil || pref.UserID == "" {
		return domain.ErrInvalidPreference
	}
	return s.repo.SavePreferences(ctx, pref)
}

func (s *notificationService) ListDeliveryAttempts(ctx context.Context, limit int) ([]*domain.DeliveryAttempt, error) {
	return s.repo.ListDeliveryAttempts(ctx, limit)
}
