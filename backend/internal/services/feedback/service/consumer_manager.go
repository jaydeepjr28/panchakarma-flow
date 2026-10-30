package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/feedback/domain"
	"github.com/jaydeepjr28/panchakarma-flow/internal/services/feedback/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/kafka"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
)

type ConsumerManager struct {
	repo          repository.FeedbackRepository
	cfg           *config.KafkaConfig
	consumers     []*kafka.Consumer
	consumerGroup string
	stopCh        chan struct{}
	cancelFn      context.CancelFunc
	wg            sync.WaitGroup
}

func NewConsumerManager(repo repository.FeedbackRepository, cfg *config.KafkaConfig, groupID string) *ConsumerManager {
	if groupID == "" {
		groupID = "feedback-analytics-group"
	}
	cfgCopy := *cfg
	cfgCopy.ConsumerGroup = groupID

	return &ConsumerManager{
		repo:          repo,
		cfg:           &cfgCopy,
		consumers:     make([]*kafka.Consumer, 0),
		consumerGroup: groupID,
		stopCh:        make(chan struct{}),
	}
}

func (m *ConsumerManager) Start(parentCtx context.Context) error {
	if m.cfg == nil || len(m.cfg.Brokers) == 0 {
		logger.Warn("Kafka brokers not configured, skipping feedback consumer manager")
		return nil
	}

	ctx, cancel := context.WithCancel(parentCtx)
	m.cancelFn = cancel

	topics := []string{
		kafka.TopicAppointmentsLifecycle,
		kafka.TopicSessionsLifecycle,
		kafka.TopicComplications,
		kafka.TopicFeedbackEvents,
	}

	for _, topic := range topics {
		consumer := kafka.NewConsumer(m.cfg, topic)
		m.consumers = append(m.consumers, consumer)

		m.wg.Add(1)
		go func(c *kafka.Consumer, top string) {
			defer m.wg.Done()
			logger.Info("Starting Feedback & Analytics consumer loop", slog.String("topic", top), slog.String("group", m.consumerGroup))

			err := c.Start(ctx, func(eventCtx context.Context, envelope *kafka.EventEnvelope) error {
				return m.processEvent(eventCtx, envelope)
			})
			if err != nil && ctx.Err() == nil {
				logger.Error("Feedback consumer terminated unexpectedly", slog.String("topic", top), slog.String("error", err.Error()))
			}
		}(consumer, topic)
	}

	return nil
}

func (m *ConsumerManager) Stop() {
	if m.cancelFn != nil {
		m.cancelFn()
	}
	for _, c := range m.consumers {
		_ = c.Close()
	}
	m.wg.Wait()
}

func (m *ConsumerManager) processEvent(ctx context.Context, env *kafka.EventEnvelope) error {
	// 1. Deduplication Ledger check
	processed, err := m.repo.IsEventProcessed(ctx, env.EventID)
	if err != nil {
		return fmt.Errorf("deduplication check error: %w", err)
	}
	if processed {
		logger.Debug("Duplicate event skipped by feedback analytics consumer",
			slog.String("event_id", env.EventID),
			slog.String("event_type", env.EventType))
		return nil
	}

	// 2. Handle Event based on Type
	switch env.EventType {
	case "panchakarma.appointment.confirmed.v1":
		err = m.handleAppointmentConfirmed(ctx, env)
	case "panchakarma.appointment.cancelled.v1":
		err = m.handleAppointmentCancelled(ctx, env)
	case "panchakarma.appointment.rescheduled.v1":
		err = m.handleAppointmentRescheduled(ctx, env)
	case "panchakarma.session.started.v1":
		err = m.handleSessionStarted(ctx, env)
	case "panchakarma.session.completed.v1":
		err = m.handleSessionCompleted(ctx, env)
	case "panchakarma.complication.flagged.v1":
		err = m.handleComplicationFlagged(ctx, env)
	case "panchakarma.complication.resolved.v1":
		err = m.handleComplicationResolved(ctx, env)
	default:
		// Unknown or unmonitored event, commit gracefully
		return nil
	}

	if err != nil {
		return err
	}

	// 3. Record in Consumed Events
	digest := sha256.Sum256(env.Payload)
	digestStr := hex.EncodeToString(digest[:])
	return m.repo.RecordConsumedEvent(ctx, env.EventID, env.EventType, m.consumerGroup, digestStr)
}

