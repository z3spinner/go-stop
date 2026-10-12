-- name: EnqueueStartedRides :exec
-- Inserts a feedback task for every ride whose window has started (departure in
-- the past, but within the bound), that hasn't been answered, and isn't already
-- queued. send_after = window end (departure + flexibility minutes) + 1 hour.
-- Idempotent via ride_id UNIQUE.
INSERT INTO feedback_queue (ride_id, phone, origin, destination, ride_date, departure_at, send_after)
SELECT r.id, r.phone, r.origin, r.destination, r.date, r.departure_at,
       r.departure_at + (r.flexibility * INTERVAL '1 minute') + INTERVAL '1 hour'
FROM rides r
WHERE r.departure_at <= NOW()
  AND r.departure_at > sqlc.arg(window_start_after)::timestamptz
  AND r.feedback_given = false
  AND NOT EXISTS (SELECT 1 FROM feedback_queue fq WHERE fq.ride_id = r.id)
ON CONFLICT (ride_id) DO NOTHING;

-- name: FindDueFeedback :many
-- Tasks past their send time and still retry-eligible.
SELECT id, ride_id, phone, origin, destination, ride_date, departure_at, send_after, sent_count, last_sent_at, created_at
FROM feedback_queue
WHERE send_after <= NOW()
  AND sent_count < sqlc.arg(max_retries)::int
  AND (last_sent_at IS NULL OR last_sent_at < sqlc.arg(retry_before)::timestamptz)
ORDER BY send_after ASC;

-- name: GetFeedbackByRideID :one
SELECT id, ride_id, phone, origin, destination, ride_date, departure_at, send_after, sent_count, last_sent_at, created_at
FROM feedback_queue
WHERE ride_id = $1;

-- name: MarkFeedbackSent :exec
UPDATE feedback_queue
SET sent_count = sent_count + 1,
    last_sent_at = NOW()
WHERE id = $1;

-- name: DeleteFeedbackByRideID :execrows
-- :execrows returns the number of rows deleted, used as the "claim" signal in
-- RecordFeedback: only the caller that actually deletes the row records the stat.
DELETE FROM feedback_queue WHERE ride_id = $1;

-- name: DeleteExhaustedFeedback :exec
DELETE FROM feedback_queue
WHERE sent_count >= sqlc.arg(max_retries)::int
   OR created_at < sqlc.arg(ttl_before)::timestamptz;
