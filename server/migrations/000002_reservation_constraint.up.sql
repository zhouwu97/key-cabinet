-- Add exclusion constraint for reservation time conflicts
ALTER TABLE reservations
ADD CONSTRAINT reservations_no_overlap
EXCLUDE USING gist (
    key_id WITH =,
    tstzrange(start_time, end_time) WITH &&
)
WHERE (status IN ('ACTIVE', 'PENDING'));
