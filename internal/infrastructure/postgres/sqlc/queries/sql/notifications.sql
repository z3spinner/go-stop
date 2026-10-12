-- name: AddNotification :exec
-- Records one inbox entry. A repeat of the same logical event (same phone and
-- dedup_key: a push retry, a repeat ping or offer) is ignored, so it neither
-- duplicates nor resurfaces as unread, and the first recorded details (e.g. a
-- ride's time before an edit) are kept. Deliberate: one entry per event.
INSERT INTO notifications (phone, kind, contact_name, origin, destination,
                           departure_at, url, title, body, ride_id, request_id,
                           dedup_key)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (phone, dedup_key) DO NOTHING;

-- name: ListNotificationsByPhone :many
SELECT id, phone, kind, contact_name, origin, destination, departure_at,
       url, title, body, ride_id, request_id, dedup_key, created_at, read_at
FROM notifications
WHERE phone = $1
ORDER BY created_at DESC
LIMIT sqlc.arg(max_items)::int;

-- name: CountUnreadNotifications :one
-- Counts over the same newest-N window ListNotificationsByPhone returns, so
-- the badge never counts items the inbox can't show.
SELECT COUNT(*)::int AS unread FROM (
    SELECT read_at FROM notifications
    WHERE phone = $1
    ORDER BY created_at DESC
    LIMIT sqlc.arg(max_items)::int
) newest
WHERE read_at IS NULL;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET read_at = NOW() WHERE phone = $1 AND read_at IS NULL;

-- name: MarkNotificationsRead :exec
UPDATE notifications SET read_at = NOW()
WHERE phone = $1 AND read_at IS NULL AND id = ANY(sqlc.arg(ids)::uuid[]);

-- name: DeleteInboxForRide :exec
DELETE FROM notifications WHERE ride_id = $1;

-- name: DeleteInboxForRequest :exec
DELETE FROM notifications WHERE request_id = $1;

-- name: DeleteNotificationsOlderThan :execrows
DELETE FROM notifications WHERE created_at < $1;
