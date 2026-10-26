-- Schema-qualify route_norm()'s call to f_unaccent (migration 010) so it
-- resolves under any search_path. pg_dump/pg_restore run with search_path = '',
-- and CREATE TABLE inlines route_norm() while validating the generated
-- origin_norm/destination_norm/driver_name_norm columns — so the unqualified
-- call made every restore (including Scalingo's PG 18 upgrade) fail with
-- "function f_unaccent(text) does not exist" and drop rides/requests.
-- The output is unchanged, so the stored generated values stay valid.
CREATE OR REPLACE FUNCTION route_norm(text) RETURNS text
  LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT AS
$$ SELECT public.f_unaccent(lower(regexp_replace(btrim($1), '\s+', ' ', 'g'))) $$;
