package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	kafkaGo "github.com/segmentio/kafka-go"
)

// EventHandler is the callback invoked when a valid event is consumed.
type EventHandler func(ctx context.Context, event *EventEnvelope) error

// Consumer wraps kafka-go Reader for consumer group event stream processing.
type Consumer struct {
	reader *kafkaGo.Reader
	topic  string
	group  string
}

// NewConsumer initializes a consumer group reader for a specific topic.
func NewConsumer(cfg *config.KafkaConfig, topic string) *Consumer {
	reader := kafkaGo.NewReader(kafkaGo.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.ConsumerGroup,
		Topic:          topic,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafkaGo.FirstOffset,
	})

	logger.Info("Kafka Consumer initialized",
		slog.String("topic", topic),
		slog.String("group", cfg.ConsumerGroup),
	)

	return &Consumer{
		reader: reader,
		topic:  topic,
		group:  cfg.ConsumerGroup,
	}
}

// Start listens for incoming messages in a blocking loop until ctx is cancelled.
func (c *Consumer) Start(ctx context.Context, handler EventHandler) error {
	logger.Info("Starting Kafka consumer loop", slog.String("topic", c.topic), slog.String("group", c.group))

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping Kafka consumer loop on context cancellation", slog.String("topic", c.topic))
			return nil
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				logger.Error("Failed to fetch Kafka message", slog.String("topic", c.topic), slog.String("error", err.Error()))
				time.Sleep(500 * time.Millisecond) // Backoff on fetch error
				continue
			}

			var event EventEnvelope
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				logger.Error("Malformed Kafka event payload - routing to poison pill log",
					slog.String("topic", c.topic),
					slog.String("error", err.Error()),
					slog.String("raw_value", string(msg.Value)),
				)
				// Commit offset to avoid poison pill blocking consumer partition
				_ = c.reader.CommitMessages(ctx, msg)
				continue
			}

			// Execute business handler
			eventCtx := logger.ContextWithTraceID(ctx, event.TraceID)
			if err := handler(eventCtx, &event); err != nil {
				logger.WithContext(eventCtx).Error("Error handling Kafka event",
					slog.String("event_type", event.EventType),
					slog.String("event_id", event.EventID),
					slog.String("error", err.Error()),
				)
				// In production: route to DLQ after retries
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				logger.Error("Failed to commit Kafka offset", slog.String("topic", c.topic), slog.String("error", err.Error()))
			}
		}
	}
}

// Close gracefully closes the consumer reader.
func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
