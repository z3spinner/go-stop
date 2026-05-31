# Ride Stats & Feedback — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Track confirmed ride matches by collecting post-ride feedback from drivers and display a weekly top-routes leaderboard on the home page.

**Architecture:** New `ride_stats` PostgreSQL table persists stats after rides expire. Drivers are prompted via an inline card in "Mes trajets" and a push notification (30 min–23h post-departure). `GET /api/stats` serves aggregated data. Frontend adds a stats section on the home page and a full `/stats` page.

**Tech Stack:** Go, Gin, pgx/v5, existing Web Push infrastructure, vanilla JS SPA

---

## File Map

```
db/migrations/002_add_stats.sql          — new table + feedback_given column on rides

internal/domain/
  ride.go                               MODIFY — add FeedbackGiven bool field
  stat.go                               CREATE — RouteStat, Stats types

internal/boundaries/repository/
  ride_repository.go                    MODIFY — add FindPendingFeedback, SetFeedbackGiven
  stat_repository.go                    CREATE — StatRepository interface

internal/infrastructure/postgres/
  ride_repo.go                          MODIFY — add feedback_given to all SELECTs/scans, add 2 new methods
  stat_repo.go                          CREATE — StatRepo implementation

internal/usecase/
  record_feedback.go                    CREATE
  record_feedback_test.go               CREATE
  get_stats.go                          CREATE
  get_stats_test.go                     CREATE
  send_feedback_reminders.go            CREATE
  send_feedback_reminders_test.go       CREATE
  post_ride_test.go                     MODIFY — add FindPendingFeedback/SetFeedbackGiven stubs to mockRideRepo
  post_request_test.go                  MODIFY — add stubs to mockRideRepoWithMatch
  delete_ride_test.go                   MODIFY — add stubs to mockRideRepoDelete
  search_rides_test.go                  MODIFY — add stubs to mockRideRepoSearch
  expire_test.go                        MODIFY — add stubs to expiringRideRepo

internal/boundaries/handler/
  feedback_handler.go                   CREATE
  stats_handler.go                      CREATE
  integration_test.go                   MODIFY — add GET /api/stats and POST feedback routes to setupRouter

main.go                                 MODIFY — wire new repos, use cases, handlers, routes, cron task

web/js/app.js                           MODIFY — i18n strings, home stats, stats page, feedback in Mes Trajets
web/css/style.css                       MODIFY — stats and feedback styles
web/index.html                          MODIFY — bump cache buster to v=4
```

---

## Task 1: Database Migration

**Files:**
- Create: `db/migrations/002_add_stats.sql`

- [ ] **Step 1: Write migration**

```sql
-- db/migrations/002_add_stats.sql

ALTER TABLE rides ADD COLUMN IF NOT EXISTS feedback_given BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS ride_stats (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    origin       VARCHAR(200) NOT NULL,
    destination  VARCHAR(200) NOT NULL,
    ride_date    DATE         NOT NULL,
    taken        BOOLEAN      NOT NULL,
    recorded_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ride_stats_route ON ride_stats(origin, destination);
CREATE INDEX IF NOT EXISTS idx_ride_stats_date  ON ride_stats(recorded_at);
```

- [ ] **Step 2: Apply migration to running devstack**

```bash
docker compose exec db psql -U gostop < db/migrations/002_add_stats.sql
```

Expected output:
```
ALTER TABLE
CREATE TABLE
CREATE INDEX
CREATE INDEX
```

Verify: `docker compose exec db psql -U gostop -c "\d rides" | grep feedback`
Expected: `feedback_given | boolean | not null | false`

- [ ] **Step 3: Commit**

```bash
git add db/migrations/002_add_stats.sql
git commit -m "feat: add ride_stats table and feedback_given column"
git push
```

---

## Task 2: Domain Types

**Files:**
- Modify: `internal/domain/ride.go`
- Create: `internal/domain/stat.go`

- [ ] **Step 1: Add FeedbackGiven to Ride struct**

In `internal/domain/ride.go`, add `FeedbackGiven bool` as the last field:

```go
package domain

import "time"

type Ride struct {
	ID            string
	DriverName    string
	Phone         string
	Origin        string
	Destination   string
	Date          time.Time
	DepartureAt   time.Time
	Flexibility   Flexibility
	PostedAt      time.Time
	ExpiresAt     time.Time
	FeedbackGiven bool
}
```

- [ ] **Step 2: Create stat.go**

```go
// internal/domain/stat.go
package domain

type RouteStat struct {
	Origin      string
	Destination string
	Count       int
}

type Stats struct {
	TopRoutes      []RouteStat
	TotalConfirmed int
	TotalThisWeek  int
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/domain/...`
Expected: no output

- [ ] **Step 4: Commit**

```bash
git add internal/domain/ride.go internal/domain/stat.go
git commit -m "feat: add FeedbackGiven to Ride, add Stats domain types"
git push
```

---

## Task 3: Repository Interfaces

**Files:**
- Modify: `internal/boundaries/repository/ride_repository.go`
- Create: `internal/boundaries/repository/stat_repository.go`

- [ ] **Step 1: Add two methods to RideRepository**

```go
// internal/boundaries/repository/ride_repository.go
package repository

import "github.com/z3spinner/go-stop/internal/domain"

type RideRepository interface {
	Save(ride domain.Ride) error
	FindByID(id string) (domain.Ride, error)
	FindAll() ([]domain.Ride, error)
	FindByPhone(phone string) ([]domain.Ride, error)
	FindByOriginAndDestination(origin, destination string) ([]domain.Ride, error)
	FindMatching(request domain.Request) ([]domain.Ride, error)
	FindPendingFeedback() ([]domain.Ride, error)
	Delete(id string) error
	DeleteExpired() error
	SetFeedbackGiven(id string) error
}
```

- [ ] **Step 2: Create stat_repository.go**

```go
// internal/boundaries/repository/stat_repository.go
package repository

import (
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

type StatRepository interface {
	Save(origin, destination string, rideDate time.Time, taken bool) error
	GetStats() (domain.Stats, error)
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/boundaries/...`
Expected: compilation errors in main.go and usecase tests (interfaces not yet implemented) — that is fine. Check only the boundaries package itself:
`go vet ./internal/boundaries/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/boundaries/repository/ride_repository.go internal/boundaries/repository/stat_repository.go
git commit -m "feat: add FindPendingFeedback/SetFeedbackGiven to RideRepository, add StatRepository interface"
git push
```

