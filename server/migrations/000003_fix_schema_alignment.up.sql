-- Sprint 4.1.1: Contract Alignment - Schema Fixes
-- Fix field naming and add missing columns to align with frozen data model

-- ============================================================================
-- Users Table Fixes
-- ============================================================================

-- Rename student_id to student_no for clarity
ALTER TABLE users RENAME COLUMN student_id TO student_no;

-- Add credit_score (business core field)
ALTER TABLE users ADD COLUMN credit_score INTEGER NOT NULL DEFAULT 100;

-- Fix role default value (STUDENT -> USER to align with frontend)
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'USER';

-- Update all TIMESTAMP to TIMESTAMPTZ
ALTER TABLE users ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';
ALTER TABLE users ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- Add department field
ALTER TABLE users ADD COLUMN department VARCHAR(200);

-- Update index name
DROP INDEX IF EXISTS idx_users_student_id;
CREATE INDEX idx_users_student_no ON users(student_no);

-- ============================================================================
-- User Identities Table Fixes
-- ============================================================================

-- Rename identity_type to provider (clearer semantics)
ALTER TABLE user_identities RENAME COLUMN identity_type TO provider;

-- Rename provider_user_id to subject (OAuth2 standard terminology)
ALTER TABLE user_identities RENAME COLUMN provider_user_id TO subject;

-- Update TIMESTAMP to TIMESTAMPTZ
ALTER TABLE user_identities ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';

-- ============================================================================
-- Devices Table Fixes
-- ============================================================================

-- Add last_heartbeat_at for online status tracking
ALTER TABLE devices ADD COLUMN last_heartbeat_at TIMESTAMPTZ;

-- Update TIMESTAMP to TIMESTAMPTZ
ALTER TABLE devices ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';
ALTER TABLE devices ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- ============================================================================
-- Slots Table Fixes
-- ============================================================================

ALTER TABLE slots ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';
ALTER TABLE slots ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- ============================================================================
-- Keys Table Fixes
-- ============================================================================

ALTER TABLE keys ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';
ALTER TABLE keys ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- ============================================================================
-- Reservations Table Fixes (CRITICAL: Time semantics)
-- ============================================================================

-- Add pickup window and expected return time fields
ALTER TABLE reservations ADD COLUMN pickup_window_start TIMESTAMPTZ;
ALTER TABLE reservations ADD COLUMN pickup_window_end TIMESTAMPTZ;
ALTER TABLE reservations ADD COLUMN expected_return_at TIMESTAMPTZ;

-- Migrate existing data: start_time -> pickup_window_start
-- Assume 2-hour pickup window and end_time as expected return
UPDATE reservations SET
  pickup_window_start = start_time,
  pickup_window_end = start_time + INTERVAL '2 hours',
  expected_return_at = end_time
WHERE pickup_window_start IS NULL;

-- Make new fields NOT NULL after migration
ALTER TABLE reservations ALTER COLUMN pickup_window_start SET NOT NULL;
ALTER TABLE reservations ALTER COLUMN pickup_window_end SET NOT NULL;
ALTER TABLE reservations ALTER COLUMN expected_return_at SET NOT NULL;

-- Drop old ambiguous fields
ALTER TABLE reservations DROP COLUMN start_time;
ALTER TABLE reservations DROP COLUMN end_time;

-- Update TIMESTAMP to TIMESTAMPTZ
ALTER TABLE reservations ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';
ALTER TABLE reservations ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- ============================================================================
-- Borrow Records Table Fixes
-- ============================================================================

-- Make borrowed_at nullable (for BORROWING state)
ALTER TABLE borrow_records ALTER COLUMN borrowed_at DROP NOT NULL;

-- Update all TIMESTAMP to TIMESTAMPTZ
ALTER TABLE borrow_records ALTER COLUMN borrowed_at TYPE TIMESTAMPTZ USING borrowed_at AT TIME ZONE 'UTC';
ALTER TABLE borrow_records ALTER COLUMN expected_return_at TYPE TIMESTAMPTZ USING expected_return_at AT TIME ZONE 'UTC';
ALTER TABLE borrow_records ALTER COLUMN returned_at TYPE TIMESTAMPTZ USING returned_at AT TIME ZONE 'UTC';
ALTER TABLE borrow_records ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';
ALTER TABLE borrow_records ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- ============================================================================
-- Device Operations Table Fixes (CRITICAL: Idempotency)
-- ============================================================================

-- Add request_id for idempotency
ALTER TABLE device_operations ADD COLUMN request_id VARCHAR(64);

-- Add user_id for audit trail
ALTER TABLE device_operations ADD COLUMN user_id VARCHAR(64);

-- Add error_code for structured error handling
ALTER TABLE device_operations ADD COLUMN error_code VARCHAR(50);

-- Create unique index on request_id (after adding the column)
CREATE UNIQUE INDEX idx_device_operations_request_id ON device_operations(request_id) WHERE request_id IS NOT NULL;

-- Update TIMESTAMP to TIMESTAMPTZ
ALTER TABLE device_operations ALTER COLUMN initiated_at TYPE TIMESTAMPTZ USING initiated_at AT TIME ZONE 'UTC';
ALTER TABLE device_operations ALTER COLUMN completed_at TYPE TIMESTAMPTZ USING completed_at AT TIME ZONE 'UTC';
ALTER TABLE device_operations ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';
ALTER TABLE device_operations ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';

-- Add foreign key for user_id
ALTER TABLE device_operations ADD CONSTRAINT fk_device_operations_user FOREIGN KEY (user_id) REFERENCES users(id);
