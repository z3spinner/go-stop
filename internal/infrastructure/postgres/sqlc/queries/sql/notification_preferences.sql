-- name: GetNotificationPreferences :one
SELECT phone, morning_digest, updated_at FROM notification_preferences WHERE phone = $1;

-- name: UpsertMorningDigestPreference :exec
INSERT INTO notification_preferences (phone, morning_digest, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (phone) DO UPDATE SET morning_digest = EXCLUDED.morning_digest, updated_at = NOW();

-- name: ListMorningDigestOptOuts :many
SELECT phone FROM notification_preferences WHERE morning_digest = FALSE;
