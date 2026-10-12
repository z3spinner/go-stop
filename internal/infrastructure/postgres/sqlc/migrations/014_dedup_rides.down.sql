DROP INDEX IF EXISTS uq_rides_dedup;
ALTER TABLE rides DROP COLUMN IF EXISTS driver_name_norm;
