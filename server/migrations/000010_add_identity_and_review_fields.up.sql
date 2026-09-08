ALTER TABLE users
    ADD COLUMN IF NOT EXISTS identity_verified BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS identity_verified_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS identity_verified_by VARCHAR(64) REFERENCES users(id);

ALTER TABLE reservations
    ADD COLUMN IF NOT EXISTS reviewed_by VARCHAR(64) REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS rejection_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_users_identity_verified
    ON users(identity_verified);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_verified_student_no
    ON users(student_no)
    WHERE identity_verified = TRUE AND student_no IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_reservations_pending_review
    ON reservations(created_at)
    WHERE status = 'PENDING';
