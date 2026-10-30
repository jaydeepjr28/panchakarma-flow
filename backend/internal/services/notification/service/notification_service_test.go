package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/provider"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/service"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/kafka"
)

// In-Memory Mock Repository for unit testing
type mockNotificationRepository struct {
	mu             sync.Mutex
	notifications  map[string]*domain.Notification
	preferences    map[string]*domain.NotificationPreference
	consumedEvents map[string]bool // key: eventID:consumerGroup
	attempts       []*domain.DeliveryAttempt
	reminders      map[string]*domain.ScheduledReminder
	outbox         map[string]*domain.OutboxEvent
}

func newMockRepository() *mockNotificationRepository {
	return &mockNotificationRepository{
		notifications:  make(map[string]*domain.Notification),
		preferences:    make(map[string]*domain.NotificationPreference),
		consumedEvents: make(map[string]bool),
		attempts:       make([]*domain.DeliveryAttempt, 0),
		reminders:      make(map[string]*domain.ScheduledReminder),
		outbox:         make(map[string]*domain.OutboxEvent),
	}
}

func (m *mockNotificationRepository) CreateNotification(ctx context.Context, notif *domain.Notification, outbox *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if notif.ID == "" {
		notif.ID = uuid.New().String()
	}
	m.notifications[notif.ID] = notif
	if outbox != nil {
		if outbox.ID == "" {
			outbox.ID = uuid.New().String()
		}
		m.outbox[outbox.ID] = outbox
	}
	return nil
}

func (m *mockNotificationRepository) GetNotificationByID(ctx context.Context, id string) (*domain.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, exists := m.notifications[id]
	if !exists {
		return nil, domain.ErrNotificationNotFound
	}
	return n, nil
}

func (m *mockNotificationRepository) ListNotificationsByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []*domain.Notification
	unread := 0
	for _, n := range m.notifications {
		if n.UserID == userID {
			list = append(list, n)
			if !n.IsRead {
				unread++
			}
		}
	}
	return list, unread, nil
}

func (m *mockNotificationRepository) MarkNotificationRead(ctx context.Context, id, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, exists := m.notifications[id]
	if !exists || n.UserID != userID {
		return domain.ErrNotificationNotFound
	}
	n.IsRead = true
	now := time.Now().UTC()
	n.ReadAt = &now
	return nil
}

func (m *mockNotificationRepository) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for _, n := range m.notifications {
		if n.UserID == userID {
			n.IsRead = true
			n.ReadAt = &now
		}
	}
	return nil
}

func (m *mockNotificationRepository) GetPreferences(ctx context.Context, userID string) (*domain.NotificationPreference, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, exists := m.preferences[userID]
	if !exists {
		return &domain.NotificationPreference{
			UserID:          userID,
			EmailEnabled:    true,
			SMSEnabled:      true,
			WhatsAppEnabled: true,
			InAppEnabled:    true,
		}, nil
	}
	return p, nil
}

func (m *mockNotificationRepository) SavePreferences(ctx context.Context, pref *domain.NotificationPreference) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.preferences[pref.UserID] = pref
	return nil
}

func (m *mockNotificationRepository) IsEventProcessed(ctx context.Context, eventID, consumerGroup string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", eventID, consumerGroup)
	return m.consumedEvents[key], nil
}

func (m *mockNotificationRepository) RecordConsumedEvent(ctx context.Context, eventID, consumerGroup, eventType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", eventID, consumerGroup)
	m.consumedEvents[key] = true
	return nil
}

func (m *mockNotificationRepository) RecordDeliveryAttempt(ctx context.Context, attempt *domain.DeliveryAttempt) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempts = append(m.attempts, attempt)
	return nil
}

func (m *mockNotificationRepository) GetDeliveryAttempts(ctx context.Context, notifID string) ([]*domain.DeliveryAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []*domain.DeliveryAttempt
	for _, a := range m.attempts {
		if a.NotificationID == notifID {
			res = append(res, a)
		}
	}
	return res, nil
}

func (m *mockNotificationRepository) ListDeliveryAttempts(ctx context.Context, limit int) ([]*domain.DeliveryAttempt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.attempts, nil
}

func (m *mockNotificationRepository) CreateScheduledReminder(ctx context.Context, reminder *domain.ScheduledReminder) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reminders[reminder.IdempotencyKey] = reminder
	return nil
}