---

## Task 4: Update Ride Repo Scans + New Methods

**Files:**
- Modify: `internal/infrastructure/postgres/ride_repo.go`

Adding `feedback_given` to every SELECT means updating `scanRide`, `collectRides`, and every query string. Also adding two new methods.

- [ ] **Step 1: Update scanRide to include feedback_given**

Replace the entire `scanRide` function:

```go
func scanRide(row pgx.Row) (domain.Ride, error) {
	var ride domain.Ride
	var flex int
	err := row.Scan(&ride.ID, &ride.DriverName, &ride.Phone, &ride.Origin, &ride.Destination,
		&ride.Date, &ride.DepartureAt, &flex, &ride.PostedAt, &ride.ExpiresAt, &ride.FeedbackGiven)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Ride{}, errors.New("ride not found")
		}
		return domain.Ride{}, err
	}
	ride.Flexibility = domain.Flexibility(flex)
	return ride, nil
}
```

- [ ] **Step 2: Update collectRides to include feedback_given**

Replace the entire `collectRides` function:

```go
func collectRides(rows pgx.Rows) ([]domain.Ride, error) {
	var rides []domain.Ride
	for rows.Next() {
		var ride domain.Ride
		var flex int
		if err := rows.Scan(&ride.ID, &ride.DriverName, &ride.Phone, &ride.Origin, &ride.Destination,
			&ride.Date, &ride.DepartureAt, &flex, &ride.PostedAt, &ride.ExpiresAt, &ride.FeedbackGiven); err != nil {
			return nil, err
		}
		ride.Flexibility = domain.Flexibility(flex)
		rides = append(rides, ride)
	}
	if rides == nil {
		rides = []domain.Ride{}
	}
	return rides, rows.Err()
}
```

- [ ] **Step 3: Update all SELECT query strings to include feedback_given**

Every SQL SELECT in `ride_repo.go` currently ends with `...posted_at, expires_at`. Add `, feedback_given` to the end of each SELECT column list. There are 5 queries to update: `FindByID`, `FindAll`, `FindByPhone`, `FindByOriginAndDestination`, `FindMatching`.

The updated column list is:
```
id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
```

Apply this change to all 5 SELECT statements.

- [ ] **Step 4: Add FindPendingFeedback method**

Add after `DeleteExpired`:

```go
func (r *RideRepo) FindPendingFeedback() ([]domain.Ride, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
		 FROM rides
		 WHERE departure_at BETWEEN (NOW() - INTERVAL '23 hours') AND (NOW() - INTERVAL '30 minutes')
		   AND feedback_given = false
		   AND expires_at > NOW()
		 ORDER BY departure_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRides(rows)
}
```

- [ ] **Step 5: Add SetFeedbackGiven method**

```go
func (r *RideRepo) SetFeedbackGiven(id string) error {
	_, err := r.pool.Exec(context.Background(),
		`UPDATE rides SET feedback_given = true WHERE id = $1`, id)
	return err
}
```

- [ ] **Step 6: Verify compilation**

Run: `go build ./internal/infrastructure/postgres/...`
Expected: no output (success)

- [ ] **Step 7: Commit**

```bash
git add internal/infrastructure/postgres/ride_repo.go
git commit -m "feat: add feedback_given to ride repo scans, add FindPendingFeedback/SetFeedbackGiven"
git push
```

---

## Task 5: Postgres Stat Repo

**Files:**
- Create: `internal/infrastructure/postgres/stat_repo.go`

- [ ] **Step 1: Write stat_repo.go**

```go
// internal/infrastructure/postgres/stat_repo.go
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
)

type StatRepo struct{ pool *pgxpool.Pool }

func NewStatRepo(pool *pgxpool.Pool) *StatRepo { return &StatRepo{pool: pool} }

func (r *StatRepo) Save(origin, destination string, rideDate time.Time, taken bool) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO ride_stats (origin, destination, ride_date, taken) VALUES ($1, $2, $3, $4)`,
		origin, destination, rideDate, taken)
	return err
}

