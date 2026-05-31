# Go-Stop — Data Model

## Overview

Three tables: `rides`, `requests`, and `subscriptions`. Destinations are not stored in a separate table — they are derived from distinct values in the `origin` and `destination` columns of both `rides` and `requests`.

---

## Schema

```sql
-- rides
CREATE TABLE rides (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
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

-- requests
CREATE TABLE requests (
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

-- subscriptions
CREATE TABLE subscriptions (
    id       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    phone    VARCHAR(20) NOT NULL UNIQUE,
    endpoint TEXT        NOT NULL,
    p256dh   TEXT        NOT NULL,
    auth     TEXT        NOT NULL
);
```

---

## Indexes

```sql
-- Fast matching and browsing by route
CREATE INDEX idx_rides_origin_destination    ON rides(origin, destination);
CREATE INDEX idx_requests_origin_destination ON requests(origin, destination);

-- Fast expiry cleanup (cron job)
CREATE INDEX idx_rides_expires_at    ON rides(expires_at);
CREATE INDEX idx_requests_expires_at ON requests(expires_at);
```

---

## Destinations Query

No separate table needed. Destinations are derived at query time:

```sql
SELECT DISTINCT origin AS location FROM rides
UNION
SELECT DISTINCT destination      FROM rides
UNION
SELECT DISTINCT origin           FROM requests
UNION
SELECT DISTINCT destination      FROM requests
ORDER BY location;
```

---

## Matching Query

Matching uses overlapping flexibility windows. A ride and request match when:

- Same origin and destination
- Same date
- Their flexibility windows overlap:
  - `ride.departure_at - ride.flexibility` ≤ `request.departure_at + request.flexibility`
  - `ride.departure_at + ride.flexibility` ≥ `request.departure_at - request.flexibility`

```sql
-- Find requests matching a given ride
SELECT * FROM requests
WHERE origin      = $1
  AND destination = $2
  AND date        = $3
  AND expires_at  > NOW()
  AND (departure_at - (flexibility * interval '1 minute'))
        <= ($4 + ($5 * interval '1 minute'))
  AND (departure_at + (flexibility * interval '1 minute'))
        >= ($4 - ($5 * interval '1 minute'));
-- $4 = ride.departure_at, $5 = ride.flexibility
```

---

## Expiry

Rides and requests are expired by a scheduled cron job:

```sql
DELETE FROM rides    WHERE expires_at < NOW();
DELETE FROM requests WHERE expires_at < NOW();
```

The `ExpiresAt` value is set at creation time. A sensible default is the end of the ride's departure date (midnight).

---

## Notes

- `flexibility` is stored in **minutes** as an integer
- `phone` is stored as plain text — no hashing, as it is intentionally displayed to the other party
- Subscriptions are keyed by `phone` (UNIQUE) — one subscription per phone number
- No foreign keys between tables — keeping it simple and decoupled
