-- name: UpsertSubscription :exec
-- ON CONFLICT (phone, endpoint) allows multiple devices per phone.
INSERT INTO subscriptions (id, phone, endpoint, p256dh, auth)
VALUES (gen_random_uuid(), $1, $2, $3, $4)
ON CONFLICT (phone, endpoint) DO UPDATE SET p256dh = $3, auth = $4;

-- name: ListSubscriptionsByPhone :many
SELECT id, phone, endpoint, p256dh, auth
FROM subscriptions WHERE phone = $1;

-- name: DeleteSubscription :exec
DELETE FROM subscriptions WHERE phone = $1;

-- name: DeleteSubscriptionByEndpoint :exec
-- Removes a specific device subscription (e.g. when push returns 410 Gone).
DELETE FROM subscriptions WHERE endpoint = $1;
