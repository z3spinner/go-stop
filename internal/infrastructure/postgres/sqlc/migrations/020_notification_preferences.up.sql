-- Per-phone notification preferences. A phone without a row has the defaults
-- (everything on), so only people who change something get a row.
CREATE TABLE IF NOT EXISTS notification_preferences (
    phone         VARCHAR(100) PRIMARY KEY,
    -- The daily morning digest of the day's rides and requests.
    morning_digest BOOLEAN     NOT NULL DEFAULT TRUE,
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
