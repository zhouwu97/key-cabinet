-- 补齐提醒重试与投递状态跟踪字段，以及机柜设备安全通信密钥

ALTER TABLE reminders ADD COLUMN IF NOT EXISTS attempt_count INT NOT NULL DEFAULT 0;
ALTER TABLE reminders ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;
ALTER TABLE reminders ADD COLUMN IF NOT EXISTS sent_at TIMESTAMPTZ;

ALTER TABLE devices ADD COLUMN IF NOT EXISTS device_secret VARCHAR(128);

-- 为既有机柜初始化默认设备通信密钥 (若为空)
UPDATE devices SET device_secret = 'cab_sec_' || id WHERE device_secret IS NULL OR device_secret = '';
