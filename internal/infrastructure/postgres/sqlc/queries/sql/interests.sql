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
