-- Idempotent ride creation: prevent a driver re-posting an identical ride.
-- A duplicate is the same phone + driver name + route + exact departure instant.

-- driver_name_norm mirrors origin_norm/destination_norm (migration 010): a STORED
-- generated column over route_norm() so the dedup key folds case, accents and
-- whitespace ("Alice" == " alice "). A stored column (rather than an expression
-- index over route_norm(driver_name)) is required because CREATE INDEX inlines
-- route_norm with a sanitized search_path and can't resolve f_unaccent — whereas
-- a generated column resolves it at write time, exactly as origin_norm does.
-- Adding the column rewrites the table, backfilling every existing row.
ALTER TABLE rides
  ADD COLUMN driver_name_norm text GENERATED ALWAYS AS (route_norm(driver_name)) STORED;

-- The dedup key, all plain columns so the write path's ON CONFLICT target is a
-- simple column list. UNIQUE so concurrent double-submits collide at the DB.
CREATE UNIQUE INDEX uq_rides_dedup
  ON rides (phone, driver_name_norm, origin_norm, destination_norm, departure_at);
