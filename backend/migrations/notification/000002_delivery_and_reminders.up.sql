-- Migration: 000002_delivery_and_reminders.up.sql for notification_db

-- Idempotent consumer tracking
CREATE TABLE IF NOT EXISTS consumed_events (
    event_id UUID NOT NULL,
    consumer_group VARCHAR(64) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (event_id, consumer_group)
);

CREATE INDEX IF NOT EXISTS idx_consumed_events_time ON consumed_events(processed_at);

-- Granular delivery attempts audit trail
CREATE TABLE IF NOT EXISTS notification_delivery_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    channel VARCHAR(32) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'DELIVERED', 'FAILED', 'CANCELLED', 'SKIPPED')),
    provider VARCHAR(64) NOT NULL,
    attempt_number INT NOT NULL DEFAULT 1,
    error_message TEXT NULL,
    dispatched_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_delivery_attempts_notif ON notification_delivery_attempts(notification_id);
CREATE INDEX IF NOT EXISTS idx_delivery_attempts_status ON notification_delivery_attempts(status, created_at DESC);

-- Scheduled reminders engine
CREATE TABLE IF NOT EXISTS scheduled_reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appointment_id UUID NOT NULL,
    patient_id UUID NOT NULL,
    clinic_id UUID NOT NULL,
    reminder_type VARCHAR(64) NOT NULL,
    scheduled_time TIMESTAMPTZ NOT NULL,
    target_event_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'SCHEDULED' CHECK (status IN ('SCHEDULED', 'DISPATCHED', 'CANCELLED', 'SKIPPED')),
    idempotency_key VARCHAR(128) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scheduled_reminders_due ON scheduled_reminders(status, scheduled_time);
CREATE INDEX IF NOT EXISTS idx_scheduled_reminders_appt ON scheduled_reminders(appointment_id);
