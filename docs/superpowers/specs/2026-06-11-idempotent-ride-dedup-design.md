# Idempotent ride creation (deduplicate duplicate posts)

**Date:** 2026-06-11
**Status:** Approved
**Branch:** `feat/idempotent-ride-dedup`

## Problem

`POST /rides` has no uniqueness guard. The same person can post the same ride
multiple times — most often by accident: a slow network, an impatient re-tap, or
a PWA replaying the request. Every submit inserts a fresh row, cluttering the
feed and re-notifying every matching searcher.

## Goal

Make ride creation **idempotent** on the natural key the user described:
**same Name + Phone + Origin + Destination + departure Time**. Re-submitting an
identical ride must be a safe no-op that returns the *existing* ride rather than
creating a second one.

## Duplicate key

Two rides are duplicates when **all five** match:

| Component        | Comparison                                                        |
| ---------------- | ----------------------------------------------------------------- |
| `phone`          | exact — already normalized by `normalizePhone` in the handler     |
| driver name      | normalized via existing `route_norm()` (folds case/accents/space) |
| `origin_norm`    | existing generated column (`route_norm(origin)`)                  |
| `destination_norm` | existing generated column (`route_norm(destination)`)           |
| `departure_at`   | **exact timestamp** (same instant)                                |

Decisions locked during brainstorming:

- **Name is normalized, not byte-exact.** `route_norm` is the project's single
  source of truth for "same place" comparisons; reusing it means `" Alice "`,
  `alice`, and `Alice` collapse to one driver. Avoids trivially-different names
  defeating the guard.
- **Time is the exact `departure_at` instant.** `09:00` and `09:01` are
  different rides. Flexibility does **not** widen the dedup window.

## Behaviour

- New ride → insert and return `201 Created` (unchanged).
- Duplicate → return the **existing** ride with `200 OK`. The existing row keeps
  its original `id` and `posted_at`.
- On a duplicate, **matching/notification is skipped** — the searchers were
  already notified when the ride was first posted. This is the main functional
  reason the create-vs-existing distinction is threaded back to the use case.

## Enforcement: DB constraint + app handling

Race-safe against concurrent double-submits (the actual cause here), and
idempotent.

### Migration `014_dedup_rides`

Follows the migration-010 pattern (stored generated column + index):

```sql
-- up
ALTER TABLE rides
  ADD COLUMN driver_name_norm text
    GENERATED ALWAYS AS (route_norm(driver_name)) STORED;

CREATE UNIQUE INDEX uq_rides_dedup
  ON rides (phone, driver_name_norm, origin_norm, destination_norm, departure_at);

-- down
DROP INDEX IF EXISTS uq_rides_dedup;
ALTER TABLE rides DROP COLUMN IF EXISTS driver_name_norm;
```

A *stored generated column* (not an expression index) keeps the `ON CONFLICT`
target a plain column list, matching how `origin_norm` / `destination_norm` are
already done. Adding the column rewrites the table, backfilling existing rows in
one pass. `014` is strictly greater than the current head (`013`), so it applies
cleanly in ordered migration (see the prod-migration-drift note: assumes 012/013
are already applied).

> A plain (non-partial) unique index covers the whole table. Expired rides are
> reaped by `DeleteExpiredRides`, so a stale key never lingers to block a
> genuinely-new future ride at the same instant.

### sqlc queries

```sql
-- name: InsertRide :one
INSERT INTO rides (id, driver_name, phone, origin, destination, date,
                   departure_at, flexibility, posted_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (phone, driver_name_norm, origin_norm, destination_norm, departure_at)
  DO NOTHING
RETURNING id;

-- name: GetRideByDedupKey :one
SELECT <ride columns>
FROM rides
WHERE phone = $1
  AND driver_name_norm  = route_norm(sqlc.arg(driver_name)::text)
  AND origin_norm       = route_norm(sqlc.arg(origin)::text)
  AND destination_norm  = route_norm(sqlc.arg(destination)::text)
  AND departure_at      = sqlc.arg(departure_at);
```

`InsertRide` becomes `:one`. On conflict, `DO NOTHING` returns zero rows →
pgx `ErrNoRows`, which the repo treats as "duplicate" and follows up with
`GetRideByDedupKey`.

### Repository

`RideRepository.Save` changes:

```go
// Save inserts the ride. If an identical ride already exists (same
// phone + normalized name + normalized route + exact departure time), no row
// is inserted; the existing ride is returned with created=false.
Save(ride domain.Ride) (saved domain.Ride, created bool, err error)
```

Implementation: call `InsertRide`. If it returns a row → `created=true`, return
the input ride. If `ErrNoRows` → `created=false`, fetch via `GetRideByDedupKey`
and return that canonical row. Both queries run autocommit (read-committed), so
the follow-up select sees the committed conflicting row.

### Use case

```go
saved, created, err := uc.rides.Save(ride)
if err != nil { return domain.Ride{}, err }
if created {
    // existing match/notify loop, unchanged
}
return saved, nil
```

### Handler

`Post` returns `201` when created, `200` when the returned ride was an existing
duplicate. (The use case returns a small flag or the handler compares; chosen
implementation: use case returns `(ride, created, err)` so the handler picks the
status. Minimal: thread `created` through.)

## Testing

- **Use case** (`post_ride_test.go`, mocks): duplicate (`created=false`) ⇒ **no**
  notifications enqueued and the existing ride is returned; new ride ⇒ notifies
  as today. Mock `Save` signature updated.
- **Integration** (`integration_test.go` / `ride_repo_*_test.go`, real DB):
  - Identical `POST` twice ⇒ second is `200`, **same `id`**, exactly one row.
  - Vary one field at a time (phone, name case, route case, departure instant)
    ⇒ confirms normalization folds case/space and that a different instant
    creates a second ride.
  - Concurrent identical double-submit ⇒ exactly one row (race safety).

## Out of scope

- Fuzzy/near-duplicate detection (different wording of the same place beyond
  `route_norm` folding, or "close enough" times). Deliberately excluded — see the
  route-matching-scope note: matching folds accents/case/whitespace only.
- Editing or merging existing rides.
