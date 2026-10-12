-- name: AddNotification :exec
-- Records one inbox entry. A repeat of the same logical event (same phone and
-- dedup_key, e.g. a push retry) is ignored.
INSERT INTO notifications (phone, kind, contact_name, origin, destination,
                           departure_at, url, title, body, dedup_key)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (phone, dedup_key) DO NOTHING;

-- name: ListNotificationsByPhone :many
SELECT id, phone, kind, contact_name, origin, destination, departure_at,
       url, title, body, dedup_key, created_at, read_at
FROM notifications
WHERE phone = $1
ORDER BY created_at DESC
LIMIT sqlc.arg(max_items)::int;

-- name: CountUnreadNotifications :one
SELECT COUNT(*)::int AS unread FROM notifications WHERE phone = $1 AND read_at IS NULL;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET read_at = NOW() WHERE phone = $1 AND read_at IS NULL;

-- name: MarkNotificationsRead :exec
UPDATE notifications SET read_at = NOW()
WHERE phone = $1 AND read_at IS NULL AND id = ANY(sqlc.arg(ids)::uuid[]);

-- name: DeleteNotificationsOlderThan :execrows
DELETE FROM notifications WHERE created_at < $1;