func (m *mockNotificationRepository) GetDueReminders(ctx context.Context, now time.Time, limit int) ([]*domain.ScheduledReminder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var due []*domain.ScheduledReminder
	for _, r := range m.reminders {
		if r.Status == domain.ReminderStatusScheduled && (r.ScheduledTime.Before(now) || r.ScheduledTime.Equal(now)) {
			due = append(due, r)
		}
	}
	return due, nil
}

func (m *mockNotificationRepository) UpdateReminderStatus(ctx context.Context, id string, status domain.ReminderStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.reminders {
		if r.ID == id {
			r.Status = status
			return nil
		}
	}
	return nil
}

func (m *mockNotificationRepository) CancelRemindersForAppointment(ctx context.Context, appointmentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.reminders {
		if r.AppointmentID == appointmentID && r.Status == domain.ReminderStatusScheduled {
			r.Status = domain.ReminderStatusCancelled
		}
	}
	return nil
}

func (m *mockNotificationRepository) UpdateRemindersTargetTime(ctx context.Context, appointmentID string, newTargetTime, newScheduledTime time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.reminders {
		if r.AppointmentID == appointmentID && r.Status == domain.ReminderStatusScheduled {
			r.TargetEventTime = newTargetTime
			r.ScheduledTime = newScheduledTime
		}
	}
	return nil
}

func (m *mockNotificationRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []*domain.OutboxEvent
	for _, o := range m.outbox {
		if o.Status == "PENDING" {
			res = append(res, o)
		}
	}
	return res, nil
}

func (m *mockNotificationRepository) MarkOutboxEventProcessed(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if o, ok := m.outbox[id]; ok {
		o.Status = "PROCESSED"
	}
	return nil
}

func (m *mockNotificationRepository) MarkOutboxEventFailed(ctx context.Context, id string, errStr string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if o, ok := m.outbox[id]; ok {
		o.RetryCount++
		if o.RetryCount >= 3 {
			o.Status = "FAILED"
		}
	}
	return nil
}

// Mock Kafka Publisher
type mockKafkaPublisher struct {
	mu        sync.Mutex
	published []*kafka.EventEnvelope
	shouldErr bool
}

func (m *mockKafkaPublisher) PublishEvent(ctx context.Context, topic, partitionKey string, event *kafka.EventEnvelope) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.shouldErr {
		return errors.New("simulated Kafka broker network partition")
	}
	m.published = append(m.published, event)
	return nil
}

// Mock Appointment Status Checker
type mockApptChecker struct {
	active bool
}

func (m *mockApptChecker) IsAppointmentActive(ctx context.Context, appointmentID string) (bool, error) {
	return m.active, nil
}

// -----------------------------------------------------------------------------
// TESTS
// -----------------------------------------------------------------------------

func TestIdempotentEventProcessing(t *testing.T) {
	repo := newMockRepository()
	devAdapter := provider.NewDevelopmentDeliveryAdapter()
	svc := service.NewNotificationService(repo, devAdapter, nil, "test-group", nil)

	payloadBytes, _ := json.Marshal(domain.AppointmentConfirmedPayload{
		AppointmentID:  "apt-101",
		PatientID:      "pat-201",
		ClinicID:       "cln-301",
		DoctorID:       "doc-401",
		PackageName:    "Abhyanga Detox",
		ScheduledStart: time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339),
	})

	envelope := &kafka.EventEnvelope{
		EventID:       "evt-unique-12345",
		EventType:     "panchakarma.appointment.confirmed.v1",
		AggregateType: "APPOINTMENT",
		AggregateID:   "apt-101",
		TraceID:       "trace-001",
		Payload:       payloadBytes,
	}

	ctx := context.Background()

	// First execution -> Should process successfully
	if err := svc.ProcessKafkaEvent(ctx, envelope); err != nil {
		t.Fatalf("first event execution failed: %v", err)
	}

	if len(repo.notifications) != 1 {
		t.Fatalf("expected 1 notification created, got %d", len(repo.notifications))
	}

	firstMessageCount := len(devAdapter.GetSentMessages())
	if firstMessageCount == 0 {
		t.Fatalf("expected delivery adapter to record sent messages")
	}

	// Second execution with SAME event_id -> Should be detected as duplicate and ignored
	if err := svc.ProcessKafkaEvent(ctx, envelope); err != nil {
		t.Fatalf("duplicate event execution should return nil, got: %v", err)
	}

	// Verification: Notification count and sent messages MUST NOT increase
	if len(repo.notifications) != 1 {
		t.Fatalf("duplicate event caused duplicate notification creation! count = %d", len(repo.notifications))
	}
	if len(devAdapter.GetSentMessages()) != firstMessageCount {
		t.Fatalf("duplicate event caused duplicate message dispatch!")
	}
}

