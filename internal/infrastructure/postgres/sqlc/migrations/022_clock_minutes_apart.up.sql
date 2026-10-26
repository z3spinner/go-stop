-- Minutes between two clock times, measured the short way round midnight
-- (0–720): 23:30 and 00:10 are 40 minutes apart, not 1400.
-- Time-of-day matching (time-only search, daily alerts) treats two symmetric
-- windows as overlapping when their centres are at most the sum of their
-- half-widths apart. Comparing `time ± interval` bounds directly breaks near
-- midnight, because time arithmetic wraps (00:10 - 30 min = 23:40) while the
-- comparisons don't.
CREATE OR REPLACE FUNCTION public.clock_minutes_apart(a time, b time) RETURNS numeric
  LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT AS
$$ SELECT least(abs(extract(epoch FROM a - b)), 86400 - abs(extract(epoch FROM a - b))) / 60 $$;
