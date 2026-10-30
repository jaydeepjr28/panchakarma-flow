package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
	kafkaGo "github.com/segmentio/kafka-go"
)

// Producer provides pure-Go event publishing to Kafka topics.
type Producer struct {
	writer *kafkaGo.Writer
	cfg    *config.KafkaConfig
}

// NewProducer initializes a Kafka Writer with connection pooling and retries.
func NewProducer(cfg *config.KafkaConfig) *Producer {
	writer := &kafkaGo.Writer{
		Addr:         kafkaGo.TCP(cfg.Brokers...),
		Balancer:     &kafkaGo.LeastBytes{},
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
		RequiredAcks: kafkaGo.RequireOne,
		MaxAttempts:  3,
	}

	logger.Info("Kafka Producer initialized",
		slog.Any("brokers", cfg.Brokers),
		slog.String("client_id", cfg.ClientID),
	)

	return &Producer{
		writer: writer,
		cfg:    cfg,
	}
}

// PublishEvent serializes and publishes an EventEnvelope to the specified topic.
func (p *Producer) PublishEvent(ctx context.Context, topic, partitionKey string, event *EventEnvelope) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := kafkaGo.Message{
		Topic: topic,
		Key:   []byte(partitionKey),
		Value: data,
		Headers: []kafkaGo.Header{
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "trace_id", Value: []byte(event.TraceID)},
			{Key: "event_id", Value: []byte(event.EventID)},
		},
		Time: time.Now().UTC(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		logger.WithContext(ctx).Error("Failed to publish Kafka event",
			slog.String("topic", topic),
			slog.String("event_type", event.EventType),
			slog.String("key", partitionKey),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to publish message to topic %s: %w", topic, err)
	}

	logger.WithContext(ctx).Debug("Kafka event published successfully",
		slog.String("topic", topic),
		slog.String("event_type", event.EventType),
		slog.String("event_id", event.EventID),
	)

	return nil
}

// Close gracefully closes the producer writer.
func (p *Producer) Close() error {
	return p.writer.Close()
}
