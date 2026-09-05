DROP INDEX IF EXISTS idx_device_operations_active_key;
DROP INDEX IF EXISTS idx_device_operations_active_device;
DROP INDEX IF EXISTS idx_borrow_records_reservation_id;

ALTER TABLE device_operations DROP COLUMN IF EXISTS ack_at;
ALTER TABLE device_operations DROP COLUMN IF EXISTS sent_at;

ALTER TABLE borrow_records DROP COLUMN IF EXISTS overdue_at;
ALTER TABLE borrow_records DROP COLUMN IF EXISTS purpose;
ALTER TABLE borrow_records DROP COLUMN IF EXISTS slot_id;
ALTER TABLE borrow_records DROP COLUMN IF EXISTS device_id;

ALTER TABLE reservations DROP COLUMN IF EXISTS cancelled_at;
ALTER TABLE reservations DROP COLUMN IF EXISTS used_at;
ALTER TABLE reservations DROP COLUMN IF EXISTS approved_at;