func (m *ConsumerManager) handleAppointmentConfirmed(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload struct {
		AppointmentID  string `json:"appointment_id"`
		PatientID      string `json:"patient_id"`
		ClinicID       string `json:"clinic_id"`
		DoctorID       string `json:"doctor_id"`
		RoomID         string `json:"room_id"`
		PackageID      string `json:"package_id"`
		PackageName    string `json:"package_name"`
		ScheduledStart string `json:"scheduled_start"`
		ScheduledEnd   string `json:"scheduled_end"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return err
	}

	startTime, _ := time.Parse(time.RFC3339, payload.ScheduledStart)
	endTime, _ := time.Parse(time.RFC3339, payload.ScheduledEnd)
	if startTime.IsZero() {
		startTime = time.Now().UTC()
	}
	if endTime.IsZero() {
		endTime = startTime.Add(2 * time.Hour)
	}

	apt := &domain.AnalyticsAppointment{
		ID:             payload.AppointmentID,
		ClinicID:       payload.ClinicID,
		PatientID:      payload.PatientID,
		DoctorID:       &payload.DoctorID,
		RoomID:         &payload.RoomID,
		PackageID:      &payload.PackageID,
		PackageName:    &payload.PackageName,
		Status:         "CONFIRMED",
		ScheduledStart: startTime,
		ScheduledEnd:   endTime,
	}

	return m.repo.UpsertAppointmentProjection(ctx, apt)
}

func (m *ConsumerManager) handleAppointmentCancelled(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload struct {
		AppointmentID string `json:"appointment_id"`
		PatientID     string `json:"patient_id"`
		ClinicID      string `json:"clinic_id"`
		Reason        string `json:"reason"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return err
	}

	apt, err := m.repo.GetAppointmentProjection(ctx, payload.AppointmentID)
	if err != nil || apt == nil {
		apt = &domain.AnalyticsAppointment{
			ID:             payload.AppointmentID,
			ClinicID:       payload.ClinicID,
			PatientID:      payload.PatientID,
			Status:         "CANCELLED",
			ScheduledStart: time.Now().UTC(),
			ScheduledEnd:   time.Now().UTC().Add(time.Hour),
		}
	} else {
		apt.Status = "CANCELLED"
	}

	return m.repo.UpsertAppointmentProjection(ctx, apt)
}

func (m *ConsumerManager) handleAppointmentRescheduled(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload struct {
		AppointmentID string `json:"appointment_id"`
		PatientID     string `json:"patient_id"`
		ClinicID      string `json:"clinic_id"`
		NewStartTime  string `json:"new_start_time"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return err
	}

	startTime, _ := time.Parse(time.RFC3339, payload.NewStartTime)
	if startTime.IsZero() {
		startTime = time.Now().UTC()
	}

	apt, err := m.repo.GetAppointmentProjection(ctx, payload.AppointmentID)
	if err != nil || apt == nil {
		apt = &domain.AnalyticsAppointment{
			ID:             payload.AppointmentID,
			ClinicID:       payload.ClinicID,
			PatientID:      payload.PatientID,
			Status:         "RESCHEDULED",
			ScheduledStart: startTime,
			ScheduledEnd:   startTime.Add(2 * time.Hour),
		}
	} else {
		apt.Status = "RESCHEDULED"
		apt.ScheduledStart = startTime
		apt.ScheduledEnd = startTime.Add(2 * time.Hour)
	}

	return m.repo.UpsertAppointmentProjection(ctx, apt)
}

func (m *ConsumerManager) handleSessionStarted(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload struct {
		SessionID     string `json:"session_id"`
		AppointmentID string `json:"appointment_id"`
		PatientID     string `json:"patient_id"`
		ClinicID      string `json:"clinic_id"`
		StartedAt     string `json:"started_at"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return err
	}

	startedTime, _ := time.Parse(time.RFC3339, payload.StartedAt)
	if startedTime.IsZero() {
		startedTime = time.Now().UTC()
	}

	sess := &domain.AnalyticsSession{
		ID:                payload.SessionID,
		AppointmentID:     payload.AppointmentID,
		ClinicID:          payload.ClinicID,
		PatientID:         payload.PatientID,
		Status:            "IN_PROGRESS",
		AllStepsCompleted: true,
		StartedAt:         &startedTime,
	}

	if err := m.repo.UpsertSessionProjection(ctx, sess); err != nil {
		return err
	}

	// Update appointment status to IN_PROGRESS
	apt, err := m.repo.GetAppointmentProjection(ctx, payload.AppointmentID)
	if err == nil && apt != nil {
		apt.Status = "IN_PROGRESS"
		_ = m.repo.UpsertAppointmentProjection(ctx, apt)
	}

	return nil
}

