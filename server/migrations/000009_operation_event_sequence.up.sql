-- 操作事件流需要稳定序号，供小程序 Polling 增量消费。
ALTER TABLE operation_events ADD COLUMN IF NOT EXISTS seq INTEGER;

WITH numbered AS (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY operation_id ORDER BY occurred_at, created_at, id) AS seq
    FROM operation_events
)
UPDATE operation_events AS events
SET seq = numbered.seq
FROM numbered
WHERE events.id = numbered.id
  AND events.seq IS NULL;

ALTER TABLE operation_events ALTER COLUMN seq SET DEFAULT 1;
ALTER TABLE operation_events ALTER COLUMN seq SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_operation_events_operation_seq
    ON operation_events(operation_id, seq);
