-- name: InsertRequest :exec
INSERT INTO requests (id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetRequestByID :one
SELECT id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
FROM requests WHERE id = $1;

-- name: ListRequestsByPhone :many
SELECT id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
FROM requests WHERE phone = $1 AND expires_at > NOW()
ORDER BY COALESCE(departure_at, date, expires_at) ASC;

-- name: ListActiveRequests :many
-- Public feed of all non-expired requests, ordered so concrete demand comes
-- first and the vaguest alerts sink to the bottom:
--   0 dated (a one-off date+time, OR a date-only "any time that day")
--   1 a daily recurring time   2 anytime
-- Within the dated group, sort chronologically by the effective moment — a
-- date-only alert sorts at the END of its day (a one-second-before-midnight
-- key), so it sits below same-day date+time entries yet above any later day.
-- A daily alert carries a 1970-01-01 sentinel departure_at, so any later year
-- marks a concrete one-off. Newest breaks ties.
SELECT id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
FROM requests WHERE expires_at > NOW()
ORDER BY
  CASE
    WHEN date IS NOT NULL OR (departure_at IS NOT NULL AND EXTRACT(YEAR FROM departure_at) > 1970) THEN 0
    WHEN departure_at IS NOT NULL THEN 1
    ELSE 2
  END,
  COALESCE(departure_at, date::timestamptz + interval '1 day' - interval '1 second') ASC NULLS LAST,
  posted_at DESC;

-- name: FindRequestsMatchingRide :many
-- Matches all alert modes inferred from NULL state of date/departure_at:
--   anytime: date IS NULL AND departure_at IS NULL
--   daily:   date IS NULL AND departure_at IS NOT NULL (time-only match)
--   day:     date set, departure_at IS NULL
--   time:    both set (overlapping window)
SELECT id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
FROM requests
WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2)
  AND expires_at > NOW()
  AND (
    (date IS NULL AND departure_at IS NULL)
    OR (date IS NULL AND departure_at IS NOT NULL
        AND (departure_at::time - (flexibility * interval '1 minute')) <= ($4::timestamptz::time + ($5::int * interval '1 minute'))
        AND (departure_at::time + (flexibility * interval '1 minute')) >= ($4::timestamptz::time - ($5::int * interval '1 minute')))
    OR (date = $3 AND departure_at IS NULL)
    OR (date = $3
        AND (departure_at - (flexibility * interval '1 minute')) <= ($4::timestamptz + ($5::int * interval '1 minute'))
        AND (departure_at + (flexibility * interval '1 minute')) >= ($4::timestamptz - ($5::int * interval '1 minute')))
  );

-- name: DeleteRequest :exec
DELETE FROM requests WHERE id = $1;

-- name: DeleteExpiredRequests :exec
DELETE FROM requests WHERE expires_at < NOW();
