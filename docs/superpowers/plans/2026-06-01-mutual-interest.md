# Mutual Interest (Contact Consent) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a searcher express interest in a ride; the driver's phone is only revealed to the searcher (and vice-versa) after the driver explicitly accepts — enforced server-side, never just hidden in the frontend.

**Architecture:** New `interests` table stores pending/accepted connections between a searcher and a ride. Public ride endpoints return a phone-free `PublicRide` struct. Phone numbers only flow through the accept endpoint (to the driver) and the contact endpoint (to the searcher) after auth + status checks. Push notifications deliver the driver's phone to the searcher on accept.

**Tech Stack:** Go, Gin, pgx/v5, existing Web Push infrastructure, vanilla JS SPA

---

## File Map

```
db/migrations/004_interests.sql            CREATE — interests table + indexes

internal/domain/interest.go               CREATE — Interest type

internal/boundaries/repository/
  interest_repository.go                  CREATE — InterestRepository interface

internal/infrastructure/postgres/
  interest_repo.go                        CREATE — postgres implementation

internal/usecase/
  express_interest.go                     CREATE
  express_interest_test.go               CREATE
  accept_interest.go                      CREATE
  accept_interest_test.go                CREATE
  get_interest_contact.go                CREATE
  get_interest_contact_test.go           CREATE

internal/boundaries/handler/
  interest_handler.go                     CREATE
  ride_handler.go                         MODIFY — public List returns PublicRide (no phone/name)

main.go                                   MODIFY — wire new repo, use cases, handlers, routes

internal/boundaries/handler/
  integration_test.go                     MODIFY — add setupRouter wiring, add interest tests

web/js/app.js                             MODIFY — interest button, Mes Trajets interest UI,
                                                    deep link, i18n strings
web/css/style.css                         MODIFY — interest UI styles
web/index.html                            MODIFY — bump cache buster
```

---

## Task 1: Database Migration

**Files:**
- Create: `db/migrations/004_interests.sql`

- [ ] **Step 1: Write migration**

```sql
-- db/migrations/004_interests.sql
-- No FK to rides: rides get deleted on expiry but interests should survive
-- long enough to complete the contact exchange.
CREATE TABLE IF NOT EXISTS interests (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    ride_id        UUID        NOT NULL,
    searcher_phone VARCHAR(20) NOT NULL,
    status         VARCHAR(20) NOT NULL DEFAULT 'pending',  -- 'pending' | 'accepted'
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(ride_id, searcher_phone)
);

CREATE INDEX IF NOT EXISTS idx_interests_ride_id        ON interests(ride_id);
CREATE INDEX IF NOT EXISTS idx_interests_searcher_phone ON interests(searcher_phone);
```

- [ ] **Step 2: Apply to devstack**

```bash
docker compose exec db psql -U gostop < db/migrations/004_interests.sql
```

Expected:
```
CREATE TABLE
CREATE INDEX
CREATE INDEX
```

Verify: `docker compose exec db psql -U gostop -c "\d interests"`
Expected: table with columns `id`, `ride_id`, `searcher_phone`, `status`, `created_at`.

- [ ] **Step 3: Update Procfile postdeploy hook**

In `Procfile`, append to the postdeploy line:
```
&& psql $DATABASE_URL < db/migrations/004_interests.sql
```

Full line:
```
postdeploy: psql $DATABASE_URL < db/migrations/001_create_tables.sql && psql $DATABASE_URL < db/migrations/002_add_stats.sql && psql $DATABASE_URL < db/migrations/003_case_insensitive_indexes.sql && psql $DATABASE_URL < db/migrations/004_interests.sql
```

- [ ] **Step 4: Commit**

```bash
git add db/migrations/004_interests.sql Procfile
git commit -m "feat: add interests table for contact consent"
git push
```

---

## Task 2: Domain Type + Repository Interface

**Files:**
- Create: `internal/domain/interest.go`
- Create: `internal/boundaries/repository/interest_repository.go`

- [ ] **Step 1: Create interest.go**

```go
// internal/domain/interest.go
package domain

import "time"

type Interest struct {
    ID            string
    RideID        string
    SearcherPhone string
    Status        string // "pending" | "accepted"
    CreatedAt     time.Time
}
```

- [ ] **Step 2: Create interest_repository.go**

```go
// internal/boundaries/repository/interest_repository.go
package repository

import "github.com/z3spinner/go-stop/internal/domain"

type InterestRepository interface {
    Save(interest domain.Interest) error
    FindByID(id string) (domain.Interest, error)
    FindByRideAndSearcher(rideID, searcherPhone string) (domain.Interest, error)
    FindByRide(rideID string) ([]domain.Interest, error)
    Accept(id string) error
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/domain/... ./internal/boundaries/repository/...`
Expected: no output

- [ ] **Step 4: Commit**

```bash
git add internal/domain/interest.go internal/boundaries/repository/interest_repository.go
git commit -m "feat: add Interest domain type and InterestRepository interface"
git push
```

---

## Task 3: Postgres Interest Repo

**Files:**
- Create: `internal/infrastructure/postgres/interest_repo.go`

- [ ] **Step 1: Write interest_repo.go**

```go
// internal/infrastructure/postgres/interest_repo.go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
)

type InterestRepo struct{ pool *pgxpool.Pool }

func NewInterestRepo(pool *pgxpool.Pool) *InterestRepo { return &InterestRepo{pool: pool} }

func (r *InterestRepo) Save(i domain.Interest) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO interests (id, ride_id, searcher_phone, status)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (ride_id, searcher_phone) DO NOTHING`,
		i.ID, i.RideID, i.SearcherPhone, i.Status)
	return err
}

func (r *InterestRepo) FindByID(id string) (domain.Interest, error) {
	row := r.pool.QueryRow(context.Background(),
		`SELECT id, ride_id, searcher_phone, status, created_at FROM interests WHERE id = $1`, id)
	return scanInterest(row)
}

func (r *InterestRepo) FindByRideAndSearcher(rideID, searcherPhone string) (domain.Interest, error) {
	row := r.pool.QueryRow(context.Background(),
		`SELECT id, ride_id, searcher_phone, status, created_at
		 FROM interests WHERE ride_id = $1 AND searcher_phone = $2`,
		rideID, searcherPhone)
	return scanInterest(row)
}