func TestAppointmentCancelledCancelsScheduledReminders(t *testing.T) {
	repo := newMockRepository()
	devAdapter := provider.NewDevelopmentDeliveryAdapter()
	svc := service.NewNotificationService(repo, devAdapter, nil, "test-group", nil)
	ctx := context.Background()

	// 1. Confirm appointment
	confirmPayload, _ := json.Marshal(domain.AppointmentConfirmedPayload{
		AppointmentID:  "apt-cancel-test",
		PatientID:      "pat-cancel-test",
		ClinicID:       "cln-001",
		PackageName:    "Shirodhara Special",
		ScheduledStart: time.Now().UTC().Add(36 * time.Hour).Format(time.RFC3339),
	})
	_ = svc.ProcessKafkaEvent(ctx, &kafka.EventEnvelope{
		EventID:   "evt-confirm",
		EventType: "panchakarma.appointment.confirmed.v1",
		Payload:   confirmPayload,
	})

	if len(repo.reminders) != 1 {
		t.Fatalf("expected 1 scheduled reminder, got %d", len(repo.reminders))
	}

	// 2. Cancel appointment
	cancelPayload, _ := json.Marshal(domain.AppointmentCancelledPayload{
		AppointmentID: "apt-cancel-test",
		PatientID:     "pat-cancel-test",
		Reason:        "Patient sudden travel",
	})
	_ = svc.ProcessKafkaEvent(ctx, &kafka.EventEnvelope{
		EventID:   "evt-cancel",
		EventType: "panchakarma.appointment.cancelled.v1",
		Payload:   cancelPayload,
	})

	// Check reminder status
	rem := repo.reminders["rem-24h-apt-cancel-test"]
	if rem == nil || rem.Status != domain.ReminderStatusCancelled {
		t.Fatalf("expected reminder to be CANCELLED, got %v", rem)
	}

	// Running due reminders check must NOT dispatch cancelled reminder
	dispatched, err := svc.ProcessDueReminders(ctx)
	if err != nil {
		t.Fatalf("process due reminders failed: %v", err)
	}
	if dispatched != 0 {
		t.Fatalf("expected 0 dispatched reminders for cancelled appointment, got %d", dispatched)
	}
}

func TestRecheckAppointmentStatusBeforeReminderDelivery(t *testing.T) {
	repo := newMockRepository()
	devAdapter := provider.NewDevelopmentDeliveryAdapter()
	// Appointment checker returns FALSE (appointment cancelled externally in appointment-service)
	checker := &mockApptChecker{active: false}
	svc := service.NewNotificationService(repo, devAdapter, checker, "test-group", nil)
	ctx := context.Background()

	// Register due reminder
	rem := &domain.ScheduledReminder{
		ID:              "rem-due-1",
		AppointmentID:   "apt-recheck-inactive",
		PatientID:       "pat-001",
		ClinicID:        "cln-001",
		ReminderType:    "pre_therapy_prep",
		ScheduledTime:   time.Now().UTC().Add(-5 * time.Minute), // already due
		TargetEventTime: time.Now().UTC().Add(2 * time.Hour),
		Status:          domain.ReminderStatusScheduled,
		IdempotencyKey:  "key-due-1",
	}
	_ = repo.CreateScheduledReminder(ctx, rem)

	// Process due reminders
	dispatched, err := svc.ProcessDueReminders(ctx)
	if err != nil {
		t.Fatalf("process due reminders error: %v", err)
	}
	if dispatched != 0 {
		t.Fatalf("expected reminder delivery to be skipped due to inactive appointment status, got %d", dispatched)
	}

	if rem.Status != domain.ReminderStatusCancelled {
		t.Fatalf("expected reminder to transition to CANCELLED, got %s", rem.Status)
	}
}

