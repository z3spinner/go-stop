-- Accent/case/whitespace-insensitive route matching.
-- Extensions: first ones declared in this project (gen_random_uuid is PG core).
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- unaccent() is only STABLE, so it can't appear in a generated column or index.
-- Wrap it to assert IMMUTABLE (safe as long as the unaccent dictionary is fixed).
CREATE OR REPLACE FUNCTION f_unaccent(text) RETURNS text
  LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT AS
$$ SELECT public.unaccent('public.unaccent', $1) $$;

-- Single source of truth for "normalise a route name": strip accents, lowercase,
-- trim, and collapse internal whitespace. Used by both the write path (generated
-- columns below) and the read path (queries call route_norm() on search input).
CREATE OR REPLACE FUNCTION route_norm(text) RETURNS text
  LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT AS
$$ SELECT f_unaccent(lower(regexp_replace(btrim($1), '\s+', ' ', 'g'))) $$;

-- Write-time normalisation: STORED generated columns recompute on every
-- INSERT/UPDATE, and adding them backfills all existing rows in the table
-- rewrite below — so this migration also normalises the live data in one pass.
ALTER TABLE rides
  ADD COLUMN origin_norm      text GENERATED ALWAYS AS (route_norm(origin))      STORED,
  ADD COLUMN destination_norm text GENERATED ALWAYS AS (route_norm(destination)) STORED;

ALTER TABLE requests
  ADD COLUMN origin_norm      text GENERATED ALWAYS AS (route_norm(origin))      STORED,
  ADD COLUMN destination_norm text GENERATED ALWAYS AS (route_norm(destination)) STORED;

-- Exact-match lookups on the normalised columns (replaces the LOWER() indexes).
CREATE INDEX idx_rides_route_norm    ON rides(origin_norm, destination_norm);
CREATE INDEX idx_requests_route_norm ON requests(origin_norm, destination_norm);

-- Trigram GIN indexes power pg_trgm fuzzy search (% operator, similarity()).
CREATE INDEX idx_rides_origin_norm_trgm         ON rides    USING gin (origin_norm      gin_trgm_ops);
CREATE INDEX idx_rides_destination_norm_trgm    ON rides    USING gin (destination_norm gin_trgm_ops);
CREATE INDEX idx_requests_origin_norm_trgm      ON requests USING gin (origin_norm      gin_trgm_ops);
CREATE INDEX idx_requests_destination_norm_trgm ON requests USING gin (destination_norm gin_trgm_ops);

-- The LOWER() functional indexes from migration 003 are now redundant.
DROP INDEX IF EXISTS idx_rides_origin_lower;
DROP INDEX IF EXISTS idx_rides_destination_lower;
DROP INDEX IF EXISTS idx_requests_origin_lower;
DROP INDEX IF EXISTS idx_requests_destination_lower;
