CREATE TABLE IF NOT EXISTS rides (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_name  VARCHAR(100) NOT NULL,
    phone        VARCHAR(20)  NOT NULL,
    origin       VARCHAR(200) NOT NULL,
    destination  VARCHAR(200) NOT NULL,
    date         DATE         NOT NULL,
    departure_at TIMESTAMPTZ  NOT NULL,
    flexibility  INT          NOT NULL DEFAULT 0,
    posted_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ  NOT NULL
);

CREATE TABLE IF NOT EXISTS requests (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    searcher_name VARCHAR(100) NOT NULL,
    phone         VARCHAR(20)  NOT NULL,
    origin        VARCHAR(200) NOT NULL,
    destination   VARCHAR(200) NOT NULL,
    date          DATE         NOT NULL,
    departure_at  TIMESTAMPTZ  NOT NULL,
    flexibility   INT          NOT NULL DEFAULT 0,
    posted_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ  NOT NULL
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    phone    VARCHAR(20) NOT NULL UNIQUE,
    endpoint TEXT        NOT NULL,
    p256dh   TEXT        NOT NULL,
    auth     TEXT        NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rides_origin_destination    ON rides(origin, destination);
CREATE INDEX IF NOT EXISTS idx_requests_origin_destination ON requests(origin, destination);
CREATE INDEX IF NOT EXISTS idx_rides_expires_at            ON rides(expires_at);
CREATE INDEX IF NOT EXISTS idx_requests_expires_at         ON requests(expires_at);
