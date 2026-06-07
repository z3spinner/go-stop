# Decoupled Feedback Queue & Ask-on-Delete Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make post-ride feedback survive ride deletion/expiry by moving it into a self-contained `feedback_queue` table, and ask "did someone come along?" at delete time for trips that have already departed.

**Architecture:** A feedback task is enqueued when a trip reaches its window (cron, runs before expiry so soon-to-expire rides are captured), pushed to the driver one hour after the window ends, retried with the same pattern as `notification_queue`, and deleted once answered or exhausted. `RecordFeedback` is rewritten to be idempotent and ride-independent (ownership/stats come from the live ride if present, else the queue task) so the in-app prompt, push reminder, and delete flow all converge. Ask-on-delete is frontend-only — it calls the existing `feedback()` then `del()` endpoints.

**Tech Stack:** Go (gin, pgx/v5, sqlc), Postgres, golang-migrate (embedded migrations); SvelteKit 2 + Svelte 5 runes, Paraglide i18n (7 locales), Vitest.

**Spec:** `docs/superpowers/specs/2026-06-07-feedback-queue-decoupling-design.md`

---

## File Structure

**Backend — create:**
- `internal/infrastructure/postgres/sqlc/migrations/012_feedback_queue.up.sql` — table + index
- `internal/infrastructure/postgres/sqlc/migrations/012_feedback_queue.down.sql` — drop
- `internal/infrastructure/postgres/sqlc/queries/sql/feedback_queue.sql` — sqlc queries (source)
- `internal/domain/feedback_task.go` — `FeedbackTask` struct
- `internal/boundaries/repository/feedback_queue_repository.go` — interface
- `internal/infrastructure/postgres/feedback_queue_repo.go` — postgres impl
- `internal/usecase/enqueue_feedback.go` — enqueue use case
- `internal/usecase/enqueue_feedback_test.go` — test
- `internal/usecase/feedback_queue_mock_test.go` — shared test mock for the queue repo

**Backend — modify:**
- `internal/infrastructure/postgres/convert.go` — add `feedbackTaskFromRow`
- `internal/usecase/errors.go` — add `ErrNotFound`
- `internal/usecase/record_feedback.go` — decoupled, idempotent rewrite
- `internal/usecase/record_feedback_test.go` — updated tests
- `internal/usecase/send_feedback_reminders.go` — queue-driven rewrite
- `internal/usecase/send_feedback_reminders_test.go` — rewritten tests
- `internal/boundaries/handler/feedback_handler.go` — map `ErrNotFound` → 404
- `main.go` — wire repo + use cases, reorder cron
- `internal/boundaries/repository/ride_repository.go` — remove `FindPendingFeedback`
- `internal/infrastructure/postgres/ride_repo.go` — remove `FindPendingFeedback`
- `internal/infrastructure/postgres/sqlc/queries/sql/rides.sql` — remove `ListRidesPendingFeedback`
- test mocks dropping `FindPendingFeedback`: `delete_ride_test.go`, `expire_test.go`, `post_request_test.go`, `post_ride_test.go`, `record_feedback_test.go`, `search_rides_test.go`
- regenerated sqlc output: `queries/models.go`, `queries/querier.go`, `queries/feedback_queue.sql.go`, `queries/rides.sql.go`

**Frontend — modify:**
- `frontend/src/messages/{fr,en,es,it,de,nl,el}.json` — `deleteAskCameAlong`, `btnCancel`
- `frontend/src/lib/components/rides/MyRideCard.svelte` — ask-on-delete
- `frontend/src/lib/components/rides/MyRideCard.test.ts` — tests

---

## Task 1: Migration — `feedback_queue` table

**Files:**
- Create: `internal/infrastructure/postgres/sqlc/migrations/012_feedback_queue.up.sql`
- Create: `internal/infrastructure/postgres/sqlc/migrations/012_feedback_queue.down.sql`

- [ ] **Step 1: Write the up migration**

`012_feedback_queue.up.sql`:
```sql
-- A self-contained queue of post-ride feedback tasks. Decoupled from `rides`
-- (no FK) so a task survives ride deletion and expiry. Carries `phone` for the
-- ownership check and origin/destination/ride_date so the stat can be written
-- without the ride. Created at window start; pushed at send_after; retried via
-- sent_count/last_sent_at; deleted once answered or exhausted.
CREATE TABLE IF NOT EXISTS feedback_queue (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id       UUID         NOT NULL UNIQUE,
    phone         VARCHAR(100) NOT NULL,
    origin        VARCHAR(200) NOT NULL,
    destination   VARCHAR(200) NOT NULL,
    ride_date     DATE         NOT NULL,
    departure_at  TIMESTAMPTZ  NOT NULL,
    send_after    TIMESTAMPTZ  NOT NULL,
    sent_count    INT          NOT NULL DEFAULT 0,
    last_sent_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fq_send_after ON feedback_queue(send_after);
```

- [ ] **Step 2: Write the down migration**

`012_feedback_queue.down.sql`:
```sql
DROP TABLE IF EXISTS feedback_queue;
```

- [ ] **Step 3: Verify the migration files are embedded**

Run: `go build ./internal/infrastructure/postgres/sqlc/migrations/`
Expected: builds cleanly (the `//go:embed *.up.sql *.down.sql` glob already picks up the new files; no code change needed).

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/postgres/sqlc/migrations/012_feedback_queue.up.sql internal/infrastructure/postgres/sqlc/migrations/012_feedback_queue.down.sql
git commit -m "feat(db): add feedback_queue migration (012)"
```

---

## Task 2: sqlc queries + generate

**Files:**
- Create: `internal/infrastructure/postgres/sqlc/queries/sql/feedback_queue.sql`
- Generated (do not hand-edit): `queries/models.go`, `queries/querier.go`, `queries/feedback_queue.sql.go`

- [ ] **Step 1: Write the query source**

`internal/infrastructure/postgres/sqlc/queries/sql/feedback_queue.sql`:
```sql
-- name: EnqueueStartedRides :exec
-- Inserts a feedback task for every ride whose window has started (departure in
-- the past, but within the bound), that hasn't been answered, and isn't already
-- queued. send_after = window end (departure + flexibility minutes) + 1 hour.
-- Idempotent via ride_id UNIQUE.
INSERT INTO feedback_queue (ride_id, phone, origin, destination, ride_date, departure_at, send_after)
SELECT r.id, r.phone, r.origin, r.destination, r.date, r.departure_at,
       r.departure_at + (r.flexibility * INTERVAL '1 minute') + INTERVAL '1 hour'
FROM rides r
WHERE r.departure_at <= NOW()
  AND r.departure_at > sqlc.arg(window_start_after)::timestamptz
  AND r.feedback_given = false
  AND NOT EXISTS (SELECT 1 FROM feedback_queue fq WHERE fq.ride_id = r.id)
ON CONFLICT (ride_id) DO NOTHING;

