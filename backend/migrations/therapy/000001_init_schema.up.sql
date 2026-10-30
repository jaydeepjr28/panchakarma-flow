-- Migration: 000001_init_schema.up.sql for therapy_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DO $$ BEGIN
    CREATE TYPE session_status AS ENUM (
        'CHECKED_IN',
        'IN_PROGRESS',
        'COMPLICATION_ALERT',
        'COMPLETED',
        'ABORTED_CLINICAL'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE step_status AS ENUM (
        'NOT_STARTED',
        'IN_PROGRESS',
        'PAUSED',
        'COMPLETED',
        'SKIPPED'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS therapy_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appointment_id UUID NOT NULL UNIQUE,
    patient_id UUID NOT NULL,
    clinic_id UUID NOT NULL,
    doctor_id UUID NOT NULL,
    therapist_id UUID NOT NULL,
    room_id UUID NOT NULL,
    package_id UUID NOT NULL,
    package_version INT NOT NULL DEFAULT 1,
    status session_status NOT NULL DEFAULT 'CHECKED_IN',
    baseline_vitals JSONB NOT NULL,
    pre_checklist_verified BOOLEAN NOT NULL DEFAULT FALSE,
    pre_checklist_data JSONB NULL,
    therapist_notes TEXT NULL,
    observations TEXT NULL,
    post_care_instructions TEXT NULL,
    post_care_items JSONB NULL,
    post_care_dispatched BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_therapist ON therapy_sessions(therapist_id, status);
CREATE INDEX IF NOT EXISTS idx_sessions_patient ON therapy_sessions(patient_id);
CREATE INDEX IF NOT EXISTS idx_sessions_clinic ON therapy_sessions(clinic_id);

CREATE TABLE IF NOT EXISTS session_step_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES therapy_sessions(id) ON DELETE CASCADE,
    step_id UUID NOT NULL,
    step_order INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    planned_duration_minutes INT NOT NULL,
    actual_duration_seconds INT NOT NULL DEFAULT 0,
    status step_status NOT NULL DEFAULT 'NOT_STARTED',
    started_at TIMESTAMPTZ NULL,
    paused_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,
    therapist_notes TEXT NULL,
    observations TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(session_id, step_order)
);

CREATE INDEX IF NOT EXISTS idx_step_exec_session ON session_step_executions(session_id);

CREATE TABLE IF NOT EXISTS session_materials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES therapy_sessions(id) ON DELETE CASCADE,
    material_name VARCHAR(255) NOT NULL,
    quantity NUMERIC(10, 2) NOT NULL,
    unit VARCHAR(32) NOT NULL,
    batch_number VARCHAR(64) NULL,
    notes TEXT NULL,
    recorded_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_materials_session ON session_materials(session_id);

CREATE TABLE IF NOT EXISTS complication_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES therapy_sessions(id) ON DELETE CASCADE,
    reporter_id UUID NOT NULL,
    reporter_role VARCHAR(32) NOT NULL,
    severity VARCHAR(32) NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    symptoms TEXT NOT NULL,
    vitals_at_incident JSONB NOT NULL,
    immediate_actions_taken TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'UNRESOLVED' CHECK (status IN ('UNRESOLVED', 'DOCTOR_REVIEWED', 'RESOLVED')),
    attending_doctor_id UUID NOT NULL,
    doctor_notes TEXT NULL,
    doctor_cleared_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_complications_session ON complication_events(session_id);
CREATE INDEX IF NOT EXISTS idx_complications_doctor ON complication_events(attending_doctor_id, status);

CREATE TABLE IF NOT EXISTS session_follow_ups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES therapy_sessions(id) ON DELETE CASCADE,
    recommended_by UUID NOT NULL,
    recommended_package_id UUID NULL,
    recommended_interval_days INT NULL,
    recommended_date DATE NULL,
    instructions TEXT NOT NULL,
    linked_appointment_id UUID NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'SCHEDULED', 'DISMISSED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_follow_ups_session ON session_follow_ups(session_id);

CREATE TABLE IF NOT EXISTS session_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES therapy_sessions(id) ON DELETE CASCADE,
    field_name VARCHAR(64) NOT NULL,
    old_value TEXT NULL,
    new_value TEXT NOT NULL,
    reason TEXT NOT NULL,
    corrected_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_session_audit ON session_audit_logs(session_id);

CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(64) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    payload JSONB NOT NULL,
    trace_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PROCESSED', 'FAILED')),
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_therapy_outbox ON outbox_events(status, created_at) WHERE status = 'PENDING';