func (r *StatRepo) GetStats() (domain.Stats, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT origin, destination, COUNT(*) AS count
		 FROM ride_stats
		 WHERE taken = true
		   AND recorded_at >= DATE_TRUNC('week', NOW())
		 GROUP BY origin, destination
		 ORDER BY count DESC
		 LIMIT 5`)
	if err != nil {
		return domain.Stats{}, err
	}
	defer rows.Close()

	var topRoutes []domain.RouteStat
	for rows.Next() {
		var rs domain.RouteStat
		if err := rows.Scan(&rs.Origin, &rs.Destination, &rs.Count); err != nil {
			return domain.Stats{}, err
		}
		topRoutes = append(topRoutes, rs)
	}
	if err := rows.Err(); err != nil {
		return domain.Stats{}, err
	}
	if topRoutes == nil {
		topRoutes = []domain.RouteStat{}
	}

	var totalConfirmed, totalThisWeek int
	err = r.pool.QueryRow(context.Background(),
		`SELECT
		   COUNT(*) FILTER (WHERE taken = true) AS total_confirmed,
		   COUNT(*) FILTER (WHERE taken = true AND recorded_at >= DATE_TRUNC('week', NOW())) AS total_this_week
		 FROM ride_stats`).Scan(&totalConfirmed, &totalThisWeek)
	if err != nil {
		return domain.Stats{}, err
	}

	return domain.Stats{
		TopRoutes:      topRoutes,
		TotalConfirmed: totalConfirmed,
		TotalThisWeek:  totalThisWeek,
	}, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/infrastructure/...`
Expected: no output

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/postgres/stat_repo.go
git commit -m "feat: add PostgreSQL stat repository"
git push
```

---

## Task 6: Update All Existing Ride Repo Mocks

Adding `FindPendingFeedback` and `SetFeedbackGiven` to the `RideRepository` interface breaks all existing mocks in the usecase tests. Add stub implementations to each.

**Files (all modify):**
- `internal/usecase/post_ride_test.go` — `mockRideRepo`
- `internal/usecase/post_request_test.go` — `mockRideRepoWithMatch`
- `internal/usecase/delete_ride_test.go` — `mockRideRepoDelete`
- `internal/usecase/search_rides_test.go` — `mockRideRepoSearch`
- `internal/usecase/expire_test.go` — `expiringRideRepo`

- [ ] **Step 1: Add stubs to mockRideRepo (post_ride_test.go)**

Add these two methods to `mockRideRepo`:

```go
func (m *mockRideRepo) FindPendingFeedback() ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepo) SetFeedbackGiven(string) error               { return nil }
```

- [ ] **Step 2: Add stubs to mockRideRepoWithMatch (post_request_test.go)**

```go
func (m *mockRideRepoWithMatch) FindPendingFeedback() ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoWithMatch) SetFeedbackGiven(string) error               { return nil }
```

- [ ] **Step 3: Add stubs to mockRideRepoDelete (delete_ride_test.go)**

```go
func (m *mockRideRepoDelete) FindPendingFeedback() ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoDelete) SetFeedbackGiven(string) error               { return nil }
```

- [ ] **Step 4: Add stubs to mockRideRepoSearch (search_rides_test.go)**

```go
func (m *mockRideRepoSearch) FindPendingFeedback() ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoSearch) SetFeedbackGiven(string) error               { return nil }
```

- [ ] **Step 5: Add stubs to expiringRideRepo (expire_test.go)**

```go
func (r *expiringRideRepo) FindPendingFeedback() ([]domain.Ride, error) { return nil, nil }
func (r *expiringRideRepo) SetFeedbackGiven(string) error               { return nil }
```

- [ ] **Step 6: Run all existing tests to confirm nothing broken**

Run: `go test ./internal/usecase/... -v`
Expected: all tests PASS (should be 32 tests)

- [ ] **Step 7: Commit**

```bash
git add internal/usecase/post_ride_test.go internal/usecase/post_request_test.go \
        internal/usecase/delete_ride_test.go internal/usecase/search_rides_test.go \
        internal/usecase/expire_test.go
git commit -m "fix: add FindPendingFeedback/SetFeedbackGiven stubs to ride repo mocks"
git push
```

---

## Task 7: RecordFeedback Use Case (TDD)

**Files:**
- Create: `internal/usecase/record_feedback.go`
- Create: `internal/usecase/record_feedback_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/usecase/record_feedback_test.go
package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// mockRideRepoFeedback — minimal ride repo for feedback tests
type mockRideRepoFeedback struct {
	rides       map[string]domain.Ride
	feedbackSet []string
}

func (m *mockRideRepoFeedback) Save(domain.Ride) error                              { return nil }
func (m *mockRideRepoFeedback) FindByID(id string) (domain.Ride, error) {
	r, ok := m.rides[id]
	if !ok {
		return domain.Ride{}, errors.New("not found")
	}
	return r, nil
}
func (m *mockRideRepoFeedback) FindAll() ([]domain.Ride, error)                     { return nil, nil }
func (m *mockRideRepoFeedback) FindByPhone(string) ([]domain.Ride, error)           { return nil, nil }
func (m *mockRideRepoFeedback) FindByOriginAndDestination(string, string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoFeedback) FindMatching(domain.Request) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoFeedback) FindPendingFeedback() ([]domain.Ride, error)        { return nil, nil }
func (m *mockRideRepoFeedback) Delete(string) error                                 { return nil }
func (m *mockRideRepoFeedback) DeleteExpired() error                                { return nil }
func (m *mockRideRepoFeedback) SetFeedbackGiven(id string) error {
	m.feedbackSet = append(m.feedbackSet, id)
	return nil
}

// mockStatRepo
type mockStatRepo struct {
	saved   []savedStat
	saveErr error
	stats   domain.Stats
}

type savedStat struct {
	origin, destination string
	rideDate            time.Time
	taken               bool
}

func (m *mockStatRepo) Save(origin, destination string, rideDate time.Time, taken bool) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, savedStat{origin, destination, rideDate, taken})
	return nil
}
func (m *mockStatRepo) GetStats() (domain.Stats, error) { return m.stats, nil }

