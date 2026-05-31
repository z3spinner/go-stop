ALTER TABLE rides ADD COLUMN IF NOT EXISTS feedback_given BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS ride_stats (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    origin       VARCHAR(200) NOT NULL,
    destination  VARCHAR(200) NOT NULL,
    ride_date    DATE         NOT NULL,
    taken        BOOLEAN      NOT NULL,
    recorded_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ride_stats_route ON ride_stats(origin, destination);
CREATE INDEX IF NOT EXISTS idx_ride_stats_date  ON ride_stats(recorded_at);
