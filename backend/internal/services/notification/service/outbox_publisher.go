package service

import (
	"context"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/notification/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/kafka"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
)

// KafkaEventPublisher publishes domain event envelopes to Kafka.
type KafkaEventPublisher interface {
	PublishEvent(ctx context.Context, topic, partitionKey string, event *kafka.EventEnvelope) error
}

// OutboxPublisher background worker dispatches pending transactional outbox events to Kafka.
type OutboxPublisher struct {
	repo      repository.NotificationRepository
	publisher KafkaEventPublisher
	interval  time.Duration
	batchSize int
	stopCh    chan struct{}
}

// NewOutboxPublisher initializes the transactional outbox polling publisher for Notification Service.
func NewOutboxPublisher(repo repository.NotificationRepository, publisher KafkaEventPublisher, interval time.Duration, batchSize int) *OutboxPublisher {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 50
	}
	return &OutboxPublisher{
		repo:      repo,
		publisher: publisher,
		interval:  interval,
		batchSize: batchSize,
		stopCh:    make(chan struct{}),
	}
}

// Start launches the outbox background polling loop.
func (p *OutboxPublisher) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-p.stopCh:
				return
			case <-ticker.C:
				_, _ = p.ProcessBatch(ctx)
			}
		}
	}()
}

// Stop terminates the polling worker loop.
func (p *OutboxPublisher) Stop() {
	select {
	case <-p.stopCh:
		// already closed
	default:
		close(p.stopCh)
	}
}

// ProcessBatch reads up to batchSize PENDING outbox events and dispatches them to Kafka.
func (p *OutboxPublisher) ProcessBatch(ctx context.Context) (int, error) {
	events, err := p.repo.GetPendingOutboxEvents(ctx, p.batchSize)
	if err != nil {
		return 0, err
	}

	processedCount := 0
	for _, event := range events {
		if p.publisher != nil {
			envelope := &kafka.EventEnvelope{
				EventID:       event.ID,
				EventType:     event.EventType,
				AggregateType: event.AggregateType,
				AggregateID:   event.AggregateID,
				TraceID:       event.TraceID,
				Timestamp:     event.CreatedAt.Format(time.RFC3339),
				Producer:      "notification-service",
				Version:       "1.0",
				Payload:       event.Payload,
			}

			topic := kafka.TopicFeedbackEvents
			err := p.publisher.PublishEvent(ctx, topic, event.AggregateID, envelope)
			if err != nil {
				logger.Warn("Failed to dispatch notification outbox event to Kafka", "event_id", event.ID, "error", err)
				_ = p.repo.MarkOutboxEventFailed(ctx, event.ID, err.Error())
				continue
			}
		}

		if err := p.repo.MarkOutboxEventProcessed(ctx, event.ID); err != nil {
			logger.Error("Failed to mark outbox event as processed", "event_id", event.ID, "error", err)
			continue
		}
		processedCount++
	}

	return processedCount, nil
}
