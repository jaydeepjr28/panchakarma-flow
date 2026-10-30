-- Migration: 000002_seed_staff.down.sql
DELETE FROM users WHERE email IN (
  'admin.ayur@somatheeram.in',
  'ramesh.vaidya@somatheeram.in',
  'dr.anand@somatheeram.in'
);
