CREATE OR REPLACE FUNCTION route_norm(text) RETURNS text
  LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT AS
$$ SELECT f_unaccent(lower(regexp_replace(btrim($1), '\s+', ' ', 'g'))) $$;
