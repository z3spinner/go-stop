# Decoupled Feedback Queue & Ask-on-Delete — Design Spec

## Problem

Post-ride feedback ("did someone come along?") is currently coupled to the `rides`
row, which produces a strong selection bias in the answers:

1. **Manual delete bias.** Deleting a ride is a hard `DELETE FROM rides`. The
   reminder query (`ListRidesPendingFeedback`) reads live `rides`, so a deleted
   ride is never asked. A driver who *found* a passenger is the one most likely to
   delete their ad ("c'est bon, trouvé"), so "yes" answers are preferentially
   stripped out.
2. **Evening-expiry bias.** `expires_at` = midnight after the departure day, and
   `DeleteExpiredRides` runs *before* `SendFeedbackReminders` in the same hourly
   cron tick. An evening ride can expire before any reminder fires.
3. **Recording requires the ride.** `RecordFeedback` does `FindByID(rideID)` and
   checks `ride.Phone` for ownership, so once the ride is gone feedback cannot be
   recorded at all.

Observed in production (logs 06-01 → 06-06): 133 rides created, **59 deleted
manually (44%)**, and of the answers with a logged outcome, all were "drove
alone" — consistent with the biases above.

## Goal

Decouple the feedback opportunity from the ride lifecycle so feedback can be
solicited and recorded even after the ride is deleted or expired, and ask the
question at the moment of deletion for trips that have already departed.

---

## Approach (queue of feedback tasks)

A feedback task becomes a self-contained row in a new `feedback_queue` table,
created when a trip actually reaches its window (not at post time), scheduled to
send one hour after the trip window ends, retried with the same pattern as the
existing notification queue, and deleted once answered or exhausted.

Because the entry is created **at window start, only if the ride still exists**,
cancelling a future ride (deleting before departure) never creates a task — this
is exactly the "silent cancel" behaviour for future deletes, with no cleanup
needed.

### Ride-window semantics

`Flexibility` is in minutes: `Exact 0 / Approximate 30 / Flexible 60`.

- window **start** = `DepartureAt`
- window **end**   = `DepartureAt + Flexibility`
- **send_after**   = window end **+ 1h** = `DepartureAt + Flexibility + 1h`

---

## Data Model

### New table: `feedback_queue` (migration `012_feedback_queue`)

Self-contained — no foreign key to `rides`, so it survives ride deletion and
expiry. Carries everything needed to (a) verify ownership without the ride
(`phone`) and (b) write the stat without the ride (`origin`, `destination`,
`ride_date`).

```sql
CREATE TABLE IF NOT EXISTS feedback_queue (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id       UUID         NOT NULL UNIQUE,
    phone         VARCHAR(100) NOT NULL,
    origin        VARCHAR(200) NOT NULL,
    destination   VARCHAR(200) NOT NULL,
    ride_date     DATE         NOT NULL,
    departure_at  TIMESTAMPTZ  NOT NULL,
    send_after    TIMESTAMPTZ  NOT NULL,   -- window end + 1h
    sent_count    INT          NOT NULL DEFAULT 0,
    last_sent_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fq_send_after ON feedback_queue(send_after);
```

`ride_id` is `UNIQUE` so enqueue is idempotent (`ON CONFLICT DO NOTHING`) and
lookups on the answer path are by `ride_id`.

The `rides.feedback_given` column is **kept** — it still drives the immediate
in-app prompt in "Mes trajets" and lets enqueue skip already-answered rides.

---

## Backend

### Domain — `internal/domain/feedback_task.go`

```go
type FeedbackTask struct {
    ID          string
    RideID      string
    Phone       string
    Origin      string
    Destination string
    RideDate    time.Time
    DepartureAt time.Time
    SendAfter   time.Time
    SentCount   int
    LastSentAt  time.Time // zero = never sent
    CreatedAt   time.Time
}
```

### Repository interface — `internal/boundaries/repository/feedback_queue_repository.go`

```go
type FeedbackQueueRepository interface {
    // EnqueueStartedRides inserts a task for every ride whose window has started
    // (within the bound), is not yet answered, and is not already queued.
    // Idempotent via ride_id UNIQUE.
    EnqueueStartedRides(bound time.Duration, sendDelay time.Duration) error
    // FindDue returns tasks where send_after <= now, retry-eligible (sent_count <
    // max AND (last_sent_at IS NULL OR last_sent_at < now-interval)).
    FindDue(retryAfter time.Time, maxRetries int) ([]domain.FeedbackTask, error)
    FindByRideID(rideID string) (domain.FeedbackTask, error) // ErrNotFound if absent
    MarkSent(id string) error
    DeleteByRideID(rideID string) error
    DeleteExhausted(maxRetries int, ttl time.Duration) error
}
```

> Implementation note: `EnqueueStartedRides` is a single `INSERT ... SELECT ...
> FROM rides r WHERE r.departure_at <= NOW() AND r.departure_at > NOW() - bound
> AND r.feedback_given = false AND NOT EXISTS (SELECT 1 FROM feedback_queue fq
> WHERE fq.ride_id = r.id) ON CONFLICT (ride_id) DO NOTHING`, computing
> `send_after` from `departure_at + flexibility (minutes) + sendDelay`. Flexibility
> is stored on `rides` as an int (minutes), so the SQL is
> `r.departure_at + (r.flexibility * INTERVAL '1 minute') + INTERVAL '1 hour'`.

