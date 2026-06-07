-- Activity log of "connections made": the moments two people exchanged contact,
-- either because a driver accepted a searcher's interest or a driver proactively
-- pinged a searcher (driver_shared). Counted on the stats page like search_events
-- and ride_events. Connections are only ever shown as a count, so unlike its
-- sibling tables this one carries no route — just the timestamp.
CREATE TABLE IF NOT EXISTS connection_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_connection_events_at ON connection_events(connected_at);

-- Backfill historical connections from existing interests. There is no
-- accepted_at column, so created_at is used as the connection moment: exact for
-- driver_shared (created already-shared) and an approximation for accepted rows.
INSERT INTO connection_events (connected_at)
SELECT created_at FROM interests WHERE status IN ('accepted', 'driver_shared');
