-- Migration: 000002_seed_clinic.up.sql for clinic_db
-- Seeds baseline clinics, treatment rooms, packages, checklists, doctors, and therapists

-- 1. Insert Clinics
INSERT INTO clinics (id, name, registration_number, address_street, address_city, address_state, address_postal_code, contact_phone, contact_email, description, supported_languages, operating_hours)
VALUES 
  ('a3f8c872-9112-4c22-b5e1-7e8913c2f101', 'Ayur Wellness Center', 'AYUR-REG-2024-001', '123 Wellness Street', 'Health City', 'Kerala', '682001', '+919876543210', 'info@ayurwellness.com', 'Premier Panchakarma therapy center with traditional authentic methods.', '{"English", "Hindi", "Malayalam"}', '{"monday": [{"open": "08:00", "close": "18:00"}], "tuesday": [{"open": "08:00", "close": "18:00"}], "wednesday": [{"open": "08:00", "close": "18:00"}], "thursday": [{"open": "08:00", "close": "18:00"}], "friday": [{"open": "08:00", "close": "18:00"}], "saturday": [{"open": "08:00", "close": "14:00"}]}'),
  ('a3f8c872-9112-4c22-b5e1-7e8913c2f102', 'Traditional Ayurveda Clinic', 'AYUR-REG-2024-002', '456 Herbal Avenue', 'Wellness Town', 'Karnataka', '560001', '+919876543211', 'contact@traditionalayur.com', 'Authentic Ayurvedic treatments following classical Vedic texts.', '{"English", "Kannada", "Tamil"}', '{"monday": [{"open": "09:00", "close": "17:00"}], "tuesday": [{"open": "09:00", "close": "17:00"}], "wednesday": [{"open": "09:00", "close": "17:00"}], "thursday": [{"open": "09:00", "close": "17:00"}], "friday": [{"open": "09:00", "close": "17:00"}]}')
ON CONFLICT (registration_number) DO NOTHING;

-- 2. Insert Treatment Rooms
INSERT INTO treatment_rooms (id, clinic_id, name, room_type, sanitation_buffer_minutes)
VALUES
  ('b1b2c3d4-e89b-12d3-a456-426614174001', 'a3f8c872-9112-4c22-b5e1-7e8913c2f101', 'Kaksha 101 - Dhanvantari', 'panchakarma_full_suite', 15),
  ('b1b2c3d4-e89b-12d3-a456-426614174002', 'a3f8c872-9112-4c22-b5e1-7e8913c2f101', 'Kaksha 102 - Ashwini', 'steam_and_massage', 15),
  ('b1b2c3d4-e89b-12d3-a456-426614174003', 'a3f8c872-9112-4c22-b5e1-7e8913c2f101', 'Kaksha 103 - Shirodhara Suite', 'shirodhara_quiet_room', 20)
ON CONFLICT (clinic_id, name) DO NOTHING;

-- 3. Insert Therapy Packages
INSERT INTO therapy_packages (id, clinic_id, name, category, description, price_cents, total_sessions, duration_minutes, version, is_published)
VALUES
  ('c1b2c3d4-e89b-12d3-a456-426614174001', 'a3f8c872-9112-4c22-b5e1-7e8913c2f101', 'Complete Panchakarma Detox', 'detox', 'A comprehensive multi-procedure program including oleation, sudation, and controlled purgation for full cellular detox.', 250000, 7, 95, 1, true),
  ('c1b2c3d4-e89b-12d3-a456-426614174002', 'a3f8c872-9112-4c22-b5e1-7e8913c2f101', 'Rejuvenation & Stress Relief Shirodhara', 'rejuvenation', 'A soothing therapy focused on mental relaxation, pouring medicated oil in continuous stream over the third eye.', 120000, 3, 90, 1, true)
ON CONFLICT DO NOTHING;

