# Edit / update my rides

**Date:** 2026-06-11
**Status:** Approved
**Branch:** `feat/edit-rides`

## Problem

A driver who posts a ride cannot change it. The only mutation paths are DELETE
and a re-POST — and because the dedup key is
`(phone, driver_name, origin, destination, departure_at)`, changing the route or
time via re-post creates a *new* ride and **loses the interests** people have
expressed (they are tied to the original ride ID). Drivers need to fix a typo'd
place, move the time, or widen flexibility without losing their connections.

## Scope (decided during brainstorming)

- **Editable:** origin, destination, departure time, flexibility.
- **Fixed:** driver name and phone. Phone is the ownership identity; name stays
  as posted.
- **In place:** the edit keeps the same ride ID, so expressed and accepted
  **interests survive**.
- **Notify on edit:** after the change, re-run searcher matching and push
  **newly-matching** searchers — the per-ride notification dedup
  (`UNIQUE(ride_id, request_id)`) means anyone already pinged for this ride is
  not pinged again.
- Derived `date` and `expires_at` recompute from the new departure time;
  `posted_at`, `feedback_given`, `driver_name`, `phone`, `id` are preserved.
- **No past-ride restriction:** any ride the driver owns that still exists can be
  edited (editing a just-passed ride's time effectively re-offers it).

## Backend

### Endpoint

`PUT /api/rides/:id` → `RideHandler.Update`. Body:

```json
{ "phone": "...", "origin": "...", "destination": "...",
  "departure_at": "2030-06-01T09:00:00Z", "flexibility": 30 }
```

`phone` authorizes the edit (mirrors `Delete`, which takes `{phone}` in the body).
Responses: `200` + updated `domain.Ride`; `400` bad body / departure_at;
`403` wrong phone; `404` no such ride; `409` the edit collides with another of
the driver's own rides (dedup-key clash).

### Usecase `UpdateRide`

`Execute(id, phone, origin, destination, departureAt, flexibility) (domain.Ride, error)`:

1. `rides.FindByID(id)` → `ErrNotFound` if missing.
2. `ride.Phone != normalizePhone(phone)` → `ErrUnauthorized`.
3. Apply edits: `Origin`/`Destination` (normalized via `normalizeLocation` in the
   handler), `DepartureAt`, `Flexibility`; recompute `Date` and `ExpiresAt` with
   the same derivation `PostRide` uses.
4. `rides.UpdateByID(ride)` → persists and returns the row; a Postgres unique
   violation (`23505`) on `uq_rides_dedup` maps to `ErrDuplicateRide`.
5. `requests.FindMatching(updated)` → notify newly-enqueued matches (shared
   helper below).

Dependencies mirror `PostRide`: rides, requests, subs, notifQueue, notifier.

### Repository

New `RideRepository.UpdateByID(ride domain.Ride) (domain.Ride, error)`:

```sql
-- name: UpdateRideByID :one
UPDATE rides SET
  origin = $2, destination = $3, date = $4,
  departure_at = $5, flexibility = $6, expires_at = $7
WHERE id = $1
RETURNING <all ride columns incl. *_norm>;
```

The generated `origin_norm`/`destination_norm`/`driver_name_norm` columns
recompute automatically. The repo translates a `*pgconn.PgError` with code
`23505` into `repository.ErrDuplicateRide`.

### Shared notify helper

Today `PostRide` enqueues, then calls `NotifySearcher` **unconditionally**, then
marks sent. `Enqueue` is `ON CONFLICT (ride_id, request_id) DO NOTHING` but
returns only `error`, so it can't tell a new match from an already-handled one.

Change:
- `NotificationQueueRepository.Enqueue(rideID, requestID, searcherPhone) (inserted bool, err error)`
  backed by `EnqueueNotification :execrows` (rows affected = 1 on insert, 0 on
  conflict).
- Extract the loop into a shared helper used by both `PostRide` and `UpdateRide`:

  ```go
  for _, req := range matches {
      inserted, _ := q.Enqueue(ride.ID, req.ID, req.Phone)
      if !inserted { continue } // already handled for this ride
      NotifySearcher(req.Phone, ride, subs, notifier)
      _ = q.MarkSentByRideAndRequest(ride.ID, req.ID)
  }
  ```

On a fresh post no queue rows exist, so every match is `inserted` → identical
behavior to today. On an edit, only genuinely new matches are pushed.

## Frontend

- **`MyRideCard`**: add an **Edit** button (next to Delete) linking to
  `/rides/[id]/edit`.
- **`/rides/[id]/edit/+page.svelte`**: fetch the ride via `api.rides.get(id)`
  (`PublicRide` exposes Origin/Destination/DepartureAt/Flexibility), prefill, and
  use the localStorage phone for authorization.
- Reuse **`RideForm`** with an `editing` prop that: hides the name/phone
  (`ProfileFields`) and the trip-type / return-trip section, relabels the submit
  button (`btnSaveChanges`), and submits via a new
  `api.rides.update(id, { phone, origin, destination, departure_at, flexibility })`.
  On success → navigate to `/my-rides`.
- **`api.ts`**: add `rides.update`; regenerate the typed client via
  `npm run api:generate` (orval) from the updated swagger.
- **i18n:** add `btnEdit`, `editRideTitle`, `btnSaveChanges` to all locale files
  (en, fr, de, el, es, it, nl).

## Testing

- **Usecase unit** (`update_ride_test.go`, mocks): updates the four fields and
  recomputes date/expiry; `404` / `403` / `409`; preserves
  name/phone/posted_at/feedback; notifies a newly-matching searcher but **not**
  one already notified for the ride. Mock `RideRepo.UpdateByID` and the
  `Enqueue (bool, error)` signature.
- **Integration (HTTP)**: `PUT` happy path (fields changed, an existing
  **interest preserved** across the edit), `403` wrong phone, `404` missing,
  `409` collision with another owned ride, and the new-vs-already-notified push
  behavior.
- **Frontend**: a light test of the edit page's `update` call (mirrors
  `feedback.test.ts`).

## Out of scope

- Editing driver name or phone.
- A dedicated edit history / audit trail.
- Notifying already-interested searchers that the ride changed (only
  newly-matching searchers are pushed; the driver keeps existing connections and
  can contact them directly).
