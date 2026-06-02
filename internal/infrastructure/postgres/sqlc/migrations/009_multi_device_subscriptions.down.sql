ALTER TABLE subscriptions DROP CONSTRAINT subscriptions_phone_endpoint_key;
ALTER TABLE subscriptions ADD CONSTRAINT subscriptions_phone_key UNIQUE (phone);
