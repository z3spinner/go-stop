-- Stores pending/sent ride↔request notification pairs so they can be retried.
-- A row is inserted when a ride matches a request. The job re-notifies
-- until sent_count reaches the max or the ride/request expires.
CREATE TABLE IF NOT EXISTS notification_queue (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id         UUID        NOT NULL,
    request_id      UUID        NOT NULL,
    searcher_phone  VARCHAR(100) NOT NULL,
    sent_count      INT         NOT NULL DEFAULT 0,
    last_sent_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(ride_id, request_id)
);
CREATE INDEX IF NOT EXISTS idx_nq_last_sent ON notification_queue(last_sent_at);
