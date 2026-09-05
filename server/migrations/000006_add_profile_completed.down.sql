-- Sprint 4.2: Remove profile_completed column from users table

ALTER TABLE users DROP COLUMN IF EXISTS profile_completed;