-- 4. Insert Package Steps
INSERT INTO therapy_package_steps (package_id, step_order, name, description, duration_minutes, instructions, required_equipment)
VALUES
  ('c1b2c3d4-e89b-12d3-a456-426614174001', 1, 'Abhyanga (Warm Oil Massage)', 'Full body synchronized oleation therapy', 45, '{"Ensure oil is heated to 38C", "Long rhythmic strokes in direction of hair follicles"}', '{"teak_droni_bed"}'),
  ('c1b2c3d4-e89b-12d3-a456-426614174001', 2, 'Svedana (Herbal Steam Therapy)', 'Medicated steam sudation for channel opening', 20, '{"Monitor forehead perspiration", "Place cold compress over head and heart"}', '{"swedana_steam_chamber"}'),
  ('c1b2c3d4-e89b-12d3-a456-426614174001', 3, 'Post-Steam Rest & Wipe', 'Dry sponge wipe and 15-minute vitals recovery', 15, '{"Offer warm herbal water", "Measure resting pulse"}', '{}')
ON CONFLICT (package_id, step_order) DO NOTHING;

-- 5. Insert Pre & Post Care Checklists
INSERT INTO care_checklists (package_id, checklist_type, version, items)
VALUES
  ('c1b2c3d4-e89b-12d3-a456-426614174001', 'PRE_PROCEDURE', 1, '[
    {"id": "pre_1", "text": "Fasted for at least 2 hours prior to arrival", "is_mandatory": true},
    {"id": "pre_2", "text": "Systolic blood pressure between 100 and 140 mmHg", "is_mandatory": true},
    {"id": "pre_3", "text": "No acute fever or systemic infection", "is_mandatory": true},
    {"id": "pre_4", "text": "Signed informed consent verified on file", "is_mandatory": true}
  ]'),
  ('c1b2c3d4-e89b-12d3-a456-426614174001', 'POST_PROCEDURE', 1, '[
    {"id": "post_1", "text": "Drink only warm water or boiled herbal decoction for 24 hours", "is_mandatory": true},
    {"id": "post_2", "text": "Avoid cold breeze, air conditioning, and direct hot sunlight", "is_mandatory": true},
    {"id": "post_3", "text": "Consume light kitchari diet; no heavy or fermented food", "is_mandatory": true},
    {"id": "post_4", "text": "Rest quietly; refrain from strenuous physical exercise", "is_mandatory": true}
  ]')
ON CONFLICT DO NOTHING;

-- 6. Insert Staff Records (Doctor and Therapist)
INSERT INTO doctors (id, user_id, clinic_id, qualifications, specialties, experience_years, max_sessions_per_day, is_active)
VALUES (
  'd1111111-1111-1111-1111-111111111101',
  'a1111111-1111-1111-1111-111111111103',
  'a3f8c872-9112-4c22-b5e1-7e8913c2f101',
  'BAMS, MD (Kayachikitsa), PhD (Panchakarma)',
  '{"Vamana", "Virechana", "Basti", "Nasya"}',
  15,
  8,
  true
)
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO therapists (id, user_id, clinic_id, specializations, experience_years, gender, is_active)
VALUES (
  'e1111111-1111-1111-1111-111111111101',
  'a1111111-1111-1111-1111-111111111102',
  'a3f8c872-9112-4c22-b5e1-7e8913c2f101',
  '{"Abhyanga", "Svedana", "Shirodhara", "Pinda Sveda"}',
  10,
  'male',
  true
)
ON CONFLICT (user_id) DO NOTHING;

-- 7. Insert Doctor Schedules (Sunday through Saturday 08:00 - 18:00)
INSERT INTO doctor_schedules (doctor_id, day_of_week, start_time, end_time, slot_duration_minutes, is_active)
VALUES
  ('d1111111-1111-1111-1111-111111111101', 0, '08:00', '18:00', 60, true),
  ('d1111111-1111-1111-1111-111111111101', 1, '08:00', '18:00', 60, true),
  ('d1111111-1111-1111-1111-111111111101', 2, '08:00', '18:00', 60, true),
  ('d1111111-1111-1111-1111-111111111101', 3, '08:00', '18:00', 60, true),
  ('d1111111-1111-1111-1111-111111111101', 4, '08:00', '18:00', 60, true),
  ('d1111111-1111-1111-1111-111111111101', 5, '08:00', '18:00', 60, true),
  ('d1111111-1111-1111-1111-111111111101', 6, '08:00', '18:00', 60, true);
