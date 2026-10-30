-- Migration: 000001_init_schema.up.sql for appointment_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

DO $$ BEGIN
    CREATE TYPE appointment_status AS ENUM (
        'HELD',
        'PENDING_APPROVAL',
        'CONFIRMED',
        'CHECKED_IN',
        'IN_PROGRESS',
        'COMPLETED',
        'CANCELLED',
        'RESCHEDULED',
        'NO_SHOW',
        'REJECTED'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL,
    clinic_id UUID NOT NULL,
    doctor_id UUID NOT NULL,
    room_id UUID NOT NULL,
    package_id UUID NOT NULL,
    slot_range TSTZRANGE NOT NULL,
    status appointment_status NOT NULL DEFAULT 'CONFIRMED',
    session_number INT NOT NULL DEFAULT 1,
    total_sessions INT NOT NULL DEFAULT 1,
    package_booking_id UUID NULL,
    idempotency_key VARCHAR(128) NULL UNIQUE,
    notes TEXT NULL,
    cancellation_reason TEXT NULL,
    approved_by UUID NULL,
    approved_at TIMESTAMPTZ NULL,
    rejection_reason TEXT NULL,
    rescheduled_to_id UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT no_overlapping_doctor_appointments EXCLUDE USING gist (
        doctor_id WITH =,
        slot_range WITH &&
    ) WHERE (status IN ('CONFIRMED', 'CHECKED_IN', 'IN_PROGRESS')),

    CONSTRAINT no_overlapping_room_appointments EXCLUDE USING gist (
        room_id WITH =,
        slot_range WITH &&
    ) WHERE (status IN ('CONFIRMED', 'CHECKED_IN', 'IN_PROGRESS'))
);

CREATE INDEX IF NOT EXISTS idx_appointments_patient ON appointments(patient_id);
CREATE INDEX IF NOT EXISTS idx_appointments_clinic_date ON appointments(clinic_id, status);
CREATE INDEX IF NOT EXISTS idx_appointments_doctor ON appointments(doctor_id, status);
CREATE INDEX IF NOT EXISTS idx_appointments_package_booking ON appointments(package_booking_id) WHERE package_booking_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS appointment_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appointment_id UUID NOT NULL REFERENCES appointments(id) ON DELETE CASCADE,
    from_status appointment_status NULL,
    to_status appointment_status NOT NULL,
    changed_by UUID NULL,
    reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_appointment_history_appt ON appointment_history(appointment_id);

CREATE TABLE IF NOT EXISTS appointment_holds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hold_token VARCHAR(64) NOT NULL UNIQUE,
    patient_id UUID NOT NULL,
    clinic_id UUID NOT NULL,
    doctor_id UUID NOT NULL,
    room_id UUID NOT NULL,
    package_id UUID NOT NULL,
    slot_range TSTZRANGE NOT NULL,
    session_number INT NOT NULL DEFAULT 1,
    total_sessions INT NOT NULL DEFAULT 1,
    package_booking_id UUID NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'CONFIRMED', 'EXPIRED', 'RELEASED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_holds_expiry ON appointment_holds(expires_at) WHERE status = 'ACTIVE';
CREATE INDEX IF NOT EXISTS idx_holds_doctor_range ON appointment_holds(doctor_id) WHERE status = 'ACTIVE';

CREATE TABLE IF NOT EXISTS appointment_waitlists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id UUID NOT NULL,
    patient_id UUID NOT NULL,
    package_id UUID NOT NULL,
    preferred_doctor_id UUID NULL,
    preferred_date_start DATE NOT NULL,
    preferred_date_end DATE NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'WAITING' CHECK (status IN ('WAITING', 'OFFERED', 'EXPIRED', 'FULFILLED', 'CANCELLED')),
    offered_slot_start TIMESTAMPTZ NULL,
    offered_slot_end TIMESTAMPTZ NULL,
    offer_expires_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_waitlists_clinic_status ON appointment_waitlists(clinic_id, status);
CREATE INDEX IF NOT EXISTS idx_waitlists_patient ON appointment_waitlists(patient_id);

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

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox_events(status, created_at) WHERE status = 'PENDING';