func (r *InterestRepo) FindByRide(rideID string) ([]domain.Interest, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, ride_id, searcher_phone, status, created_at
		 FROM interests WHERE ride_id = $1 ORDER BY created_at ASC`, rideID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var interests []domain.Interest
	for rows.Next() {
		i, err := scanInterestRow(rows)
		if err != nil {
			return nil, err
		}
		interests = append(interests, i)
	}
	if interests == nil {
		interests = []domain.Interest{}
	}
	return interests, rows.Err()
}

func (r *InterestRepo) Accept(id string) error {
	_, err := r.pool.Exec(context.Background(),
		`UPDATE interests SET status = 'accepted' WHERE id = $1`, id)
	return err
}

func scanInterest(row pgx.Row) (domain.Interest, error) {
	var i domain.Interest
	err := row.Scan(&i.ID, &i.RideID, &i.SearcherPhone, &i.Status, &i.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Interest{}, errors.New("interest not found")
		}
		return domain.Interest{}, err
	}
	return i, nil
}

func scanInterestRow(rows pgx.Rows) (domain.Interest, error) {
	var i domain.Interest
	err := rows.Scan(&i.ID, &i.RideID, &i.SearcherPhone, &i.Status, &i.CreatedAt)
	return i, err
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/infrastructure/postgres/...`
Expected: no output

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/postgres/interest_repo.go
git commit -m "feat: add postgres interest repository"
git push
```

---

## Task 4: Update All Existing Ride Repo Mocks

Adding `InterestRepository` is new — no existing mocks need updating for `RideRepository`. But the usecase tests need an `InterestRepository` mock. Define it once in `express_interest_test.go` in Task 5. It will be reused across the three usecase test files since they share `package usecase_test`.

---

## Task 5: ExpressInterest Use Case (TDD)

**Files:**
- Create: `internal/usecase/express_interest.go`
- Create: `internal/usecase/express_interest_test.go`

The use case:
1. Looks up the ride (to get driver phone + route details for push)
2. Generates a new UUID for the interest
3. Saves the interest (ON CONFLICT DO NOTHING handles duplicates)
4. Re-fetches the interest (to get existing ID if it was a duplicate)
5. Sends push notification to driver's subscription (skips silently if no subscription)
6. Returns the interest

- [ ] **Step 1: Write failing tests**

```go
// internal/usecase/express_interest_test.go
package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// ── Shared mock for InterestRepository ───────────────────────────────────────

type mockInterestRepo struct {
	saved    []domain.Interest
	byID     map[string]domain.Interest
	saveErr  error
	acceptCalled []string
}

func (m *mockInterestRepo) Save(i domain.Interest) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, i)
	if m.byID == nil {
		m.byID = make(map[string]domain.Interest)
	}
	m.byID[i.ID] = i
	return nil
}
func (m *mockInterestRepo) FindByID(id string) (domain.Interest, error) {
	i, ok := m.byID[id]
	if !ok {
		return domain.Interest{}, errors.New("not found")
	}
	return i, nil
}
func (m *mockInterestRepo) FindByRideAndSearcher(rideID, phone string) (domain.Interest, error) {
	for _, i := range m.saved {
		if i.RideID == rideID && i.SearcherPhone == phone {
			return i, nil
		}
	}
	return domain.Interest{}, errors.New("not found")
}
func (m *mockInterestRepo) FindByRide(rideID string) ([]domain.Interest, error) {
	var result []domain.Interest
	for _, i := range m.saved {
		if i.RideID == rideID {
			result = append(result, i)
		}
	}
	return result, nil
}
func (m *mockInterestRepo) Accept(id string) error {
	m.acceptCalled = append(m.acceptCalled, id)
	if i, ok := m.byID[id]; ok {
		i.Status = "accepted"
		m.byID[id] = i
	}
	return nil
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestExpressInterest_CreatesInterestAndNotifiesDriver(t *testing.T) {
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {
				ID: "ride-1", Phone: "555-driver",
				Origin: "Saillans", Destination: "Crest",
				DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	interests := &mockInterestRepo{}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-driver": {Phone: "555-driver", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewExpressInterest(rides, interests, subs, n)
	interest, err := uc.Execute("ride-1", "555-searcher")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if interest.ID == "" {
		t.Error("expected interest to have an ID")
	}
	if interest.Status != "pending" {
		t.Errorf("expected status pending, got %s", interest.Status)
	}
	if interest.SearcherPhone != "555-searcher" {
		t.Errorf("expected searcher phone 555-searcher, got %s", interest.SearcherPhone)
	}
	if len(interests.saved) != 1 {
		t.Errorf("expected 1 saved interest, got %d", len(interests.saved))
	}
	if !n.called {
		t.Error("expected push notification sent to driver")
	}
	if n.lastMsg.URL == "" {
		t.Error("expected notification URL to be set")
	}
}

func TestExpressInterest_SkipsPushIfDriverHasNoSubscription(t *testing.T) {
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-driver"},
		},
	}
	interests := &mockInterestRepo{}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}} // no subscription
	n := &mockNotifier{}

	uc := usecase.NewExpressInterest(rides, interests, subs, n)
	_, err := uc.Execute("ride-1", "555-searcher")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send push when driver has no subscription")
	}
}

func TestExpressInterest_RejectsIfSearcherIsDriver(t *testing.T) {
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-same"},
		},
	}
	interests := &mockInterestRepo{}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewExpressInterest(rides, interests, subs, n)
	_, err := uc.Execute("ride-1", "555-same")

	if err == nil {
		t.Error("expected error when searcher is the driver")
	}
}

