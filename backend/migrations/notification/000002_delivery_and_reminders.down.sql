-- Migration: 000002_delivery_and_reminders.down.sql for notification_db
DROP TABLE IF EXISTS scheduled_reminders;
DROP TABLE IF EXISTS notification_delivery_attempts;
DROP TABLE IF EXISTS consumed_events;
