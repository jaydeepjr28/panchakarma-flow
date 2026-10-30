package kafka_test

import (
	"encoding/json"
	"testing"

	"github.com/jaydeepjr28/panchakarma-flow/pkg/kafka"
)

func TestEventEnvelope(t *testing.T) {
	type SamplePayload struct {
		AppointmentID string `json:"appointment_id"`
		Status        string `json:"status"`
	}

	payload := SamplePayload{
		AppointmentID: "apt-12345",
		Status:        "CONFIRMED",
	}

	env, err := kafka.NewEventEnvelope(
		"panchakarma.appointment.confirmed.v1",
		"APPOINTMENT",
		"apt-12345",
		"clinic-99",
		"trace-abc",
		"appointment-service",
		payload,
	)
	if err != nil {
		t.Fatalf("failed to create event envelope: %v", err)
	}

	if env.EventID == "" {
		t.Errorf("expected generated event_id")
	}
	if env.EventType != "panchakarma.appointment.confirmed.v1" {
		t.Errorf("expected event_type 'panchakarma.appointment.confirmed.v1', got %s", env.EventType)
	}

	var decoded SamplePayload
	if err := json.Unmarshal(env.Payload, &decoded); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if decoded.AppointmentID != "apt-12345" {
		t.Errorf("expected appointment_id 'apt-12345', got %s", decoded.AppointmentID)
	}
}