func TestRecordFeedback_SavesStatAndMarksFeedbackGiven(t *testing.T) {
	rides := &mockRideRepoFeedback{
		rides: map[string]domain.Ride{
			"ride-1": {
				ID: "ride-1", Phone: "555-0001",
				Origin: "Saillans", Destination: "Crest",
				Date:    time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
				DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	stats := &mockStatRepo{}

	uc := usecase.NewRecordFeedback(rides, stats)
	err := uc.Execute("ride-1", "555-0001", true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats.saved) != 1 {
		t.Errorf("expected 1 stat saved, got %d", len(stats.saved))
	}
	if !stats.saved[0].taken {
		t.Error("expected taken=true")
	}
	if stats.saved[0].origin != "Saillans" {
		t.Errorf("expected origin Saillans, got %s", stats.saved[0].origin)
	}
	if len(rides.feedbackSet) != 1 || rides.feedbackSet[0] != "ride-1" {
		t.Error("expected feedback_given set on ride-1")
	}
}

func TestRecordFeedback_SavesNegativeFeedback(t *testing.T) {
	rides := &mockRideRepoFeedback{
		rides: map[string]domain.Ride{
			"ride-2": {ID: "ride-2", Phone: "555-0001", Origin: "A", Destination: "B",
				Date: time.Now()},
		},
	}
	stats := &mockStatRepo{}

	uc := usecase.NewRecordFeedback(rides, stats)
	err := uc.Execute("ride-2", "555-0001", false)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.saved[0].taken {
		t.Error("expected taken=false")
	}
}

func TestRecordFeedback_RejectsWrongPhone(t *testing.T) {
	rides := &mockRideRepoFeedback{
		rides: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-0001"},
		},
	}
	stats := &mockStatRepo{}

	uc := usecase.NewRecordFeedback(rides, stats)
	err := uc.Execute("ride-1", "555-9999", true)

	if !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
	if len(stats.saved) != 0 {
		t.Error("should not save stat on unauthorized")
	}
}

func TestRecordFeedback_ReturnsErrorIfRideNotFound(t *testing.T) {
	rides := &mockRideRepoFeedback{rides: map[string]domain.Ride{}}
	stats := &mockStatRepo{}

	uc := usecase.NewRecordFeedback(rides, stats)
	err := uc.Execute("nonexistent", "555-0001", true)

	if err == nil {
		t.Error("expected not found error")
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

Run: `go test ./internal/usecase/... -run TestRecordFeedback -v`
Expected: compilation error — `usecase.NewRecordFeedback undefined`

- [ ] **Step 3: Write record_feedback.go**

```go
// internal/usecase/record_feedback.go
package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

type RecordFeedback struct {
	rides repository.RideRepository
	stats repository.StatRepository
}

func NewRecordFeedback(rides repository.RideRepository, stats repository.StatRepository) *RecordFeedback {
	return &RecordFeedback{rides: rides, stats: stats}
}

func (uc *RecordFeedback) Execute(rideID, phone string, taken bool) error {
	ride, err := uc.rides.FindByID(rideID)
	if err != nil {
		return err
	}
	if ride.Phone != phone {
		return ErrUnauthorized
	}
	if err := uc.stats.Save(ride.Origin, ride.Destination, ride.Date, taken); err != nil {
		return err
	}
	return uc.rides.SetFeedbackGiven(rideID)
}
```

- [ ] **Step 4: Run tests to confirm they pass**

Run: `go test ./internal/usecase/... -run TestRecordFeedback -v`
Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/record_feedback.go internal/usecase/record_feedback_test.go
git commit -m "feat: add RecordFeedback use case"
git push
```

---

## Task 8: GetStats Use Case (TDD)

**Files:**
- Create: `internal/usecase/get_stats.go`
- Create: `internal/usecase/get_stats_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/usecase/get_stats_test.go
package usecase_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestGetStats_ReturnsDelegatedStats(t *testing.T) {
	expectedStats := domain.Stats{
		TopRoutes: []domain.RouteStat{
			{Origin: "Saillans", Destination: "Crest", Count: 4},
		},
		TotalConfirmed: 42,
		TotalThisWeek:  4,
	}
	stats := &mockStatRepo{stats: expectedStats}

	uc := usecase.NewGetStats(stats)
	result, err := uc.Execute()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalConfirmed != 42 {
		t.Errorf("expected TotalConfirmed 42, got %d", result.TotalConfirmed)
	}
	if len(result.TopRoutes) != 1 {
		t.Errorf("expected 1 top route, got %d", len(result.TopRoutes))
	}
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./internal/usecase/... -run TestGetStats -v`
Expected: compilation error — `usecase.NewGetStats undefined`

- [ ] **Step 3: Write get_stats.go**

```go
// internal/usecase/get_stats.go
package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type GetStats struct {
	stats repository.StatRepository
}

func NewGetStats(stats repository.StatRepository) *GetStats {
	return &GetStats{stats: stats}
}

func (uc *GetStats) Execute() (domain.Stats, error) {
	return uc.stats.GetStats()
}
```

- [ ] **Step 4: Run to confirm pass**

Run: `go test ./internal/usecase/... -run TestGetStats -v`
Expected: 1 test PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/get_stats.go internal/usecase/get_stats_test.go
git commit -m "feat: add GetStats use case"
git push
```

---

## Task 9: SendFeedbackReminders Use Case (TDD)

**Files:**
- Create: `internal/usecase/send_feedback_reminders.go`
- Create: `internal/usecase/send_feedback_reminders_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/usecase/send_feedback_reminders_test.go
package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// mockRideRepoPendingFeedback controls FindPendingFeedback return value
type mockRideRepoPendingFeedback struct {
	pending []domain.Ride
}

func (m *mockRideRepoPendingFeedback) Save(domain.Ride) error                              { return nil }
func (m *mockRideRepoPendingFeedback) FindByID(string) (domain.Ride, error)               { return domain.Ride{}, nil }
func (m *mockRideRepoPendingFeedback) FindAll() ([]domain.Ride, error)                     { return nil, nil }
func (m *mockRideRepoPendingFeedback) FindByPhone(string) ([]domain.Ride, error)           { return nil, nil }
func (m *mockRideRepoPendingFeedback) FindByOriginAndDestination(string, string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoPendingFeedback) FindMatching(domain.Request) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoPendingFeedback) FindPendingFeedback() ([]domain.Ride, error) {
	return m.pending, nil
}
func (m *mockRideRepoPendingFeedback) Delete(string) error      { return nil }
func (m *mockRideRepoPendingFeedback) DeleteExpired() error     { return nil }
func (m *mockRideRepoPendingFeedback) SetFeedbackGiven(string) error { return nil }

func TestSendFeedbackReminders_SendsPushToDriver(t *testing.T) {
	rides := &mockRideRepoPendingFeedback{
		pending: []domain.Ride{
			{
				ID: "ride-1", DriverName: "Alice", Phone: "555-0001",
				Origin: "Saillans", Destination: "Crest",
				DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0001": {Phone: "555-0001", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackReminders(rides, subs, n)
	err := uc.Execute()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !n.called {
		t.Error("expected push notification to be sent")
	}
	if n.lastMsg.URL != "/my-rides" {
		t.Errorf("expected URL /my-rides, got %s", n.lastMsg.URL)
	}
}

func TestSendFeedbackReminders_SkipsIfNoSubscription(t *testing.T) {
	rides := &mockRideRepoPendingFeedback{
		pending: []domain.Ride{
			{ID: "ride-1", Phone: "555-no-sub"},
		},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackReminders(rides, subs, n)
	err := uc.Execute()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send notification when driver has no subscription")
	}
}

func TestSendFeedbackReminders_NoPendingRides_NoNotifications(t *testing.T) {
	rides := &mockRideRepoPendingFeedback{pending: []domain.Ride{}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackReminders(rides, subs, n)
	err := uc.Execute()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send any notification with no pending rides")
	}
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./internal/usecase/... -run TestSendFeedbackReminders -v`
Expected: compilation error — `usecase.NewSendFeedbackReminders undefined`

- [ ] **Step 3: Write send_feedback_reminders.go**

```go
// internal/usecase/send_feedback_reminders.go
package usecase

import (
	"fmt"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type SendFeedbackReminders struct {
	rides    repository.RideRepository
	subs     repository.SubscriptionRepository
	notifier notification.Notifier
}

func NewSendFeedbackReminders(
	rides repository.RideRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *SendFeedbackReminders {
	return &SendFeedbackReminders{rides: rides, subs: subs, notifier: notifier}
}

func (uc *SendFeedbackReminders) Execute() error {
	pending, err := uc.rides.FindPendingFeedback()
	if err != nil {
		return err
	}
	for _, ride := range pending {
		sub, err := uc.subs.FindByPhone(ride.Phone)
		if err != nil {
			continue
		}
		msg := domain.Message{
			Title:       "Votre trajet est-il parti avec des passagers ?",
			Body:        fmt.Sprintf("%s → %s", ride.Origin, ride.Destination),
			URL:         "/my-rides",
			ContactName: ride.DriverName,
			Phone:       ride.Phone,
			Origin:      ride.Origin,
			Destination: ride.Destination,
			DepartureAt: ride.DepartureAt,
		}
		_ = uc.notifier.Send(sub, msg)
	}
	return nil
}
```

- [ ] **Step 4: Run to confirm pass**

Run: `go test ./internal/usecase/... -run TestSendFeedbackReminders -v`
Expected: all 3 tests PASS

- [ ] **Step 5: Run full test suite**

Run: `go test ./internal/usecase/... -v`
Expected: all tests PASS (35+ tests)

- [ ] **Step 6: Commit**

```bash
git add internal/usecase/send_feedback_reminders.go internal/usecase/send_feedback_reminders_test.go
git commit -m "feat: add SendFeedbackReminders use case"
git push
```

---

## Task 10: HTTP Handlers

**Files:**
- Create: `internal/boundaries/handler/feedback_handler.go`
- Create: `internal/boundaries/handler/stats_handler.go`

- [ ] **Step 1: Write feedback_handler.go**

```go
// internal/boundaries/handler/feedback_handler.go
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type FeedbackHandler struct {
	recordFeedback *usecase.RecordFeedback
}

func NewFeedbackHandler(recordFeedback *usecase.RecordFeedback) *FeedbackHandler {
	return &FeedbackHandler{recordFeedback: recordFeedback}
}

type feedbackRequest struct {
	Phone string `json:"phone" binding:"required"`
	Taken bool   `json:"taken"`
}

func (h *FeedbackHandler) Post(c *gin.Context) {
	var req feedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.recordFeedback.Execute(c.Param("id"), req.Phone, req.Taken); err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
```

- [ ] **Step 2: Write stats_handler.go**

```go
// internal/boundaries/handler/stats_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type StatsHandler struct {
	getStats *usecase.GetStats
}

func NewStatsHandler(getStats *usecase.GetStats) *StatsHandler {
	return &StatsHandler{getStats: getStats}
}

func (h *StatsHandler) Get(c *gin.Context) {
	stats, err := h.getStats.Execute()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/boundaries/handler/...`
Expected: no output

- [ ] **Step 4: Commit**

```bash
git add internal/boundaries/handler/feedback_handler.go internal/boundaries/handler/stats_handler.go
git commit -m "feat: add FeedbackHandler and StatsHandler"
git push
```

---

## Task 11: Wire Everything in main.go

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Update main.go**

Replace the contents of `main.go` with:

```go
package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/boundaries/handler"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
	"github.com/z3spinner/go-stop/internal/infrastructure/webpush"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func main() {
	pool, err := postgres.NewPool()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	rideRepo := postgres.NewRideRepo(pool)
	requestRepo := postgres.NewRequestRepo(pool)
	destRepo := postgres.NewDestinationRepo(pool)
	subRepo := postgres.NewSubscriptionRepo(pool)
	statRepo := postgres.NewStatRepo(pool)

	notifier := webpush.New(
		os.Getenv("VAPID_PUBLIC_KEY"),
		os.Getenv("VAPID_PRIVATE_KEY"),
		os.Getenv("VAPID_EMAIL"),
	)

	postRide := usecase.NewPostRide(rideRepo, requestRepo, subRepo, notifier)
	postRequest := usecase.NewPostRequest(requestRepo, rideRepo, subRepo, notifier)
	getRides := usecase.NewGetRides(rideRepo)
	getMyRides := usecase.NewGetMyRides(rideRepo)
	searchRides := usecase.NewSearchRides(rideRepo)
	getDests := usecase.NewGetDestinations(destRepo)
	subscribe := usecase.NewSubscribe(subRepo)
	unsubscribe := usecase.NewUnsubscribe(subRepo)
	deleteRide := usecase.NewDeleteRide(rideRepo)
	deleteRequest := usecase.NewDeleteRequest(requestRepo)
	getMyRequests := usecase.NewGetMyRequests(requestRepo)
	expireRides := usecase.NewExpireRides(rideRepo)
	expireRequests := usecase.NewExpireRequests(requestRepo)
	recordFeedback := usecase.NewRecordFeedback(rideRepo, statRepo)
	getStats := usecase.NewGetStats(statRepo)
	sendFeedbackReminders := usecase.NewSendFeedbackReminders(rideRepo, subRepo, notifier)

	rideH := handler.NewRideHandler(postRide, getRides, getMyRides, searchRides, deleteRide, rideRepo)
	requestH := handler.NewRequestHandler(postRequest, getMyRequests, deleteRequest, requestRepo)
	destH := handler.NewDestinationHandler(getDests)
	subH := handler.NewSubscriptionHandler(subscribe, unsubscribe)
	vapidH := handler.NewVapidHandler(os.Getenv("VAPID_PUBLIC_KEY"))
	siteName := os.Getenv("SITE_NAME")
	if siteName == "" {
		siteName = "Go-Stop"
	}
	configH := handler.NewConfigHandler(siteName)
	feedbackH := handler.NewFeedbackHandler(recordFeedback)
	statsH := handler.NewStatsHandler(getStats)

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := expireRides.Execute(); err != nil {
				log.Printf("expire rides: %v", err)
			}
			if err := expireRequests.Execute(); err != nil {
				log.Printf("expire requests: %v", err)
			}
			if err := sendFeedbackReminders.Execute(); err != nil {
				log.Printf("send feedback reminders: %v", err)
			}
		}
	}()

	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.Static("/css", "./web/css")
	r.Static("/js", "./web/js")
	r.StaticFile("/manifest.json", "./web/manifest.json")
	r.StaticFile("/sw.js", "./web/js/sw.js")
	r.StaticFile("/logo.svg", "./web/logo.svg")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/index.html")
	})

	api := r.Group("/api")
	{
		api.POST("/rides", rideH.Post)
		api.GET("/rides", rideH.List)
		api.GET("/rides/:id", rideH.Get)
		api.DELETE("/rides/:id", rideH.Delete)
		api.POST("/rides/:id/feedback", feedbackH.Post)

		api.POST("/requests", requestH.Post)
		api.GET("/requests", requestH.List)
		api.GET("/requests/:id", requestH.Get)
		api.DELETE("/requests/:id", requestH.Delete)

		api.GET("/destinations", destH.List)

		api.POST("/subscriptions", subH.Subscribe)
		api.DELETE("/subscriptions/:phone", subH.Unsubscribe)

		api.GET("/vapid-public-key", vapidH.GetPublicKey)
		api.GET("/config", configH.Get)
		api.GET("/stats", statsH.Get)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
```

- [ ] **Step 2: Verify full build**

Run: `go build ./...`
Expected: no output

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat: wire stats, feedback handler, and feedback reminder cron"
git push
```

---

## Task 12: Integration Tests for Stats + Feedback

**Files:**
- Modify: `internal/boundaries/handler/integration_test.go`

- [ ] **Step 1: Update setupRouter to add new routes**

In `setupRouter()`, add after the existing use cases:

```go
recordFeedback := usecase.NewRecordFeedback(rideRepo, statRepo)
getStats := usecase.NewGetStats(statRepo)
feedbackH := handler.NewFeedbackHandler(recordFeedback)
statsH := handler.NewStatsHandler(getStats)
```

Add `statRepo` at the top of `setupRouter`:
```go
statRepo := postgres.NewStatRepo(handlerPool)
```

Register the routes:
```go
r.POST("/api/rides/:id/feedback", feedbackH.Post)
r.GET("/api/stats", statsH.Get)
```

- [ ] **Step 2: Add TestHTTP_Feedback_RecordsStatAndMarksFeedbackGiven**

```go
func TestHTTP_Feedback_RecordsStatAndMarksFeedbackGiven(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	// Post a ride
	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "555-0001",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["ID"].(string)

	// Submit positive feedback
	w2 := postJSON(r, "/api/rides/"+id+"/feedback", map[string]interface{}{
		"phone": "555-0001",
		"taken": true,
	})
	if w2.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w2.Code, w2.Body.String())
	}

	// Stats should now show the route
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/api/stats", nil)
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w3.Code)
	}
	var stats map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &stats)
	if stats["total_this_week"].(float64) != 1 {
		t.Errorf("expected total_this_week=1, got %v", stats["total_this_week"])
	}
}

