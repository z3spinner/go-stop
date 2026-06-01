-- name: UpsertSubscription :exec
INSERT INTO subscriptions (id, phone, endpoint, p256dh, auth)
VALUES (gen_random_uuid(), $1, $2, $3, $4)
ON CONFLICT (phone) DO UPDATE SET endpoint = $2, p256dh = $3, auth = $4;

-- name: GetSubscriptionByPhone :one
SELECT id, phone, endpoint, p256dh, auth
FROM subscriptions WHERE phone = $1;

-- name: DeleteSubscription :exec
DELETE FROM subscriptions WHERE phone = $1;
