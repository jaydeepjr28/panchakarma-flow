package service

import (
	"context"
	"strings"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/therapy/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/kafka"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
)

// KafkaEventPublisher defines the contract for dispatching envelopes to Kafka.
type KafkaEventPublisher interface {
	PublishEvent(ctx context.Context, topic, partitionKey string, event *kafka.EventEnvelope) error
}

// OutboxPublisher background worker dispatches pending transactional outbox events to Kafka.
type OutboxPublisher struct {
	repo      repository.TherapyRepository
	publisher KafkaEventPublisher
	interval  time.Duration
	batchSize int
	stopCh    chan struct{}
}

// NewOutboxPublisher initializes the transactional outbox polling publisher for Therapy Service.
func NewOutboxPublisher(repo repository.TherapyRepository, publisher KafkaEventPublisher, interval time.Duration, batchSize int) *OutboxPublisher {
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

// Start launches the background worker polling loop.
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

// Stop terminates the polling loop.
func (p *OutboxPublisher) Stop() {
	select {
	case <-p.stopCh:
		// already closed
	default:
		close(p.stopCh)
	}
}

// ProcessBatch queries pending outbox events, formats standard CloudEvents envelopes, and produces to Kafka.
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
				Producer:      "therapy-service",
				Version:       "1.0",
				Payload:       event.Payload,
			}

			// Route complications to emergency topic, otherwise sessions topic
			topic := kafka.TopicSessionsLifecycle
			if event.AggregateType == "complication" || strings.Contains(event.EventType, "complication") {
				topic = kafka.TopicComplications
			}

			err := p.publisher.PublishEvent(ctx, topic, event.AggregateID, envelope)
			if err != nil {
				logger.Warn("Failed to dispatch outbox event to Kafka", "event_id", event.ID, "error", err)
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