func TestHTTP_Feedback_WrongPhone_Returns403(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "555-0001",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["ID"].(string)

	w2 := postJSON(r, "/api/rides/"+id+"/feedback", map[string]interface{}{
		"phone": "555-9999",
		"taken": true,
	})
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w2.Code)
	}
}
```

- [ ] **Step 3: Run integration tests**

```bash
docker compose -f docker-compose.yml -f docker-compose.test.yml up db -d
TEST_DATABASE_URL="postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable" \
  go test -tags integration -count=1 -v -run "TestHTTP_Feedback" ./internal/boundaries/handler/...
```

Expected: both tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/boundaries/handler/integration_test.go
git commit -m "test: add integration tests for feedback and stats endpoints"
git push
```

---

## Task 13: Frontend — i18n, Stats Page, Home Stats, Mes Trajets Feedback

**Files:**
- Modify: `web/js/app.js`
- Modify: `web/css/style.css`
- Modify: `web/index.html`

### Step group A — i18n strings

- [ ] **Step 1: Add EN strings to STRINGS.en**

Add these keys inside `STRINGS.en` (before `privacyBody`):

```js
feedbackTitle:   'Did anyone join your ride?',
feedbackYes:     'Yes, someone joined',
feedbackNo:      'No, I drove alone',
feedbackThanks:  'Thanks!',
statsTitle:      'This week',
statsEmpty:      'No confirmed rides yet this week.',
statsAllTime:    (n) => `All time: ${n} confirmed`,
btnAllStats:     'All stats →',
statsPageTitle:  'Stats',
statsRouteCount: (n) => `${n} ✓`,
```

