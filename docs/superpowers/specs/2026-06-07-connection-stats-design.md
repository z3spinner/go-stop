# Connections & unanswered-request statistics — design

**Date:** 2026-06-07
**Status:** Approved (pending implementation plan)

## Goal

Add two new statistics to the stats page, alongside the existing
**Searches** and **Rides posted** activity counts:

- **Connections made** — how many times two people actually exchanged contact.
- **Went unanswered** — how many contact requests were never answered before
  the ride was gone.

Both are presented as `ActivityCounts` (this month / this year / all time), the
same shape as the existing activity stats.

## Definitions

### Connection made
A connection is an interest that reached one of the two states where contact is
actually exchanged:

- `accepted` — the searcher expressed interest and the **driver accepted**
  (mutual contact unlocked).
- `driver_shared` — the **driver proactively pinged** a searcher, sharing their
  contact (one-way handshake initiated by the driver).

A `pending` interest (expressed but not yet answered) is **not** a connection.

### Went unanswered
A contact request counts as unanswered when it is still `pending` **and the ride
is no longer available**. An hourly cron (`expireRides` → `DELETE FROM rides
WHERE expires_at < NOW()`) physically removes rides once past their `expires_at`
(departure + flexibility + grace), and interests have no FK to rides — so a
genuinely-unanswered request is, in the common case, a `pending` interest whose
**ride row is already gone**. The `expires_at < NOW()` check additionally covers
the brief window between a ride expiring and the next cron tick.

So: a `pending` interest counts when its ride is missing **or** present-but-expired.

This is **derived from current state**, not recorded as an event:
- It self-corrects: if the driver accepts in time, the interest is no longer
  `pending`, so it never counts.
- Cancelled requests are already deleted (cancel performs a row `DELETE`), so
  they never count.
- It is bucketed by the request's `created_at`.

## Architecture

The app already has an established activity-stats pattern: dedicated append-only
event tables (`search_events`, `ride_events`) counted with time-windowed
`COUNT(*)` queries, assembled into `domain.Stats` by `StatRepo.GetStats`, served
by `StatsHandler`, and rendered on the stats page. Both new stats plug into this
pattern.

- **Connections** are discrete events (an accept / a ping happens at a moment),
  so they follow the event-table pattern: a new `connection_events` table with a
  row appended at the moment of connection.
- **Unanswered** is the *absence* of a reply — there is no single moment it
  "happens" — so it is computed live from the `interests` table at read time. No
  new table.

## Components

### 1. Migration `013_connection_events`

> Note: `012_feedback_queue` already exists (committed). Production
> `schema_migrations` is currently at version 11, so `012` and this `013` both
> apply on the next deploy via `migratedb up` in the Procfile.

```sql
-- up
CREATE TABLE connection_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_connection_events_at ON connection_events(connected_at);

-- Backfill historical connections from existing interests. No accepted_at
-- column exists, so created_at is used as the connection moment for historical
-- rows (an approximation for `accepted`; exact for `driver_shared`, which is
-- created already-shared). Going forward the real moment is stamped.
INSERT INTO connection_events (connected_at)
SELECT created_at FROM interests WHERE status IN ('accepted', 'driver_shared');
```

```sql
-- down
DROP TABLE connection_events;
```

The table is deliberately leaner than `search_events`/`ride_events`, which carry
an `(origin, destination)` route that is never displayed. Connections are only
ever surfaced as a count, so route columns are omitted (YAGNI; also avoids the
backfill failing for interests whose ride has since been deleted).

### 2. Recording connections

`stats.sql`:
```sql
-- name: InsertConnectionEvent :exec
INSERT INTO connection_events DEFAULT VALUES;

-- name: GetConnectionEventCounts :one
SELECT
  COUNT(*)                                                            AS all_time,
  COUNT(*) FILTER (WHERE connected_at >= DATE_TRUNC('year',  NOW()))  AS this_year,
  COUNT(*) FILTER (WHERE connected_at >= DATE_TRUNC('month', NOW()))  AS this_month
FROM connection_events;
```