func TestDeliveryProviderFailureResilience(t *testing.T) {
	repo := newMockRepository()
	devAdapter := provider.NewDevelopmentDeliveryAdapter()
	// Simulate email gateway failure
	devAdapter.SetFailureSimulation("email", true)

	svc := service.NewNotificationService(repo, devAdapter, nil, "test-group", nil)
	ctx := context.Background()

	confirmPayload, _ := json.Marshal(domain.AppointmentConfirmedPayload{
		AppointmentID:  "apt-fail-test",
		PatientID:      "pat-fail-test",
		ClinicID:       "cln-001",
		PackageName:    "Panchakarma",
		ScheduledStart: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
	})

	// Process event — MUST NOT fail the overarching transaction
	err := svc.ProcessKafkaEvent(ctx, &kafka.EventEnvelope{
		EventID:   "evt-provider-fail",
		EventType: "panchakarma.appointment.confirmed.v1",
		Payload:   confirmPayload,
	})
	if err != nil {
		t.Fatalf("provider failure should not reverse confirmed booking event: %v", err)
	}

	// Verify notification record exists
	if len(repo.notifications) != 1 {
		t.Fatalf("notification should still be saved in DB")
	}

	// Verify delivery attempt was logged as FAILED
	attempts := repo.attempts
	var foundFailedEmail bool
	for _, a := range attempts {
		if a.Channel == "email" && a.Status == domain.DeliveryStatusFailed {
			foundFailedEmail = true
			if a.ErrorMessage == nil || *a.ErrorMessage == "" {
				t.Errorf("expected error message recorded on failed delivery attempt")
			}
		}
	}
	if !foundFailedEmail {
		t.Fatalf("expected to find a FAILED delivery attempt for email channel")
	}
}

func TestEmergencyComplicationAlert(t *testing.T) {
	repo := newMockRepository()
	devAdapter := provider.NewDevelopmentDeliveryAdapter()
	svc := service.NewNotificationService(repo, devAdapter, nil, "test-group", nil)
	ctx := context.Background()

	payload, _ := json.Marshal(domain.ComplicationFlaggedPayload{
		ComplicationID:    "comp-911",
		SessionID:         "ses-888",
		AppointmentID:     "apt-777",
		PatientID:         "pat-666",
		ClinicID:          "cln-555",
		AttendingDoctorID: "doc-anand-emergency",
		Severity:          "HIGH",
		Symptoms:          "Hypotension and dizziness",
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
	})

	err := svc.ProcessKafkaEvent(ctx, &kafka.EventEnvelope{
		EventID:   "evt-emerg-01",
		EventType: "panchakarma.complication.flagged.v1",
		Payload:   payload,
	})
	if err != nil {
		t.Fatalf("complication event processing failed: %v", err)
	}

	// Verify EMERGENCY priority notification created
	var foundEmergency bool
	for _, n := range repo.notifications {
		if n.Priority == domain.PriorityEmergency && n.UserID == "doc-anand-emergency" {
			foundEmergency = true
			break
		}
	}
	if !foundEmergency {
		t.Fatalf("expected emergency priority notification addressed to attending physician")
	}

	// Verify push alert dispatched
	var foundPush bool
	for _, msg := range devAdapter.GetSentMessages() {
		if msg.Channel == "push" {
			foundPush = true
			break
		}
	}
	if !foundPush {
		t.Fatalf("expected emergency push alert dispatched to doctor")
	}
}

func TestOutboxPublisherWorker(t *testing.T) {
	repo := newMockRepository()
	kafkaMock := &mockKafkaPublisher{}
	publisher := service.NewOutboxPublisher(repo, kafkaMock, 100*time.Millisecond, 10)

	// Add pending outbox item
	outboxID := "outbox-test-1"
	repo.outbox[outboxID] = &domain.OutboxEvent{
		ID:            outboxID,
		AggregateType: "NOTIFICATION",
		AggregateID:   "notif-101",
		EventType:     "panchakarma.notification.dispatched.v1",
		Payload:       []byte(`{"status":"SENT"}`),
		Status:        "PENDING",
		CreatedAt:     time.Now().UTC(),
	}

	ctx := context.Background()
	count, err := publisher.ProcessBatch(ctx)
	if err != nil {
		t.Fatalf("process batch failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 processed outbox event, got %d", count)
	}

	if repo.outbox[outboxID].Status != "PROCESSED" {
		t.Fatalf("expected outbox event status PROCESSED, got %s", repo.outbox[outboxID].Status)
	}
	if len(kafkaMock.published) != 1 {
		t.Fatalf("expected 1 event published to Kafka, got %d", len(kafkaMock.published))
	}
}