- [ ] **Step 2: Add FR strings to STRINGS.fr**

Add these keys inside `STRINGS.fr` (before `privacyBody`):

```js
feedbackTitle:   'Quelqu\'un est-il venu ?',
feedbackYes:     'Oui, quelqu\'un est venu',
feedbackNo:      'Non, j\'ai conduit seul(e)',
feedbackThanks:  'Merci !',
statsTitle:      'Cette semaine',
statsEmpty:      'Aucun trajet confirmé cette semaine.',
statsAllTime:    (n) => `Depuis le début : ${n} confirmés`,
btnAllStats:     'Toutes les stats →',
statsPageTitle:  'Statistiques',
statsRouteCount: (n) => `${n} ✓`,
```

### Step group B — home page stats section

- [ ] **Step 3: Update renderHome() to load and display stats**

Replace `renderHome()` with:

```js
async function renderHome() {
  history.replaceState({ path: '/' }, '', '/');
  const s = t();
  app.innerHTML = `
    ${topBar()}
    <div class="hero">
      <h1>${esc(SITE_NAME)}</h1>
      <p class="tagline">${s.tagline}</p>
      <button class="btn btn-primary" id="btn-driver">${s.btnDriver}</button>
      <button class="btn btn-secondary" id="btn-searcher">${s.btnSearcher}</button>
      <div class="ghost-row">
        <button class="btn-ghost-inline" id="btn-my-rides">${s.btnMyRides}</button>
        <span class="ghost-sep">·</span>
        <button class="btn-ghost-inline" id="btn-my-alerts">${s.btnMyAlerts}</button>
      </div>
    </div>
    <div id="home-stats"></div>`;
  document.getElementById('btn-driver').onclick = renderPostRide;
  document.getElementById('btn-searcher').onclick = renderSearchRides;
  document.getElementById('btn-my-rides').onclick = renderMyRides;
  document.getElementById('btn-my-alerts').onclick = renderMyAlerts;
  bindControls();
  loadHomeStats();
}

async function loadHomeStats() {
  const s = t();
  try {
    const stats = await api('GET', '/stats');
    if (!stats.top_routes || !stats.top_routes.length) return;
    const rows = stats.top_routes.map(r =>
      `<div class="stats-row">
        <span class="stats-route">${esc(r.Origin)} → ${esc(r.Destination)}</span>
        <span class="stats-count">${s.statsRouteCount(r.Count)}</span>
      </div>`
    ).join('');
    document.getElementById('home-stats').innerHTML = `
      <div class="stats-widget">
        <div class="stats-widget-title">${s.statsTitle}</div>
        ${rows}
        <button class="btn-all-stats" id="btn-all-stats">${s.btnAllStats}</button>
      </div>`;
    document.getElementById('btn-all-stats').onclick = renderStats;
  } catch {
    // silently omit stats if unavailable
  }
}
```