`bound` = 24h (don't back-fill old rides on first deploy). `sendDelay` = 1h.

### Use cases

**`EnqueueFeedback`** (new) — calls `EnqueueStartedRides(24h, 1h)`. Runs **first**
in the cron, *before* `ExpireRides`, so a ride about to be expired/deleted is
still captured.

**`SendFeedbackReminders`** (rewritten) — reads from `feedback_queue` instead of
`rides`:
```
retryAfter = now - interval
for task in queue.FindDue(retryAfter, maxRetries):
    sendToAll(task.Phone, feedbackMessage(task), subs, notifier)   // url /rides/<id>/feedback
    queue.MarkSent(task.id)
queue.DeleteExhausted(maxRetries, ttl)
```
Uses the same `interval` / `maxRetries` config and `sendToAll` helper as
`RetryNotifications` (mirrored pattern, not the `notification_queue` table).
`ttl` = a safety net (e.g. 7 days) so abandoned tasks are eventually removed.

**`RecordFeedback`** (rewritten — idempotent, decoupled):
```
task, taskErr := queue.FindByRideID(rideID)      // may be ErrNotFound
ride, rideErr := rides.FindByID(rideID)          // may be ErrNotFound

// Ownership: prefer the live ride, else the queue task.
ownerPhone := ride.Phone (if rideErr == nil) else task.Phone (if taskErr == nil)
if ownerPhone == "" -> ErrNotFound            // neither exists → nothing to record
if ownerPhone != phone -> ErrUnauthorized

// Stat data: prefer the live ride, else the queue task.
origin, destination, date := (ride or task)
stats.Save(origin, destination, date, taken)

queue.DeleteByRideID(rideID)                   // cancel any pending reminder (no-op if none)
if rideErr == nil { rides.SetFeedbackGiven(rideID) }
```
This makes every answer path converge: in-app prompt (ride alive), push reminder
(ride may be gone), and the delete flow.

### Cron ordering (`main.go` goroutine — once at startup, then hourly)

The cycle is extracted into a closure called once at boot (so the first
enqueue/send doesn't wait an hour after a deploy) and then on every hourly tick.
It runs in the background goroutine and does not delay the HTTP server.

```
1. enqueueFeedback.Execute()      // NEW — before expiry, captures soon-to-expire rides
2. expireRides.Execute()
3. expireRequests.Execute()
4. sendFeedbackReminders.Execute() // rewritten — queue-driven
5. retryNotifications.Execute()
```

### Removed

- `RideRepository.FindPendingFeedback()` and the `ListRidesPendingFeedback` sqlc
  query (rides-based reminder path). `SetFeedbackGiven` stays.

---

## API

No new endpoints. Existing endpoints are reused:

- `POST /api/rides/:id/feedback` — same contract `{ phone, taken }`; handler
  unchanged, only the `RecordFeedback` use case behind it changes (above).
  - 403 if phone mismatch, 404 if neither ride nor queue task exists, 204 on success.
- `DELETE /api/rides/:id` — unchanged.

The ask-on-delete flow is **frontend-only**: it calls `feedback()` then `del()`.

---

## Frontend (`frontend/`)

### `MyRideCard.svelte` — ask-on-delete

Current `del()` deletes immediately. New behaviour:

- **Past ride, not yet answered** (`isPast && !ride.FeedbackGiven && !fbDone`):
  clicking Delete swaps the delete button for an inline confirm asking the
  feedback question with two buttons. Choosing yes/no:
  1. `await api.rides.feedback(ride.ID, phone, taken)` (best-effort)
  2. `await api.rides.del(ride.ID, phone)`
  Plus a small "annuler" to back out without deleting.
- **Past ride, already answered**, or **future ride**: delete immediately as
  today (future = silent cancel; no queue entry exists yet).

The existing post-departure inline feedback prompt stays as the immediate channel;
answering it still works and (via the rewritten `RecordFeedback`) cancels the
queued reminder.

### `/rides/[id]/feedback/+page.svelte`

Unchanged. Already tolerant of a missing ride (fetches ride for display, ignores
failure). With the decoupled `RecordFeedback`, the answer succeeds even when the
ride has been deleted/expired — this is the push-reminder path.

### i18n (all 7 locales: fr base, en, es, it, de, nl, el)

Two new keys (no `btnCancel` exists today), reusing `feedbackYes`/`feedbackNo`:

| Key | fr | en |
|---|---|---|
| `deleteAskCameAlong` | Avant de supprimer : quelqu'un est-il venu ? | Before deleting: did someone come along? |
| `btnCancel` | Annuler | Cancel |

---

## Testing

- **`record_feedback_test.go`** (rewrite): records via live ride; records via queue
  task when ride is gone; ownership from ride and from task; rejects phone
  mismatch; 404 when neither exists; deletes queue entry on success; idempotent.
- **`enqueue_feedback_test.go`** (new): enqueues started rides within bound; skips
  future rides, already-answered rides, already-queued rides; idempotent on
  re-run; computes `send_after` from flexibility.
- **`send_feedback_reminders_test.go`** (rewrite): only `send_after <= now` tasks
  sent; retry interval and max-retries respected; `sent_count`/`last_sent_at`
  bumped; exhausted/TTL tasks deleted.
- **`MyRideCard.test.ts`**: delete on past unanswered ride shows the question and
  records before deleting; delete on future ride deletes silently; delete on
  answered past ride deletes silently.

---

## Migration strategy

`012_feedback_queue.up.sql` / `.down.sql`, embedded via `migrations.go`, applied
by `cmd/migratedb` in the web boot command (per the postdeploy-ordering note —
boot-time reads must be migrated in the web command, not postdeploy).

No backfill: the 24h enqueue bound means only rides departing within the last day
get a task on first run; older rides are intentionally left alone.

---

## What is NOT in scope

- Recording a distinct "cancelled" outcome for future-ride deletions (silent
  cancel only).
- Language-aware push text (stays French, as today).
- Changing `rides.expires_at` semantics or the rides cleanup.
- Routing feedback pushes through the existing `notification_queue` table.
