-- Migration: 000002_analytics_and_adverse_events.up.sql for feedback_analytics_db

-- 1. Enhance patient_feedback table
ALTER TABLE patient_feedback ALTER COLUMN session_id DROP NOT NULL;
ALTER TABLE patient_feedback ADD COLUMN IF NOT EXISTS status VARCHAR(32) NOT NULL DEFAULT 'SUBMITTED';
ALTER TABLE patient_feedback ADD COLUMN IF NOT EXISTS acknowledged_at TIMESTAMPTZ NULL;
ALTER TABLE patient_feedback ADD COLUMN IF NOT EXISTS acknowledged_by UUID NULL;
ALTER TABLE patient_feedback ADD COLUMN IF NOT EXISTS staff_notes TEXT NULL;

-- 2. Create adverse_events table (Complications and Clinical Adverse Incidents)
CREATE TABLE IF NOT EXISTS adverse_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NULL,
    appointment_id UUID NULL,
    patient_id UUID NOT NULL,
    clinic_id UUID NOT NULL,
    severity VARCHAR(32) NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    symptoms TEXT NOT NULL,
    vitals_at_incident JSONB NULL DEFAULT '{}'::jsonb,
    immediate_actions_taken TEXT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'REPORTED' CHECK (status IN ('REPORTED', 'UNDER_REVIEW', 'ACKNOWLEDGED', 'RESOLVED')),
    reported_by UUID NOT NULL,
    reporter_role VARCHAR(32) NOT NULL,
    attending_doctor_id UUID NULL,
    acknowledged_at TIMESTAMPTZ NULL,
    acknowledged_by UUID NULL,
    staff_notes TEXT NULL,
    resolved_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_adverse_clinic_created ON adverse_events(clinic_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_adverse_severity ON adverse_events(severity);
CREATE INDEX IF NOT EXISTS idx_adverse_status ON adverse_events(status);
CREATE INDEX IF NOT EXISTS idx_adverse_patient ON adverse_events(patient_id);

-- 3. Local Operational Projection: analytics_appointments
CREATE TABLE IF NOT EXISTS analytics_appointments (
    id UUID PRIMARY KEY,
    clinic_id UUID NOT NULL,
    patient_id UUID NOT NULL,
    doctor_id UUID NULL,
    room_id UUID NULL,
    package_id UUID NULL,
    package_name VARCHAR(128) NULL,
    status VARCHAR(32) NOT NULL,
    scheduled_start TIMESTAMPTZ NOT NULL,
    scheduled_end TIMESTAMPTZ NOT NULL,
    is_follow_up BOOLEAN NOT NULL DEFAULT FALSE,
    parent_appointment_id UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_analytics_apt_clinic_sched ON analytics_appointments(clinic_id, scheduled_start);
CREATE INDEX IF NOT EXISTS idx_analytics_apt_status ON analytics_appointments(status);
CREATE INDEX IF NOT EXISTS idx_analytics_apt_patient ON analytics_appointments(patient_id);

-- 4. Local Operational Projection: analytics_sessions
CREATE TABLE IF NOT EXISTS analytics_sessions (
    id UUID PRIMARY KEY,
    appointment_id UUID NOT NULL,
    clinic_id UUID NOT NULL,
    patient_id UUID NOT NULL,
    therapist_id UUID NULL,
    status VARCHAR(32) NOT NULL,
    all_steps_completed BOOLEAN NOT NULL DEFAULT TRUE,
    total_steps INT NOT NULL DEFAULT 0,
    completed_steps INT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,
    duration_minutes NUMERIC(8,2) NULL,
    follow_up_recommended BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_analytics_sess_clinic_started ON analytics_sessions(clinic_id, started_at);
CREATE INDEX IF NOT EXISTS idx_analytics_sess_therapist ON analytics_sessions(therapist_id);
CREATE INDEX IF NOT EXISTS idx_analytics_sess_status ON analytics_sessions(status);

-- 5. Kafka Deduplication Ledger: consumed_events
CREATE TABLE IF NOT EXISTS consumed_events (
    event_id VARCHAR(128) PRIMARY KEY,
    event_type VARCHAR(128) NOT NULL,
    consumer_group VARCHAR(128) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payload_digest VARCHAR(64) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_consumed_events_processed ON consumed_events(processed_at);

-- 6. Transactional Outbox: outbox_events
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(64) NOT NULL,
    aggregate_id VARCHAR(128) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    payload JSONB NOT NULL,
    trace_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_fb_outbox_pending ON outbox_events(created_at) WHERE status = 'PENDING';