### Step group C — stats page

- [ ] **Step 4: Add renderStats() function**

Add before `renderHome()`:

```js
async function renderStats() {
  pushRoute('/stats');
  const s = t();
  app.innerHTML = `
    ${pageBar()}
    <h2>${s.statsPageTitle}</h2>
    <div id="stats-content"><p class="section-hint">…</p></div>`;
  document.getElementById('back').onclick = renderHome;
  bindControls();

  try {
    const stats = await api('GET', '/stats');
    const totalLine = stats.total_confirmed > 0
      ? `<p class="stats-total">${s.statsAllTime(stats.total_confirmed)}</p>`
      : '';
    const rows = stats.top_routes.length
      ? stats.top_routes.map(r => `
          <div class="stats-row">
            <span class="stats-route">${esc(r.Origin)} → ${esc(r.Destination)}</span>
            <span class="stats-count">${s.statsRouteCount(r.Count)}</span>
          </div>`).join('')
      : `<p class="section-hint">${s.statsEmpty}</p>`;

    document.getElementById('stats-content').innerHTML = `
      ${totalLine}
      <div class="stats-week-title">${s.statsTitle}</div>
      ${rows}`;
  } catch (err) {
    document.getElementById('stats-content').innerHTML =
      `<p class="error">${err.message}</p>`;
  }
}
```

### Step group D — feedback prompt in Mes Trajets

- [ ] **Step 5: Update renderMyRides to show feedback prompt for past rides**

Update the section that renders ride cards inside `renderMyRides`. Replace the template that renders each ride card to add a feedback section when `departure_at < now` and `!r.FeedbackGiven`:

```js
// Inside renderMyRides, replace the rides.map(...) call with:
list.innerHTML = rides.map(r => {
  const isPast = new Date(r.DepartureAt) < new Date();
  const needsFeedback = isPast && !r.FeedbackGiven;
  const feedbackSection = needsFeedback ? `
    <div class="feedback-prompt" id="fb-${esc(r.ID)}">
      <span class="feedback-question">${s.feedbackTitle}</span>
      <div class="feedback-btns">
        <button class="btn-fb-yes" data-id="${esc(r.ID)}" data-phone="${esc(phone)}">${s.feedbackYes}</button>
        <button class="btn-fb-no"  data-id="${esc(r.ID)}" data-phone="${esc(phone)}">${s.feedbackNo}</button>
      </div>
      <div class="feedback-thanks hidden" id="fb-thanks-${esc(r.ID)}">${s.feedbackThanks}</div>
    </div>` : '';
  return `
    <div class="card" id="card-${esc(r.ID)}">
      <div class="card-route">${esc(r.Origin)} → ${esc(r.Destination)}</div>
      <div class="card-meta">${formatTime(r.DepartureAt)} <span class="tag">${s.flexLabel[r.Flexibility] || esc(r.Flexibility) + ' min'}</span></div>
      ${feedbackSection}
      <button class="btn btn-danger btn-delete" data-id="${esc(r.ID)}" data-phone="${esc(phone)}">${s.btnDelete}</button>
      <div class="delete-msg" id="msg-${esc(r.ID)}"></div>
    </div>`;
}).join('');

// Bind feedback buttons
list.querySelectorAll('.btn-fb-yes, .btn-fb-no').forEach(btn => {
  btn.onclick = async () => {
    const taken = btn.classList.contains('btn-fb-yes');
    try {
      await api('POST', `/rides/${btn.dataset.id}/feedback`, {
        phone: btn.dataset.phone, taken,
      });
      const prompt = document.getElementById('fb-' + btn.dataset.id);
      prompt.querySelector('.feedback-btns').remove();
      prompt.querySelector('.feedback-question').remove();
      document.getElementById('fb-thanks-' + btn.dataset.id).classList.remove('hidden');
    } catch {
      // silently fail — will retry next visit
    }
  };
});

// Bind delete buttons (existing logic, unchanged)
list.querySelectorAll('.btn-delete').forEach(btn => {
  btn.onclick = async () => {
    try {
      await api('DELETE', `/rides/${btn.dataset.id}`, { phone: btn.dataset.phone });
      const card = document.getElementById('card-' + btn.dataset.id);
      card.style.opacity = '0.4';
      btn.disabled = true;
      document.getElementById('msg-' + btn.dataset.id).textContent = s.deleteOk;
    } catch {
      document.getElementById('msg-' + btn.dataset.id).textContent = s.deleteErr;
    }
  };
});
```

### Step group E — handleDeepLink + cache buster

- [ ] **Step 6: Add /stats to handleDeepLink switch**

In `handleDeepLink()`, add to the switch:

```js
case '/stats':        await renderStats();          return true;
```

