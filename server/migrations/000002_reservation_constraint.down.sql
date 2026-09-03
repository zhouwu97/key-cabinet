-- Remove exclusion constraint for reservation time conflicts
ALTER TABLE reservations DROP CONSTRAINT IF EXISTS reservations_no_overlap;
