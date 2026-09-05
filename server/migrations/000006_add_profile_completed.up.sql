-- Sprint 4.2: Add profile_completed column to users table

ALTER TABLE users ADD COLUMN profile_completed BOOLEAN NOT NULL DEFAULT FALSE;

-- Existing users with valid student_no and name are marked as profile completed
UPDATE users SET profile_completed = TRUE WHERE student_no IS NOT NULL AND student_no != '';
