package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/internal/services/feedback/repository"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/kafka"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
)

type OutboxPublisher struct {
	repo     repository.FeedbackRepository
	producer *kafka.Producer
	interval time.Duration
	stopCh   chan struct{}
}

func NewOutboxPublisher(repo repository.FeedbackRepository, producer *kafka.Producer, interval time.Duration) *OutboxPublisher {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &OutboxPublisher{
		repo:     repo,
		producer: producer,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (p *OutboxPublisher) Start() {
	go func() {
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.publishPendingBatch()
			case <-p.stopCh:
				return
			}
		}
	}()
}

func (p *OutboxPublisher) Stop() {
	close(p.stopCh)
}

func (p *OutboxPublisher) publishPendingBatch() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	events, err := p.repo.GetPendingOutboxEvents(ctx, 50)
	if err != nil {
		logger.Warn("Failed to query pending feedback outbox events", slog.String("error", err.Error()))
		return
	}

	for _, ev := range events {
		if p.producer == nil {
			_ = p.repo.MarkOutboxEventPublished(ctx, ev.ID)
			continue
		}

		envelope := &kafka.EventEnvelope{
			EventID:       ev.ID,
			EventType:     ev.EventType,
			AggregateType: ev.AggregateType,
			AggregateID:   ev.AggregateID,
			TraceID:       ev.TraceID,
			Timestamp:     ev.CreatedAt.Format(time.RFC3339),
			Producer:      "feedback-analytics-service",
			Version:       "1.0",
			Payload:       ev.Payload,
		}

		topic := kafka.TopicFeedbackEvents
		err := p.producer.PublishEvent(ctx, topic, ev.AggregateID, envelope)
		if err != nil {
			logger.Warn("Failed to publish feedback outbox event to Kafka",
				slog.String("id", ev.ID),
				slog.String("event_type", ev.EventType),
				slog.String("error", err.Error()))
			_ = p.repo.MarkOutboxEventFailed(ctx, ev.ID)
			continue
		}

		if err := p.repo.MarkOutboxEventPublished(ctx, ev.ID); err != nil {
			logger.Warn("Failed to mark feedback outbox event published",
				slog.String("id", ev.ID),
				slog.String("error", err.Error()))
		}
	}
}
