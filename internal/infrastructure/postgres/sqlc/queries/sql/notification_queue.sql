-- name: EnqueueNotification :execrows
-- Returns rows affected: 1 when a new ride↔request pair is inserted, 0 when it
-- already existed. Callers use this to notify only newly-matched searchers.
-- (The match is pushed once; repeats are inbox reminders.)
INSERT INTO notification_queue (ride_id, request_id, searcher_phone)
VALUES ($1, $2, $3)
ON CONFLICT (ride_id, request_id) DO NOTHING;

-- name: DeleteExpiredNotifications :exec
DELETE FROM notification_queue
WHERE NOT EXISTS (SELECT 1 FROM rides    WHERE id = ride_id    AND expires_at > NOW())
   OR NOT EXISTS (SELECT 1 FROM requests WHERE id = request_id AND expires_at > NOW());

-- name: MarkNotificationSentByRideAndRequest :exec
UPDATE notification_queue
SET sent_count   = sent_count + 1,
    last_sent_at = NOW()
WHERE ride_id = $1 AND request_id = $2;

-- name: DeleteNotificationsForRide :exec
DELETE FROM notification_queue WHERE ride_id = $1;

-- name: ListNotificationsForSearcher :many
-- Returns pending notifications for a searcher (for UI display).
SELECT nq.id, nq.ride_id, nq.request_id, nq.searcher_phone,
       nq.sent_count, nq.last_sent_at, nq.created_at
FROM notification_queue nq
JOIN rides r ON r.id = nq.ride_id AND r.expires_at > NOW()
WHERE nq.searcher_phone = $1
ORDER BY nq.created_at DESC;
