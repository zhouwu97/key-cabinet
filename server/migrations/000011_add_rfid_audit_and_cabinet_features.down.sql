DROP TABLE IF EXISTS reminders;
ALTER TABLE device_operations DROP COLUMN IF EXISTS scanned_rfid;
ALTER TABLE borrow_records DROP COLUMN IF EXISTS scanned_rfid;
