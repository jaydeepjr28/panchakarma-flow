-- Migration: 000002_seed_staff.up.sql for identity_db
-- Seeds authorized Clinic Admin and Therapist staff accounts into PostgreSQL

INSERT INTO users (id, email, phone, password_hash, role, clinic_id, is_active, created_at, updated_at)
VALUES 
  (
    'a1111111-1111-1111-1111-111111111101',
    'admin.ayur@somatheeram.in',
    '+919447012345',
    crypt('AdminPass123!', gen_salt('bf', 12)),
    'clinic_admin',
    'a3f8c872-9112-4c22-b5e1-7e8913c2f101',
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
  ),
  (
    'a1111111-1111-1111-1111-111111111102',
    'ramesh.vaidya@somatheeram.in',
    '+919447054321',
    crypt('TherapistPass123!', gen_salt('bf', 12)),
    'therapist',
    'a3f8c872-9112-4c22-b5e1-7e8913c2f101',
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
  ),
  (
    'a1111111-1111-1111-1111-111111111103',
    'dr.anand@somatheeram.in',
    '+919447099887',
    crypt('DoctorPass123!', gen_salt('bf', 12)),
    'doctor',
    'a3f8c872-9112-4c22-b5e1-7e8913c2f101',
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
  )
ON CONFLICT (email) DO UPDATE SET 
  password_hash = EXCLUDED.password_hash,
  role = EXCLUDED.role,
  clinic_id = EXCLUDED.clinic_id,
  is_active = true,
  updated_at = CURRENT_TIMESTAMP;
