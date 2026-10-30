-- Migration: 000002_seed_clinic.down.sql
DELETE FROM clinics WHERE id IN (
  'a3f8c872-9112-4c22-b5e1-7e8913c2f101',
  'a3f8c872-9112-4c22-b5e1-7e8913c2f102'
);
