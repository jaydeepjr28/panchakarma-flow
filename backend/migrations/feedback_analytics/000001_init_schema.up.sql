-- Migration: 000001_init_schema.up.sql for feedback_analytics_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS patient_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appointment_id UUID NOT NULL UNIQUE,
    session_id UUID NOT NULL UNIQUE,
    patient_id UUID NOT NULL,
    clinic_id UUID NOT NULL,
    doctor_id UUID NOT NULL,
    therapist_id UUID NOT NULL,
    overall_rating INT NOT NULL CHECK (overall_rating BETWEEN 1 AND 5),
    therapist_rating INT NOT NULL CHECK (therapist_rating BETWEEN 1 AND 5),
    facility_cleanliness_rating INT NOT NULL CHECK (facility_cleanliness_rating BETWEEN 1 AND 5),
    comments TEXT NULL,
    is_anonymous BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_feedback_clinic ON patient_feedback(clinic_id);
CREATE INDEX IF NOT EXISTS idx_feedback_doctor ON patient_feedback(doctor_id);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID DEFAULT gen_random_uuid(),
    event_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actor_id UUID NOT NULL,
    actor_role VARCHAR(32) NOT NULL,
    action VARCHAR(128) NOT NULL,
    target_resource_type VARCHAR(64) NOT NULL,
    target_resource_id UUID NOT NULL,
    clinic_id UUID NULL,
    ip_address INET NOT NULL,
    user_agent VARCHAR(512) NOT NULL,
    previous_state_hash VARCHAR(64) NULL,
    new_state_hash VARCHAR(64) NULL,
    purpose TEXT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY (id, event_time)
) PARTITION BY RANGE (event_time);

CREATE TABLE IF NOT EXISTS audit_logs_default PARTITION OF audit_logs DEFAULT;

CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor_id, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_audit_clinic ON audit_logs(clinic_id, event_time DESC);

CREATE TABLE IF NOT EXISTS operational_daily_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id UUID NOT NULL,
    metric_date DATE NOT NULL,
    total_appointments INT NOT NULL DEFAULT 0,
    completed_sessions INT NOT NULL DEFAULT 0,
    cancelled_count INT NOT NULL DEFAULT 0,
    no_show_count INT NOT NULL DEFAULT 0,
    room_utilization_pct NUMERIC(5,2) NOT NULL DEFAULT 0.00,
    doctor_utilization_pct NUMERIC(5,2) NOT NULL DEFAULT 0.00,
    average_overall_rating NUMERIC(3,2) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(clinic_id, metric_date)
);

CREATE INDEX IF NOT EXISTS idx_metrics_clinic_date ON operational_daily_metrics(clinic_id, metric_date DESC);