func TestEventSchemaCompatibility(t *testing.T) {
	// Verify JSON unmarshaling for all core event schemas
	confirmJSON := `{"appointment_id":"a1","patient_id":"p1","clinic_id":"c1","scheduled_start":"2026-11-01T09:00:00Z"}`
	var confirm domain.AppointmentConfirmedPayload
	if err := json.Unmarshal([]byte(confirmJSON), &confirm); err != nil {
		t.Errorf("failed unmarshaling AppointmentConfirmedPayload: %v", err)
	}

	sessionJSON := `{"session_id":"s1","appointment_id":"a1","completed_at":"2026-11-01T10:30:00Z","post_care_instructions":"Drink warm ginger water"}`
	var session domain.SessionCompletedPayload
	if err := json.Unmarshal([]byte(sessionJSON), &session); err != nil {
		t.Errorf("failed unmarshaling SessionCompletedPayload: %v", err)
	}

	compJSON := `{"complication_id":"c1","session_id":"s1","severity":"HIGH","symptoms":"Fever"}`
	var comp domain.ComplicationFlaggedPayload
	if err := json.Unmarshal([]byte(compJSON), &comp); err != nil {
		t.Errorf("failed unmarshaling ComplicationFlaggedPayload: %v", err)
	}

	fbJSON := `{"feedback_id":"fb1","appointment_id":"a1","patient_id":"p1","rating":5}`
	var fb domain.FeedbackSubmittedPayload
	if err := json.Unmarshal([]byte(fbJSON), &fb); err != nil {
		t.Errorf("failed unmarshaling FeedbackSubmittedPayload: %v", err)
	}

	fuJSON := `{"follow_up_id":"fu1","session_id":"s1","patient_id":"p1","recommended_date":"2026-11-15"}`
	var fu domain.FollowUpReminderPayload
	if err := json.Unmarshal([]byte(fuJSON), &fu); err != nil {
		t.Errorf("failed unmarshaling FollowUpReminderPayload: %v", err)
	}
}

func TestFollowUpAndFeedbackEvents(t *testing.T) {
	repo := newMockRepository()
	devProvider := provider.NewDevelopmentDeliveryAdapter()
	svc := service.NewNotificationService(repo, devProvider, nil, "test-group", nil)
	ctx := context.Background()

	// 1. Feedback submitted event
	feedbackPayload, _ := json.Marshal(domain.FeedbackSubmittedPayload{
		FeedbackID:    "fb-101",
		AppointmentID: "apt-101",
		PatientID:     "pat-101",
		ClinicID:      "cln-101",
		Rating:        5,
		Comments:      "Excellent treatment",
	})
	fbEnv := &kafka.EventEnvelope{
		EventID:     "evt-fb-1",
		EventType:   "panchakarma.feedback.submitted.v1",
		AggregateID: "fb-101",
		Payload:     feedbackPayload,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	if err := svc.ProcessKafkaEvent(ctx, fbEnv); err != nil {
		t.Fatalf("failed processing feedback submitted event: %v", err)
	}

	// 2. Follow-up reminder event
	followUpPayload, _ := json.Marshal(domain.FollowUpReminderPayload{
		FollowUpID:      "fu-201",
		SessionID:       "ses-201",
		PatientID:       "pat-101",
		ClinicID:        "cln-101",
		RecommendedDate: "2026-11-15",
		Instructions:    "Follow-up Nasya treatment",
	})
	fuEnv := &kafka.EventEnvelope{
		EventID:     "evt-fu-1",
		EventType:   "panchakarma.followup.reminder.v1",
		AggregateID: "fu-201",
		Payload:     followUpPayload,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	if err := svc.ProcessKafkaEvent(ctx, fuEnv); err != nil {
		t.Fatalf("failed processing follow up reminder event: %v", err)
	}

	// Verify both notifications persisted for patient pat-101
	notifs, count, err := svc.ListUserNotifications(ctx, "pat-101", 10, 0)
	if err != nil {
		t.Fatalf("failed listing user notifications: %v", err)
	}
	if count != 2 || len(notifs) != 2 {
		t.Fatalf("expected 2 notifications, got count=%d, len=%d", count, len(notifs))
	}
}

func TestDLQRoutingOnFailedEvent(t *testing.T) {
	repo := newMockRepository()
	devProvider := provider.NewDevelopmentDeliveryAdapter()

	svc := service.NewNotificationService(repo, devProvider, nil, "test-group", nil)
	ctx := context.Background()

	env := &kafka.EventEnvelope{
		EventID:     "evt-fail-1",
		EventType:   "panchakarma.session.started.v1",
		AggregateID: "ses-fail",
		Payload:     []byte("invalid json"),
	}
	err := svc.RouteToDLQ(ctx, env, errors.New("malformed session started payload"))
	if err != nil {
		t.Fatalf("expected RouteToDLQ with nil producer to return nil, got: %v", err)
	}
}
