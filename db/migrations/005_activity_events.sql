CREATE TABLE IF NOT EXISTS search_events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    origin      VARCHAR(200) NOT NULL,
    destination VARCHAR(200) NOT NULL,
    searched_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_search_events_at ON search_events(searched_at);

CREATE TABLE IF NOT EXISTS ride_events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    origin      VARCHAR(200) NOT NULL,
    destination VARCHAR(200) NOT NULL,
    posted_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ride_events_at ON ride_events(posted_at);