-- name: FindDueFeedback :many
-- Tasks past their send time and still retry-eligible.
SELECT id, ride_id, phone, origin, destination, ride_date, departure_at, send_after, sent_count, last_sent_at, created_at
FROM feedback_queue
WHERE send_after <= NOW()
  AND sent_count < sqlc.arg(max_retries)::int
  AND (last_sent_at IS NULL OR last_sent_at < sqlc.arg(retry_before)::timestamptz)
ORDER BY send_after ASC;

-- name: GetFeedbackByRideID :one
SELECT id, ride_id, phone, origin, destination, ride_date, departure_at, send_after, sent_count, last_sent_at, created_at
FROM feedback_queue
WHERE ride_id = $1;

-- name: MarkFeedbackSent :exec
UPDATE feedback_queue
SET sent_count = sent_count + 1,
    last_sent_at = NOW()
WHERE id = $1;

-- name: DeleteFeedbackByRideID :execrows
-- :execrows returns the number of rows deleted, used as the "claim" signal in
-- RecordFeedback: only the caller that actually deletes the row records the stat.
DELETE FROM feedback_queue WHERE ride_id = $1;

-- name: DeleteExhaustedFeedback :exec
DELETE FROM feedback_queue
WHERE sent_count >= sqlc.arg(max_retries)::int
   OR created_at < sqlc.arg(ttl_before)::timestamptz;
