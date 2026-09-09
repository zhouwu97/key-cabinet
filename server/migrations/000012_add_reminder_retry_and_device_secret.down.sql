ALTER TABLE reminders DROP COLUMN IF EXISTS attempt_count;
ALTER TABLE reminders DROP COLUMN IF EXISTS next_retry_at;
ALTER TABLE reminders DROP COLUMN IF EXISTS sent_at;

ALTER TABLE devices DROP COLUMN IF EXISTS device_secret;
