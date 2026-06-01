-- name: InsertInterest :exec
INSERT INTO interests (id, ride_id, searcher_phone, searcher_name, status)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (ride_id, searcher_phone) DO NOTHING;

-- name: GetInterestByID :one
SELECT id, ride_id, searcher_phone, searcher_name, status, created_at
FROM interests WHERE id = $1;

-- name: GetInterestByRideAndSearcher :one
SELECT id, ride_id, searcher_phone, searcher_name, status, created_at
FROM interests WHERE ride_id = $1 AND searcher_phone = $2;

-- name: ListInterestsByRide :many
SELECT id, ride_id, searcher_phone, searcher_name, status, created_at
FROM interests WHERE ride_id = $1
ORDER BY created_at ASC;

-- name: AcceptInterest :exec
UPDATE interests SET status = 'accepted' WHERE id = $1;

-- name: ListInterestsBySearcher :many
-- Returns all interests made by a searcher, joined with ride info for display.
-- Includes rides that may have expired (so the searcher can see their full history).
SELECT i.id, i.ride_id, i.searcher_phone, i.searcher_name, i.status, i.created_at,
       r.origin, r.destination, r.departure_at, r.driver_name
FROM interests i
JOIN rides r ON r.id = i.ride_id
WHERE i.searcher_phone = $1
ORDER BY i.created_at DESC;

-- name: CountInterestsByRides :many
-- Returns interest counts for a set of ride IDs.
SELECT ride_id, COUNT(*) AS count
FROM interests
WHERE ride_id = ANY($1::uuid[])
GROUP BY ride_id;
