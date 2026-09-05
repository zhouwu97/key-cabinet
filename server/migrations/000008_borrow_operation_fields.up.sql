-- Sprint 4.4-4.6：补齐借还记录与设备操作的审计和状态字段。

ALTER TABLE reservations ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ;
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS used_at TIMESTAMPTZ;
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ;

ALTER TABLE borrow_records ADD COLUMN IF NOT EXISTS device_id VARCHAR(64) REFERENCES devices(id);
ALTER TABLE borrow_records ADD COLUMN IF NOT EXISTS slot_id VARCHAR(64) REFERENCES slots(id);
ALTER TABLE borrow_records ADD COLUMN IF NOT EXISTS purpose TEXT;
ALTER TABLE borrow_records ADD COLUMN IF NOT EXISTS overdue_at TIMESTAMPTZ;

ALTER TABLE device_operations ADD COLUMN IF NOT EXISTS sent_at TIMESTAMPTZ;
ALTER TABLE device_operations ADD COLUMN IF NOT EXISTS ack_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_borrow_records_reservation_id
    ON borrow_records(reservation_id);

-- 同一设备/钥匙在同一时刻只能存在一个未结束操作，最终一致性由设备事件和事务更新保证。
CREATE UNIQUE INDEX IF NOT EXISTS idx_device_operations_active_device
    ON device_operations(device_id)
    WHERE status IN ('CREATED', 'AUTHORIZED', 'SENT', 'EXECUTING');

CREATE UNIQUE INDEX IF NOT EXISTS idx_device_operations_active_key
    ON device_operations(key_id)
    WHERE key_id IS NOT NULL
      AND status IN ('CREATED', 'AUTHORIZED', 'SENT', 'EXECUTING');
