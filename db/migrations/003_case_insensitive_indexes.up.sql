-- Case-insensitive functional indexes for origin/destination lookups.
-- Required for good performance with LOWER(origin) = LOWER($1) queries.
CREATE INDEX IF NOT EXISTS idx_rides_origin_lower       ON rides(LOWER(origin));
CREATE INDEX IF NOT EXISTS idx_rides_destination_lower  ON rides(LOWER(destination));
CREATE INDEX IF NOT EXISTS idx_requests_origin_lower    ON requests(LOWER(origin));
CREATE INDEX IF NOT EXISTS idx_requests_destination_lower ON requests(LOWER(destination));