func TestExpressInterest_ReturnsErrorIfRideNotFound(t *testing.T) {
	rides := &mockRideRepo{byID: map[string]domain.Ride{}}
	interests := &mockInterestRepo{}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewExpressInterest(rides, interests, subs, n)
	_, err := uc.Execute("nonexistent", "555-searcher")

	if err == nil {
		t.Error("expected error for missing ride")
	}
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./internal/usecase/... -run TestExpressInterest -v`
Expected: compilation error — `usecase.NewExpressInterest undefined`

- [ ] **Step 3: Write express_interest.go**

```go
// internal/usecase/express_interest.go
package usecase

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type ExpressInterest struct {
	rides     repository.RideRepository
	interests repository.InterestRepository
	subs      repository.SubscriptionRepository
	notifier  notification.Notifier
}

func NewExpressInterest(
	rides repository.RideRepository,
	interests repository.InterestRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *ExpressInterest {
	return &ExpressInterest{rides: rides, interests: interests, subs: subs, notifier: notifier}
}

func (uc *ExpressInterest) Execute(rideID, searcherPhone string) (domain.Interest, error) {
	ride, err := uc.rides.FindByID(rideID)
	if err != nil {
		return domain.Interest{}, err
	}
	if ride.Phone == searcherPhone {
		return domain.Interest{}, errors.New("searcher cannot be the driver")
	}

	interest := domain.Interest{
		ID:            uuid.New().String(),
		RideID:        rideID,
		SearcherPhone: searcherPhone,
		Status:        "pending",
	}
	if err := uc.interests.Save(interest); err != nil {
		return domain.Interest{}, err
	}

	// Re-fetch in case a duplicate already existed (Save uses ON CONFLICT DO NOTHING)
	existing, err := uc.interests.FindByRideAndSearcher(rideID, searcherPhone)
	if err == nil {
		interest = existing
	}

	// Notify driver (best-effort — no error if no subscription)
	sub, err := uc.subs.FindByPhone(ride.Phone)
	if err == nil {
		msg := domain.Message{
			Title:       "Quelqu'un est intéressé par votre trajet",
			Body:        fmt.Sprintf("%s → %s", ride.Origin, ride.Destination),
			URL:         "/my-rides",
			ContactName: "",
			Phone:       "",
			Origin:      ride.Origin,
			Destination: ride.Destination,
			DepartureAt: ride.DepartureAt,
		}
		_ = uc.notifier.Send(sub, msg)
	}

	return interest, nil
}
```

- [ ] **Step 4: Run to confirm pass**

Run: `go test ./internal/usecase/... -run TestExpressInterest -v`
Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/express_interest.go internal/usecase/express_interest_test.go
git commit -m "feat: add ExpressInterest use case"
git push
```

---

## Task 6: AcceptInterest Use Case (TDD)

**Files:**
- Create: `internal/usecase/accept_interest.go`
- Create: `internal/usecase/accept_interest_test.go`

The use case:
1. Finds the interest by ID
2. Finds the ride (to auth the driver + get route for push body)
3. Verifies `ride.Phone == driverPhone`
4. Marks interest as accepted
5. Sends push to searcher's subscription with driver's phone in the payload URL
6. Returns searcher's phone (to be given to the driver)

- [ ] **Step 1: Write failing tests**

```go
// internal/usecase/accept_interest_test.go
package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestAcceptInterest_AcceptsAndReturnsSearcherPhone(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "pending",
	}
	interests := &mockInterestRepo{
		byID:  map[string]domain.Interest{"int-1": interest},
		saved: []domain.Interest{interest},
	}
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-driver",
				Origin: "Saillans", Destination: "Crest",
				DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC)},
		},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-searcher": {Phone: "555-searcher", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewAcceptInterest(interests, rides, subs, n)
	searcherPhone, err := uc.Execute("int-1", "555-driver")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if searcherPhone != "555-searcher" {
		t.Errorf("expected searcher phone 555-searcher, got %s", searcherPhone)
	}
	if len(interests.acceptCalled) == 0 || interests.acceptCalled[0] != "int-1" {
		t.Error("expected Accept called on interest int-1")
	}
	if !n.called {
		t.Error("expected push notification sent to searcher")
	}
}

func TestAcceptInterest_RejectsWrongDriverPhone(t *testing.T) {
	interest := domain.Interest{ID: "int-1", RideID: "ride-1", SearcherPhone: "555-searcher", Status: "pending"}
	interests := &mockInterestRepo{
		byID:  map[string]domain.Interest{"int-1": interest},
		saved: []domain.Interest{interest},
	}
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-driver"},
		},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewAcceptInterest(interests, rides, subs, n)
	_, err := uc.Execute("int-1", "555-wrong")

	if !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
	if len(interests.acceptCalled) != 0 {
		t.Error("Accept should not be called on unauthorized")
	}
}

func TestAcceptInterest_ReturnsErrorIfInterestNotFound(t *testing.T) {
	interests := &mockInterestRepo{byID: map[string]domain.Interest{}}
	rides := &mockRideRepo{byID: map[string]domain.Ride{}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewAcceptInterest(interests, rides, subs, n)
	_, err := uc.Execute("nonexistent", "555-driver")

	if err == nil {
		t.Error("expected error for missing interest")
	}
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./internal/usecase/... -run TestAcceptInterest -v`
Expected: compilation error — `usecase.NewAcceptInterest undefined`

- [ ] **Step 3: Write accept_interest.go**

```go
// internal/usecase/accept_interest.go
package usecase

import (
	"fmt"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type AcceptInterest struct {
	interests repository.InterestRepository
	rides     repository.RideRepository
	subs      repository.SubscriptionRepository
	notifier  notification.Notifier
}

func NewAcceptInterest(
	interests repository.InterestRepository,
	rides repository.RideRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *AcceptInterest {
	return &AcceptInterest{interests: interests, rides: rides, subs: subs, notifier: notifier}
}

// Execute accepts the interest and returns the searcher's phone to the driver.
// It also pushes a notification to the searcher with a deep link to retrieve
// the driver's phone via GET /api/interests/:id/contact.
func (uc *AcceptInterest) Execute(interestID, driverPhone string) (string, error) {
	interest, err := uc.interests.FindByID(interestID)
	if err != nil {
		return "", err
	}

	ride, err := uc.rides.FindByID(interest.RideID)
	if err != nil {
		return "", err
	}
	if ride.Phone != driverPhone {
		return "", ErrUnauthorized
	}

	if err := uc.interests.Accept(interestID); err != nil {
		return "", err
	}

	// Notify searcher (best-effort)
	sub, err := uc.subs.FindByPhone(interest.SearcherPhone)
	if err == nil {
		msg := domain.Message{
			Title:       "Le conducteur accepte le contact",
			Body:        fmt.Sprintf("%s → %s", ride.Origin, ride.Destination),
			URL:         "/interests/" + interestID,
			Origin:      ride.Origin,
			Destination: ride.Destination,
			DepartureAt: ride.DepartureAt,
		}
		_ = uc.notifier.Send(sub, msg)
	}

	return interest.SearcherPhone, nil
}
```

- [ ] **Step 4: Run to confirm pass**

Run: `go test ./internal/usecase/... -run TestAcceptInterest -v`
Expected: all 3 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/accept_interest.go internal/usecase/accept_interest_test.go
git commit -m "feat: add AcceptInterest use case"
git push
```

---

## Task 7: GetInterestContact Use Case (TDD)

**Files:**
- Create: `internal/usecase/get_interest_contact.go`
- Create: `internal/usecase/get_interest_contact_test.go`

Returns the OTHER party's phone. Driver gets searcher's phone; searcher gets driver's phone.
Only works after status == "accepted". Enforces authorization.

- [ ] **Step 1: Write failing tests**

```go
// internal/usecase/get_interest_contact_test.go
package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestGetInterestContact_DriverGetsSearcherPhone(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "accepted",
	}
	interests := &mockInterestRepo{
		byID:  map[string]domain.Interest{"int-1": interest},
		saved: []domain.Interest{interest},
	}
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-driver",
				DepartureAt: time.Now()},
		},
	}

	uc := usecase.NewGetInterestContact(interests, rides)
	phone, err := uc.Execute("int-1", "555-driver")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phone != "555-searcher" {
		t.Errorf("expected 555-searcher, got %s", phone)
	}
}

