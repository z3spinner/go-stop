DROP INDEX IF EXISTS idx_rides_origin_norm_trgm;
DROP INDEX IF EXISTS idx_rides_destination_norm_trgm;
DROP INDEX IF EXISTS idx_requests_origin_norm_trgm;
DROP INDEX IF EXISTS idx_requests_destination_norm_trgm;
DROP INDEX IF EXISTS idx_rides_route_norm;
DROP INDEX IF EXISTS idx_requests_route_norm;

ALTER TABLE rides    DROP COLUMN IF EXISTS origin_norm, DROP COLUMN IF EXISTS destination_norm;
ALTER TABLE requests DROP COLUMN IF EXISTS origin_norm, DROP COLUMN IF EXISTS destination_norm;

-- Restore migration 003's case-insensitive functional indexes.
CREATE INDEX IF NOT EXISTS idx_rides_origin_lower         ON rides(LOWER(origin));
CREATE INDEX IF NOT EXISTS idx_rides_destination_lower    ON rides(LOWER(destination));
CREATE INDEX IF NOT EXISTS idx_requests_origin_lower      ON requests(LOWER(origin));
CREATE INDEX IF NOT EXISTS idx_requests_destination_lower ON requests(LOWER(destination));

DROP FUNCTION IF EXISTS route_norm(text);
DROP FUNCTION IF EXISTS f_unaccent(text);
