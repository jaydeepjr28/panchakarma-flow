package kafka

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Standard Kafka Topics
const (
	TopicAppointmentsLifecycle = "panchakarma.appointments.events"
	TopicSessionsLifecycle     = "panchakarma.sessions.events"
	TopicComplications         = "panchakarma.complications.emergency"
	TopicIdentityEvents        = "panchakarma.identity.events"
	TopicFeedbackEvents        = "panchakarma.feedback.events"
)

// EventEnvelope defines the standard CloudEvents-inspired JSON envelope.
type EventEnvelope struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	ClinicID      string          `json:"clinic_id,omitempty"`
	TraceID       string          `json:"trace_id"`
	Timestamp     string          `json:"timestamp"`
	Producer      string          `json:"producer"`
	Version       string          `json:"version"`
	Payload       json.RawMessage `json:"payload"`
}

// NewEventEnvelope wraps domain data into the standard event envelope.
func NewEventEnvelope(eventType, aggregateType, aggregateID, clinicID, traceID, producer string, payload any) (*EventEnvelope, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &EventEnvelope{
		EventID:       uuid.New().String(),
		EventType:     eventType,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		ClinicID:      clinicID,
		TraceID:       traceID,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Producer:      producer,
		Version:       "1.0",
		Payload:       bytes,
	}, nil
}
