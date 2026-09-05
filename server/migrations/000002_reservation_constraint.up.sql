-- Add exclusion constraint for reservation time conflicts
ALTER TABLE reservations
ADD CONSTRAINT reservations_no_overlap
EXCLUDE USING gist (
    key_id WITH =,
    -- 初始版本使用不带时区的时间列，这里明确按 UTC 转换后再构造 tstzrange。
    tstzrange(
        start_time AT TIME ZONE 'UTC',
        end_time AT TIME ZONE 'UTC',
        '[)'
    ) WITH &&
)
WHERE (status IN ('ACTIVE', 'PENDING'));
