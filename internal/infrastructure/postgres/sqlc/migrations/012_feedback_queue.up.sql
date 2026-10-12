-- A self-contained queue of post-ride feedback tasks. Decoupled from `rides`
-- (no FK) so a task survives ride deletion and expiry. Carries `phone` for the
-- ownership check and origin/destination/ride_date so the stat can be written
-- without the ride. Created at window start; pushed at send_after; retried via
-- sent_count/last_sent_at; deleted once answered or exhausted.
CREATE TABLE IF NOT EXISTS feedback_queue (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id       UUID         NOT NULL UNIQUE,
    phone         VARCHAR(100) NOT NULL,
    origin        VARCHAR(200) NOT NULL,
    destination   VARCHAR(200) NOT NULL,
    ride_date     DATE         NOT NULL,
    departure_at  TIMESTAMPTZ  NOT NULL,
    send_after    TIMESTAMPTZ  NOT NULL,
    sent_count    INT          NOT NULL DEFAULT 0,
    last_sent_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fq_send_after ON feedback_queue(send_after);
