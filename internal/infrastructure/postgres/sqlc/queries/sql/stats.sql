-- name: InsertRideStat :exec
INSERT INTO ride_stats (origin, destination, ride_date, taken) VALUES ($1, $2, $3, $4);

-- name: InsertSearchEvent :exec
INSERT INTO search_events (origin, destination) VALUES ($1, $2);

-- name: InsertRideEvent :exec
INSERT INTO ride_events (origin, destination) VALUES ($1, $2);

-- name: GetTopRoutes :many
SELECT origin, destination, COUNT(*) AS count
FROM ride_stats
WHERE taken = true
  AND recorded_at >= DATE_TRUNC('week', NOW())
GROUP BY origin, destination
ORDER BY count DESC
LIMIT 5;

-- name: GetRideStatsTotals :one
SELECT
  COUNT(*) FILTER (WHERE taken = true)                                                     AS total_confirmed,
  COUNT(*) FILTER (WHERE taken = true AND recorded_at >= DATE_TRUNC('week', NOW()))        AS total_this_week
FROM ride_stats;

-- name: GetSearchEventCounts :one
SELECT
  COUNT(*)                                                                           AS all_time,
  COUNT(*) FILTER (WHERE searched_at >= DATE_TRUNC('year',  NOW()))                 AS this_year,
  COUNT(*) FILTER (WHERE searched_at >= DATE_TRUNC('month', NOW()))                 AS this_month
FROM search_events;

-- name: GetRideEventCounts :one
SELECT
  COUNT(*)                                                                           AS all_time,
  COUNT(*) FILTER (WHERE posted_at >= DATE_TRUNC('year',  NOW()))                   AS this_year,
  COUNT(*) FILTER (WHERE posted_at >= DATE_TRUNC('month', NOW()))                   AS this_month
FROM ride_events;
