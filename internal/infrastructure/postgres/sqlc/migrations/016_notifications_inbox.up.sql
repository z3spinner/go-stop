-- Persistent per-user copy of every notification (the in-app inbox), so a
-- missed or swiped-away Web Push can still be read in the app. Also covers
-- users with no push subscription at all (e.g. iOS without the installed PWA).
-- Rows are garbage-collected after NOTIFICATION_RETENTION_DAYS (default 30),
-- and deleted early when the owner deletes the ride/request they are about.
--
-- Stores the notification kind plus the few parameters the client needs to
-- render it in the reader's language. The other party's phone number is
-- deliberately NOT stored: contact details stay behind the consent flow.
CREATE TABLE IF NOT EXISTS notifications (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    -- recipient; VARCHAR(100) like notification_queue/feedback_queue
    phone         VARCHAR(100) NOT NULL,
    kind          VARCHAR(40)  NOT NULL,
    contact_name  VARCHAR(100) NOT NULL DEFAULT '',
    origin        VARCHAR(200) NOT NULL DEFAULT '',
    destination   VARCHAR(200) NOT NULL DEFAULT '',
    departure_at  TIMESTAMPTZ,
    url           TEXT         NOT NULL DEFAULT '/',
    title         TEXT         NOT NULL DEFAULT '',
    body          TEXT         NOT NULL DEFAULT '',
    -- The post this notification is about (no FK, like the other tables).
    ride_id       UUID,
    request_id    UUID,
    -- Identifies one logical event so push retries don't create duplicates.
    -- NULL = never deduplicated (NULLs are distinct in a UNIQUE constraint).
    dedup_key     VARCHAR(200),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    read_at       TIMESTAMPTZ,
    UNIQUE (phone, dedup_key)
);
CREATE INDEX IF NOT EXISTS idx_notifications_phone_created ON notifications(phone, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at);
CREATE INDEX IF NOT EXISTS idx_notifications_ride ON notifications(ride_id) WHERE ride_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notifications_request ON notifications(request_id) WHERE request_id IS NOT NULL;
