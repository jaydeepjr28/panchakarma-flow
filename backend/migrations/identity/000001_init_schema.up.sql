-- Migration: 000001_init_schema.up.sql for identity_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

DO $$ BEGIN
    CREATE TYPE user_role AS ENUM (
        'patient',
        'doctor',
        'therapist',
        'clinic_admin',
        'super_admin'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(32) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role user_role NOT NULL,
    clinic_id UUID NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role_clinic ON users(role, clinic_id) WHERE clinic_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS refresh_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    family_id UUID NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    user_agent VARCHAR(512) NULL,
    ip_address INET NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_refresh_user ON refresh_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_family ON refresh_sessions(family_id);

CREATE TABLE IF NOT EXISTS patient_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    date_of_birth DATE NOT NULL,
    biological_sex VARCHAR(16) NOT NULL CHECK (biological_sex IN ('male', 'female', 'other')),
    blood_group VARCHAR(8) NULL,
    emergency_contact_name VARCHAR(255) NOT NULL,
    emergency_contact_phone VARCHAR(32) NOT NULL,
    address_street VARCHAR(255) NULL,
    address_city VARCHAR(100) NULL,
    address_state VARCHAR(100) NULL,
    address_postal_code VARCHAR(20) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS health_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    allergies JSONB NOT NULL DEFAULT '[]'::jsonb,
    chronic_conditions JSONB NOT NULL DEFAULT '[]'::jsonb,
    current_medications JSONB NOT NULL DEFAULT '[]'::jsonb,
    dietary_habits VARCHAR(255) NULL,
    prakriti_baseline VARCHAR(64) NULL,
    encrypted_medical_history TEXT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS consent_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL REFERENCES users(id),
    package_id UUID NOT NULL,
    consent_version VARCHAR(32) NOT NULL,
    consent_text TEXT NOT NULL,
    acknowledged_risks JSONB NOT NULL,
    signature_vector TEXT NOT NULL,
    signature_hash VARCHAR(64) NOT NULL,
    ip_address INET NOT NULL,
    user_agent VARCHAR(512) NOT NULL,
    signed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_consent_patient ON consent_records(patient_id);
CREATE INDEX IF NOT EXISTS idx_consent_package ON consent_records(package_id);
