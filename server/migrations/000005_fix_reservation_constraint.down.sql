-- Rollback reservation constraint fix
ALTER TABLE reservations DROP CONSTRAINT IF EXISTS reservations_no_overlap;

-- Restore original constraint (with old field names)
-- Note: This will fail if 000003 migration has already renamed fields
-- In that case, this down migration should only be run after 000003 down
