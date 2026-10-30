-- Migration: 000001_init_schema.up.sql for clinic_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS clinics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    registration_number VARCHAR(100) NOT NULL UNIQUE,
    address_street VARCHAR(255) NOT NULL,
    address_city VARCHAR(100) NOT NULL,
    address_state VARCHAR(100) NOT NULL,
    address_postal_code VARCHAR(20) NOT NULL,
    contact_phone VARCHAR(32) NOT NULL,
    contact_email VARCHAR(255) NOT NULL,
    description TEXT NULL,
    supported_languages TEXT[] NOT NULL DEFAULT '{"English"}',
    operating_hours JSONB NOT NULL,
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Kolkata',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_clinics_city ON clinics(address_city);

CREATE TABLE IF NOT EXISTS treatment_rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    room_type VARCHAR(64) NOT NULL DEFAULT 'standard_panchakarma',
    capacity INT NOT NULL DEFAULT 1,
    sanitation_buffer_minutes INT NOT NULL DEFAULT 15,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(clinic_id, name)
);

CREATE INDEX IF NOT EXISTS idx_rooms_clinic ON treatment_rooms(clinic_id);

CREATE TABLE IF NOT EXISTS room_equipment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES treatment_rooms(id) ON DELETE CASCADE,
    equipment_type VARCHAR(64) NOT NULL,
    serial_number VARCHAR(100) NULL,
    is_operational BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS doctors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    clinic_id UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    qualifications VARCHAR(255) NOT NULL,
    specialties TEXT[] NOT NULL,
    experience_years INT NOT NULL DEFAULT 0,
    max_sessions_per_day INT NOT NULL DEFAULT 8 CHECK (max_sessions_per_day > 0 AND max_sessions_per_day <= 16),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_doctors_clinic ON doctors(clinic_id);

CREATE TABLE IF NOT EXISTS therapists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    clinic_id UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    specializations TEXT[] NOT NULL DEFAULT '{}',
    experience_years INT NOT NULL DEFAULT 0,
    gender VARCHAR(16) NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_therapists_clinic ON therapists(clinic_id);

CREATE TABLE IF NOT EXISTS doctor_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    doctor_id UUID NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    slot_duration_minutes INT NOT NULL DEFAULT 60,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    CHECK (end_time > start_time)
);

CREATE INDEX IF NOT EXISTS idx_schedules_doctor ON doctor_schedules(doctor_id, day_of_week);

CREATE TABLE IF NOT EXISTS doctor_leaves (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    doctor_id UUID NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    reason TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'APPROVED' CHECK (status IN ('PENDING', 'APPROVED', 'CANCELLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (end_time > start_time)
);

CREATE INDEX IF NOT EXISTS idx_leaves_doctor_range ON doctor_leaves(doctor_id, start_time, end_time);

CREATE TABLE IF NOT EXISTS therapy_packages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(64) NOT NULL,
    description TEXT NOT NULL,
    price_cents INT NOT NULL CHECK (price_cents >= 0),
    total_sessions INT NOT NULL DEFAULT 1 CHECK (total_sessions >= 1),
    duration_minutes INT NOT NULL CHECK (duration_minutes > 0),
    recommended_intervals_days INT NOT NULL DEFAULT 1,
    version INT NOT NULL DEFAULT 1,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_packages_clinic ON therapy_packages(clinic_id);

CREATE TABLE IF NOT EXISTS therapy_package_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    package_id UUID NOT NULL REFERENCES therapy_packages(id) ON DELETE CASCADE,
    step_order INT NOT NULL CHECK (step_order >= 1),
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    duration_minutes INT NOT NULL CHECK (duration_minutes > 0),
    instructions TEXT[] NOT NULL DEFAULT '{}',
    required_equipment TEXT[] NOT NULL DEFAULT '{}',
    UNIQUE(package_id, step_order)
);

CREATE TABLE IF NOT EXISTS care_checklists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    package_id UUID NOT NULL REFERENCES therapy_packages(id) ON DELETE CASCADE,
    checklist_type VARCHAR(32) NOT NULL CHECK (checklist_type IN ('PRE_PROCEDURE', 'POST_PROCEDURE')),
    version INT NOT NULL DEFAULT 1,
    items JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_checklists_package ON care_checklists(package_id, checklist_type);