func TestGetInterestContact_SearcherGetsDriverPhone(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "accepted",
	}
	interests := &mockInterestRepo{
		byID:  map[string]domain.Interest{"int-1": interest},
		saved: []domain.Interest{interest},
	}
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-driver",
				DepartureAt: time.Now()},
		},
	}

	uc := usecase.NewGetInterestContact(interests, rides)
	phone, err := uc.Execute("int-1", "555-searcher")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phone != "555-driver" {
		t.Errorf("expected 555-driver, got %s", phone)
	}
}

func TestGetInterestContact_ReturnsErrorIfPending(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "pending",
	}
	interests := &mockInterestRepo{
		byID:  map[string]domain.Interest{"int-1": interest},
		saved: []domain.Interest{interest},
	}
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-driver",
				DepartureAt: time.Now()},
		},
	}

	uc := usecase.NewGetInterestContact(interests, rides)
	_, err := uc.Execute("int-1", "555-driver")

	if err == nil {
		t.Error("expected error for pending interest")
	}
}

func TestGetInterestContact_RejectsUnauthorizedPhone(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "accepted",
	}
	interests := &mockInterestRepo{
		byID:  map[string]domain.Interest{"int-1": interest},
		saved: []domain.Interest{interest},
	}
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-driver",
				DepartureAt: time.Now()},
		},
	}

	uc := usecase.NewGetInterestContact(interests, rides)
	_, err := uc.Execute("int-1", "555-stranger")

	if err == nil {
		t.Error("expected error for unauthorized phone")
	}
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./internal/usecase/... -run TestGetInterestContact -v`
Expected: compilation error — `usecase.NewGetInterestContact undefined`

- [ ] **Step 3: Write get_interest_contact.go**

```go
// internal/usecase/get_interest_contact.go
package usecase

import (
	"errors"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

type GetInterestContact struct {
	interests repository.InterestRepository
	rides     repository.RideRepository
}

func NewGetInterestContact(
	interests repository.InterestRepository,
	rides repository.RideRepository,
) *GetInterestContact {
	return &GetInterestContact{interests: interests, rides: rides}
}

func (uc *GetInterestContact) Execute(interestID, requesterPhone string) (string, error) {
	interest, err := uc.interests.FindByID(interestID)
	if err != nil {
		return "", err
	}
	if interest.Status != "accepted" {
		return "", errors.New("interest not yet accepted")
	}

	ride, err := uc.rides.FindByID(interest.RideID)
	if err != nil {
		return "", err
	}

	switch requesterPhone {
	case ride.Phone:
		return interest.SearcherPhone, nil
	case interest.SearcherPhone:
		return ride.Phone, nil
	default:
		return "", ErrUnauthorized
	}
}
```

- [ ] **Step 4: Run to confirm pass**

Run: `go test ./internal/usecase/... -run TestGetInterestContact -v`
Expected: all 4 tests PASS

- [ ] **Step 5: Run full suite**

Run: `go test ./internal/usecase/... -v`
Expected: all tests PASS (45+ tests)

- [ ] **Step 6: Commit**

```bash
git add internal/usecase/get_interest_contact.go internal/usecase/get_interest_contact_test.go
git commit -m "feat: add GetInterestContact use case"
git push
```

---

## Task 8: HTTP Handlers + Public Ride Response

**Files:**
- Create: `internal/boundaries/handler/interest_handler.go`
- Modify: `internal/boundaries/handler/ride_handler.go`

### Part A: Public ride response (strip phone + name)

The `List` handler currently returns the full `domain.Ride` for all callers. Public callers (no `X-Phone` header) must not receive phone numbers or names.

Add this struct and conversion function to `ride_handler.go`:

```go
// publicRide is returned for public search/feed requests.
// Phone and DriverName are intentionally absent.
type publicRide struct {
	ID            string    `json:"ID"`
	Origin        string    `json:"Origin"`
	Destination   string    `json:"Destination"`
	Date          time.Time `json:"Date"`
	DepartureAt   time.Time `json:"DepartureAt"`
	Flexibility   int       `json:"Flexibility"`
	PostedAt      time.Time `json:"PostedAt"`
	ExpiresAt     time.Time `json:"ExpiresAt"`
	FeedbackGiven bool      `json:"FeedbackGiven"`
}

func toPublicRides(rides []domain.Ride) []publicRide {
	out := make([]publicRide, len(rides))
	for i, r := range rides {
		out[i] = publicRide{
			ID:            r.ID,
			Origin:        r.Origin,
			Destination:   r.Destination,
			Date:          r.Date,
			DepartureAt:   r.DepartureAt,
			Flexibility:   int(r.Flexibility),
			PostedAt:      r.PostedAt,
			ExpiresAt:     r.ExpiresAt,
			FeedbackGiven: r.FeedbackGiven,
		}
	}
	return out
}
```

Update `List` to use `toPublicRides` when no `X-Phone` is set:

```go
func (h *RideHandler) List(c *gin.Context) {
	origin := c.Query("origin")
	destination := c.Query("destination")
	phone := c.GetHeader("X-Phone")
	phone = normalizePhone(phone)

	var rides []domain.Ride
	var err error
	switch {
	case phone != "":
		rides, err = h.getMyRides.Execute(phone)
	case origin != "" && destination != "":
		rides, err = h.searchRides.Execute(origin, destination)
	default:
		rides, err = h.getRides.Execute()
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if phone != "" {
		// Owner view: full ride including phone
		c.JSON(http.StatusOK, rides)
	} else {
		// Public view: strip phone and driver name
		c.JSON(http.StatusOK, toPublicRides(rides))
	}
}
```

### Part B: InterestHandler

```go
// internal/boundaries/handler/interest_handler.go
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type InterestHandler struct {
	expressInterest    *usecase.ExpressInterest
	acceptInterest     *usecase.AcceptInterest
	getInterestContact *usecase.GetInterestContact
}

func NewInterestHandler(
	expressInterest *usecase.ExpressInterest,
	acceptInterest *usecase.AcceptInterest,
	getInterestContact *usecase.GetInterestContact,
) *InterestHandler {
	return &InterestHandler{
		expressInterest:    expressInterest,
		acceptInterest:     acceptInterest,
		getInterestContact: getInterestContact,
	}
}

type expressInterestRequest struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *InterestHandler) Express(c *gin.Context) {
	var req expressInterestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	interest, err := h.expressInterest.Execute(c.Param("id"), normalizePhone(req.Phone))
	if err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "searcher cannot be the driver"})
			return
		}
		if err.Error() == "ride not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "ride not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Return only non-PII fields
	c.JSON(http.StatusCreated, gin.H{
		"id":     interest.ID,
		"status": interest.Status,
	})
}

