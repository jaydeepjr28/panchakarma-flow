-- Migration: 000002_analytics_and_adverse_events.down.sql for feedback_analytics_db

DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS consumed_events;
DROP TABLE IF EXISTS analytics_sessions;
DROP TABLE IF EXISTS analytics_appointments;
DROP TABLE IF EXISTS adverse_events;
ALTER TABLE patient_feedback DROP COLUMN IF EXISTS staff_notes;
ALTER TABLE patient_feedback DROP COLUMN IF EXISTS acknowledged_by;
ALTER TABLE patient_feedback DROP COLUMN IF EXISTS acknowledged_at;
ALTER TABLE patient_feedback DROP COLUMN IF EXISTS status;
