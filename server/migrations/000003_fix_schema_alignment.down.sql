-- Rollback Sprint 4.1.1 schema alignment changes

-- Device Operations
ALTER TABLE device_operations DROP CONSTRAINT IF EXISTS fk_device_operations_user;
ALTER TABLE device_operations ALTER COLUMN updated_at TYPE TIMESTAMP USING updated_at AT TIME ZONE 'UTC';
ALTER TABLE device_operations ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC';
ALTER TABLE device_operations ALTER COLUMN completed_at TYPE TIMESTAMP USING completed_at AT TIME ZONE 'UTC';
ALTER TABLE device_operations ALTER COLUMN initiated_at TYPE TIMESTAMP USING initiated_at AT TIME ZONE 'UTC';
DROP INDEX IF EXISTS idx_device_operations_request_id;
ALTER TABLE device_operations DROP COLUMN IF EXISTS error_code;
ALTER TABLE device_operations DROP COLUMN IF EXISTS user_id;
ALTER TABLE device_operations DROP COLUMN IF EXISTS request_id;

-- Borrow Records
ALTER TABLE borrow_records ALTER COLUMN updated_at TYPE TIMESTAMP USING updated_at AT TIME ZONE 'UTC';
ALTER TABLE borrow_records ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC';
ALTER TABLE borrow_records ALTER COLUMN returned_at TYPE TIMESTAMP USING returned_at AT TIME ZONE 'UTC';
ALTER TABLE borrow_records ALTER COLUMN expected_return_at TYPE TIMESTAMP USING expected_return_at AT TIME ZONE 'UTC';
ALTER TABLE borrow_records ALTER COLUMN borrowed_at TYPE TIMESTAMP USING borrowed_at AT TIME ZONE 'UTC';
ALTER TABLE borrow_records ALTER COLUMN borrowed_at SET NOT NULL;

-- Reservations
ALTER TABLE reservations ALTER COLUMN updated_at TYPE TIMESTAMP USING updated_at AT TIME ZONE 'UTC';
ALTER TABLE reservations ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC';
ALTER TABLE reservations ADD COLUMN end_time TIMESTAMP;
ALTER TABLE reservations ADD COLUMN start_time TIMESTAMP;
UPDATE reservations SET
  start_time = pickup_window_start,
  end_time = expected_return_at;
ALTER TABLE reservations ALTER COLUMN start_time SET NOT NULL;
ALTER TABLE reservations ALTER COLUMN end_time SET NOT NULL;
ALTER TABLE reservations DROP COLUMN expected_return_at;
ALTER TABLE reservations DROP COLUMN pickup_window_end;
ALTER TABLE reservations DROP COLUMN pickup_window_start;

-- Keys
ALTER TABLE keys ALTER COLUMN updated_at TYPE TIMESTAMP USING updated_at AT TIME ZONE 'UTC';
ALTER TABLE keys ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC';

-- Slots
ALTER TABLE slots ALTER COLUMN updated_at TYPE TIMESTAMP USING updated_at AT TIME ZONE 'UTC';
ALTER TABLE slots ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC';

-- Devices
ALTER TABLE devices ALTER COLUMN updated_at TYPE TIMESTAMP USING updated_at AT TIME ZONE 'UTC';
ALTER TABLE devices ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC';
ALTER TABLE devices DROP COLUMN IF EXISTS last_heartbeat_at;

-- User Identities
ALTER TABLE user_identities ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC';
ALTER TABLE user_identities RENAME COLUMN subject TO provider_user_id;
ALTER TABLE user_identities RENAME COLUMN provider TO identity_type;

-- Users
DROP INDEX IF EXISTS idx_users_student_no;
CREATE INDEX idx_users_student_id ON users(student_id);
ALTER TABLE users DROP COLUMN IF EXISTS department;
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'STUDENT';
ALTER TABLE users ALTER COLUMN updated_at TYPE TIMESTAMP USING updated_at AT TIME ZONE 'UTC';
ALTER TABLE users ALTER COLUMN created_at TYPE TIMESTAMP USING created_at AT TIME ZONE 'UTC';
ALTER TABLE users DROP COLUMN IF EXISTS credit_score;
ALTER TABLE users RENAME COLUMN student_no TO student_id;
