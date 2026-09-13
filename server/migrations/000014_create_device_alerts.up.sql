CREATE TABLE IF NOT EXISTS device_alerts (
    id VARCHAR(64) PRIMARY KEY,
    device_id VARCHAR(64) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    slot_id VARCHAR(64) REFERENCES slots(id) ON DELETE SET NULL,
    slot_no INT,
    type VARCHAR(64) NOT NULL,
    expected_rfid VARCHAR(64),
    actual_rfid VARCHAR(64),
    message TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'OPEN',
    detected_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_device_alerts_device_status ON device_alerts(device_id, status);
CREATE INDEX IF NOT EXISTS idx_device_alerts_slot ON device_alerts(slot_id, status);
