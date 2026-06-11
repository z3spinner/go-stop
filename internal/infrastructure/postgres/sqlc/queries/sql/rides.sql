-- name: InsertRide :one
-- Idempotent insert. ON CONFLICT on the dedup key (phone + normalized driver
-- name + normalized route + exact departure instant) means a re-posted ride
-- inserts nothing and returns zero rows; the caller then re-reads the existing
-- ride via GetRideByDedupKey.
INSERT INTO rides (id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (phone, driver_name_norm, origin_norm, destination_norm, departure_at) DO NOTHING
RETURNING id;

-- name: GetRideByDedupKey :one
-- Fetch the ride that owns a given dedup key (used after InsertRide hits a
-- conflict). Mirrors the unique index in migration 014.
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides
WHERE phone = $1
  AND driver_name_norm = route_norm(sqlc.arg(driver_name)::text)
  AND origin_norm = route_norm(sqlc.arg(origin)::text)
  AND destination_norm = route_norm(sqlc.arg(destination)::text)
  AND departure_at = sqlc.arg(departure_at);

-- name: GetRideByID :one
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides WHERE id = $1;

-- name: ListRidesActive :many
-- grace_minutes: hides rides whose flex window ended more than N minutes ago
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides
WHERE expires_at > NOW()
  AND departure_at + (flexibility * interval '1 minute') + (sqlc.arg(grace_minutes)::int * interval '1 minute') > NOW()
ORDER BY departure_at ASC;

-- name: ListRidesByPhone :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides WHERE phone = $1 AND expires_at > NOW()
ORDER BY departure_at ASC;

-- name: SearchRidesByDate :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides
WHERE origin_norm = route_norm(sqlc.arg(origin)::text) AND destination_norm = route_norm(sqlc.arg(destination)::text)
  AND date = sqlc.arg(date)
  AND expires_at > NOW()
  AND departure_at + (flexibility * interval '1 minute') + (sqlc.arg(grace_minutes)::int * interval '1 minute') > NOW()
ORDER BY departure_at ASC;

-- name: SearchRidesByTime :many
-- Time-only search: any date, departure window overlaps search_time ± tolerance.
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides
WHERE origin_norm = route_norm(sqlc.arg(origin)::text) AND destination_norm = route_norm(sqlc.arg(destination)::text)
  AND expires_at > NOW()
  AND departure_at + (flexibility * interval '1 minute') + (sqlc.arg(grace_minutes)::int * interval '1 minute') > NOW()
  AND (departure_at::time - (flexibility * interval '1 minute')) <= (sqlc.arg(search_time)::timestamptz::time + (sqlc.arg(search_tolerance_minutes)::int * interval '1 minute'))
  AND (departure_at::time + (flexibility * interval '1 minute')) >= (sqlc.arg(search_time)::timestamptz::time - (sqlc.arg(search_tolerance_minutes)::int * interval '1 minute'))
ORDER BY departure_at ASC;

-- name: SearchRidesByDateTime :many
-- Returns rides on the given date whose departure window (±flexibility) overlaps
-- the search time ± search_tolerance_minutes. Hides expired/past-grace rides.
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides
WHERE origin_norm = route_norm(sqlc.arg(origin)::text) AND destination_norm = route_norm(sqlc.arg(destination)::text)
  AND date = sqlc.arg(date)
  AND expires_at > NOW()
  AND departure_at + (flexibility * interval '1 minute') + (sqlc.arg(grace_minutes)::int * interval '1 minute') > NOW()
  AND (departure_at - (flexibility * interval '1 minute')) <= (sqlc.arg(search_time)::timestamptz + (sqlc.arg(search_tolerance_minutes)::int * interval '1 minute'))
  AND (departure_at + (flexibility * interval '1 minute')) >= (sqlc.arg(search_time)::timestamptz - (sqlc.arg(search_tolerance_minutes)::int * interval '1 minute'))
ORDER BY departure_at ASC;

-- name: SearchRides :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides
WHERE origin_norm = route_norm(sqlc.arg(origin)::text) AND destination_norm = route_norm(sqlc.arg(destination)::text)
  AND expires_at > NOW()
  AND departure_at + (flexibility * interval '1 minute') + (sqlc.arg(grace_minutes)::int * interval '1 minute') > NOW()
ORDER BY departure_at ASC;

-- name: SearchRidesFuzzy :many
-- Trigram fuzzy fallback for typos/spelling variants. The `%` operator uses the
-- GIN indexes and respects pg_trgm.similarity_threshold (default 0.3). Used only
-- as a search fallback when the exact lookup returns nothing — NEVER for the
-- notification matching path, where a loose match would ping the wrong driver.
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides
WHERE origin_norm % route_norm(sqlc.arg(origin)::text)
  AND destination_norm % route_norm(sqlc.arg(destination)::text)
  AND expires_at > NOW()
  AND departure_at + (flexibility * interval '1 minute') + (sqlc.arg(grace_minutes)::int * interval '1 minute') > NOW()
ORDER BY similarity(origin_norm, route_norm(sqlc.arg(origin)::text))
       + similarity(destination_norm, route_norm(sqlc.arg(destination)::text)) DESC,
         departure_at ASC;

-- name: FindRidesMatchingAnytimeRequest :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides
WHERE origin_norm = route_norm(sqlc.arg(origin)::text) AND destination_norm = route_norm(sqlc.arg(destination)::text)
  AND expires_at > NOW();

-- name: FindRidesMatchingDailyRequest :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides
WHERE origin_norm = route_norm(sqlc.arg(origin)::text) AND destination_norm = route_norm(sqlc.arg(destination)::text)
  AND expires_at > NOW()
  AND (departure_at::time - (flexibility * interval '1 minute')) <= (sqlc.arg(departure_at)::timestamptz::time + (sqlc.arg(window_minutes)::int * interval '1 minute'))
  AND (departure_at::time + (flexibility * interval '1 minute')) >= (sqlc.arg(departure_at)::timestamptz::time - (sqlc.arg(window_minutes)::int * interval '1 minute'));

-- name: FindRidesMatchingDayRequest :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides
WHERE origin_norm = route_norm(sqlc.arg(origin)::text) AND destination_norm = route_norm(sqlc.arg(destination)::text)
  AND date = sqlc.arg(date)
  AND expires_at > NOW();

-- name: FindRidesMatchingTimeRequest :many
SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given, origin_norm, destination_norm, driver_name_norm
FROM rides
WHERE origin_norm = route_norm(sqlc.arg(origin)::text) AND destination_norm = route_norm(sqlc.arg(destination)::text)
  AND date = sqlc.arg(date)
  AND expires_at > NOW()
  AND (departure_at - (flexibility * interval '1 minute')) <= (sqlc.arg(departure_at)::timestamptz + (sqlc.arg(window_minutes)::int * interval '1 minute'))
  AND (departure_at + (flexibility * interval '1 minute')) >= (sqlc.arg(departure_at)::timestamptz - (sqlc.arg(window_minutes)::int * interval '1 minute'));

-- name: DeleteRide :exec
DELETE FROM rides WHERE id = $1;

-- name: DeleteExpiredRides :exec
DELETE FROM rides WHERE expires_at < NOW();

-- name: ClaimRideFeedback :execrows
UPDATE rides SET feedback_given = true WHERE id = $1 AND feedback_given = false;
