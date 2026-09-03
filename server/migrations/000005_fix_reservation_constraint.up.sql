-- Fix reservation constraint to use new field names
-- Drop old constraint if exists
ALTER TABLE reservations DROP CONSTRAINT IF EXISTS reservations_no_overlap;

-- Add exclusion constraint with correct field names and statuses
-- PENDING: awaiting approval
-- APPROVED: approved but pickup window not started
-- ACTIVE: within pickup window
-- These three statuses should prevent time conflicts
ALTER TABLE reservations
ADD CONSTRAINT reservations_no_overlap
EXCLUDE USING gist (
    key_id WITH =,
    tstzrange(pickup_window_start, expected_return_at) WITH &&
)
WHERE (status IN ('PENDING', 'APPROVED', 'ACTIVE'));
