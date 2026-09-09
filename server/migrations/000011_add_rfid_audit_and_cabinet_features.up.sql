-- 补齐 RFID 实际读卡审计字段与逾期/临期提醒落库表

ALTER TABLE borrow_records ADD COLUMN IF NOT EXISTS scanned_rfid VARCHAR(64);
ALTER TABLE device_operations ADD COLUMN IF NOT EXISTS scanned_rfid VARCHAR(64);

CREATE TABLE IF NOT EXISTS reminders (
    id VARCHAR(64) PRIMARY KEY,
    borrow_record_id VARCHAR(64) NOT NULL REFERENCES borrow_records(id),
    user_id VARCHAR(64) NOT NULL REFERENCES users(id),
    type VARCHAR(32) NOT NULL, -- 'APPROACHING_OVERDUE', 'OVERDUE'
    status VARCHAR(32) NOT NULL DEFAULT 'SENT', -- 'PENDING', 'SENT', 'FAILED'
    attempt_count INT NOT NULL DEFAULT 1,
    next_retry_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    channel VARCHAR(32) NOT NULL DEFAULT 'WECHAT_SUBSCRIBE',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_reminders_type_record
    ON reminders(borrow_record_id, type);

CREATE INDEX IF NOT EXISTS idx_reminders_user_id
    ON reminders(user_id);
