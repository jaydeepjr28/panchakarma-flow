package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/kafka"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
)

// ConsumerManager manages multiple Kafka topic consumers concurrently.
type ConsumerManager struct {
	svc        NotificationService
	cfg        *config.KafkaConfig
	consumers  []*kafka.Consumer
	maxRetries int
	wg         sync.WaitGroup
	cancelFn   context.CancelFunc
}

// NewConsumerManager initializes a multi-topic consumer manager.
func NewConsumerManager(svc NotificationService, cfg *config.KafkaConfig) *ConsumerManager {
	return &ConsumerManager{
		svc:        svc,
		cfg:        cfg,
		consumers:  make([]*kafka.Consumer, 0),
		maxRetries: 3,
	}
}

// Start launches consumers for all required domain topics in background goroutines.
func (m *ConsumerManager) Start(parentCtx context.Context) error {
	if m.cfg == nil || len(m.cfg.Brokers) == 0 {
		logger.Warn("Kafka brokers not configured, skipping consumer manager start")
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
			logger.Info("Starting notification consumer loop", slog.String("topic", top))

			err := c.Start(ctx, func(eventCtx context.Context, envelope *kafka.EventEnvelope) error {
				return m.processWithRetries(eventCtx, envelope)
			})
			if err != nil && ctx.Err() == nil {
				logger.Error("Consumer terminated unexpectedly", slog.String("topic", top), slog.String("error", err.Error()))
			}
		}(consumer, topic)
	}

	return nil
}

// processWithRetries executes event handling with bounded retries and exponential backoff.
func (m *ConsumerManager) processWithRetries(ctx context.Context, env *kafka.EventEnvelope) error {
	var lastErr error
	backoffs := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	for attempt := 0; attempt <= m.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := backoffs[attempt-1]
			logger.Warn("Retrying Kafka event processing",
				slog.String("event_id", env.EventID),
				slog.String("event_type", env.EventType),
				slog.Int("attempt", attempt),
				slog.Duration("backoff", backoff),
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := m.svc.ProcessKafkaEvent(ctx, env)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	// Max retries exceeded: Route to Dead Letter Queue (DLQ)
	if lastErr != nil {
		_ = m.svc.RouteToDLQ(ctx, env, lastErr)
	}

	return fmt.Errorf("exceeded max retries (%d) processing event %s: %w", m.maxRetries, env.EventID, lastErr)
}

// Stop gracefully stops all running consumers and waits for them to terminate.
func (m *ConsumerManager) Stop() {
	if m.cancelFn != nil {
		m.cancelFn()
	}
	for _, c := range m.consumers {
		_ = c.Close()
	}
	m.wg.Wait()
	logger.Info("All notification Kafka consumers shut down gracefully")
}