type acceptInterestRequest struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *InterestHandler) Accept(c *gin.Context) {
	var req acceptInterestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	searcherPhone, err := h.acceptInterest.Execute(c.Param("id"), normalizePhone(req.Phone))
	if err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	// Phone revealed only to the authenticated driver
	c.JSON(http.StatusOK, gin.H{"searcher_phone": searcherPhone})
}

func (h *InterestHandler) GetContact(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	otherPhone, err := h.getInterestContact.Execute(c.Param("id"), phone)
	if err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"phone": otherPhone})
}
```

- [ ] **Step 1: Make the changes above to ride_handler.go and create interest_handler.go**

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/boundaries/handler/...`
Expected: no output

- [ ] **Step 3: Commit**

```bash
git add internal/boundaries/handler/interest_handler.go internal/boundaries/handler/ride_handler.go
git commit -m "feat: add InterestHandler, strip phone/name from public ride responses"
git push
```

---

## Task 9: Wire main.go + Routes

**Files:**
- Modify: `main.go`

Add after `statRepo`:
```go
interestRepo := postgres.NewInterestRepo(pool)
```

Add use cases after `sendFeedbackReminders`:
```go
expressInterest    := usecase.NewExpressInterest(rideRepo, interestRepo, subRepo, notifier)
acceptInterest     := usecase.NewAcceptInterest(interestRepo, rideRepo, subRepo, notifier)
getInterestContact := usecase.NewGetInterestContact(interestRepo, rideRepo)
```

Add handler:
```go
interestH := handler.NewInterestHandler(expressInterest, acceptInterest, getInterestContact)
```

Add routes inside the `api` group:
```go
api.POST("/rides/:id/interest",    interestH.Express)
api.POST("/interests/:id/accept",  interestH.Accept)
api.GET("/interests/:id/contact",  interestH.GetContact)
```

- [ ] **Step 1: Make the changes to main.go**

- [ ] **Step 2: Verify full build + tests**

