-- No FK to rides: rides get deleted on expiry but interests should survive
-- long enough to complete the contact exchange.
CREATE TABLE IF NOT EXISTS interests (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id        UUID         NOT NULL,
    searcher_phone VARCHAR(20)  NOT NULL,
    searcher_name  VARCHAR(100) NOT NULL DEFAULT '',
    status         VARCHAR(20)  NOT NULL DEFAULT 'pending',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(ride_id, searcher_phone)
);

CREATE INDEX IF NOT EXISTS idx_interests_ride_id        ON interests(ride_id);
CREATE INDEX IF NOT EXISTS idx_interests_searcher_phone ON interests(searcher_phone);
