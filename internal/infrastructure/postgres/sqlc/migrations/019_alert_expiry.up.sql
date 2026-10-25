-- Open-ended alerts (anytime, or a daily time: no date of their own) used to
-- last a year, long after most people had stopped using them: they still showed
-- in the public feed and still pinged drivers. They now last 30 days from
-- posting or from their last renewal, and their owner is warned 3 days before
-- so they can keep them (domain.AlertLifetime / AlertExpiryNotice).

-- One row per alert whose current expiry has been warned about. Renewing moves
-- requests.expires_at, so the row no longer matches and the next expiry is
-- warned about again.
CREATE TABLE IF NOT EXISTS request_expiry_warnings (
    request_id UUID        PRIMARY KEY REFERENCES requests(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    warned_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Bring existing year-long alerts in line: 30 days from posting, but never
-- sooner than a week from now, so their owners get the warning and a few days
-- to answer it.
UPDATE requests
SET expires_at = GREATEST(posted_at + INTERVAL '30 days', NOW() + INTERVAL '7 days')
WHERE date IS NULL
  AND (departure_at IS NULL OR EXTRACT(YEAR FROM departure_at) = 1970)
  AND expires_at > GREATEST(posted_at + INTERVAL '30 days', NOW() + INTERVAL '7 days');
