DROP INDEX IF EXISTS idx_operation_events_operation_seq;
ALTER TABLE operation_events DROP COLUMN IF EXISTS seq;