Run: `go build ./... && go test ./internal/usecase/...`
Expected: both succeed

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "feat: wire interest use cases, handlers, and routes"
git push
```

---

## Task 10: Integration Tests

**Files:**
- Modify: `internal/boundaries/handler/integration_test.go`

- [ ] **Step 1: Add interestRepo + use cases to setupRouter**

In `setupRouter()`, add after `statRepo`:
```go
interestRepo := postgres.NewInterestRepo(handlerPool)
expressInterest    := usecase.NewExpressInterest(rideRepo, interestRepo, subRepo, n)
acceptInterest     := usecase.NewAcceptInterest(interestRepo, rideRepo, subRepo, n)
getInterestContact := usecase.NewGetInterestContact(interestRepo, rideRepo)
interestH := handler.NewInterestHandler(expressInterest, acceptInterest, getInterestContact)
```

Add routes:
```go
r.POST("/api/rides/:id/interest",   interestH.Express)
r.POST("/api/interests/:id/accept", interestH.Accept)
r.GET("/api/interests/:id/contact", interestH.GetContact)
```

- [ ] **Step 2: Add test: express interest creates record**

```go
func TestHTTP_Interest_ExpressCreatesRecord(t *testing.T) {
	truncateAll(t)
	handlerPool.Exec(context.Background(), `TRUNCATE interests`)
	r := setupRouter()

	// Post a ride (driver)
	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "555-driver",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 30,
	})
	var ride map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ride)
	rideID := ride["ID"].(string)

	// Searcher expresses interest
	w2 := postJSON(r, "/api/rides/"+rideID+"/interest", map[string]interface{}{
		"phone": "555-searcher",
	})
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w2.Code, w2.Body.String())
	}
	var interest map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &interest)
	if interest["id"] == nil || interest["id"] == "" {
		t.Error("expected interest ID in response")
	}
	if interest["status"] != "pending" {
		t.Errorf("expected pending status, got %v", interest["status"])
	}
	// Phone must NOT be in response
	if interest["searcher_phone"] != nil || interest["phone"] != nil {
		t.Error("phone must not appear in express-interest response")
	}
}
```

- [ ] **Step 3: Add test: driver cannot be searcher**

```go
func TestHTTP_Interest_DriverCannotBeSearcher(t *testing.T) {
	truncateAll(t)
	handlerPool.Exec(context.Background(), `TRUNCATE interests`)
	r := setupRouter()

	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "555-driver",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})
	var ride map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ride)
	rideID := ride["ID"].(string)

	w2 := postJSON(r, "/api/rides/"+rideID+"/interest", map[string]interface{}{
		"phone": "555-driver", // same as driver
	})
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w2.Code)
	}
}
```

- [ ] **Step 4: Add test: full accept flow reveals phones correctly**

```go
func TestHTTP_Interest_AcceptRevealsPhonesCorrectly(t *testing.T) {
	truncateAll(t)
	handlerPool.Exec(context.Background(), `TRUNCATE interests`)
	r := setupRouter()

	// Driver posts ride
	w := postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "555-driver",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 30,
	})
	var ride map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &ride)
	rideID := ride["ID"].(string)

	// Searcher expresses interest
	w2 := postJSON(r, "/api/rides/"+rideID+"/interest", map[string]interface{}{
		"phone": "555-searcher",
	})
	var interest map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &interest)
	interestID := interest["id"].(string)

	// Contact endpoint returns 404 while pending
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/api/interests/"+interestID+"/contact", nil)
	req3.Header.Set("X-Phone", "555-searcher")
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusNotFound {
		t.Errorf("expected 404 while pending, got %d", w3.Code)
	}

	// Driver accepts
	w4 := postJSON(r, "/api/interests/"+interestID+"/accept", map[string]interface{}{
		"phone": "555-driver",
	})
	if w4.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w4.Code, w4.Body.String())
	}
	var acceptResp map[string]interface{}
	json.Unmarshal(w4.Body.Bytes(), &acceptResp)
	if acceptResp["searcher_phone"] != "555-searcher" {
		t.Errorf("driver should receive searcher phone, got %v", acceptResp["searcher_phone"])
	}
	// Driver response must NOT include driver's own phone (it's already known to them)

	// Searcher can now get driver's phone
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest(http.MethodGet, "/api/interests/"+interestID+"/contact", nil)
	req5.Header.Set("X-Phone", "555-searcher")
	r.ServeHTTP(w5, req5)
	if w5.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w5.Code, w5.Body.String())
	}
	var contactResp map[string]interface{}
	json.Unmarshal(w5.Body.Bytes(), &contactResp)
	if contactResp["phone"] != "555-driver" {
		t.Errorf("searcher should receive driver phone, got %v", contactResp["phone"])
	}

	// Stranger gets 403
	w6 := httptest.NewRecorder()
	req6, _ := http.NewRequest(http.MethodGet, "/api/interests/"+interestID+"/contact", nil)
	req6.Header.Set("X-Phone", "555-stranger")
	r.ServeHTTP(w6, req6)
	if w6.Code != http.StatusForbidden {
		t.Errorf("expected 403 for stranger, got %d", w6.Code)
	}
}
```

- [ ] **Step 5: Add test: public ride list strips phone**

```go
func TestHTTP_PublicRideList_StripsPIIFields(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	postJSON(r, "/api/rides", map[string]interface{}{
		"driver_name": "Alice", "phone": "555-secret",
		"origin": "Saillans", "destination": "Crest",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	})

	// Public request (no X-Phone)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/rides", nil)
	r.ServeHTTP(w, req)

	var rides []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &rides)
	if len(rides) != 1 {
		t.Fatalf("expected 1 ride, got %d", len(rides))
	}
	if rides[0]["Phone"] != nil {
		t.Errorf("Phone must not appear in public ride list, got %v", rides[0]["Phone"])
	}
	if rides[0]["DriverName"] != nil {
		t.Errorf("DriverName must not appear in public ride list, got %v", rides[0]["DriverName"])
	}
	// Route details must still be present
	if rides[0]["Origin"] == nil {
		t.Error("Origin must be present in public ride list")
	}
}
```

- [ ] **Step 6: Also add truncation of interests table in TestMain and truncateAll**

In `truncateAll`:
```go
func truncateAll(t *testing.T) {
	t.Helper()
	if _, err := handlerPool.Exec(context.Background(),
		`TRUNCATE rides, requests, subscriptions, ride_stats, interests`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
```

In `TestMain`, update the TRUNCATE:
```go
_, truncErr = handlerPool.Exec(context.Background(),
    `TRUNCATE rides, requests, subscriptions, ride_stats, interests`)
```

- [ ] **Step 7: Run integration tests**

```bash
docker compose -f docker-compose.yml -f docker-compose.test.yml up db -d
TEST_DATABASE_URL="postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable" \
  go test -tags integration -count=1 -v \
  -run "TestHTTP_Interest|TestHTTP_PublicRide" \
  ./internal/boundaries/handler/...
```

Expected: all 5 tests PASS

- [ ] **Step 8: Run full integration suite for regressions**

```bash
TEST_DATABASE_URL="postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable" \
  go test -tags integration -count=1 ./internal/boundaries/handler/...
```

Expected: all tests PASS

- [ ] **Step 9: Commit**

```bash
git add internal/boundaries/handler/integration_test.go
git commit -m "test: add interest and public-ride-list integration tests"
git push
```

---

## Task 11: Frontend

**Files:**
- Modify: `web/js/app.js`
- Modify: `web/css/style.css`
- Modify: `web/index.html`

### i18n strings (add to all 6 languages)

Add these after `noActiveRides` in each language block:

**English:**
```js
btnInterest:      'I\'m interested',
interestSent:     'Request sent — you\'ll be notified when the driver accepts.',
interestPending:  'Waiting for driver',
pendingInterests: (n) => n === 1 ? '1 person interested' : `${n} people interested`,
btnAccept:        'Accept & share my number',
contactRevealed:  'Contact accepted',
theirNumber:      'Their number:',
```

**French:**
```js
btnInterest:      'Je suis intéressé(e)',
interestSent:     'Demande envoyée — vous serez alerté(e) lorsque le conducteur accepte.',
interestPending:  'En attente du conducteur',
pendingInterests: (n) => n === 1 ? '1 personne intéressée' : `${n} personnes intéressées`,
btnAccept:        'Accepter et partager mon numéro',
contactRevealed:  'Contact accepté',
theirNumber:      'Leur numéro :',
```

Add equivalent translations for ES, IT, DE, NL.

### Home feed and search results: "I'm interested" button

In `loadHomeFeed()`, replace the simple ride card with one that has an interest button. The public ride objects from `GET /api/rides` now have NO phone or name:

```js
el.innerHTML = `
  <div class="home-feed">
    <div class="home-feed-title">${s.homeFeedTitle}</div>
    ${rides.map(r => `
      <div class="home-feed-card">
        <span class="home-feed-route">${esc(r.Origin)} → ${esc(r.Destination)}</span>
        <span class="home-feed-meta">${formatTime(r.DepartureAt)} <span class="tag">${s.flexLabel[r.Flexibility] || esc(r.Flexibility) + ' min'}</span></span>
        <button class="btn-interest" data-ride-id="${esc(r.ID)}">${s.btnInterest}</button>
        <div class="interest-state hidden" id="int-state-${esc(r.ID)}"></div>
      </div>`).join('')}
  </div>`;
el.querySelectorAll('.btn-interest').forEach(btn => {
  btn.onclick = () => handleInterestClick(btn, s);
});
```

In `renderSearchRides`, similarly add the interest button to each ride card (replacing the phone display):

```js
function rideCard(r) {
  return `<div class="card card-compact">
    <div class="card-meta">${formatTime(r.DepartureAt)} <span class="tag">${s.flexLabel[r.Flexibility] || esc(r.Flexibility) + ' min'}</span></div>
    <button class="btn-interest" data-ride-id="${esc(r.ID)}">${s.btnInterest}</button>
    <div class="interest-state hidden" id="int-state-${esc(r.ID)}"></div>
  </div>`;
}
```

After `results.innerHTML = ...`:
```js
results.querySelectorAll('.btn-interest').forEach(btn => {
  btn.onclick = () => handleInterestClick(btn, s);
});
```

### handleInterestClick function

```js
async function handleInterestClick(btn, s) {
  const rideID = btn.dataset.rideId;
  const p = getProfile();
  let phone = p.phone;
  if (!phone) {
    phone = prompt(s.labelPhone);
    if (!phone) return;
    saveProfile('', phone);
  }
  try {
    btn.disabled = true;
    const res = await api('POST', `/rides/${rideID}/interest`, { phone });
    localStorage.setItem('interest_' + rideID, res.id);
    btn.textContent = s.interestPending;
    const stateEl = document.getElementById('int-state-' + rideID);
    if (stateEl) {
      stateEl.textContent = s.interestSent;
      stateEl.classList.remove('hidden');
    }
  } catch (err) {
    btn.disabled = false;
    alert(err.message);
  }
}
```

### Mes Trajets: show pending interests per ride, with accept button

In `renderMyRides`, after rendering ride cards and loading seekers, also fetch interests for each ride and show an accept button for pending ones:

Replace the seekers fetch section with one that also handles interests:

```js
rides.forEach(r => {
  // Seekers (matching requests)
  api('GET', `/rides/${r.ID}/requests`, null, { 'X-Phone': phone }).then(reqs => {
    const el = document.getElementById('seekers-' + r.ID);
    if (!el) return;
    if (!reqs || !reqs.length) {
      el.innerHTML = `<span class="seekers-empty">${s.noSeekers}</span>`;
    } else {
      el.innerHTML = `<div class="seekers-title">${s.seekersTitle}</div>` +
        reqs.map(req => `
          <div class="seeker-row">
            <strong>${esc(req.SearcherName)}</strong>
            <span class="seeker-meta">${formatTime(req.DepartureAt)} <span class="tag">${s.flexLabel[req.Flexibility] || esc(req.Flexibility) + ' min'}</span></span>
            <a href="tel:${esc(req.Phone)}" class="seeker-phone">${esc(req.Phone)}</a>
          </div>`).join('');
    }
  }).catch(() => {
    const el = document.getElementById('seekers-' + r.ID);
    if (el) el.innerHTML = '';
  });

  // Interests (consent-based contact)
  api('GET', `/rides/${r.ID}/interests`, null, { 'X-Phone': phone }).then(interests => {
    const el = document.getElementById('interests-' + r.ID);
    if (!el || !interests || !interests.length) return;
    el.innerHTML = `<div class="interests-title">${s.pendingInterests(interests.length)}</div>` +
      interests.map(i => `
        <div class="interest-row" id="irow-${esc(i.id)}">
          ${i.status === 'pending'
            ? `<button class="btn-accept-interest" data-id="${esc(i.id)}" data-phone="${esc(phone)}">${s.btnAccept}</button>`
            : `<span class="interest-accepted">${s.contactRevealed}: <a href="tel:${esc(i.searcher_phone)}">${esc(i.searcher_phone)}</a></span>`
          }
        </div>`).join('');
    el.querySelectorAll('.btn-accept-interest').forEach(btn => {
      btn.onclick = async () => {
        try {
          btn.disabled = true;
          const res = await api('POST', `/interests/${btn.dataset.id}/accept`, { phone: btn.dataset.phone });
          document.getElementById('irow-' + btn.dataset.id).innerHTML =
            `<span class="interest-accepted">${s.contactRevealed}: <a href="tel:${esc(res.searcher_phone)}">${esc(res.searcher_phone)}</a></span>`;
        } catch { btn.disabled = false; }
      };
    });
  }).catch(() => {});
});
```

Add `<div id="interests-${esc(r.ID)}"></div>` placeholder inside each ride card in `renderMyRides`.

Note: `GET /api/rides/:id/interests` requires a new backend endpoint — add `rideH.ListInterests` in Task 9 (update main.go and setupRouter). The handler returns `[]map[string]interface{}` with `{id, status, searcher_phone (only if accepted)}`.

### Deep link: /interests/:id

Add to `handleDeepLink` switch:
```js
case '/interests': {
  // path is /interests/:id
  const id = path.split('/')[2];
  if (id) {
    await renderInterestContact(id);
    return true;
  }
  break;
}
```

Add function:
```js
async function renderInterestContact(interestID) {
  pushRoute('/interests/' + interestID);
  const s = t();
  const p = getProfile();
  app.innerHTML = `
    ${pageBar()}
    <h2>${s.contactRevealed}</h2>
    <div id="contact-result"><p class="section-hint">…</p></div>`;
  document.getElementById('back').onclick = renderHome;
  bindControls();

  try {
    const res = await api('GET', `/interests/${interestID}/contact`, null, { 'X-Phone': p.phone });
    document.getElementById('contact-result').innerHTML = `
      <div class="card">
        <div class="card-contact">
          <p>${s.theirNumber} <a href="tel:${esc(res.phone)}">${esc(res.phone)}</a></p>
        </div>
      </div>`;
  } catch (err) {
    document.getElementById('contact-result').innerHTML =
      `<p class="error">${esc(err.message)}</p>`;
  }
}
```

### CSS additions

```css
.btn-interest {
  margin-top: 6px;
  background: none;
  border: 1px solid var(--blue);
  border-radius: var(--radius);
  color: var(--blue);
  font-size: 0.85rem;
  padding: 5px 10px;
  cursor: pointer;
  width: 100%;
}
.btn-interest:hover { background: var(--blue); color: white; }
.btn-interest:disabled { opacity: 0.5; cursor: default; }
.interest-state { font-size: 0.8rem; color: var(--green); margin-top: 4px; }
.interests-title { font-size: 0.75rem; font-weight: 700; text-transform: uppercase;
  letter-spacing: 0.05em; color: var(--gray-600); margin: 8px 0 4px; }
.interest-row { padding: 4px 0; border-top: 1px solid var(--gray-100); }
.interest-accepted { font-size: 0.85rem; color: var(--green); }
.btn-accept-interest {
  background: var(--green); color: white; border: none;
  border-radius: var(--radius); padding: 6px 10px; font-size: 0.85rem; cursor: pointer; width: 100%;
}
.btn-accept-interest:hover { opacity: 0.9; }
```

### Cache buster + push to devstack

```bash
# Bump index.html cache buster (increment v= by 1)
docker cp web go-stop-app-1:/app/
```

- [ ] **Step 1: Add i18n strings to all 6 languages**
- [ ] **Step 2: Update home feed ride cards with interest button**
- [ ] **Step 3: Update search results ride cards with interest button**
- [ ] **Step 4: Add `handleInterestClick` function**
- [ ] **Step 5: Update `renderMyRides` to show interest rows + accept buttons**
- [ ] **Step 6: Add `renderInterestContact` function and `/interests/:id` deep link**
- [ ] **Step 7: Add `GET /api/rides/:id/interests` endpoint (see Task 9 addendum)**
- [ ] **Step 8: Add CSS**
- [ ] **Step 9: Bump cache buster in index.html**
- [ ] **Step 10: Push to devstack and browser test the full flow:**
  - Open home page → click "Je suis intéressé(e)" on a ride
  - Open "Mes Trajets" → see the interest count → click Accept
  - Verify contact phone appears after accept
  - Verify phone does NOT appear in the public ride list response (check Network tab)
- [ ] **Step 11: Commit**

```bash
git add web/
git commit -m "feat: interest button on ride cards, accept flow in Mes Trajets, contact deep link"
git push
```

---

## Task 9 Addendum: GET /api/rides/:id/interests Endpoint

Add to `ride_handler.go`:

```go
func (h *RideHandler) ListInterests(c *gin.Context) {
	phone := normalizePhone(c.GetHeader("X-Phone"))
	if phone == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Phone header required"})
		return
	}
	ride, err := h.rideRepo.FindByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if ride.Phone != phone {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}
	interests, err := h.interestRepo.FindByRide(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Return phone only for accepted interests
	type interestResponse struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		SearcherPhone string `json:"searcher_phone,omitempty"`
	}
	out := make([]interestResponse, len(interests))
	for i, interest := range interests {
		out[i] = interestResponse{ID: interest.ID, Status: interest.Status}
		if interest.Status == "accepted" {
			out[i].SearcherPhone = interest.SearcherPhone
		}
	}
	c.JSON(http.StatusOK, out)
}
```

Add `interestRepo repository.InterestRepository` to `RideHandler` struct and `NewRideHandler` params.

Register route: `api.GET("/rides/:id/interests", rideH.ListInterests)`

---

## Self-Review

### Spec Coverage

| Requirement | Task |
|---|---|
| interests table (no FK to rides) | Task 1 |
| Interest domain type | Task 2 |
| InterestRepository interface | Task 2 |
| Postgres implementation | Task 3 |
| ExpressInterest: creates interest, notifies driver, rejects driver=searcher | Task 5 |
| AcceptInterest: auth check, marks accepted, notifies searcher, returns searcher phone | Task 6 |
| GetInterestContact: returns other party's phone, only after accepted, only to authorized | Task 7 |
| Phone stripped from public GET /rides (backend, not just frontend) | Task 8 |
| Interest handler: Express, Accept, GetContact | Task 8 |
| GET /api/rides/:id/interests endpoint (driver only, phone only on accepted) | Task 9 Addendum |
| interests table in test truncation | Task 10 |
| Integration test: express creates record, no phone in response | Task 10 |
| Integration test: driver cannot be searcher | Task 10 |
| Integration test: full accept flow, phones correct per party | Task 10 |
| Integration test: public ride list strips phone | Task 10 |
| Integration test: stranger gets 403 on contact | Task 10 |
| Frontend interest button on home feed | Task 11 |
| Frontend interest button on search results | Task 11 |
| Frontend Mes Trajets accept button | Task 11 |
| Frontend deep link /interests/:id | Task 11 |
| Browser test: full flow | Task 11 |
| Commit before push | All tasks |

### Placeholder Scan

No TBDs found.

### Type Consistency

- `domain.Interest.Status` is `string` throughout — set to `"pending"` at creation, `"accepted"` on accept ✓
- `interestResponse.SearcherPhone` uses `omitempty` — absent when pending ✓
- `handleInterestClick` stores interest ID in `localStorage` as `interest_${rideID}` for potential reuse ✓
- `normalizePhone` applied to all phone inputs in interest handler ✓
- The `GET /api/rides/:id/interests` endpoint requires the `RideHandler` to hold `interestRepo` — Task 9 Addendum adds this field; Task 9 must wire it before Task 11 tests ✓
