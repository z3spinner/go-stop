-- name: InsertRide :exec
INSERT INTO rides (id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetRideByID :one
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
FROM rides WHERE id = $1;

-- name: ListRidesActive :many
-- grace_minutes: hides rides whose flex window ended more than N minutes ago
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
FROM rides
WHERE expires_at > NOW()
  AND departure_at + (flexibility * interval '1 minute') + (sqlc.arg(grace_minutes)::int * interval '1 minute') > NOW()
ORDER BY departure_at ASC;

-- name: ListRidesByPhone :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
FROM rides WHERE phone = $1 AND expires_at > NOW()
ORDER BY departure_at ASC;

-- name: SearchRidesByDate :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
FROM rides
WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2)
  AND date = $3
  AND expires_at > NOW()
  AND departure_at + (flexibility * interval '1 minute') + (sqlc.arg(grace_minutes)::int * interval '1 minute') > NOW()
ORDER BY departure_at ASC;

-- name: SearchRidesByDateTime :many
-- Returns rides on the given date whose departure window (±flexibility) overlaps
-- the search time ± search_tolerance_minutes. Hides expired/past-grace rides.
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
FROM rides
WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2)
  AND date = $3
  AND expires_at > NOW()
  AND departure_at + (flexibility * interval '1 minute') + (sqlc.arg(grace_minutes)::int * interval '1 minute') > NOW()
  AND (departure_at - (flexibility * interval '1 minute')) <= ($4::timestamptz + (sqlc.arg(search_tolerance_minutes)::int * interval '1 minute'))
  AND (departure_at + (flexibility * interval '1 minute')) >= ($4::timestamptz - (sqlc.arg(search_tolerance_minutes)::int * interval '1 minute'))
ORDER BY departure_at ASC;

-- name: SearchRides :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
FROM rides
WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2)
  AND expires_at > NOW()
  AND departure_at + (flexibility * interval '1 minute') + (sqlc.arg(grace_minutes)::int * interval '1 minute') > NOW()
ORDER BY departure_at ASC;

-- name: FindRidesMatchingAnytimeRequest :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
FROM rides
WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2)
  AND expires_at > NOW();

-- name: FindRidesMatchingDailyRequest :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
FROM rides
WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2)
  AND expires_at > NOW()
  AND (departure_at::time - (flexibility * interval '1 minute')) <= ($3::timestamptz::time + ($4::int * interval '1 minute'))
  AND (departure_at::time + (flexibility * interval '1 minute')) >= ($3::timestamptz::time - ($4::int * interval '1 minute'));

-- name: FindRidesMatchingDayRequest :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
FROM rides
WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2)
  AND date = $3
  AND expires_at > NOW();

-- name: FindRidesMatchingTimeRequest :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
FROM rides
WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2)
  AND date = $3
  AND expires_at > NOW()
  AND (departure_at - (flexibility * interval '1 minute')) <= ($4::timestamptz + ($5::int * interval '1 minute'))
  AND (departure_at + (flexibility * interval '1 minute')) >= ($4::timestamptz - ($5::int * interval '1 minute'));

-- name: DeleteRide :exec
DELETE FROM rides WHERE id = $1;

-- name: DeleteExpiredRides :exec
DELETE FROM rides WHERE expires_at < NOW();

-- name: ListRidesPendingFeedback :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
FROM rides
WHERE departure_at BETWEEN (NOW() - INTERVAL '23 hours') AND (NOW() - INTERVAL '30 minutes')
  AND feedback_given = false
  AND expires_at > NOW()
ORDER BY departure_at ASC;

-- name: SetRideFeedbackGiven :exec
UPDATE rides SET feedback_given = true WHERE id = $1;
