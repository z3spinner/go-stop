-- name: ListDestinations :many
-- Returns known locations sorted by popularity. Combines active rides/requests
-- with historical ride_stats so locations persist after rides expire.
SELECT MIN(location)::text AS location FROM (
  SELECT origin      AS location FROM rides
  UNION ALL SELECT destination   FROM rides
  UNION ALL SELECT origin        FROM requests
  UNION ALL SELECT destination   FROM requests
  UNION ALL SELECT origin        FROM ride_stats
  UNION ALL SELECT destination   FROM ride_stats
) all_locs
GROUP BY LOWER(location)
ORDER BY COUNT(*) DESC, MIN(location) ASC;
