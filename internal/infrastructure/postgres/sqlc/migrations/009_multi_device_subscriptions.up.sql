-- Allow multiple devices per phone number.
-- Old: UNIQUE(phone) — only one subscription per phone.
-- New: UNIQUE(phone, endpoint) — one row per device/browser per phone.
ALTER TABLE subscriptions DROP CONSTRAINT subscriptions_phone_key;
ALTER TABLE subscriptions ADD CONSTRAINT subscriptions_phone_endpoint_key UNIQUE (phone, endpoint);