- [ ] **Step 7: Bump cache buster in index.html to v=4**

```html
<link rel="stylesheet" href="/css/style.css?v=4">
<script src="/js/app.js?v=4"></script>
```

### Step group F — CSS

- [ ] **Step 8: Add stats and feedback styles to style.css**

Append to `web/css/style.css`:

```css
/* Stats widget on home page */
.stats-widget {
  margin-top: 24px;
  border-top: 1px solid var(--gray-300);
  padding-top: 16px;
}
.stats-widget-title, .stats-week-title {
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--gray-600);
  margin-bottom: 10px;
}
.stats-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 0;
  border-bottom: 1px solid var(--gray-100);
  font-size: 0.9rem;
}
.stats-route { color: var(--gray-900); }
.stats-count { color: var(--green); font-weight: 600; font-size: 0.85rem; }
.btn-all-stats {
  display: block;
  width: 100%;
  margin-top: 10px;
  background: none;
  border: none;
  color: var(--blue);
  font-size: 0.85rem;
  cursor: pointer;
  text-align: right;
  padding: 4px 0;
}
.btn-all-stats:hover { text-decoration: underline; }
.stats-total { font-size: 0.85rem; color: var(--gray-600); margin-bottom: 12px; }

/* Feedback prompt in Mes Trajets */
.feedback-prompt {
  background: var(--gray-50);
  border: 1px solid var(--gray-300);
  border-radius: var(--radius);
  padding: 10px 12px;
  margin: 8px 0;
}
.feedback-question { font-size: 0.9rem; font-weight: 500; display: block; margin-bottom: 8px; }
.feedback-btns { display: flex; gap: 8px; }
.btn-fb-yes, .btn-fb-no {
  flex: 1;
  padding: 7px;
  border: 1px solid var(--gray-300);
  border-radius: var(--radius);
  background: white;
  font-size: 0.85rem;
  cursor: pointer;
}
.btn-fb-yes:hover { background: var(--green); color: white; border-color: var(--green); }
.btn-fb-no:hover  { background: var(--gray-100); }
.feedback-thanks  { font-size: 0.85rem; color: var(--green); font-weight: 500; }
```

- [ ] **Step 9: Push web files to running container**

```bash
docker cp web go-stop-app-1:/app/
echo "pushed"
```

- [ ] **Step 10: Smoke test in browser**

Navigate to `http://localhost:8080/?v=4`.

Verify:
1. Home page loads with "Cette semaine" stats section visible (if any stats exist) or hidden (if empty)
2. `curl -s http://localhost:8080/api/stats` returns `{"top_routes":[],"total_confirmed":0,"total_this_week":0}`
3. Navigate to "Mes trajets" — past rides show "Quelqu'un est-il venu ?" prompt

- [ ] **Step 11: Commit**

```bash
git add web/js/app.js web/css/style.css web/index.html
git commit -m "feat: stats home widget, stats page, feedback prompt in Mes Trajets"
git push
```

---

## Self-Review

### Spec Coverage

| Requirement | Task |
|---|---|
| `ride_stats` table persists after ride deletion | Task 1 (migration) |
| `feedback_given` column on rides | Task 1 (migration) |
| `domain.RouteStat`, `domain.Stats` types | Task 2 |
| `StatRepository` interface | Task 3 |
| `FindPendingFeedback`, `SetFeedbackGiven` on RideRepository | Tasks 3, 4 |
| Postgres stat_repo implementation | Task 5 |
| All existing ride repo mocks updated | Task 6 |
| `RecordFeedback` use case (phone auth + save stat + mark given) | Task 7 |
| `GetStats` use case | Task 8 |
| `SendFeedbackReminders` use case (30min–23h window) | Task 9 |
| `POST /api/rides/:id/feedback` handler | Task 10 |
| `GET /api/stats` handler | Task 10 |
| `SendFeedbackReminders` wired into hourly cron | Task 11 |
| Integration tests for feedback + stats | Task 12 |
| Home page top-5 stats widget | Task 13 |
| Stats page at `/stats` | Task 13 |
| "Mes trajets" inline feedback prompt | Task 13 |
| `/stats` in `handleDeepLink` | Task 13 |
| Cache buster bumped | Task 13 |

### Placeholder Scan

No TBDs detected.

### Type Consistency

- `domain.Stats.TopRoutes` is `[]RouteStat` — used as `stats.top_routes` in JSON (Go serialises without json tags, so it becomes `TopRoutes` not `top_routes`)

⚠️ **Fix required:** `domain.Stats` fields need json tags so they serialize as camelCase for the frontend.

Update `internal/domain/stat.go`:

```go
package domain

type RouteStat struct {
	Origin      string `json:"Origin"`
	Destination string `json:"Destination"`
	Count       int    `json:"Count"`
}

type Stats struct {
	TopRoutes      []RouteStat `json:"top_routes"`
	TotalConfirmed int         `json:"total_confirmed"`
	TotalThisWeek  int         `json:"total_this_week"`
}
```

Wait — the frontend JS uses `r.Origin`, `r.Destination`, `r.Count` (PascalCase as used elsewhere for domain types without json tags). Let me check: other domain types like `domain.Ride` have NO json tags and serialize as PascalCase. The frontend already uses `r.Origin`, `r.DriverName` etc.

**Decision:** Add json tags only on `Stats` to control the top-level field names (`top_routes` etc.) but leave `RouteStat` fields as PascalCase (consistent with other domain types). Update `loadHomeStats` and `renderStats` in the frontend to use `r.Origin`, `r.Destination`, `r.Count`.

Final `stat.go`:

```go
package domain

type RouteStat struct {
	Origin      string
	Destination string
	Count       int
}

type Stats struct {
	TopRoutes      []RouteStat `json:"top_routes"`
	TotalConfirmed int         `json:"total_confirmed"`
	TotalThisWeek  int         `json:"total_this_week"`
}
```

Frontend uses `r.Origin`, `r.Destination`, `r.Count` for `RouteStat` — consistent with PascalCase convention used elsewhere. ✓

Note: Task 2 must use the json-tagged `Stats` struct (above) rather than the one without tags shown earlier. Update Task 2 accordingly.
