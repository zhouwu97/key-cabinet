DROP INDEX IF EXISTS idx_reservations_pending_review;
DROP INDEX IF EXISTS idx_users_identity_verified;
DROP INDEX IF EXISTS idx_users_verified_student_no;

ALTER TABLE reservations
    DROP COLUMN IF EXISTS rejection_reason,
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS reviewed_by;

ALTER TABLE users
    DROP COLUMN IF EXISTS identity_verified_by,
    DROP COLUMN IF EXISTS identity_verified_at,
    DROP COLUMN IF EXISTS identity_verified;