```

- [ ] **Step 2: Generate sqlc code**

Run: `make sqlc`
Expected: regenerates `internal/infrastructure/postgres/sqlc/queries/*.go` with no errors. New symbols become available:
- `queries.FeedbackQueue` (model struct in `models.go`)
- `queries.EnqueueStartedRides(ctx, windowStartAfter pgtype.Timestamptz)`
- `queries.FindDueFeedback(ctx, FindDueFeedbackParams{MaxRetries int32, RetryBefore pgtype.Timestamptz})` → `[]queries.FeedbackQueue`
- `queries.GetFeedbackByRideID(ctx, rideID pgtype.UUID)` → `queries.FeedbackQueue`
- `queries.MarkFeedbackSent(ctx, id pgtype.UUID)`
- `queries.DeleteFeedbackByRideID(ctx, rideID pgtype.UUID) (int64, error)` — note: returns rows-affected (`:execrows`)
- `queries.DeleteExhaustedFeedback(ctx, DeleteExhaustedFeedbackParams{MaxRetries int32, TtlBefore pgtype.Timestamptz})`

- [ ] **Step 3: Verify it builds**

Run: `go build ./...`
Expected: builds cleanly.

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/postgres/sqlc/queries/
git commit -m "feat(db): sqlc queries for feedback_queue"
```

---

## Task 3: Domain type + repository interface

**Files:**
- Create: `internal/domain/feedback_task.go`
- Create: `internal/boundaries/repository/feedback_queue_repository.go`

- [ ] **Step 1: Write the domain type**

`internal/domain/feedback_task.go`:
```go
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "time"

// FeedbackTask is a queued post-ride feedback request. It is self-contained:
// it carries the owner's phone (for the ownership check) and origin/destination/
// ride_date (to write the stat) so feedback can be solicited and recorded even
// after the ride row has been deleted or expired.
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

- [ ] **Step 2: Write the repository interface**

`internal/boundaries/repository/feedback_queue_repository.go`:
```go
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

import (
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

type FeedbackQueueRepository interface {
	// EnqueueStartedRides inserts a task for every ride whose window has started
	// (departure_at in the past but after windowStartAfter), is not yet answered,
	// and is not already queued. Idempotent via ride_id UNIQUE.
	EnqueueStartedRides(windowStartAfter time.Time) error

	// FindDue returns tasks past send_after that are still retry-eligible
	// (sent_count < maxRetries AND (last_sent_at IS NULL OR last_sent_at < retryAfter)).
	FindDue(retryAfter time.Time, maxRetries int) ([]domain.FeedbackTask, error)

	// FindByRideID returns the task for a ride; a non-nil error means absent.
	FindByRideID(rideID string) (domain.FeedbackTask, error)

	// MarkSent increments sent_count and sets last_sent_at.
	MarkSent(id string) error

	// DeleteByRideID removes the task for a ride and reports whether a row was
	// actually deleted. The bool is the concurrency "claim": when feedback is
	// recorded for a ride whose row is gone, only the caller that deletes the
	// task (returns true) writes the stat; concurrent callers get false and no-op.
	DeleteByRideID(rideID string) (bool, error)

	// DeleteExhausted removes tasks that hit maxRetries or are older than ttl.
	DeleteExhausted(maxRetries int, ttl time.Duration) error
}
```

- [ ] **Step 3: Verify it builds**

Run: `go build ./...`
Expected: builds cleanly.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/feedback_task.go internal/boundaries/repository/feedback_queue_repository.go
git commit -m "feat(feedback): domain FeedbackTask + queue repository interface"
```

---

## Task 4: Postgres `FeedbackQueueRepo`

**Files:**
- Create: `internal/infrastructure/postgres/feedback_queue_repo.go`
- Modify: `internal/infrastructure/postgres/convert.go`

- [ ] **Step 1: Add the row converter**

Append to `internal/infrastructure/postgres/convert.go` (after the existing converters):
```go
func feedbackTaskFromRow(q queries.FeedbackQueue) domain.FeedbackTask {
	return domain.FeedbackTask{
		ID:          uuidTo(q.ID),
		RideID:      uuidTo(q.RideID),
		Phone:       q.Phone,
		Origin:      q.Origin,
		Destination: q.Destination,
		RideDate:    dateTo(q.RideDate),
		DepartureAt: tsTo(q.DepartureAt),
		SendAfter:   tsTo(q.SendAfter),
		SentCount:   int(q.SentCount),
		LastSentAt:  tsTo(q.LastSentAt),
		CreatedAt:   tsTo(q.CreatedAt),
	}
}
```

- [ ] **Step 2: Write the repo**

`internal/infrastructure/postgres/feedback_queue_repo.go`:
```go
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type FeedbackQueueRepo struct{ q *queries.Queries }

func NewFeedbackQueueRepo(pool *pgxpool.Pool) *FeedbackQueueRepo {
	return &FeedbackQueueRepo{q: queries.New(pool)}
}

func (r *FeedbackQueueRepo) EnqueueStartedRides(windowStartAfter time.Time) error {
	return r.q.EnqueueStartedRides(context.Background(), tsFrom(windowStartAfter))
}

func (r *FeedbackQueueRepo) FindDue(retryAfter time.Time, maxRetries int) ([]domain.FeedbackTask, error) {
	rows, err := r.q.FindDueFeedback(context.Background(), queries.FindDueFeedbackParams{
		MaxRetries:  int32(maxRetries),
		RetryBefore: tsFrom(retryAfter),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.FeedbackTask, len(rows))
	for i, row := range rows {
		out[i] = feedbackTaskFromRow(row)
	}
	return out, nil
}

func (r *FeedbackQueueRepo) FindByRideID(rideID string) (domain.FeedbackTask, error) {
	row, err := r.q.GetFeedbackByRideID(context.Background(), uuidFrom(rideID))
	if err != nil {
		return domain.FeedbackTask{}, err
	}
	return feedbackTaskFromRow(row), nil
}

func (r *FeedbackQueueRepo) MarkSent(id string) error {
	return r.q.MarkFeedbackSent(context.Background(), uuidFrom(id))
}

func (r *FeedbackQueueRepo) DeleteByRideID(rideID string) (bool, error) {
	n, err := r.q.DeleteFeedbackByRideID(context.Background(), uuidFrom(rideID))
	return n > 0, err
}

func (r *FeedbackQueueRepo) DeleteExhausted(maxRetries int, ttl time.Duration) error {
	return r.q.DeleteExhaustedFeedback(context.Background(), queries.DeleteExhaustedFeedbackParams{
		MaxRetries: int32(maxRetries),
		TtlBefore:  tsFrom(time.Now().Add(-ttl)),
	})
}
```

- [ ] **Step 3: Verify the repo satisfies the interface and builds**

Run: `go build ./...`
Expected: builds cleanly. (If you want a compile-time interface assertion, it is added at wiring time in Task 9 via the `main.go` usage.)

- [ ] **Step 4: Commit**

```bash
git add internal/infrastructure/postgres/feedback_queue_repo.go internal/infrastructure/postgres/convert.go
git commit -m "feat(feedback): postgres FeedbackQueueRepo"
```

---

## Task 5: Shared test mock + `EnqueueFeedback` use case

**Files:**
- Create: `internal/usecase/feedback_queue_mock_test.go`
- Create: `internal/usecase/enqueue_feedback.go`
- Create: `internal/usecase/enqueue_feedback_test.go`

- [ ] **Step 1: Write the shared queue mock**

`internal/usecase/feedback_queue_mock_test.go`:
```go
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"errors"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

// mockFeedbackQueue is the shared in-memory FeedbackQueueRepository for usecase tests.
type mockFeedbackQueue struct {
	enqueueCalled     bool
	enqueueWindowArg  time.Time
	enqueueErr        error
	due               []domain.FeedbackTask
	byRideID          map[string]domain.FeedbackTask
	claimedRides      map[string]bool // ride IDs already claimed via DeleteByRideID
	marked            []string
	deletedByRideID   []string
	deleteExhaustedOK bool
}

func (m *mockFeedbackQueue) EnqueueStartedRides(windowStartAfter time.Time) error {
	m.enqueueCalled = true
	m.enqueueWindowArg = windowStartAfter
	return m.enqueueErr
}
func (m *mockFeedbackQueue) FindDue(retryAfter time.Time, maxRetries int) ([]domain.FeedbackTask, error) {
	return m.due, nil
}
func (m *mockFeedbackQueue) FindByRideID(rideID string) (domain.FeedbackTask, error) {
	if m.byRideID != nil {
		if t, ok := m.byRideID[rideID]; ok {
			return t, nil
		}
	}
	return domain.FeedbackTask{}, errors.New("not found")
}
func (m *mockFeedbackQueue) MarkSent(id string) error {
	m.marked = append(m.marked, id)
	return nil
}
// DeleteByRideID models the atomic claim: the first call for a ride that has a
// queued task returns true (claimed); later calls return false. byRideID is left
// intact so FindByRideID still serves the phone check, modelling the real race
// where concurrent callers both read the task before either claims it.
func (m *mockFeedbackQueue) DeleteByRideID(rideID string) (bool, error) {
	m.deletedByRideID = append(m.deletedByRideID, rideID)
	if m.byRideID == nil {
		return false, nil
	}
	if _, ok := m.byRideID[rideID]; !ok {
		return false, nil
	}
	if m.claimedRides[rideID] {
		return false, nil
	}
	if m.claimedRides == nil {
		m.claimedRides = map[string]bool{}
	}
	m.claimedRides[rideID] = true
	return true, nil
}
func (m *mockFeedbackQueue) DeleteExhausted(maxRetries int, ttl time.Duration) error {
	m.deleteExhaustedOK = true
	return nil
}
```

- [ ] **Step 2: Write the failing test**

`internal/usecase/enqueue_feedback_test.go`:
```go
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestEnqueueFeedback_CallsRepoWith24hBound(t *testing.T) {
	q := &mockFeedbackQueue{}
	uc := usecase.NewEnqueueFeedback(q)

	before := time.Now()
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()

	if !q.enqueueCalled {
		t.Fatal("expected EnqueueStartedRides to be called")
	}
	// windowStartAfter should be ~24h before now.
	wantLo := before.Add(-usecase.FeedbackEnqueueBound)
	wantHi := after.Add(-usecase.FeedbackEnqueueBound)
	if q.enqueueWindowArg.Before(wantLo) || q.enqueueWindowArg.After(wantHi) {
		t.Errorf("windowStartAfter %v not within [%v, %v]", q.enqueueWindowArg, wantLo, wantHi)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/usecase/ -run TestEnqueueFeedback -v`
Expected: FAIL — `usecase.NewEnqueueFeedback` / `usecase.FeedbackEnqueueBound` undefined.

- [ ] **Step 4: Write the use case**

`internal/usecase/enqueue_feedback.go`:
```go
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

// FeedbackEnqueueBound limits enqueue to rides whose window started within the
// last day, so old rides are not back-filled on first deploy.
const FeedbackEnqueueBound = 24 * time.Hour

type EnqueueFeedback struct {
	queue repository.FeedbackQueueRepository
}

func NewEnqueueFeedback(queue repository.FeedbackQueueRepository) *EnqueueFeedback {
	return &EnqueueFeedback{queue: queue}
}

func (uc *EnqueueFeedback) Execute() error {
	return uc.queue.EnqueueStartedRides(time.Now().Add(-FeedbackEnqueueBound))
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/usecase/ -run TestEnqueueFeedback -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/usecase/feedback_queue_mock_test.go internal/usecase/enqueue_feedback.go internal/usecase/enqueue_feedback_test.go
git commit -m "feat(feedback): EnqueueFeedback use case"
```

---

## Task 6: Rewrite `RecordFeedback` (decoupled, idempotent, claim-guarded) + `ErrNotFound` + handler 404

This task also introduces the **double-record guard**: a live ride is claimed by flipping `feedback_given` false→true conditionally (`ClaimFeedback`), and a gone ride is claimed by deleting its queue row (`DeleteByRideID` returning rows-affected). Only the caller that wins the claim writes the `ride_stats` row, so a double-click or an in-app answer racing the push answer can't produce duplicate stats. The claim happens *before* the stat write.

**Files:**
- Modify: `internal/usecase/errors.go`
- Modify: `internal/infrastructure/postgres/sqlc/queries/sql/rides.sql` (SetRideFeedbackGiven → ClaimRideFeedback)
- Modify: `internal/boundaries/repository/ride_repository.go` (SetFeedbackGiven → ClaimFeedback)
- Modify: `internal/infrastructure/postgres/ride_repo.go` (method swap)
- Modify test mocks: `internal/usecase/{delete_ride,expire,post_request,post_ride,search_rides}_test.go`
- Modify: `internal/usecase/record_feedback.go`
- Modify: `internal/usecase/record_feedback_test.go`
- Modify: `internal/boundaries/handler/feedback_handler.go`
- Regenerated: `queries/querier.go`, `queries/rides.sql.go`

- [ ] **Step 1: Add the `ErrNotFound` sentinel**

`internal/usecase/errors.go` — add alongside `ErrUnauthorized`:
```go
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import "errors"

var ErrUnauthorized = errors.New("unauthorized")
var ErrNotFound = errors.New("not found")
```

- [ ] **Step 1b: Replace `SetFeedbackGiven` with a conditional `ClaimFeedback` (live-ride double-record guard)**

`SetFeedbackGiven` set the flag unconditionally; `ClaimFeedback` only flips it when still false and reports whether *this* call won, so exactly one concurrent caller records the stat.

(a) In `internal/infrastructure/postgres/sqlc/queries/sql/rides.sql`, replace the `SetRideFeedbackGiven` query with:
```sql
-- name: ClaimRideFeedback :execrows
UPDATE rides SET feedback_given = true WHERE id = $1 AND feedback_given = false;
```

(b) Regenerate: `make sqlc` → generates `queries.ClaimRideFeedback(ctx, id pgtype.UUID) (int64, error)` (replaces `SetRideFeedbackGiven`).

(c) In `internal/boundaries/repository/ride_repository.go:28`, replace `SetFeedbackGiven(id string) error` with:
```go
	// ClaimFeedback flips feedback_given false→true and reports whether this call
	// performed the flip (true = caller won the claim and should record the stat).
	ClaimFeedback(id string) (bool, error)
```

(d) In `internal/infrastructure/postgres/ride_repo.go` (~line 184), replace the `SetFeedbackGiven` method with:
```go
func (r *RideRepo) ClaimFeedback(id string) (bool, error) {
	n, err := r.q.ClaimRideFeedback(context.Background(), uuidFrom(id))
	return n == 1, err
}
```

(e) In each of these test mocks, replace the `SetFeedbackGiven(string) error { return nil }` line with `ClaimFeedback(string) (bool, error) { return true, nil }`:
- `internal/usecase/delete_ride_test.go:52`
- `internal/usecase/expire_test.go:44`
- `internal/usecase/post_request_test.go:46`
- `internal/usecase/post_ride_test.go:58`
- `internal/usecase/search_rides_test.go:44`

(The `mockRideRepoFeedback` in `record_feedback_test.go` is updated in Step 2; the `mockRideRepoPendingFeedback` in `send_feedback_reminders_test.go` is removed in Task 7.)

- [ ] **Step 2: Update the tests (failing)**

First update the `mockRideRepoFeedback` definition in this file: add a `claimed map[string]bool` field, and replace its `SetFeedbackGiven` method with a stateful `ClaimFeedback` (so the double-submit test sees the second claim fail):
```go
type mockRideRepoFeedback struct {
	rides       map[string]domain.Ride
	feedbackSet []string
	claimed     map[string]bool
}

func (m *mockRideRepoFeedback) ClaimFeedback(id string) (bool, error) {
	if m.claimed[id] {
		return false, nil
	}
	if m.claimed == nil {
		m.claimed = map[string]bool{}
	}
	m.claimed[id] = true
	m.feedbackSet = append(m.feedbackSet, id)
	return true, nil
}
```
(Keep the `FindPendingFeedback` stub for now — Task 8 removes it. Keep `mockStatRepo`.)

Then replace the test functions. Every `usecase.NewRecordFeedback(rides, stats)` call becomes `usecase.NewRecordFeedback(rides, stats, queue)`. The cases:

```go
func TestRecordFeedback_SavesStatAndMarksFeedbackGiven(t *testing.T) {
	rides := &mockRideRepoFeedback{
		rides: map[string]domain.Ride{
			"ride-1": {
				ID: "ride-1", Phone: "555-0001",
				Origin: "Saillans", Destination: "Crest",
				Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
				DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	stats := &mockStatRepo{}
	queue := &mockFeedbackQueue{}

	uc := usecase.NewRecordFeedback(rides, stats, queue)
	if err := uc.Execute("ride-1", "555-0001", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats.saved) != 1 || !stats.saved[0].taken || stats.saved[0].origin != "Saillans" {
		t.Errorf("unexpected stat saved: %+v", stats.saved)
	}
	if len(rides.feedbackSet) != 1 || rides.feedbackSet[0] != "ride-1" {
		t.Error("expected feedback_given set on ride-1")
	}
	if len(queue.deletedByRideID) != 1 || queue.deletedByRideID[0] != "ride-1" {
		t.Error("expected queue entry for ride-1 to be deleted")
	}
}

func TestRecordFeedback_RecordsViaQueueWhenRideGone(t *testing.T) {
	rides := &mockRideRepoFeedback{rides: map[string]domain.Ride{}} // ride deleted/expired
	stats := &mockStatRepo{}
	queue := &mockFeedbackQueue{byRideID: map[string]domain.FeedbackTask{
		"ride-9": {RideID: "ride-9", Phone: "555-0001", Origin: "Die", Destination: "Crest",
			RideDate: time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC)},
	}}

	uc := usecase.NewRecordFeedback(rides, stats, queue)
	if err := uc.Execute("ride-9", "555-0001", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats.saved) != 1 || stats.saved[0].origin != "Die" {
		t.Errorf("expected stat from queue task, got %+v", stats.saved)
	}
	if len(rides.feedbackSet) != 0 {
		t.Error("must not set feedback_given when the ride is gone")
	}
	if len(queue.deletedByRideID) != 1 || queue.deletedByRideID[0] != "ride-9" {
		t.Error("expected queue entry for ride-9 to be deleted")
	}
}

func TestRecordFeedback_SavesNegativeFeedback(t *testing.T) {
	rides := &mockRideRepoFeedback{rides: map[string]domain.Ride{
		"ride-2": {ID: "ride-2", Phone: "555-0001", Origin: "A", Destination: "B", Date: time.Now()},
	}}
	stats := &mockStatRepo{}
	uc := usecase.NewRecordFeedback(rides, stats, &mockFeedbackQueue{})
	if err := uc.Execute("ride-2", "555-0001", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.saved[0].taken {
		t.Error("expected taken=false")
	}
}

func TestRecordFeedback_RejectsWrongPhone(t *testing.T) {
	rides := &mockRideRepoFeedback{rides: map[string]domain.Ride{"ride-1": {ID: "ride-1", Phone: "555-0001"}}}
	stats := &mockStatRepo{}
	uc := usecase.NewRecordFeedback(rides, stats, &mockFeedbackQueue{})
	if err := uc.Execute("ride-1", "555-9999", true); !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
	if len(stats.saved) != 0 {
		t.Error("should not save stat on unauthorized")
	}
}

func TestRecordFeedback_RejectsWrongPhoneViaQueue(t *testing.T) {
	rides := &mockRideRepoFeedback{rides: map[string]domain.Ride{}}
	stats := &mockStatRepo{}
	queue := &mockFeedbackQueue{byRideID: map[string]domain.FeedbackTask{
		"ride-9": {RideID: "ride-9", Phone: "555-0001", Origin: "Die", Destination: "Crest"},
	}}
	uc := usecase.NewRecordFeedback(rides, stats, queue)
	if err := uc.Execute("ride-9", "555-9999", true); !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestRecordFeedback_ReturnsNotFoundWhenNeitherExists(t *testing.T) {
	rides := &mockRideRepoFeedback{rides: map[string]domain.Ride{}}
	stats := &mockStatRepo{}
	uc := usecase.NewRecordFeedback(rides, stats, &mockFeedbackQueue{})
	if err := uc.Execute("nonexistent", "555-0001", true); !errors.Is(err, usecase.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRecordFeedback_DoubleSubmitLiveRideRecordsOnce(t *testing.T) {
	rides := &mockRideRepoFeedback{rides: map[string]domain.Ride{
		"ride-1": {ID: "ride-1", Phone: "p", Origin: "A", Destination: "B", Date: time.Now()},
	}}
	stats := &mockStatRepo{}
	uc := usecase.NewRecordFeedback(rides, stats, &mockFeedbackQueue{})
	_ = uc.Execute("ride-1", "p", true)
	_ = uc.Execute("ride-1", "p", true) // second call loses the claim
	if len(stats.saved) != 1 {
		t.Errorf("expected exactly 1 stat, got %d", len(stats.saved))
	}
}

func TestRecordFeedback_DoubleSubmitGoneRideRecordsOnce(t *testing.T) {
	rides := &mockRideRepoFeedback{rides: map[string]domain.Ride{}}
	stats := &mockStatRepo{}
	queue := &mockFeedbackQueue{byRideID: map[string]domain.FeedbackTask{
		"ride-9": {RideID: "ride-9", Phone: "p", Origin: "Die", Destination: "Crest"},
	}}
	uc := usecase.NewRecordFeedback(rides, stats, queue)
	_ = uc.Execute("ride-9", "p", true)
	_ = uc.Execute("ride-9", "p", true) // second call loses the claim (DeleteByRideID returns false)
	if len(stats.saved) != 1 {
		t.Errorf("expected exactly 1 stat, got %d", len(stats.saved))
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/usecase/ -run TestRecordFeedback -v`
Expected: FAIL (compile error) — the still-old `record_feedback.go` calls `uc.rides.SetFeedbackGiven` (now removed) and `NewRecordFeedback` takes 2 args, not 3. Both are fixed in Step 4.

- [ ] **Step 4: Rewrite the use case**

Replace `internal/usecase/record_feedback.go` with:
```go
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"log"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

type RecordFeedback struct {
	rides repository.RideRepository
	stats repository.StatRepository
	queue repository.FeedbackQueueRepository
}

func NewRecordFeedback(
	rides repository.RideRepository,
	stats repository.StatRepository,
	queue repository.FeedbackQueueRepository,
) *RecordFeedback {
	return &RecordFeedback{rides: rides, stats: stats, queue: queue}
}

// Execute records the driver's answer to "did someone come along?". It is
// idempotent and ride-independent: ownership and route/date come from the live
// ride if it still exists, otherwise from the queued task — so the in-app
// prompt, the push reminder (ride may be gone), and the delete flow all converge.
// The "claim" (conditional feedback_given flip, or queue-row delete) happens
// before the stat write, so only one concurrent caller records a stat.
func (uc *RecordFeedback) Execute(rideID, phone string, taken bool) error {
	if ride, err := uc.rides.FindByID(rideID); err == nil {
		// Ride still exists: verify ownership, then claim by flipping feedback_given.
		if ride.Phone != phone {
			return ErrUnauthorized
		}
		claimed, err := uc.rides.ClaimFeedback(rideID)
		if err != nil {
			return err
		}
		// Cancel any pending push reminder (idempotent; safe even if we lost the claim).
		_, _ = uc.queue.DeleteByRideID(rideID)
		if !claimed {
			return nil // already answered — idempotent no-op
		}
		return uc.record(rideID, ride.Origin, ride.Destination, ride.Date, taken)
	}

	// Ride gone (deleted/expired): fall back to the queued task.
	task, err := uc.queue.FindByRideID(rideID)
	if err != nil {
		return ErrNotFound
	}
	if task.Phone != phone {
		return ErrUnauthorized
	}
	// Claim by deleting the queue row; only the caller that deletes it records.
	claimed, err := uc.queue.DeleteByRideID(rideID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil // already answered — idempotent no-op
	}
	return uc.record(rideID, task.Origin, task.Destination, task.RideDate, taken)
}

func (uc *RecordFeedback) record(rideID, origin, destination string, date time.Time, taken bool) error {
	if err := uc.stats.Save(origin, destination, date, taken); err != nil {
		return err
	}
	// Logged explicitly so yes/no is visible in the logs — the HTTP access log
	// only shows the path, and `taken` travels in the request body.
	outcome := "drove_alone"
	if taken {
		outcome = "shared"
	}
	log.Printf("ride feedback ride=%s outcome=%s route=%q->%q", rideID, outcome, origin, destination)
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/usecase/ -run TestRecordFeedback -v`
Expected: PASS (all 8 cases, including the two double-submit guards).

- [ ] **Step 6: Map `ErrNotFound` → 404 in the handler**

In `internal/boundaries/handler/feedback_handler.go`, inside `Post`, add a not-found branch before the 500 fallback:
```go
	if err := h.recordFeedback.Execute(c.Param("id"), normalizePhone(req.Phone), req.Taken); err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		if errors.Is(err, usecase.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
```

- [ ] **Step 7: Verify build (main.go will not compile yet — that's fine for the package build of usecase/handler)**

Run: `go build ./internal/...`
Expected: builds cleanly (only `main.go` wiring is stale; fixed in Task 9).

- [ ] **Step 8: Commit**

```bash
git add internal/usecase/errors.go internal/usecase/record_feedback.go internal/usecase/record_feedback_test.go \
  internal/boundaries/handler/feedback_handler.go \
  internal/boundaries/repository/ride_repository.go internal/infrastructure/postgres/ride_repo.go \
  internal/infrastructure/postgres/sqlc/queries/ \
  internal/usecase/delete_ride_test.go internal/usecase/expire_test.go internal/usecase/post_request_test.go \
  internal/usecase/post_ride_test.go internal/usecase/search_rides_test.go
git commit -m "feat(feedback): decouple RecordFeedback + claim-based double-record guard"
```

---

## Task 7: Rewrite `SendFeedbackReminders` (queue-driven)

**Files:**
- Modify: `internal/usecase/send_feedback_reminders.go`
- Modify: `internal/usecase/send_feedback_reminders_test.go`

- [ ] **Step 1: Rewrite the tests (failing)**

Replace `internal/usecase/send_feedback_reminders_test.go` entirely (drops the `mockRideRepoPendingFeedback`; uses the shared `mockFeedbackQueue`, plus `mockSubRepo` from `post_ride_test.go` and `mockNotifier` from `notify_test.go`):
```go
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestSendFeedbackReminders_SendsPushToDriver(t *testing.T) {
	queue := &mockFeedbackQueue{due: []domain.FeedbackTask{
		{ID: "fq-1", RideID: "ride-1", Phone: "555-0001", Origin: "Saillans", Destination: "Crest",
			DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC)},
	}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0001": {Phone: "555-0001", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackReminders(queue, subs, n, 2, 3)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !n.called {
		t.Error("expected push notification to be sent")
	}
	if n.lastMsg.URL != "/rides/ride-1/feedback" {
		t.Errorf("expected URL /rides/ride-1/feedback, got %s", n.lastMsg.URL)
	}
	if len(queue.marked) != 1 || queue.marked[0] != "fq-1" {
		t.Error("expected task fq-1 marked sent")
	}
	if !queue.deleteExhaustedOK {
		t.Error("expected DeleteExhausted to be called")
	}
}

func TestSendFeedbackReminders_SkipsIfNoSubscription(t *testing.T) {
	queue := &mockFeedbackQueue{due: []domain.FeedbackTask{
		{ID: "fq-1", RideID: "ride-1", Phone: "555-no-sub"},
	}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackReminders(queue, subs, n, 2, 3)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send when driver has no subscription")
	}
}

func TestSendFeedbackReminders_NoDueTasks_NoNotifications(t *testing.T) {
	queue := &mockFeedbackQueue{due: []domain.FeedbackTask{}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackReminders(queue, subs, n, 2, 3)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send any notification with no due tasks")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/usecase/ -run TestSendFeedbackReminders -v`
Expected: FAIL — `NewSendFeedbackReminders` signature mismatch (compile error).

- [ ] **Step 3: Rewrite the use case**

Replace `internal/usecase/send_feedback_reminders.go` with:
```go
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"fmt"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// feedbackTTL bounds how long an unanswered task lingers before cleanup.
const feedbackTTL = 7 * 24 * time.Hour

type SendFeedbackReminders struct {
	queue      repository.FeedbackQueueRepository
	subs       repository.SubscriptionRepository
	notifier   notification.Notifier
	interval   time.Duration
	maxRetries int
}

func NewSendFeedbackReminders(
	queue repository.FeedbackQueueRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
	intervalHours, maxRetries int,
) *SendFeedbackReminders {
	if intervalHours <= 0 {
		intervalHours = DefaultRetryIntervalHours
	}
	if maxRetries <= 0 {
		maxRetries = DefaultMaxRetries
	}
	return &SendFeedbackReminders{
		queue:      queue,
		subs:       subs,
		notifier:   notifier,
		interval:   time.Duration(intervalHours) * time.Hour,
		maxRetries: maxRetries,
	}
}

func (uc *SendFeedbackReminders) Execute() error {
	retryAfter := time.Now().Add(-uc.interval)
	due, err := uc.queue.FindDue(retryAfter, uc.maxRetries)
	if err != nil {
		return err
	}
	for _, task := range due {
		sendToAll(task.Phone, domain.Message{
			Title:       "Votre trajet est-il parti avec des passagers ?",
			Body:        fmt.Sprintf("%s → %s", task.Origin, task.Destination),
			URL:         "/rides/" + task.RideID + "/feedback",
			Phone:       task.Phone,
			Origin:      task.Origin,
			Destination: task.Destination,
			DepartureAt: task.DepartureAt,
		}, uc.subs, uc.notifier)
		_ = uc.queue.MarkSent(task.ID)
	}
	_ = uc.queue.DeleteExhausted(uc.maxRetries, feedbackTTL)
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/usecase/ -run TestSendFeedbackReminders -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/send_feedback_reminders.go internal/usecase/send_feedback_reminders_test.go
git commit -m "feat(feedback): queue-driven SendFeedbackReminders with retry"
```

---

## Task 8: Remove the dead `FindPendingFeedback` path

This removes the rides-based reminder query now that the queue drives reminders. It must be one atomic commit: the interface method and every mock implementing it are removed together, or the package won't compile.

**Files:**
- Modify: `internal/boundaries/repository/ride_repository.go`
- Modify: `internal/infrastructure/postgres/ride_repo.go`
- Modify: `internal/infrastructure/postgres/sqlc/queries/sql/rides.sql`
- Modify mocks: `internal/usecase/delete_ride_test.go`, `expire_test.go`, `post_request_test.go`, `post_ride_test.go`, `record_feedback_test.go`, `search_rides_test.go`
- Regenerated: `queries/querier.go`, `queries/rides.sql.go`

- [ ] **Step 1: Remove from the interface**

In `internal/boundaries/repository/ride_repository.go`, delete the line:
```go
	FindPendingFeedback() ([]domain.Ride, error)
```

- [ ] **Step 2: Remove from the postgres repo**

In `internal/infrastructure/postgres/ride_repo.go`, delete the whole method:
```go
func (r *RideRepo) FindPendingFeedback() ([]domain.Ride, error) {
	rows, err := r.q.ListRidesPendingFeedback(context.Background())
	...
}
```
(Delete the entire function body — lines around 176–183.)

- [ ] **Step 3: Remove the sqlc query**

In `internal/infrastructure/postgres/sqlc/queries/sql/rides.sql`, delete the `ListRidesPendingFeedback` query block (the `-- name: ListRidesPendingFeedback :many` statement and its SQL).

- [ ] **Step 4: Regenerate sqlc**

Run: `make sqlc`
Expected: `queries/querier.go` and `queries/rides.sql.go` no longer reference `ListRidesPendingFeedback`.

- [ ] **Step 5: Remove the method from every test mock**

Delete the `FindPendingFeedback` line from each of these (one line each):
- `internal/usecase/delete_ride_test.go:51`
- `internal/usecase/expire_test.go:43`
- `internal/usecase/post_request_test.go:45`
- `internal/usecase/post_ride_test.go:57`
- `internal/usecase/record_feedback_test.go:49`
- `internal/usecase/search_rides_test.go:43`

Each line looks like:
```go
func (m *mockRideRepo...) FindPendingFeedback() ([]domain.Ride, error) { return nil, nil }
```

- [ ] **Step 6: Verify build + all usecase tests**

Run: `go build ./internal/...` then `go test ./internal/usecase/...`
Expected: builds; all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/boundaries/repository/ride_repository.go internal/infrastructure/postgres/ride_repo.go internal/infrastructure/postgres/sqlc/queries/ internal/usecase/delete_ride_test.go internal/usecase/expire_test.go internal/usecase/post_request_test.go internal/usecase/post_ride_test.go internal/usecase/record_feedback_test.go internal/usecase/search_rides_test.go
git commit -m "refactor(feedback): remove dead rides-based FindPendingFeedback path"
```

---

## Task 9: Wire it up in `main.go`

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Construct the repo and use cases**

In `main.go`, after `notifQueueRepo := postgres.NewNotificationQueueRepo(pool)`, add:
```go
	feedbackQueueRepo := postgres.NewFeedbackQueueRepo(pool)
```

Change the `recordFeedback` line to pass the queue:
```go
	recordFeedback := usecase.NewRecordFeedback(rideRepo, statRepo, feedbackQueueRepo)
```

Change the `sendFeedbackReminders` line to the new signature and add `enqueueFeedback`:
```go
	enqueueFeedback := usecase.NewEnqueueFeedback(feedbackQueueRepo)
	sendFeedbackReminders := usecase.NewSendFeedbackReminders(feedbackQueueRepo, subRepo, notifier, 2, 3)
```

- [ ] **Step 2: Reorder the cron — enqueue first, before expiry; run once at startup**

Extract the cycle into a closure and call it once before entering the ticker loop, so the first run happens at boot rather than an hour later. Replace the whole `go func() { ... }()` block with:
```go
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		runCronCycle := func() {
			// enqueueFeedback runs FIRST, before expireRides, so an evening ride
			// about to be expired/deleted is still captured into the feedback queue.
			if err := enqueueFeedback.Execute(); err != nil {
				log.Printf("enqueue feedback: %v", err)
			}
			if err := expireRides.Execute(); err != nil {
				log.Printf("expire rides: %v", err)
			}
			if err := expireRequests.Execute(); err != nil {
				log.Printf("expire requests: %v", err)
			}
			if err := sendFeedbackReminders.Execute(); err != nil {
				log.Printf("send feedback reminders: %v", err)
			}
			if err := retryNotifications.Execute(); err != nil {
				log.Printf("retry notifications: %v", err)
			}
		}
		runCronCycle() // tick at startup — don't wait an hour for the first cycle
		for range ticker.C {
			runCronCycle()
		}
	}()
```

> Note: the startup tick runs in a background goroutine, so it does not delay the HTTP server coming up. With a single web instance this is the only cron runner; if web is ever scaled >1, every instance runs its own startup cycle (enqueue is idempotent via `ON CONFLICT`, so that is safe — see the concurrency note below on send/record).

- [ ] **Step 3: Verify the whole module builds and the full unit suite passes**

Run: `go build ./...` then `go test ./internal/usecase/...`
Expected: builds cleanly; all tests pass.

- [ ] **Step 4: Run `go vet` and gofmt**

Run: `go vet ./... && make fmt`
Expected: no vet errors; formatting clean.

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "feat(feedback): wire feedback queue, enqueue-before-expiry cron ordering"
```

---

## Task 10: i18n keys (7 locales)

**Files:**
- Modify: `frontend/src/messages/{fr,en,es,it,de,nl,el}.json`

- [ ] **Step 1: Add the two keys to each locale**

Add `deleteAskCameAlong` and `btnCancel` near the existing `deleteOk`/`btnDelete` keys (placement is cosmetic; JSON is a flat object). Values per locale:

`fr.json`:
```json
	"deleteAskCameAlong": "Avant de supprimer : quelqu'un est-il venu ?",
	"btnCancel": "Annuler",
```
`en.json`:
```json
	"deleteAskCameAlong": "Before deleting: did someone come along?",
	"btnCancel": "Cancel",
```
`es.json`:
```json
	"deleteAskCameAlong": "Antes de eliminar: ¿vino alguien?",
	"btnCancel": "Cancelar",
```
`it.json`:
```json
	"deleteAskCameAlong": "Prima di eliminare: è venuto qualcuno?",
	"btnCancel": "Annulla",
```
`de.json`:
```json
	"deleteAskCameAlong": "Vor dem Löschen: ist jemand mitgefahren?",
	"btnCancel": "Abbrechen",
```
`nl.json`:
```json
	"deleteAskCameAlong": "Voor het verwijderen: is er iemand meegereden?",
	"btnCancel": "Annuleren",
```
`el.json`:
```json
	"deleteAskCameAlong": "Πριν τη διαγραφή: ήρθε κάποιος μαζί σου;",
	"btnCancel": "Άκυρο",
```

(Watch JSON commas — add a trailing comma to the preceding line if you insert before the closing `}`.)

- [ ] **Step 2: Verify the frontend still builds/typechecks**

Run: `npm --prefix frontend run build`
Expected: build succeeds (Paraglide compiles the new message keys; no "missing message" errors).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/messages/
git commit -m "i18n(feedback): add deleteAskCameAlong + btnCancel (7 locales)"
```

---

## Task 11: Ask-on-delete in `MyRideCard.svelte`

**Files:**
- Modify: `frontend/src/lib/components/rides/MyRideCard.svelte`
- Modify: `frontend/src/lib/components/rides/MyRideCard.test.ts`

- [ ] **Step 1: Write the failing tests**

Replace `frontend/src/lib/components/rides/MyRideCard.test.ts` with (extends the api mock with `del` + `feedback`, adds delete-flow cases; keeps the name-display cases):
```ts
// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';

const { listInterests, del, feedback } = vi.hoisted(() => ({
	listInterests: vi.fn(),
	del: vi.fn(async () => null),
	feedback: vi.fn(async () => null),
}));
vi.mock('$lib/api', () => ({
	api: {
		rides: {
			listMatchingRequests: vi.fn(async () => []),
			listInterests,
			del,
			feedback,
		},
	},
}));

import MyRideCard from './MyRideCard.svelte';

const futureRide = {
	ID: 'r1', Origin: 'Saillans', Destination: 'Crest',
	DepartureAt: '2030-06-01T09:00:00Z', Flexibility: 0, FeedbackGiven: false,
} as any;
const pastRide = {
	ID: 'rp', Origin: 'A', Destination: 'B',
	DepartureAt: '2020-06-01T09:00:00Z', Flexibility: 0, FeedbackGiven: false,
} as any;

beforeEach(() => {
	listInterests.mockReset();
	listInterests.mockResolvedValue([]);
	del.mockClear();
	feedback.mockClear();
});

describe('MyRideCard name display', () => {
	it('shows "{name} wants a ride" for a pending interest', async () => {
		listInterests.mockResolvedValue([{ id: 'i1', status: 'pending', searcher_name: 'Marie' }]);
		render(MyRideCard, { props: { ride: futureRide, phone: '5550001' } });
		expect(await screen.findByText(/Marie/)).toBeInTheDocument();
		expect(screen.getByText(/wants a ride|demande un trajet/)).toBeInTheDocument();
	});

	it('shows name and phone for an accepted interest', async () => {
		listInterests.mockResolvedValue([
			{ id: 'i2', status: 'accepted', searcher_name: 'Marie', searcher_phone: '0612345678' },
		]);
		const { container } = render(MyRideCard, { props: { ride: futureRide, phone: '5550001' } });
		await screen.findByText(/Marie/);
		expect(container.querySelector('a[href="tel:0612345678"]')).toBeInTheDocument();
		expect(container.querySelector('.interest-accepted')!.textContent).toMatch(/Marie\s*—\s*0612345678/);
	});

	it('falls back to a placeholder when a pending interest has no name', async () => {
		listInterests.mockResolvedValue([{ id: 'i3', status: 'pending', searcher_name: '' }]);
		render(MyRideCard, { props: { ride: futureRide, phone: '5550001' } });
		expect(await screen.findByText(/Someone|Quelqu'un/)).toBeInTheDocument();
	});
});

describe('MyRideCard ask-on-delete', () => {
	it('asks the came-along question when deleting a past, unanswered ride', async () => {
		const { container } = render(MyRideCard, { props: { ride: pastRide, phone: 'p' } });
		await fireEvent.click(container.querySelector('.btn-delete')!);
		// the confirm block appears; nothing deleted yet
		expect(container.querySelector('.delete-confirm')).toBeInTheDocument();
		expect(del).not.toHaveBeenCalled();
		// answering yes records feedback then deletes
		await fireEvent.click(container.querySelector('.btn-del-yes')!);
		expect(feedback).toHaveBeenCalledWith('rp', 'p', true);
		expect(del).toHaveBeenCalledWith('rp', 'p');
	});

	it('records "no" then deletes', async () => {
		const { container } = render(MyRideCard, { props: { ride: pastRide, phone: 'p' } });
		await fireEvent.click(container.querySelector('.btn-delete')!);
		await fireEvent.click(container.querySelector('.btn-del-no')!);
		expect(feedback).toHaveBeenCalledWith('rp', 'p', false);
		expect(del).toHaveBeenCalledWith('rp', 'p');
	});

	it('deletes a future ride silently (no question)', async () => {
		const { container } = render(MyRideCard, { props: { ride: futureRide, phone: 'p' } });
		await fireEvent.click(container.querySelector('.btn-delete')!);
		expect(container.querySelector('.delete-confirm')).toBeNull();
		expect(feedback).not.toHaveBeenCalled();
		expect(del).toHaveBeenCalledWith('r1', 'p');
	});
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm --prefix frontend test`
Expected: the three ask-on-delete cases FAIL (no `.delete-confirm` / `.btn-del-yes`; delete fires immediately).

- [ ] **Step 3: Implement the component change**

In `frontend/src/lib/components/rides/MyRideCard.svelte`:

(a) Add a confirm-state rune near the other `$state` declarations:
```svelte
	let confirmingDelete = $state(false);
```

(b) Replace the `del()` function with `del()` + the new control functions:
```svelte
	async function del() {
		try { await api.rides.del(ride.ID, phone); deleted = true; delMsg = m.deleteOk(); }
		catch { delMsg = m.deleteErr(); }
	}
	function requestDelete() {
		// Ask "did someone come along?" only for trips that have actually departed
		// and haven't been answered yet. Future deletes are silent cancellations.
		if (isPast && !ride.FeedbackGiven && !fbDone) { confirmingDelete = true; }
		else { del(); }
	}
	async function deleteWithFeedback(taken: boolean) {
		try { await api.rides.feedback(ride.ID, phone, taken); } catch { /* best-effort */ }
		confirmingDelete = false;
		await del();
	}