func (m *ConsumerManager) handleSessionCompleted(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload struct {
		SessionID     string `json:"session_id"`
		AppointmentID string `json:"appointment_id"`
		PatientID     string `json:"patient_id"`
		ClinicID      string `json:"clinic_id"`
		CompletedAt   string `json:"completed_at"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return err
	}

	completedTime, _ := time.Parse(time.RFC3339, payload.CompletedAt)
	if completedTime.IsZero() {
		completedTime = time.Now().UTC()
	}

	sess, err := m.repo.GetSessionProjection(ctx, payload.SessionID)
	if err != nil || sess == nil {
		started := completedTime.Add(-45 * time.Minute)
		dur := 45.0
		sess = &domain.AnalyticsSession{
			ID:                payload.SessionID,
			AppointmentID:     payload.AppointmentID,
			ClinicID:          payload.ClinicID,
			PatientID:         payload.PatientID,
			Status:            "COMPLETED",
			AllStepsCompleted: true,
			StartedAt:         &started,
			CompletedAt:       &completedTime,
			DurationMinutes:   &dur,
		}
	} else {
		sess.Status = "COMPLETED"
		sess.CompletedAt = &completedTime
		if sess.StartedAt != nil && completedTime.After(*sess.StartedAt) {
			dur := completedTime.Sub(*sess.StartedAt).Minutes()
			sess.DurationMinutes = &dur
		}
	}

	if err := m.repo.UpsertSessionProjection(ctx, sess); err != nil {
		return err
	}

	// Mark appointment completed
	apt, err := m.repo.GetAppointmentProjection(ctx, payload.AppointmentID)
	if err == nil && apt != nil {
		apt.Status = "COMPLETED"
		_ = m.repo.UpsertAppointmentProjection(ctx, apt)
	}

	return nil
}

func (m *ConsumerManager) handleComplicationFlagged(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload struct {
		ComplicationID    string `json:"complication_id"`
		SessionID         string `json:"session_id"`
		AppointmentID     string `json:"appointment_id"`
		PatientID         string `json:"patient_id"`
		ClinicID          string `json:"clinic_id"`
		AttendingDoctorID string `json:"attending_doctor_id"`
		Severity          string `json:"severity"`
		Symptoms          string `json:"symptoms"`
		Timestamp         string `json:"timestamp"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return err
	}

	ae := &domain.AdverseEvent{
		ID:                payload.ComplicationID,
		SessionID:         &payload.SessionID,
		AppointmentID:     &payload.AppointmentID,
		PatientID:         payload.PatientID,
		ClinicID:          payload.ClinicID,
		Severity:          payload.Severity,
		Symptoms:          payload.Symptoms,
		Status:            "REPORTED",
		ReportedBy:        payload.AttendingDoctorID,
		ReporterRole:      "doctor",
		AttendingDoctorID: &payload.AttendingDoctorID,
	}

	return m.repo.CreateAdverseEvent(ctx, ae)
}

func (m *ConsumerManager) handleComplicationResolved(ctx context.Context, env *kafka.EventEnvelope) error {
	var payload struct {
		ComplicationID string `json:"complication_id"`
		SessionID      string `json:"session_id"`
		DoctorID       string `json:"doctor_id"`
		ResolvedAt     string `json:"resolved_at"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return err
	}

	return m.repo.AcknowledgeAdverseEvent(ctx, payload.ComplicationID, payload.DoctorID, "Resolved by attending physician", "RESOLVED", true)
}
