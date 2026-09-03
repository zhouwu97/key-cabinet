-- Create operation_events table for detailed device operation tracking
-- This table stores the event stream of each device operation

CREATE TABLE operation_events (
    id VARCHAR(64) PRIMARY KEY,
    operation_id VARCHAR(64) NOT NULL REFERENCES device_operations(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_operation_events_operation_id ON operation_events(operation_id);
CREATE INDEX idx_operation_events_occurred_at ON operation_events(occurred_at);
CREATE INDEX idx_operation_events_type ON operation_events(event_type);

COMMENT ON TABLE operation_events IS 'Device operation event stream for detailed tracking';
COMMENT ON COLUMN operation_events.event_type IS 'Event types: COMMAND_SENT, ACK, POSITIONING, DOOR_OPEN, KEY_REMOVED, RFID_VERIFIED, DOOR_CLOSED, COMPLETED, TIMEOUT, ERROR';