```

(c) Replace the existing delete button markup:
```svelte
	<button type="button" class="btn btn-danger btn-delete" data-id={ride.ID} data-phone={phone} disabled={deleted} onclick={del}>{m.btnDelete()}</button>
	<div class="delete-msg" id="msg-{ride.ID}">{delMsg}</div>
```
with:
```svelte
	{#if confirmingDelete}
		<div class="delete-confirm mt-2" id="del-confirm-{ride.ID}">
			<div class="delete-confirm-q text-sm">{m.deleteAskCameAlong()}</div>
			<div class="delete-confirm-btns flex gap-2">
				<button type="button" class="btn-del-yes" onclick={() => deleteWithFeedback(true)}>{m.feedbackYes()}</button>
				<button type="button" class="btn-del-no" onclick={() => deleteWithFeedback(false)}>{m.feedbackNo()}</button>
				<button type="button" class="btn-del-cancel" onclick={() => (confirmingDelete = false)}>{m.btnCancel()}</button>
			</div>
		</div>
	{:else}
		<button type="button" class="btn btn-danger btn-delete" data-id={ride.ID} data-phone={phone} disabled={deleted} onclick={requestDelete}>{m.btnDelete()}</button>
	{/if}
	<div class="delete-msg" id="msg-{ride.ID}">{delMsg}</div>
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npm --prefix frontend test`
Expected: PASS (all name-display and ask-on-delete cases).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/components/rides/MyRideCard.svelte frontend/src/lib/components/rides/MyRideCard.test.ts
git commit -m "feat(feedback): ask 'did someone come along?' when deleting a past ride"
```

---

## Task 12: Full build + final verification

**Files:** none (verification only)

- [ ] **Step 1: Build the frontend into `web/build`**

Run: `make build-web`
Expected: `web/build` is regenerated, build succeeds.

- [ ] **Step 2: Full Go build + unit tests**

Run: `go build ./... && go test ./internal/usecase/...`
Expected: builds; all unit tests pass.

- [ ] **Step 3: Integration tests (only if a dev Postgres is available)**

Run: `make test`
Expected: integration suite passes (exercises the new `feedback_queue` table via migrations). If no DB is available, note it and skip — usecase tests already cover the logic.

- [ ] **Step 4: Manual smoke (optional, requires running app)**

Run: `make dev`, then in the app create a ride with a past departure, open "Mes trajets", click Delete → confirm the came-along question appears; answer it → ride is removed and stats updated. Verify a future ride deletes without a question.

- [ ] **Step 5: Final commit (if build artifacts changed)**

```bash
git add web/build
git commit -m "chore(web): rebuild frontend with ask-on-delete"
```

---

## Self-Review Notes (resolved)

- **Spec coverage:** queue table (T1–T2), enqueue at window start before expiry (T5, T9), send_after = departure+flex+1h (T2 SQL), retry mirrors notification_queue (T7), decoupled idempotent RecordFeedback (T6), ask-on-delete past-only / silent future (T11), 7-locale i18n (T10), removal of the rides-based reminder path (T8). All present.
- **Type consistency:** `FeedbackTask` fields used identically in domain (T3), converter (T4), mock (T5), and use cases (T6–T7). Repo method names (`EnqueueStartedRides`, `FindDue`, `FindByRideID`, `MarkSent`, `DeleteByRideID`, `DeleteExhausted`) match between interface (T3), postgres impl (T4), and mock (T5).
- **Interface-vs-spec note:** the spec sketched `EnqueueStartedRides(bound, sendDelay time.Duration)`; the implementation uses `EnqueueStartedRides(windowStartAfter time.Time)` with the +1h send delay and flexibility computed in SQL — simpler to express in sqlc and equivalent in behaviour.
- **Atomicity:** Task 8 removes the interface method and all mock implementations in a single commit so the package always compiles. Likewise Task 6 swaps `SetFeedbackGiven`→`ClaimFeedback` across interface + postgres + all mocks in one commit.
- **Concurrency guard (double-record):** the claim precedes the stat write — live ride via conditional `ClaimFeedback` (`UPDATE … WHERE feedback_given=false`), gone ride via `DeleteByRideID` rows-affected. Single statements, no explicit transaction (matches the codebase's `ON CONFLICT` idiom). Known limitation: claim-then-write means a (rare) `stats.Save` failure after a successful claim loses that one answer rather than corrupting data; not guarded because it would require threading a pgx transaction through the repos, which the codebase does not currently do. The push send itself is a non-transactional external call, so end-to-end exactly-once is not attempted — the guard targets duplicate *records*, which is what skews the metric.
- **Startup tick:** the cron cycle is extracted to a closure run once at boot and then hourly (T9), so first enqueue/send doesn't wait an hour post-deploy.