`StatRepository` interface and `StatRepo` gain:
```go
RecordConnection() error
```

Recording happens **only on a genuine state transition**, async and best-effort
(matching how `RecordSearch`/`RecordRide` are fired from the handler layer):

- **Accept** (`AcceptInterest`): record only when the interest actually flips
  `pending → accepted`. The accept path is currently idempotent (a repeat
  `UPDATE ... SET status='accepted'` is a no-op but would still "succeed"), so
  `AcceptInterest.Execute` must report whether it newly accepted (e.g. return a
  bool, or check prior status / rows affected). The handler records only when a
  new acceptance occurred.
- **Ping** (`PingSearcher`): the ping inserts a `driver_shared` interest with
  `ON CONFLICT (ride_id, searcher_phone) DO NOTHING`. A repeated ping inserts no
  row, so recording must be gated on an actual insert (report rows affected).

This guards against a repeated click double-counting.

### 3. Unanswered — derived query

`stats.sql`:
```sql
-- name: GetUnansweredCounts :one
SELECT
  COUNT(*)                                                              AS all_time,
  COUNT(*) FILTER (WHERE i.created_at >= DATE_TRUNC('year',  NOW()))    AS this_year,
  COUNT(*) FILTER (WHERE i.created_at >= DATE_TRUNC('month', NOW()))    AS this_month
FROM interests i
LEFT JOIN rides r ON r.id = i.ride_id
WHERE i.status = 'pending' AND (r.id IS NULL OR r.expires_at < NOW());
```

Left join with `r.id IS NULL OR r.expires_at < NOW()`: the dominant unanswered
case is a `pending` interest whose ride was already deleted by the expiry cron,
so an inner join would wrongly drop exactly the rows we want to count. No
backfill needed — it reads live state.

### 4. Domain + assembly

`domain.Stats` gains two fields:
```go
Connections ActivityCounts `json:"connections"`
Unanswered  ActivityCounts `json:"unanswered"`
```

`StatRepo.GetStats` calls `GetConnectionEventCounts` and `GetUnansweredCounts`
and maps them into the two new `ActivityCounts`, alongside the existing
`Searches` and `RidesPosted`. `StatsHandler` and the `GetStats` usecase are
unchanged (they pass `domain.Stats` through).

### 5. Frontend

The stats page (`frontend/src/routes/stats/+page.svelte`) currently renders a
two-tile grid: **Searches** | **Rides posted**. It becomes a 2×2 grid adding:

- **Connections made** → `stats.connections`
- **Went unanswered** → `stats.unanswered`

New i18n message keys `statsConnections` and `statsUnanswered` are added to the
message files. The plan must confirm the i18n setup's handling of missing keys
(Paraglide typically requires every key to exist in each locale): to keep the
build valid, add the keys to **all** locale files, using the real translation
where trivially known and the English source text as a placeholder otherwise.
Proper non-English translations are a follow-up.

## Testing

Integration tests (`//go:build integration`, real Postgres):

- A connection event is recorded when a driver **accepts** a pending interest.
- A connection event is recorded when a driver **pings** a searcher
  (`driver_shared`).
- A repeated accept / repeated ping does **not** add a second event.
- Expressing interest (`pending`) and cancelling record **no** connection.
- `GetUnansweredCounts`: a `pending` interest on an **expired** ride counts, and
  so does one whose **ride row is gone** (deleted by the expiry cron — the common
  case); a `pending` interest on a **future** ride does not; an `accepted` interest
  does not.
- `GET /api/stats` returns populated `connections` and `unanswered` objects.

Frontend:
- Stats page renders the two new tiles; `svelte-check` passes.

## Out of scope / future

- Per-route breakdown of connections ("top connected routes") — table is a
  count only; route columns intentionally omitted.
- Translating the new stat labels into the non-English locales.
- Distinguishing `accepted` vs `driver_shared` in the connections count (both
  count as one "connection").
