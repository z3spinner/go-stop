-- name: AddNotification :one
-- Records one inbox entry and returns its id. A repeat of the same logical
-- event (same phone and dedup_key: a push retry, a repeat ping or offer) keeps
-- the existing row untouched — it neither duplicates nor resurfaces as unread,
-- and the first recorded details (e.g. a ride's time before an edit) are kept —
-- and returns that row's id. (The no-op SET is only there so RETURNING yields
-- the existing row.) Deliberate: one entry per event.
INSERT INTO notifications (phone, kind, contact_name, origin, destination,
                           departure_at, url, title, body, ride_id, request_id,
                           dedup_key)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (phone, dedup_key) DO UPDATE SET phone = EXCLUDED.phone
RETURNING id;

-- name: ListNotificationsByPhone :many
-- Dismissed (user-deleted) items are hidden.
SELECT id, phone, kind, contact_name, origin, destination, departure_at,
       url, title, body, ride_id, request_id, dedup_key, created_at, read_at,
       dismissed_at, reminded_at, reminder_count
FROM notifications
WHERE phone = $1 AND dismissed_at IS NULL
ORDER BY created_at DESC
LIMIT sqlc.arg(max_items)::int;

-- name: CountUnreadNotifications :one
-- Counts over the same newest-N window ListNotificationsByPhone returns, so
-- the badge never counts items the inbox can't show.
SELECT COUNT(*)::int AS unread FROM (
    SELECT read_at FROM notifications
    WHERE phone = $1 AND dismissed_at IS NULL
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

-- name: DeleteInboxForGonePosts :execrows
-- Removes items about rides/requests that no longer exist — expired (deleted
-- by the hourly expiry job) or deleted by their owner — so a post's details
-- don't outlive it in other people's inboxes. The driver's feedback prompt is
-- sent after the ride is over, on purpose: it stays while it can still be
-- answered (its feedback task exists), and goes with the task.
DELETE FROM notifications n
WHERE (
    n.ride_id IS NOT NULL
    AND NOT EXISTS (SELECT 1 FROM rides r WHERE r.id = n.ride_id)
    AND (n.kind <> 'feedback_reminder'
         OR NOT EXISTS (SELECT 1 FROM feedback_queue fq WHERE fq.ride_id = n.ride_id))
  )
  OR (
    n.request_id IS NOT NULL
    AND NOT EXISTS (SELECT 1 FROM requests q WHERE q.id = n.request_id)
  );

-- name: DismissNotification :exec
-- User deleted the item from their inbox (see migration 017). Also marks it
-- read so it never counts towards the badge.
UPDATE notifications
SET dismissed_at = NOW(), read_at = COALESCE(read_at, NOW())
WHERE phone = $1 AND id = $2 AND dismissed_at IS NULL;

-- name: RestoreNotification :execrows
-- Undo of a recent dismissal; only while its content hasn't been blanked yet.
UPDATE notifications
SET dismissed_at = NULL
WHERE phone = $1 AND id = $2
  AND dismissed_at IS NOT NULL AND dismissed_at > sqlc.arg(dismissed_after)::timestamptz;

-- name: ScrubDismissedNotifications :execrows
-- Blanks the content of items dismissed before the cutoff, keeping only what
-- the tombstone needs (phone, kind, dedup key, post ids, timestamps).
UPDATE notifications
SET contact_name = '', origin = '', destination = '', title = '', body = '',
    url = '/', departure_at = NULL
WHERE dismissed_at IS NOT NULL AND dismissed_at < $1
  AND (contact_name <> '' OR origin <> '' OR destination <> '' OR title <> ''
       OR body <> '' OR url <> '/' OR departure_at IS NOT NULL);

-- name: MarkNotificationsReadByDedupKeys :exec
-- The user acted on the thing these notifications are about (accepted a
-- request, opened a shared contact, looked at offers), so they count as read.
UPDATE notifications SET read_at = NOW()
WHERE phone = $1 AND read_at IS NULL AND dedup_key = ANY(sqlc.arg(keys)::text[]);

-- name: FindNotificationRemindersDue :many
-- Unread, not deleted items of the given kinds whose last push (the original
-- one, or the last reminder) is older than the cutoff, and which have had
-- fewer than max_reminders reminders (0 = no limit). Items about an expired or
-- deleted post are already gone (see DeleteInboxForGonePosts).
SELECT id, phone, kind, contact_name, origin, destination, departure_at,
       url, title, body, ride_id, request_id, dedup_key, created_at, read_at,
       dismissed_at, reminded_at, reminder_count
FROM notifications
WHERE read_at IS NULL AND dismissed_at IS NULL
  AND kind = ANY(sqlc.arg(kinds)::text[])
  AND (sqlc.arg(max_reminders)::int = 0 OR reminder_count < sqlc.arg(max_reminders)::int)
  AND COALESCE(reminded_at, created_at) <= sqlc.arg(cutoff)::timestamptz
ORDER BY COALESCE(reminded_at, created_at)
LIMIT sqlc.arg(max_items)::int;

-- name: MarkNotificationReminded :exec
UPDATE notifications SET reminded_at = NOW(), reminder_count = reminder_count + 1
WHERE id = $1;
