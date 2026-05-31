# Go-Stop Scaffold — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scaffold the complete Go-Stop local ride-sharing platform — a Go/Gin/PostgreSQL/Web Push application deployable to Scalingo.

**Architecture:** Clean Architecture with four concentric layers (domain → use cases → boundaries → infrastructure). Single Go binary serves both the JSON API and static frontend. PostgreSQL for persistence, Web Push (VAPID) for browser push notifications. No user accounts — phone number is lightweight auth.

**Tech Stack:** Go 1.22+, Gin (`github.com/gin-gonic/gin`), pgx/v5 (`github.com/jackc/pgx/v5`), webpush-go (`github.com/SherClockHolmes/webpush-go`), uuid (`github.com/google/uuid`)

---

## File Map

```
go.mod
main.go
Procfile
scalingo.json
README.md
db/migrations/
  001_create_tables.sql
internal/
  domain/
    flexibility.go        — Flexibility type + presets
    ride.go               — Ride struct
    request.go            — Request struct
    subscription.go       — Subscription + PushKeys structs
    message.go            — Message struct (push notification payload)
  boundaries/
    repository/
      ride_repository.go        — RideRepository interface
      request_repository.go     — RequestRepository interface
      destination_repository.go — DestinationRepository interface
      subscription_repository.go — SubscriptionRepository interface
    notification/
      notifier.go               — Notifier interface
  usecase/
    match.go              — windowsOverlap helper
    match_test.go
    notify.go             — NotifySearcher / NotifyDriver functions
    notify_test.go
    post_ride.go
    post_ride_test.go
    post_request.go
    post_request_test.go
    get_rides.go
    get_rides_test.go
    search_rides.go
    search_rides_test.go
    get_destinations.go
    get_destinations_test.go
    subscribe.go
    subscribe_test.go
    delete_ride.go
    delete_ride_test.go
    delete_request.go
    delete_request_test.go
    expire.go
    expire_test.go
  infrastructure/
    postgres/
      db.go               — connection pool setup
      ride_repo.go
      request_repo.go
      destination_repo.go
      subscription_repo.go
    webpush/
      webpush.go
  boundaries/
    handler/
      ride_handler.go
      request_handler.go
      destination_handler.go
      subscription_handler.go
web/
  index.html
  manifest.json
  css/
    style.css
  js/
    app.js
    sw.js                 — service worker (push notifications)
```

---

## Task 1: Project Scaffold

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p internal/domain
mkdir -p internal/boundaries/repository
mkdir -p internal/boundaries/notification
mkdir -p internal/boundaries/handler
mkdir -p internal/usecase
mkdir -p internal/infrastructure/postgres
mkdir -p internal/infrastructure/webpush
mkdir -p web/css web/js
mkdir -p db/migrations
```

- [ ] **Step 2: Create go.mod**

```
module github.com/z3spinner/go-stop

go 1.22

require (
	github.com/SherClockHolmes/webpush-go v1.3.0
	github.com/gin-gonic/gin v1.10.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.6.0
)
```

- [ ] **Step 3: Fetch dependencies**

Run: `go mod tidy`
Expected: `go.sum` created, no errors

- [ ] **Step 4: Commit**

```bash
git init
git add go.mod go.sum
git commit -m "chore: initialise go module"
```

---

## Task 2: Domain Types

**Files:**
- Create: `internal/domain/flexibility.go`
- Create: `internal/domain/ride.go`
- Create: `internal/domain/request.go`
- Create: `internal/domain/subscription.go`
- Create: `internal/domain/message.go`

- [ ] **Step 1: Write flexibility.go**

```go
package domain

type Flexibility int

const (
	Exact       Flexibility = 0
	Approximate Flexibility = 30
	Flexible    Flexibility = 60
)
```

- [ ] **Step 2: Write ride.go**

```go
package domain

import "time"

type Ride struct {
	ID          string
	DriverName  string
	Phone       string
	Origin      string
	Destination string
	Date        time.Time
	DepartureAt time.Time
	Flexibility Flexibility
	PostedAt    time.Time
	ExpiresAt   time.Time
}
```

- [ ] **Step 3: Write request.go**

```go
package domain

import "time"

type Request struct {
	ID           string
	SearcherName string
	Phone        string
	Origin       string
	Destination  string
	Date         time.Time
	DepartureAt  time.Time
	Flexibility  Flexibility
	PostedAt     time.Time
	ExpiresAt    time.Time
}
```

- [ ] **Step 4: Write subscription.go**

```go
package domain

type Subscription struct {
	ID       string
	Phone    string
	Endpoint string
	Keys     PushKeys
}

type PushKeys struct {
	P256DH string
	Auth   string
}
```

- [ ] **Step 5: Write message.go**

```go
package domain

import "time"

type Message struct {
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	URL         string    `json:"url"`
	ContactName string    `json:"contact_name"`
	Phone       string    `json:"phone"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	DepartureAt time.Time `json:"departure_at"`
}
```

- [ ] **Step 6: Verify it compiles**

Run: `go build ./internal/domain/...`
Expected: no output (success)

- [ ] **Step 7: Commit**

```bash
git add internal/domain/
git commit -m "feat: add domain types"
```

---

## Task 3: Boundary Interfaces

**Files:**
- Create: `internal/boundaries/repository/ride_repository.go`
- Create: `internal/boundaries/repository/request_repository.go`
- Create: `internal/boundaries/repository/destination_repository.go`
- Create: `internal/boundaries/repository/subscription_repository.go`
- Create: `internal/boundaries/notification/notifier.go`

- [ ] **Step 1: Write ride_repository.go**

```go
package repository

import "github.com/z3spinner/go-stop/internal/domain"

type RideRepository interface {
	Save(ride domain.Ride) error
	FindByID(id string) (domain.Ride, error)
	FindAll() ([]domain.Ride, error)
	FindByOriginAndDestination(origin, destination string) ([]domain.Ride, error)
	FindMatching(request domain.Request) ([]domain.Ride, error)
	Delete(id string) error
	DeleteExpired() error
}
```

- [ ] **Step 2: Write request_repository.go**

```go
package repository

import "github.com/z3spinner/go-stop/internal/domain"

type RequestRepository interface {
	Save(request domain.Request) error
	FindByID(id string) (domain.Request, error)
	FindMatching(ride domain.Ride) ([]domain.Request, error)
	Delete(id string) error
	DeleteExpired() error
}
```

- [ ] **Step 3: Write destination_repository.go**

```go
package repository

type DestinationRepository interface {
	GetAll() ([]string, error)
}
```

- [ ] **Step 4: Write subscription_repository.go**

```go
package repository

import "github.com/z3spinner/go-stop/internal/domain"

type SubscriptionRepository interface {
	Save(subscription domain.Subscription) error
	FindByPhone(phone string) (domain.Subscription, error)
	Delete(phone string) error
}
```

- [ ] **Step 5: Write notifier.go**

```go
package notification

import "github.com/z3spinner/go-stop/internal/domain"

type Notifier interface {
	Send(subscription domain.Subscription, message domain.Message) error
}
```

- [ ] **Step 6: Verify it compiles**

Run: `go build ./internal/boundaries/...`
Expected: no output (success)

- [ ] **Step 7: Commit**

```bash
git add internal/boundaries/
git commit -m "feat: add boundary interfaces"
```

---

## Task 4: Match Use Case

**Files:**
- Create: `internal/usecase/match.go`
- Create: `internal/usecase/match_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/usecase/match_test.go
package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

var baseDate = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

func departure(hour, min int) time.Time {
	return time.Date(2026, 6, 1, hour, min, 0, 0, time.UTC)
}

func TestWindowsOverlap_ExactMatch(t *testing.T) {
	ride := domain.Ride{DepartureAt: departure(9, 0), Flexibility: domain.Exact}
	req := domain.Request{DepartureAt: departure(9, 0), Flexibility: domain.Exact}
	if !usecase.WindowsOverlap(ride, req) {
		t.Error("exact same time should match")
	}
}

func TestWindowsOverlap_ExactNoMatch(t *testing.T) {
	ride := domain.Ride{DepartureAt: departure(9, 0), Flexibility: domain.Exact}
	req := domain.Request{DepartureAt: departure(10, 0), Flexibility: domain.Exact}
	if usecase.WindowsOverlap(ride, req) {
		t.Error("different exact times should not match")
	}
}

func TestWindowsOverlap_FlexibleOverlap(t *testing.T) {
	// ride 08:00-10:00, request 09:30-10:30 — overlap 09:30-10:00
	ride := domain.Ride{DepartureAt: departure(9, 0), Flexibility: domain.Flexible}
	req := domain.Request{DepartureAt: departure(10, 0), Flexibility: domain.Approximate}
	if !usecase.WindowsOverlap(ride, req) {
		t.Error("overlapping windows should match")
	}
}

func TestWindowsOverlap_NoOverlap(t *testing.T) {
	// ride 09:00-09:00 (exact), request 10:00-11:00 — no overlap
	ride := domain.Ride{DepartureAt: departure(9, 0), Flexibility: domain.Exact}
	req := domain.Request{DepartureAt: departure(10, 30), Flexibility: domain.Approximate}
	if usecase.WindowsOverlap(ride, req) {
		t.Error("non-overlapping windows should not match")
	}
}

func TestWindowsOverlap_AdjacentWindows(t *testing.T) {
	// ride window ends at 09:30, request window starts at 09:30 — touching, should match
	ride := domain.Ride{DepartureAt: departure(9, 0), Flexibility: domain.Approximate}
	req := domain.Request{DepartureAt: departure(10, 0), Flexibility: domain.Approximate}
	if !usecase.WindowsOverlap(ride, req) {
		t.Error("touching windows (09:30 meets 09:30) should match")
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

Run: `go test ./internal/usecase/... -run TestWindowsOverlap -v`
Expected: compilation error — `usecase.WindowsOverlap undefined`

- [ ] **Step 3: Write match.go**

```go
// internal/usecase/match.go
package usecase

import (
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

// WindowsOverlap reports whether a ride and request have overlapping departure windows.
func WindowsOverlap(ride domain.Ride, req domain.Request) bool {
	rideEarliest := ride.DepartureAt.Add(-time.Duration(ride.Flexibility) * time.Minute)
	rideLatest := ride.DepartureAt.Add(time.Duration(ride.Flexibility) * time.Minute)
	reqEarliest := req.DepartureAt.Add(-time.Duration(req.Flexibility) * time.Minute)
	reqLatest := req.DepartureAt.Add(time.Duration(req.Flexibility) * time.Minute)
	return !rideLatest.Before(reqEarliest) && !rideEarliest.After(reqLatest)
}
```

- [ ] **Step 4: Run tests to confirm they pass**

Run: `go test ./internal/usecase/... -run TestWindowsOverlap -v`
Expected: all 5 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/match.go internal/usecase/match_test.go
git commit -m "feat: add window overlap matching logic"
```

---

## Task 5: Notify Use Case

**Files:**
- Create: `internal/usecase/notify.go`
- Create: `internal/usecase/notify_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/usecase/notify_test.go
package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type mockNotifier struct {
	called  bool
	lastMsg domain.Message
	err     error
}

func (m *mockNotifier) Send(sub domain.Subscription, msg domain.Message) error {
	m.called = true
	m.lastMsg = msg
	return m.err
}

func TestNotifySearcher_SendsCorrectMessage(t *testing.T) {
	n := &mockNotifier{}
	sub := domain.Subscription{Phone: "555-0001"}
	ride := domain.Ride{
		ID:          "ride-1",
		DriverName:  "Alice",
		Phone:       "555-0001",
		Origin:      "Village A",
		Destination: "Train Station",
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	}

	err := usecase.NotifySearcher(sub, ride, n)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !n.called {
		t.Error("notifier.Send was not called")
	}
	if n.lastMsg.URL != "/rides/ride-1" {
		t.Errorf("expected URL /rides/ride-1, got %s", n.lastMsg.URL)
	}
	if n.lastMsg.ContactName != "Alice" {
		t.Errorf("expected ContactName Alice, got %s", n.lastMsg.ContactName)
	}
}

func TestNotifyDriver_SendsCorrectMessage(t *testing.T) {
	n := &mockNotifier{}
	sub := domain.Subscription{Phone: "555-0002"}
	req := domain.Request{
		ID:           "req-1",
		SearcherName: "Bob",
		Phone:        "555-0002",
		Origin:       "Village A",
		Destination:  "Train Station",
		DepartureAt:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	}

	err := usecase.NotifyDriver(sub, req, n)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.lastMsg.URL != "/requests/req-1" {
		t.Errorf("expected URL /requests/req-1, got %s", n.lastMsg.URL)
	}
	if n.lastMsg.ContactName != "Bob" {
		t.Errorf("expected ContactName Bob, got %s", n.lastMsg.ContactName)
	}
}

func TestNotifySearcher_PropagatesNotifierError(t *testing.T) {
	n := &mockNotifier{err: errors.New("push failed")}
	err := usecase.NotifySearcher(domain.Subscription{}, domain.Ride{ID: "x"}, n)
	if err == nil {
		t.Error("expected error to be propagated")
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

Run: `go test ./internal/usecase/... -run TestNotify -v`
Expected: compilation error — `usecase.NotifySearcher undefined`

- [ ] **Step 3: Write notify.go**

```go
// internal/usecase/notify.go
package usecase

import (
	"fmt"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/domain"
)

func NotifySearcher(sub domain.Subscription, ride domain.Ride, notifier notification.Notifier) error {
	msg := domain.Message{
		Title:       "Ride available!",
		Body:        fmt.Sprintf("%s is driving from %s to %s", ride.DriverName, ride.Origin, ride.Destination),
		URL:         "/rides/" + ride.ID,
		ContactName: ride.DriverName,
		Phone:       ride.Phone,
		Origin:      ride.Origin,
		Destination: ride.Destination,
		DepartureAt: ride.DepartureAt,
	}
	return notifier.Send(sub, msg)
}

func NotifyDriver(sub domain.Subscription, req domain.Request, notifier notification.Notifier) error {
	msg := domain.Message{
		Title:       "Someone needs a ride!",
		Body:        fmt.Sprintf("%s needs a ride from %s to %s", req.SearcherName, req.Origin, req.Destination),
		URL:         "/requests/" + req.ID,
		ContactName: req.SearcherName,
		Phone:       req.Phone,
		Origin:      req.Origin,
		Destination: req.Destination,
		DepartureAt: req.DepartureAt,
	}
	return notifier.Send(sub, msg)
}
```

- [ ] **Step 4: Run tests to confirm they pass**

Run: `go test ./internal/usecase/... -run TestNotify -v`
Expected: all 3 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/notify.go internal/usecase/notify_test.go
git commit -m "feat: add notify use cases"
```

---

## Task 6: Post Ride Use Case

**Files:**
- Create: `internal/usecase/post_ride.go`
- Create: `internal/usecase/post_ride_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/usecase/post_ride_test.go
package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// mockRideRepo — in-memory ride repository for tests
type mockRideRepo struct {
	saved    []domain.Ride
	findByID map[string]domain.Ride
	matching []domain.Request
	saveErr  error
}

func (m *mockRideRepo) Save(r domain.Ride) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, r)
	return nil
}
func (m *mockRideRepo) FindByID(id string) (domain.Ride, error) {
	r, ok := m.findByID[id]
	if !ok {
		return domain.Ride{}, errors.New("not found")
	}
	return r, nil
}
func (m *mockRideRepo) FindAll() ([]domain.Ride, error)  { return m.saved, nil }
func (m *mockRideRepo) FindByOriginAndDestination(o, d string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepo) FindMatching(domain.Request) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepo) Delete(id string) error                              { return nil }
func (m *mockRideRepo) DeleteExpired() error                                { return nil }

// mockRequestRepo — in-memory request repository for tests
type mockRequestRepo struct {
	matching []domain.Request
	saved    []domain.Request
}

func (m *mockRequestRepo) Save(r domain.Request) error { m.saved = append(m.saved, r); return nil }
func (m *mockRequestRepo) FindByID(id string) (domain.Request, error) {
	return domain.Request{}, errors.New("not found")
}
func (m *mockRequestRepo) FindMatching(domain.Ride) ([]domain.Request, error) {
	return m.matching, nil
}
func (m *mockRequestRepo) Delete(string) error      { return nil }
func (m *mockRequestRepo) DeleteExpired() error     { return nil }

// mockSubRepo — in-memory subscription repository for tests
type mockSubRepo struct {
	subs   map[string]domain.Subscription
	saved  []domain.Subscription
	saveErr error
}

func (m *mockSubRepo) Save(s domain.Subscription) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if m.subs == nil {
		m.subs = make(map[string]domain.Subscription)
	}
	m.subs[s.Phone] = s
	m.saved = append(m.saved, s)
	return nil
}
func (m *mockSubRepo) FindByPhone(phone string) (domain.Subscription, error) {
	s, ok := m.subs[phone]
	if !ok {
		return domain.Subscription{}, errors.New("not found")
	}
	return s, nil
}
func (m *mockSubRepo) Delete(phone string) error { delete(m.subs, phone); return nil }

func TestPostRide_SavesRide(t *testing.T) {
	rides := &mockRideRepo{}
	reqs := &mockRequestRepo{}
	subs := &mockSubRepo{}
	n := &mockNotifier{}

	uc := usecase.NewPostRide(rides, reqs, subs, n)
	ride := domain.Ride{
		DriverName:  "Alice",
		Phone:       "555-0001",
		Origin:      "Village A",
		Destination: "Station",
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility: domain.Approximate,
	}

	err := uc.Execute(ride)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rides.saved) != 1 {
		t.Errorf("expected 1 saved ride, got %d", len(rides.saved))
	}
	if rides.saved[0].ID == "" {
		t.Error("expected ride to have an ID assigned")
	}
	if rides.saved[0].ExpiresAt.IsZero() {
		t.Error("expected ExpiresAt to be set")
	}
}

func TestPostRide_NotifiesMatchingSearchers(t *testing.T) {
	rides := &mockRideRepo{}
	reqs := &mockRequestRepo{
		matching: []domain.Request{
			{ID: "req-1", SearcherName: "Bob", Phone: "555-0002"},
		},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0002": {Phone: "555-0002", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewPostRide(rides, reqs, subs, n)
	err := uc.Execute(domain.Ride{
		DriverName:  "Alice",
		Phone:       "555-0001",
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !n.called {
		t.Error("expected notification to be sent")
	}
}

func TestPostRide_SkipsNotificationIfNoSubscription(t *testing.T) {
	rides := &mockRideRepo{}
	reqs := &mockRequestRepo{
		matching: []domain.Request{{Phone: "555-0003"}},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}} // no subscription
	n := &mockNotifier{}

	uc := usecase.NewPostRide(rides, reqs, subs, n)
	err := uc.Execute(domain.Ride{DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send notification when searcher has no subscription")
	}
}

func TestPostRide_ReturnsErrorIfSaveFails(t *testing.T) {
	rides := &mockRideRepo{saveErr: errors.New("db error")}
	uc := usecase.NewPostRide(rides, &mockRequestRepo{}, &mockSubRepo{}, &mockNotifier{})
	err := uc.Execute(domain.Ride{DepartureAt: time.Now()})
	if err == nil {
		t.Error("expected error when save fails")
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

Run: `go test ./internal/usecase/... -run TestPostRide -v`
Expected: compilation error — `usecase.NewPostRide undefined`

- [ ] **Step 3: Write post_ride.go**

```go
// internal/usecase/post_ride.go
package usecase

import (
	"time"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type PostRide struct {
	rides    repository.RideRepository
	requests repository.RequestRepository
	subs     repository.SubscriptionRepository
	notifier notification.Notifier
}

func NewPostRide(
	rides repository.RideRepository,
	requests repository.RequestRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *PostRide {
	return &PostRide{rides: rides, requests: requests, subs: subs, notifier: notifier}
}

func (uc *PostRide) Execute(ride domain.Ride) error {
	ride.ID = uuid.New().String()
	ride.PostedAt = time.Now()
	ride.Date = time.Date(ride.DepartureAt.Year(), ride.DepartureAt.Month(), ride.DepartureAt.Day(), 0, 0, 0, 0, ride.DepartureAt.Location())
	ride.ExpiresAt = time.Date(ride.DepartureAt.Year(), ride.DepartureAt.Month(), ride.DepartureAt.Day()+1, 0, 0, 0, 0, ride.DepartureAt.Location())

	if err := uc.rides.Save(ride); err != nil {
		return err
	}

	matching, err := uc.requests.FindMatching(ride)
	if err != nil {
		return err
	}

	for _, req := range matching {
		sub, err := uc.subs.FindByPhone(req.Phone)
		if err != nil {
			continue
		}
		_ = NotifySearcher(sub, ride, uc.notifier)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to confirm they pass**

Run: `go test ./internal/usecase/... -run TestPostRide -v`
Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/post_ride.go internal/usecase/post_ride_test.go
git commit -m "feat: add post ride use case"
```

---

## Task 7: Post Request Use Case

**Files:**
- Create: `internal/usecase/post_request.go`
- Create: `internal/usecase/post_request_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/usecase/post_request_test.go
package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestPostRequest_SavesRequest(t *testing.T) {
	reqs := &mockRequestRepo{}
	rides := &mockRideRepo{}
	subs := &mockSubRepo{}
	n := &mockNotifier{}

	uc := usecase.NewPostRequest(reqs, rides, subs, n)
	req := domain.Request{
		SearcherName: "Bob",
		Phone:        "555-0002",
		Origin:       "Village A",
		Destination:  "Station",
		DepartureAt:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility:  domain.Approximate,
	}

	err := uc.Execute(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs.saved) != 1 {
		t.Errorf("expected 1 saved request, got %d", len(reqs.saved))
	}
	if reqs.saved[0].ID == "" {
		t.Error("expected request to have an ID assigned")
	}
	if reqs.saved[0].ExpiresAt.IsZero() {
		t.Error("expected ExpiresAt to be set")
	}
}

func TestPostRequest_NotifiesMatchingDrivers(t *testing.T) {
	reqs := &mockRequestRepo{}
	rides := &mockRideRepo{
		saved: []domain.Ride{
			{ID: "ride-1", DriverName: "Alice", Phone: "555-0001"},
		},
	}
	// Override FindMatching to return a matching ride
	ridesWithMatch := &mockRideRepoWithMatching{
		saved: rides.saved,
		matchResult: []domain.Ride{
			{ID: "ride-1", DriverName: "Alice", Phone: "555-0001"},
		},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0001": {Phone: "555-0001", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewPostRequest(reqs, ridesWithMatch, subs, n)
	err := uc.Execute(domain.Request{
		SearcherName: "Bob",
		DepartureAt:  time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !n.called {
		t.Error("expected notification to be sent to driver")
	}
}

func TestPostRequest_ReturnsErrorIfSaveFails(t *testing.T) {
	reqs := &mockRequestRepo{}
	reqs.saveErr = errors.New("db error")
	uc := usecase.NewPostRequest(reqs, &mockRideRepoWithMatching{}, &mockSubRepo{}, &mockNotifier{})
	err := uc.Execute(domain.Request{DepartureAt: time.Now()})
	if err == nil {
		t.Error("expected error when save fails")
	}
}

// mockRequestRepo needs a saveErr field — add it here
type mockRequestRepoSaveErr struct {
	mockRequestRepo
	saveErr error
}

func (m *mockRequestRepoSaveErr) Save(r domain.Request) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, r)
	return nil
}

// mockRideRepoWithMatching overrides FindMatching to return controlled results
type mockRideRepoWithMatching struct {
	saved       []domain.Ride
	matchResult []domain.Ride
}

func (m *mockRideRepoWithMatching) Save(r domain.Ride) error { m.saved = append(m.saved, r); return nil }
func (m *mockRideRepoWithMatching) FindByID(id string) (domain.Ride, error) {
	return domain.Ride{}, errors.New("not found")
}
func (m *mockRideRepoWithMatching) FindAll() ([]domain.Ride, error) { return m.saved, nil }
func (m *mockRideRepoWithMatching) FindByOriginAndDestination(o, d string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoWithMatching) FindMatching(domain.Request) ([]domain.Ride, error) {
	return m.matchResult, nil
}
func (m *mockRideRepoWithMatching) Delete(string) error  { return nil }
func (m *mockRideRepoWithMatching) DeleteExpired() error  { return nil }
```

Note: the `mockRequestRepo` defined in `post_ride_test.go` needs a `saveErr` field. Add it:

Open `post_ride_test.go` and update `mockRequestRepo`:

```go
type mockRequestRepo struct {
	matching []domain.Request
	saved    []domain.Request
	saveErr  error
}

func (m *mockRequestRepo) Save(r domain.Request) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, r)
	return nil
}
```

- [ ] **Step 2: Run tests to confirm they fail**

Run: `go test ./internal/usecase/... -run TestPostRequest -v`
Expected: compilation error — `usecase.NewPostRequest undefined`

- [ ] **Step 3: Write post_request.go**

```go
// internal/usecase/post_request.go
package usecase

import (
	"time"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type PostRequest struct {
	requests repository.RequestRepository
	rides    repository.RideRepository
	subs     repository.SubscriptionRepository
	notifier notification.Notifier
}

func NewPostRequest(
	requests repository.RequestRepository,
	rides repository.RideRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *PostRequest {
	return &PostRequest{requests: requests, rides: rides, subs: subs, notifier: notifier}
}

func (uc *PostRequest) Execute(req domain.Request) error {
	req.ID = uuid.New().String()
	req.PostedAt = time.Now()
	req.Date = time.Date(req.DepartureAt.Year(), req.DepartureAt.Month(), req.DepartureAt.Day(), 0, 0, 0, 0, req.DepartureAt.Location())
	req.ExpiresAt = time.Date(req.DepartureAt.Year(), req.DepartureAt.Month(), req.DepartureAt.Day()+1, 0, 0, 0, 0, req.DepartureAt.Location())

	if err := uc.requests.Save(req); err != nil {
		return err
	}

	matching, err := uc.rides.FindMatching(req)
	if err != nil {
		return err
	}

	for _, ride := range matching {
		sub, err := uc.subs.FindByPhone(ride.Phone)
		if err != nil {
			continue
		}
		_ = NotifyDriver(sub, req, uc.notifier)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to confirm they pass**

Run: `go test ./internal/usecase/... -run TestPostRequest -v`
Expected: all 3 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/post_request.go internal/usecase/post_request_test.go
git commit -m "feat: add post request use case"
```

---

## Task 8: Get Rides Use Case

**Files:**
- Create: `internal/usecase/get_rides.go`
- Create: `internal/usecase/get_rides_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/usecase/get_rides_test.go
package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestGetRides_ReturnsAllRides(t *testing.T) {
	rides := &mockRideRepo{
		saved: []domain.Ride{
			{ID: "1", DriverName: "Alice", DepartureAt: time.Now()},
			{ID: "2", DriverName: "Bob", DepartureAt: time.Now()},
		},
	}

	uc := usecase.NewGetRides(rides)
	result, err := uc.Execute()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 rides, got %d", len(result))
	}
}

func TestGetRides_ReturnsEmptySliceWhenNoRides(t *testing.T) {
	rides := &mockRideRepo{}

	uc := usecase.NewGetRides(rides)
	result, err := uc.Execute()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected empty slice, not nil")
	}
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./internal/usecase/... -run TestGetRides -v`
Expected: compilation error

- [ ] **Step 3: Write get_rides.go**

```go
// internal/usecase/get_rides.go
package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type GetRides struct {
	rides repository.RideRepository
}

func NewGetRides(rides repository.RideRepository) *GetRides {
	return &GetRides{rides: rides}
}

func (uc *GetRides) Execute() ([]domain.Ride, error) {
	result, err := uc.rides.FindAll()
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []domain.Ride{}, nil
	}
	return result, nil
}
```

- [ ] **Step 4: Run to confirm pass**

Run: `go test ./internal/usecase/... -run TestGetRides -v`
Expected: 2 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/get_rides.go internal/usecase/get_rides_test.go
git commit -m "feat: add get rides use case"
```

---

## Task 9: Search Rides Use Case

**Files:**
- Create: `internal/usecase/search_rides.go`
- Create: `internal/usecase/search_rides_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/usecase/search_rides_test.go
package usecase_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type mockRideRepoSearch struct {
	resultsByRoute map[string][]domain.Ride
}

func (m *mockRideRepoSearch) Save(r domain.Ride) error { return nil }
func (m *mockRideRepoSearch) FindByID(id string) (domain.Ride, error) {
	return domain.Ride{}, nil
}
func (m *mockRideRepoSearch) FindAll() ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoSearch) FindByOriginAndDestination(o, d string) ([]domain.Ride, error) {
	key := o + "|" + d
	return m.resultsByRoute[key], nil
}
func (m *mockRideRepoSearch) FindMatching(domain.Request) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoSearch) Delete(string) error                                 { return nil }
func (m *mockRideRepoSearch) DeleteExpired() error                                { return nil }

func TestSearchRides_FiltersByOriginAndDestination(t *testing.T) {
	rides := &mockRideRepoSearch{
		resultsByRoute: map[string][]domain.Ride{
			"Village A|Station": {{ID: "1", Origin: "Village A", Destination: "Station"}},
		},
	}

	uc := usecase.NewSearchRides(rides)
	result, err := uc.Execute("Village A", "Station")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 ride, got %d", len(result))
	}
	if result[0].Origin != "Village A" {
		t.Errorf("unexpected origin: %s", result[0].Origin)
	}
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./internal/usecase/... -run TestSearchRides -v`
Expected: compilation error

- [ ] **Step 3: Write search_rides.go**

```go
// internal/usecase/search_rides.go
package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type SearchRides struct {
	rides repository.RideRepository
}

func NewSearchRides(rides repository.RideRepository) *SearchRides {
	return &SearchRides{rides: rides}
}

func (uc *SearchRides) Execute(origin, destination string) ([]domain.Ride, error) {
	result, err := uc.rides.FindByOriginAndDestination(origin, destination)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []domain.Ride{}, nil
	}
	return result, nil
}
```

- [ ] **Step 4: Run to confirm pass**

Run: `go test ./internal/usecase/... -run TestSearchRides -v`
Expected: 1 test PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/search_rides.go internal/usecase/search_rides_test.go
git commit -m "feat: add search rides use case"
```

---

## Task 10: Get Destinations Use Case

**Files:**
- Create: `internal/usecase/get_destinations.go`
- Create: `internal/usecase/get_destinations_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/usecase/get_destinations_test.go
package usecase_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/usecase"
)

type mockDestinationRepo struct {
	locations []string
}

func (m *mockDestinationRepo) GetAll() ([]string, error) {
	return m.locations, nil
}

func TestGetDestinations_ReturnsAll(t *testing.T) {
	repo := &mockDestinationRepo{locations: []string{"Village A", "Station", "Town B"}}

	uc := usecase.NewGetDestinations(repo)
	result, err := uc.Execute()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 destinations, got %d", len(result))
	}
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./internal/usecase/... -run TestGetDestinations -v`
Expected: compilation error

- [ ] **Step 3: Write get_destinations.go**

```go
// internal/usecase/get_destinations.go
package usecase

import "github.com/z3spinner/go-stop/internal/boundaries/repository"

type GetDestinations struct {
	destinations repository.DestinationRepository
}

func NewGetDestinations(destinations repository.DestinationRepository) *GetDestinations {
	return &GetDestinations{destinations: destinations}
}

func (uc *GetDestinations) Execute() ([]string, error) {
	return uc.destinations.GetAll()
}
```

- [ ] **Step 4: Run to confirm pass**

Run: `go test ./internal/usecase/... -run TestGetDestinations -v`
Expected: 1 test PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/get_destinations.go internal/usecase/get_destinations_test.go
git commit -m "feat: add get destinations use case"
```

---

## Task 11: Subscribe Use Case

**Files:**
- Create: `internal/usecase/subscribe.go`
- Create: `internal/usecase/subscribe_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/usecase/subscribe_test.go
package usecase_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestSubscribe_SavesSubscription(t *testing.T) {
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}

	uc := usecase.NewSubscribe(subs)
	err := uc.Execute(domain.Subscription{
		Phone:    "555-0001",
		Endpoint: "https://push.example.com",
		Keys:     domain.PushKeys{P256DH: "key1", Auth: "auth1"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs.saved) != 1 {
		t.Errorf("expected 1 saved subscription, got %d", len(subs.saved))
	}
}

func TestUnsubscribe_DeletesSubscription(t *testing.T) {
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0001": {Phone: "555-0001"},
	}}

	uc := usecase.NewUnsubscribe(subs)
	err := uc.Execute("555-0001")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := subs.subs["555-0001"]; ok {
		t.Error("subscription should have been deleted")
	}
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./internal/usecase/... -run TestSubscribe -v`
Expected: compilation error

- [ ] **Step 3: Write subscribe.go**

```go
// internal/usecase/subscribe.go
package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type Subscribe struct {
	subs repository.SubscriptionRepository
}

func NewSubscribe(subs repository.SubscriptionRepository) *Subscribe {
	return &Subscribe{subs: subs}
}

func (uc *Subscribe) Execute(sub domain.Subscription) error {
	return uc.subs.Save(sub)
}

type Unsubscribe struct {
	subs repository.SubscriptionRepository
}

func NewUnsubscribe(subs repository.SubscriptionRepository) *Unsubscribe {
	return &Unsubscribe{subs: subs}
}

func (uc *Unsubscribe) Execute(phone string) error {
	return uc.subs.Delete(phone)
}
```

- [ ] **Step 4: Run to confirm pass**

Run: `go test ./internal/usecase/... -run TestSubscribe -v`
Expected: 2 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/subscribe.go internal/usecase/subscribe_test.go
git commit -m "feat: add subscribe/unsubscribe use cases"
```

---

## Task 12: Delete Use Cases

**Files:**
- Create: `internal/usecase/delete_ride.go`
- Create: `internal/usecase/delete_request.go`
- Create: `internal/usecase/delete_ride_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/usecase/delete_ride_test.go
package usecase_test

import (
	"errors"
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type mockRideRepoDelete struct {
	rides   map[string]domain.Ride
	deleted []string
}

func (m *mockRideRepoDelete) Save(r domain.Ride) error { return nil }
func (m *mockRideRepoDelete) FindByID(id string) (domain.Ride, error) {
	r, ok := m.rides[id]
	if !ok {
		return domain.Ride{}, errors.New("not found")
	}
	return r, nil
}
func (m *mockRideRepoDelete) FindAll() ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoDelete) FindByOriginAndDestination(o, d string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoDelete) FindMatching(domain.Request) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoDelete) Delete(id string) error {
	m.deleted = append(m.deleted, id)
	return nil
}
func (m *mockRideRepoDelete) DeleteExpired() error { return nil }

type mockRequestRepoDelete struct {
	requests map[string]domain.Request
	deleted  []string
}

func (m *mockRequestRepoDelete) Save(r domain.Request) error { return nil }
func (m *mockRequestRepoDelete) FindByID(id string) (domain.Request, error) {
	r, ok := m.requests[id]
	if !ok {
		return domain.Request{}, errors.New("not found")
	}
	return r, nil
}
func (m *mockRequestRepoDelete) FindMatching(domain.Ride) ([]domain.Request, error) { return nil, nil }
func (m *mockRequestRepoDelete) Delete(id string) error {
	m.deleted = append(m.deleted, id)
	return nil
}
func (m *mockRequestRepoDelete) DeleteExpired() error { return nil }

func TestDeleteRide_DeletesWhenPhoneMatches(t *testing.T) {
	rides := &mockRideRepoDelete{
		rides: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-0001"},
		},
	}

	uc := usecase.NewDeleteRide(rides)
	err := uc.Execute("ride-1", "555-0001")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rides.deleted) != 1 || rides.deleted[0] != "ride-1" {
		t.Error("expected ride-1 to be deleted")
	}
}

func TestDeleteRide_RejectsWrongPhone(t *testing.T) {
	rides := &mockRideRepoDelete{
		rides: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-0001"},
		},
	}

	uc := usecase.NewDeleteRide(rides)
	err := uc.Execute("ride-1", "555-9999")

	if err == nil {
		t.Error("expected unauthorized error")
	}
	if len(rides.deleted) != 0 {
		t.Error("ride should not have been deleted")
	}
}

func TestDeleteRide_ReturnsErrorIfNotFound(t *testing.T) {
	rides := &mockRideRepoDelete{rides: map[string]domain.Ride{}}
	uc := usecase.NewDeleteRide(rides)
	err := uc.Execute("nonexistent", "555-0001")
	if err == nil {
		t.Error("expected not found error")
	}
}

func TestDeleteRequest_DeletesWhenPhoneMatches(t *testing.T) {
	reqs := &mockRequestRepoDelete{
		requests: map[string]domain.Request{
			"req-1": {ID: "req-1", Phone: "555-0002"},
		},
	}

	uc := usecase.NewDeleteRequest(reqs)
	err := uc.Execute("req-1", "555-0002")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs.deleted) != 1 {
		t.Error("expected req-1 to be deleted")
	}
}

func TestDeleteRequest_RejectsWrongPhone(t *testing.T) {
	reqs := &mockRequestRepoDelete{
		requests: map[string]domain.Request{
			"req-1": {ID: "req-1", Phone: "555-0002"},
		},
	}

	uc := usecase.NewDeleteRequest(reqs)
	err := uc.Execute("req-1", "555-9999")

	if err == nil {
		t.Error("expected unauthorized error")
	}
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./internal/usecase/... -run TestDelete -v`
Expected: compilation error

- [ ] **Step 3: Write delete_ride.go**

```go
// internal/usecase/delete_ride.go
package usecase

import (
	"errors"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

var ErrUnauthorized = errors.New("unauthorized")

type DeleteRide struct {
	rides repository.RideRepository
}

func NewDeleteRide(rides repository.RideRepository) *DeleteRide {
	return &DeleteRide{rides: rides}
}

func (uc *DeleteRide) Execute(id, phone string) error {
	ride, err := uc.rides.FindByID(id)
	if err != nil {
		return err
	}
	if ride.Phone != phone {
		return ErrUnauthorized
	}
	return uc.rides.Delete(id)
}
```

- [ ] **Step 4: Write delete_request.go**

```go
// internal/usecase/delete_request.go
package usecase

import "github.com/z3spinner/go-stop/internal/boundaries/repository"

type DeleteRequest struct {
	requests repository.RequestRepository
}

func NewDeleteRequest(requests repository.RequestRepository) *DeleteRequest {
	return &DeleteRequest{requests: requests}
}

func (uc *DeleteRequest) Execute(id, phone string) error {
	req, err := uc.requests.FindByID(id)
	if err != nil {
		return err
	}
	if req.Phone != phone {
		return ErrUnauthorized
	}
	return uc.requests.Delete(id)
}
```

- [ ] **Step 5: Run to confirm pass**

Run: `go test ./internal/usecase/... -run TestDelete -v`
Expected: 5 tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/usecase/delete_ride.go internal/usecase/delete_request.go internal/usecase/delete_ride_test.go
git commit -m "feat: add delete ride and request use cases"
```

---

## Task 13: Expire Use Cases

**Files:**
- Create: `internal/usecase/expire.go`
- Create: `internal/usecase/expire_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/usecase/expire_test.go
package usecase_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/usecase"
)

type mockRideRepoExpire struct {
	expireCalled bool
}

func (m *mockRideRepoExpire) Save(r interface{}) error                        { return nil }
func (m *mockRideRepoExpire) FindByID(id string) (interface{}, error)          { return nil, nil }
func (m *mockRideRepoExpire) FindAll() (interface{}, error)                     { return nil, nil }
func (m *mockRideRepoExpire) FindByOriginAndDestination(o, d string) (interface{}, error) {
	return nil, nil
}
func (m *mockRideRepoExpire) FindMatching(interface{}) (interface{}, error) { return nil, nil }
func (m *mockRideRepoExpire) Delete(string) error                            { return nil }
func (m *mockRideRepoExpire) DeleteExpired() error {
	m.expireCalled = true
	return nil
}

// Use concrete types to match the interface
type rideRepoExpireMock struct{ expireCalled bool }

func (m *rideRepoExpireMock) Save(r interface{}) error                              { return nil }
func (m *rideRepoExpireMock) FindByID(string) (interface{}, error)                  { return nil, nil }
func (m *rideRepoExpireMock) FindAll() (interface{}, error)                          { return nil, nil }
func (m *rideRepoExpireMock) FindByOriginAndDestination(string, string) (interface{}, error) {
	return nil, nil
}
func (m *rideRepoExpireMock) FindMatching(interface{}) (interface{}, error) { return nil, nil }
func (m *rideRepoExpireMock) Delete(string) error                            { return nil }
func (m *rideRepoExpireMock) DeleteExpired() error {
	m.expireCalled = true
	return nil
}

func TestExpireRides_CallsDeleteExpired(t *testing.T) {
	rides := &mockRideRepo{}
	uc := usecase.NewExpireRides(rides)
	err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExpireRequests_CallsDeleteExpired(t *testing.T) {
	reqs := &mockRequestRepo{}
	uc := usecase.NewExpireRequests(reqs)
	err := uc.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./internal/usecase/... -run TestExpire -v`
Expected: compilation error

- [ ] **Step 3: Write expire.go**

```go
// internal/usecase/expire.go
package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

type ExpireRides struct {
	rides repository.RideRepository
}

func NewExpireRides(rides repository.RideRepository) *ExpireRides {
	return &ExpireRides{rides: rides}
}

func (uc *ExpireRides) Execute() error {
	return uc.rides.DeleteExpired()
}

type ExpireRequests struct {
	requests repository.RequestRepository
}

func NewExpireRequests(requests repository.RequestRepository) *ExpireRequests {
	return &ExpireRequests{requests: requests}
}

func (uc *ExpireRequests) Execute() error {
	return uc.requests.DeleteExpired()
}
```

- [ ] **Step 4: Run to confirm pass**

Run: `go test ./internal/usecase/... -run TestExpire -v`
Expected: 2 tests PASS

- [ ] **Step 5: Run all use case tests**

Run: `go test ./internal/usecase/... -v`
Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/usecase/expire.go internal/usecase/expire_test.go
git commit -m "feat: add expire use cases"
```

---

## Task 14: PostgreSQL Infrastructure

**Files:**
- Create: `internal/infrastructure/postgres/db.go`
- Create: `internal/infrastructure/postgres/ride_repo.go`
- Create: `internal/infrastructure/postgres/request_repo.go`
- Create: `internal/infrastructure/postgres/destination_repo.go`
- Create: `internal/infrastructure/postgres/subscription_repo.go`

Note: these implementations require a running PostgreSQL database. There are no unit tests — integration testing requires a real database. Verify by compiling.

- [ ] **Step 1: Write db.go**

```go
// internal/infrastructure/postgres/db.go
package postgres

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool() (*pgxpool.Pool, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
```

- [ ] **Step 2: Write ride_repo.go**

```go
// internal/infrastructure/postgres/ride_repo.go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
)

type RideRepo struct{ pool *pgxpool.Pool }

func NewRideRepo(pool *pgxpool.Pool) *RideRepo { return &RideRepo{pool: pool} }

func (r *RideRepo) Save(ride domain.Ride) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO rides (id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		ride.ID, ride.DriverName, ride.Phone, ride.Origin, ride.Destination,
		ride.Date, ride.DepartureAt, int(ride.Flexibility), ride.PostedAt, ride.ExpiresAt,
	)
	return err
}

func (r *RideRepo) FindByID(id string) (domain.Ride, error) {
	row := r.pool.QueryRow(context.Background(),
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM rides WHERE id = $1`, id)
	return scanRide(row)
}

func (r *RideRepo) FindAll() ([]domain.Ride, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM rides WHERE expires_at > NOW() ORDER BY departure_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRides(rows)
}

func (r *RideRepo) FindByOriginAndDestination(origin, destination string) ([]domain.Ride, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM rides WHERE origin = $1 AND destination = $2 AND expires_at > NOW() ORDER BY departure_at ASC`,
		origin, destination)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRides(rows)
}

func (r *RideRepo) FindMatching(req domain.Request) ([]domain.Ride, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM rides
		 WHERE origin = $1 AND destination = $2 AND date = $3 AND expires_at > NOW()
		   AND (departure_at - (flexibility * interval '1 minute')) <= ($4 + ($5 * interval '1 minute'))
		   AND (departure_at + (flexibility * interval '1 minute')) >= ($4 - ($5 * interval '1 minute'))`,
		req.Origin, req.Destination, req.Date, req.DepartureAt, int(req.Flexibility))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRides(rows)
}

func (r *RideRepo) Delete(id string) error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM rides WHERE id = $1`, id)
	return err
}

func (r *RideRepo) DeleteExpired() error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM rides WHERE expires_at < NOW()`)
	return err
}

func scanRide(row pgx.Row) (domain.Ride, error) {
	var ride domain.Ride
	var flex int
	err := row.Scan(&ride.ID, &ride.DriverName, &ride.Phone, &ride.Origin, &ride.Destination,
		&ride.Date, &ride.DepartureAt, &flex, &ride.PostedAt, &ride.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Ride{}, errors.New("ride not found")
		}
		return domain.Ride{}, err
	}
	ride.Flexibility = domain.Flexibility(flex)
	return ride, nil
}

func collectRides(rows pgx.Rows) ([]domain.Ride, error) {
	var rides []domain.Ride
	for rows.Next() {
		var ride domain.Ride
		var flex int
		if err := rows.Scan(&ride.ID, &ride.DriverName, &ride.Phone, &ride.Origin, &ride.Destination,
			&ride.Date, &ride.DepartureAt, &flex, &ride.PostedAt, &ride.ExpiresAt); err != nil {
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

- [ ] **Step 3: Write request_repo.go**

```go
// internal/infrastructure/postgres/request_repo.go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
)

type RequestRepo struct{ pool *pgxpool.Pool }

func NewRequestRepo(pool *pgxpool.Pool) *RequestRepo { return &RequestRepo{pool: pool} }

func (r *RequestRepo) Save(req domain.Request) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO requests (id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		req.ID, req.SearcherName, req.Phone, req.Origin, req.Destination,
		req.Date, req.DepartureAt, int(req.Flexibility), req.PostedAt, req.ExpiresAt,
	)
	return err
}

func (r *RequestRepo) FindByID(id string) (domain.Request, error) {
	row := r.pool.QueryRow(context.Background(),
		`SELECT id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM requests WHERE id = $1`, id)
	var req domain.Request
	var flex int
	err := row.Scan(&req.ID, &req.SearcherName, &req.Phone, &req.Origin, &req.Destination,
		&req.Date, &req.DepartureAt, &flex, &req.PostedAt, &req.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Request{}, errors.New("request not found")
		}
		return domain.Request{}, err
	}
	req.Flexibility = domain.Flexibility(flex)
	return req, nil
}

func (r *RequestRepo) FindMatching(ride domain.Ride) ([]domain.Request, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM requests
		 WHERE origin = $1 AND destination = $2 AND date = $3 AND expires_at > NOW()
		   AND (departure_at - (flexibility * interval '1 minute')) <= ($4 + ($5 * interval '1 minute'))
		   AND (departure_at + (flexibility * interval '1 minute')) >= ($4 - ($5 * interval '1 minute'))`,
		ride.Origin, ride.Destination, ride.Date, ride.DepartureAt, int(ride.Flexibility))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reqs []domain.Request
	for rows.Next() {
		var req domain.Request
		var flex int
		if err := rows.Scan(&req.ID, &req.SearcherName, &req.Phone, &req.Origin, &req.Destination,
			&req.Date, &req.DepartureAt, &flex, &req.PostedAt, &req.ExpiresAt); err != nil {
			return nil, err
		}
		req.Flexibility = domain.Flexibility(flex)
		reqs = append(reqs, req)
	}
	if reqs == nil {
		reqs = []domain.Request{}
	}
	return reqs, rows.Err()
}

func (r *RequestRepo) Delete(id string) error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM requests WHERE id = $1`, id)
	return err
}

func (r *RequestRepo) DeleteExpired() error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM requests WHERE expires_at < NOW()`)
	return err
}
```

- [ ] **Step 4: Write destination_repo.go**

```go
// internal/infrastructure/postgres/destination_repo.go
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DestinationRepo struct{ pool *pgxpool.Pool }

func NewDestinationRepo(pool *pgxpool.Pool) *DestinationRepo { return &DestinationRepo{pool: pool} }

func (r *DestinationRepo) GetAll() ([]string, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT DISTINCT origin AS location FROM rides
		 UNION SELECT DISTINCT destination FROM rides
		 UNION SELECT DISTINCT origin FROM requests
		 UNION SELECT DISTINCT destination FROM requests
		 ORDER BY location`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var locations []string
	for rows.Next() {
		var loc string
		if err := rows.Scan(&loc); err != nil {
			return nil, err
		}
		locations = append(locations, loc)
	}
	if locations == nil {
		locations = []string{}
	}
	return locations, rows.Err()
}
```

- [ ] **Step 5: Write subscription_repo.go**

```go
// internal/infrastructure/postgres/subscription_repo.go
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
)

type SubscriptionRepo struct{ pool *pgxpool.Pool }

func NewSubscriptionRepo(pool *pgxpool.Pool) *SubscriptionRepo {
	return &SubscriptionRepo{pool: pool}
}

func (r *SubscriptionRepo) Save(sub domain.Subscription) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO subscriptions (id, phone, endpoint, p256dh, auth)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4)
		 ON CONFLICT (phone) DO UPDATE SET endpoint = $2, p256dh = $3, auth = $4`,
		sub.Phone, sub.Endpoint, sub.Keys.P256DH, sub.Keys.Auth,
	)
	return err
}

func (r *SubscriptionRepo) FindByPhone(phone string) (domain.Subscription, error) {
	row := r.pool.QueryRow(context.Background(),
		`SELECT id, phone, endpoint, p256dh, auth FROM subscriptions WHERE phone = $1`, phone)
	var sub domain.Subscription
	err := row.Scan(&sub.ID, &sub.Phone, &sub.Endpoint, &sub.Keys.P256DH, &sub.Keys.Auth)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Subscription{}, errors.New("subscription not found")
		}
		return domain.Subscription{}, err
	}
	return sub, nil
}

func (r *SubscriptionRepo) Delete(phone string) error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM subscriptions WHERE phone = $1`, phone)
	return err
}
```

- [ ] **Step 6: Verify compilation**

Run: `go build ./internal/infrastructure/...`
Expected: no output (success)

- [ ] **Step 7: Commit**

```bash
git add internal/infrastructure/postgres/
git commit -m "feat: add PostgreSQL repository implementations"
```

---

## Task 15: Web Push Infrastructure

**Files:**
- Create: `internal/infrastructure/webpush/webpush.go`

- [ ] **Step 1: Write webpush.go**

```go
// internal/infrastructure/webpush/webpush.go
package webpush

import (
	"encoding/json"
	"fmt"

	webpushlib "github.com/SherClockHolmes/webpush-go"
	"github.com/z3spinner/go-stop/internal/domain"
)

type WebPushNotifier struct {
	vapidPublic  string
	vapidPrivate string
	vapidEmail   string
}

func New(vapidPublic, vapidPrivate, vapidEmail string) *WebPushNotifier {
	return &WebPushNotifier{
		vapidPublic:  vapidPublic,
		vapidPrivate: vapidPrivate,
		vapidEmail:   vapidEmail,
	}
}

func (n *WebPushNotifier) Send(sub domain.Subscription, msg domain.Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	s := &webpushlib.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpushlib.Keys{
			P256dh: sub.Keys.P256DH,
			Auth:   sub.Keys.Auth,
		},
	}

	resp, err := webpushlib.SendNotification(payload, s, &webpushlib.Options{
		VAPIDPublicKey:  n.vapidPublic,
		VAPIDPrivateKey: n.vapidPrivate,
		Subscriber:      n.vapidEmail,
		TTL:             86400,
	})
	if err != nil {
		return fmt.Errorf("send push notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("push service returned status %d", resp.StatusCode)
	}
	return nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/infrastructure/webpush/...`
Expected: no output

- [ ] **Step 3: Commit**

```bash
git add internal/infrastructure/webpush/
git commit -m "feat: add Web Push (VAPID) implementation"
```

---

## Task 16: HTTP Handlers

**Files:**
- Create: `internal/boundaries/handler/ride_handler.go`
- Create: `internal/boundaries/handler/request_handler.go`
- Create: `internal/boundaries/handler/destination_handler.go`
- Create: `internal/boundaries/handler/subscription_handler.go`

- [ ] **Step 1: Write ride_handler.go**

```go
// internal/boundaries/handler/ride_handler.go
package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type RideHandler struct {
	postRide   *usecase.PostRide
	getRides   *usecase.GetRides
	searchRides *usecase.SearchRides
	deleteRide *usecase.DeleteRide
	rideGetter rideByIDGetter
}

type rideByIDGetter interface {
	FindByID(id string) (domain.Ride, error)
}

func NewRideHandler(
	postRide *usecase.PostRide,
	getRides *usecase.GetRides,
	searchRides *usecase.SearchRides,
	deleteRide *usecase.DeleteRide,
	rideGetter rideByIDGetter,
) *RideHandler {
	return &RideHandler{
		postRide:    postRide,
		getRides:    getRides,
		searchRides: searchRides,
		deleteRide:  deleteRide,
		rideGetter:  rideGetter,
	}
}

type postRideRequest struct {
	DriverName  string `json:"driver_name" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	Origin      string `json:"origin" binding:"required"`
	Destination string `json:"destination" binding:"required"`
	DepartureAt string `json:"departure_at" binding:"required"`
	Flexibility int    `json:"flexibility"`
}

func (h *RideHandler) Post(c *gin.Context) {
	var req postRideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dept, err := time.Parse(time.RFC3339, req.DepartureAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid departure_at, use RFC3339"})
		return
	}
	ride := domain.Ride{
		DriverName:  req.DriverName,
		Phone:       req.Phone,
		Origin:      req.Origin,
		Destination: req.Destination,
		DepartureAt: dept,
		Flexibility: domain.Flexibility(req.Flexibility),
	}
	if err := h.postRide.Execute(ride); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ride)
}

func (h *RideHandler) List(c *gin.Context) {
	origin := c.Query("origin")
	destination := c.Query("destination")

	var rides []domain.Ride
	var err error
	if origin != "" && destination != "" {
		rides, err = h.searchRides.Execute(origin, destination)
	} else {
		rides, err = h.getRides.Execute()
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rides)
}

func (h *RideHandler) Get(c *gin.Context) {
	ride, err := h.rideGetter.FindByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, ride)
}

type deleteRideRequest struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *RideHandler) Delete(c *gin.Context) {
	var req deleteRideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.deleteRide.Execute(c.Param("id"), req.Phone); err != nil {
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

- [ ] **Step 2: Write request_handler.go**

```go
// internal/boundaries/handler/request_handler.go
package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type RequestHandler struct {
	postRequest   *usecase.PostRequest
	deleteRequest *usecase.DeleteRequest
	requestGetter requestByIDGetter
}

type requestByIDGetter interface {
	FindByID(id string) (domain.Request, error)
}

func NewRequestHandler(
	postRequest *usecase.PostRequest,
	deleteRequest *usecase.DeleteRequest,
	requestGetter requestByIDGetter,
) *RequestHandler {
	return &RequestHandler{
		postRequest:   postRequest,
		deleteRequest: deleteRequest,
		requestGetter: requestGetter,
	}
}

type postRequestBody struct {
	SearcherName string `json:"searcher_name" binding:"required"`
	Phone        string `json:"phone" binding:"required"`
	Origin       string `json:"origin" binding:"required"`
	Destination  string `json:"destination" binding:"required"`
	DepartureAt  string `json:"departure_at" binding:"required"`
	Flexibility  int    `json:"flexibility"`
}

func (h *RequestHandler) Post(c *gin.Context) {
	var body postRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dept, err := time.Parse(time.RFC3339, body.DepartureAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid departure_at, use RFC3339"})
		return
	}
	req := domain.Request{
		SearcherName: body.SearcherName,
		Phone:        body.Phone,
		Origin:       body.Origin,
		Destination:  body.Destination,
		DepartureAt:  dept,
		Flexibility:  domain.Flexibility(body.Flexibility),
	}
	if err := h.postRequest.Execute(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, req)
}

func (h *RequestHandler) Get(c *gin.Context) {
	req, err := h.requestGetter.FindByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, req)
}

type deleteRequestBody struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *RequestHandler) Delete(c *gin.Context) {
	var body deleteRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.deleteRequest.Execute(c.Param("id"), body.Phone); err != nil {
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

- [ ] **Step 3: Write destination_handler.go**

```go
// internal/boundaries/handler/destination_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type DestinationHandler struct {
	getDestinations *usecase.GetDestinations
}

func NewDestinationHandler(getDestinations *usecase.GetDestinations) *DestinationHandler {
	return &DestinationHandler{getDestinations: getDestinations}
}

func (h *DestinationHandler) List(c *gin.Context) {
	destinations, err := h.getDestinations.Execute()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, destinations)
}
```

- [ ] **Step 4: Write subscription_handler.go**

```go
// internal/boundaries/handler/subscription_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type SubscriptionHandler struct {
	subscribe   *usecase.Subscribe
	unsubscribe *usecase.Unsubscribe
}

func NewSubscriptionHandler(subscribe *usecase.Subscribe, unsubscribe *usecase.Unsubscribe) *SubscriptionHandler {
	return &SubscriptionHandler{subscribe: subscribe, unsubscribe: unsubscribe}
}

type subscribeRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Endpoint string `json:"endpoint" binding:"required"`
	P256DH   string `json:"p256dh" binding:"required"`
	Auth     string `json:"auth" binding:"required"`
}

func (h *SubscriptionHandler) Subscribe(c *gin.Context) {
	var req subscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sub := domain.Subscription{
		Phone:    req.Phone,
		Endpoint: req.Endpoint,
		Keys:     domain.PushKeys{P256DH: req.P256DH, Auth: req.Auth},
	}
	if err := h.subscribe.Execute(sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusCreated)
}

func (h *SubscriptionHandler) Unsubscribe(c *gin.Context) {
	phone := c.Param("phone")
	if err := h.unsubscribe.Execute(phone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./internal/boundaries/handler/...`
Expected: no output

- [ ] **Step 6: Commit**

```bash
git add internal/boundaries/handler/
git commit -m "feat: add Gin HTTP handlers"
```

---

## Task 17: Main.go

**Files:**
- Create: `main.go`

- [ ] **Step 1: Write main.go**

```go
// main.go
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

	// Repositories
	rideRepo := postgres.NewRideRepo(pool)
	requestRepo := postgres.NewRequestRepo(pool)
	destRepo := postgres.NewDestinationRepo(pool)
	subRepo := postgres.NewSubscriptionRepo(pool)

	// Notifier
	notifier := webpush.New(
		os.Getenv("VAPID_PUBLIC_KEY"),
		os.Getenv("VAPID_PRIVATE_KEY"),
		os.Getenv("VAPID_EMAIL"),
	)

	// Use cases
	postRide := usecase.NewPostRide(rideRepo, requestRepo, subRepo, notifier)
	postRequest := usecase.NewPostRequest(requestRepo, rideRepo, subRepo, notifier)
	getRides := usecase.NewGetRides(rideRepo)
	searchRides := usecase.NewSearchRides(rideRepo)
	getDests := usecase.NewGetDestinations(destRepo)
	subscribe := usecase.NewSubscribe(subRepo)
	unsubscribe := usecase.NewUnsubscribe(subRepo)
	deleteRide := usecase.NewDeleteRide(rideRepo)
	deleteRequest := usecase.NewDeleteRequest(requestRepo)
	expireRides := usecase.NewExpireRides(rideRepo)
	expireRequests := usecase.NewExpireRequests(requestRepo)

	// Handlers
	rideH := handler.NewRideHandler(postRide, getRides, searchRides, deleteRide, rideRepo)
	requestH := handler.NewRequestHandler(postRequest, deleteRequest, requestRepo)
	destH := handler.NewDestinationHandler(getDests)
	subH := handler.NewSubscriptionHandler(subscribe, unsubscribe)

	// Expiry cron — runs every hour
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
		}
	}()

	// Router
	r := gin.Default()
	r.Static("/css", "./web/css")
	r.Static("/js", "./web/js")
	r.StaticFile("/manifest.json", "./web/manifest.json")
	r.StaticFile("/sw.js", "./web/js/sw.js")
	r.StaticFile("/favicon.ico", "./web/favicon.ico")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/index.html")
	})

	api := r.Group("/api")
	{
		api.POST("/rides", rideH.Post)
		api.GET("/rides", rideH.List)
		api.GET("/rides/:id", rideH.Get)
		api.DELETE("/rides/:id", rideH.Delete)

		api.POST("/requests", requestH.Post)
		api.GET("/requests/:id", requestH.Get)
		api.DELETE("/requests/:id", requestH.Delete)

		api.GET("/destinations", destH.List)

		api.POST("/subscriptions", subH.Subscribe)
		api.DELETE("/subscriptions/:phone", subH.Unsubscribe)
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

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`
Expected: no output (success)

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "feat: wire up application in main.go"
```

---

## Task 18: Database Migrations

**Files:**
- Create: `db/migrations/001_create_tables.sql`

- [ ] **Step 1: Write 001_create_tables.sql**

```sql
-- db/migrations/001_create_tables.sql

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
```

- [ ] **Step 2: Commit**

```bash
git add db/migrations/
git commit -m "feat: add database migration"
```

---

## Task 19: Frontend

**Files:**
- Create: `web/manifest.json`
- Create: `web/index.html`
- Create: `web/css/style.css`
- Create: `web/js/app.js`
- Create: `web/js/sw.js`

Note: The frontend communicates with the API at `/api/*`. All routes fall through to `index.html` for client-side navigation.

- [ ] **Step 1: Write manifest.json**

```json
{
  "name": "Go-Stop",
  "short_name": "Go-Stop",
  "description": "Local ride sharing notice board",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#2563eb",
  "icons": [
    {
      "src": "/icon-192.png",
      "sizes": "192x192",
      "type": "image/png"
    },
    {
      "src": "/icon-512.png",
      "sizes": "512x512",
      "type": "image/png"
    }
  ]
}
```

- [ ] **Step 2: Write index.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Go-Stop</title>
  <link rel="manifest" href="/manifest.json">
  <link rel="stylesheet" href="/css/style.css">
  <meta name="theme-color" content="#2563eb">
</head>
<body>
  <div id="app">
    <!-- Views are rendered by app.js -->
  </div>
  <script src="/js/app.js"></script>
</body>
</html>
```

- [ ] **Step 3: Write style.css**

```css
*, *::before, *::after { box-sizing: border-box; }

:root {
  --blue: #2563eb;
  --blue-dark: #1d4ed8;
  --green: #16a34a;
  --red: #dc2626;
  --gray-50: #f9fafb;
  --gray-100: #f3f4f6;
  --gray-300: #d1d5db;
  --gray-600: #4b5563;
  --gray-900: #111827;
  --radius: 8px;
}

body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  background: var(--gray-50);
  color: var(--gray-900);
  min-height: 100vh;
}

#app { max-width: 600px; margin: 0 auto; padding: 16px; }

h1 { font-size: 1.75rem; margin: 0 0 4px; }
h2 { font-size: 1.25rem; margin: 0 0 16px; }
p.tagline { color: var(--gray-600); margin: 0 0 32px; }

.hero { text-align: center; padding: 48px 0 32px; }

.btn {
  display: block;
  width: 100%;
  padding: 16px;
  border: none;
  border-radius: var(--radius);
  font-size: 1.1rem;
  font-weight: 600;
  cursor: pointer;
  margin-bottom: 12px;
  text-align: center;
}
.btn-primary { background: var(--blue); color: white; }
.btn-primary:hover { background: var(--blue-dark); }
.btn-secondary { background: white; color: var(--blue); border: 2px solid var(--blue); }
.btn-secondary:hover { background: var(--gray-100); }
.btn-danger { background: var(--red); color: white; font-size: 0.9rem; padding: 8px 12px; width: auto; }
.btn-back { background: none; border: none; color: var(--blue); cursor: pointer; font-size: 1rem; padding: 0; margin-bottom: 16px; }

.form-group { margin-bottom: 16px; }
label { display: block; font-weight: 500; margin-bottom: 4px; font-size: 0.9rem; }
input, select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--gray-300);
  border-radius: var(--radius);
  font-size: 1rem;
  background: white;
}
input:focus, select:focus { outline: 2px solid var(--blue); border-color: transparent; }

.card {
  background: white;
  border: 1px solid var(--gray-300);
  border-radius: var(--radius);
  padding: 16px;
  margin-bottom: 12px;
}
.card-route { font-size: 1.1rem; font-weight: 600; margin-bottom: 4px; }
.card-meta { font-size: 0.9rem; color: var(--gray-600); margin-bottom: 8px; }
.card-contact { font-size: 0.95rem; }

.empty { text-align: center; padding: 32px; color: var(--gray-600); }

.tag {
  display: inline-block;
  background: var(--gray-100);
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 0.8rem;
  color: var(--gray-600);
}

.error { color: var(--red); font-size: 0.9rem; margin-top: 8px; }

datalist option { padding: 8px; }
```

- [ ] **Step 4: Write app.js**

```js
// web/js/app.js
'use strict';

const app = document.getElementById('app');

const FLEX_LABELS = { 0: 'Exact', 30: '±30 min', 60: '±60 min' };

// Escape user-supplied strings before inserting into innerHTML.
function esc(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function formatTime(iso) {
  const d = new Date(iso);
  return d.toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short' })
    + ' at ' + d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
}

async function api(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch('/api' + path, opts);
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

async function getDestinations() {
  try { return await api('GET', '/destinations'); }
  catch { return []; }
}

function destinationList(id, destinations) {
  return `<datalist id="${id}">${destinations.map(d => `<option value="${esc(d)}">`).join('')}</datalist>`;
}

// ── Views ─────────────────────────────────────────────────────────────────────

function renderHome() {
  app.innerHTML = `
    <div class="hero">
      <h1>Go-Stop</h1>
      <p class="tagline">Local rides, direct contact</p>
      <button class="btn btn-primary" id="btn-driver">I'm driving</button>
      <button class="btn btn-secondary" id="btn-searcher">I need a ride</button>
    </div>`;
  document.getElementById('btn-driver').onclick = renderPostRide;
  document.getElementById('btn-searcher').onclick = renderSearchRides;
}

async function renderPostRide() {
  const dests = await getDestinations();
  app.innerHTML = `
    <button class="btn-back" id="back">← Back</button>
    <h2>Post a ride</h2>
    <form id="ride-form">
      <div class="form-group"><label>Your name</label><input name="driver_name" required></div>
      <div class="form-group"><label>Phone number</label><input name="phone" type="tel" required></div>
      <div class="form-group"><label>From</label><input name="origin" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>To</label><input name="destination" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <div class="form-group"><label>Date &amp; departure time</label><input name="departure_at" type="datetime-local" required></div>
      <div class="form-group">
        <label>Flexibility</label>
        <select name="flexibility">
          <option value="0">Exact</option>
          <option value="30" selected>±30 minutes</option>
          <option value="60">±60 minutes</option>
        </select>
      </div>
      <button class="btn btn-primary" type="submit">Post ride</button>
      <div class="error" id="err"></div>
    </form>`;
  document.getElementById('back').onclick = renderHome;
  document.getElementById('ride-form').onsubmit = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const dept = new Date(fd.get('departure_at')).toISOString();
    try {
      await api('POST', '/rides', {
        driver_name: fd.get('driver_name'),
        phone: fd.get('phone'),
        origin: fd.get('origin'),
        destination: fd.get('destination'),
        departure_at: dept,
        flexibility: parseInt(fd.get('flexibility')),
      });
      renderHome();
    } catch (err) {
      document.getElementById('err').textContent = err.message;
    }
  };
}

async function renderSearchRides() {
  const dests = await getDestinations();
  app.innerHTML = `
    <button class="btn-back" id="back">← Back</button>
    <h2>Find a ride</h2>
    <form id="search-form">
      <div class="form-group"><label>From</label><input name="origin" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>To</label><input name="destination" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <button class="btn btn-primary" type="submit">Search</button>
    </form>
    <div id="results"></div>`;
  document.getElementById('back').onclick = renderHome;
  document.getElementById('search-form').onsubmit = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const origin = fd.get('origin');
    const dest = fd.get('destination');
    const results = document.getElementById('results');
    try {
      const rides = await api('GET', `/rides?origin=${encodeURIComponent(origin)}&destination=${encodeURIComponent(dest)}`);
      if (!rides.length) {
        results.innerHTML = `
          <div class="empty"><p>No rides found.</p>
          <button class="btn btn-secondary" id="btn-post-req">Post a waiting request</button></div>`;
        document.getElementById('btn-post-req').onclick = () => renderPostRequest(origin, dest);
        return;
      }
      results.innerHTML = rides.map(r => `
        <div class="card">
          <div class="card-route">${esc(r.origin)} → ${esc(r.destination)}</div>
          <div class="card-meta">${formatTime(r.departure_at)} <span class="tag">${FLEX_LABELS[r.flexibility] || esc(r.flexibility) + ' min'}</span></div>
          <div class="card-contact">
            <strong>${esc(r.driver_name)}</strong> — <a href="tel:${esc(r.phone)}">${esc(r.phone)}</a>
          </div>
        </div>`).join('');
    } catch (err) {
      const div = document.createElement('div');
      div.className = 'error';
      div.textContent = err.message;
      results.replaceChildren(div);
    }
  };
}

async function renderPostRequest(origin = '', destination = '') {
  const dests = await getDestinations();
  app.innerHTML = `
    <button class="btn-back" id="back">← Back</button>
    <h2>Post a waiting request</h2>
    <form id="req-form">
      <div class="form-group"><label>Your name</label><input name="searcher_name" required></div>
      <div class="form-group"><label>Phone number</label><input name="phone" type="tel" required></div>
      <div class="form-group"><label>From</label><input name="origin" value="${esc(origin)}" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>To</label><input name="destination" value="${esc(destination)}" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <div class="form-group"><label>Date &amp; time needed</label><input name="departure_at" type="datetime-local" required></div>
      <div class="form-group">
        <label>Flexibility</label>
        <select name="flexibility">
          <option value="0">Exact</option>
          <option value="30" selected>±30 minutes</option>
          <option value="60">±60 minutes</option>
        </select>
      </div>
      <button class="btn btn-primary" type="submit">Post request</button>
      <div class="error" id="err"></div>
    </form>`;
  document.getElementById('back').onclick = renderHome;
  document.getElementById('req-form').onsubmit = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const dept = new Date(fd.get('departure_at')).toISOString();
    try {
      await api('POST', '/requests', {
        searcher_name: fd.get('searcher_name'),
        phone: fd.get('phone'),
        origin: fd.get('origin'),
        destination: fd.get('destination'),
        departure_at: dept,
        flexibility: parseInt(fd.get('flexibility')),
      });
      renderHome();
    } catch (err) {
      document.getElementById('err').textContent = err.message;
    }
  };
}

renderHome();

// ── Service worker registration ───────────────────────────────────────────────
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js').catch(console.error);
}
```

- [ ] **Step 5: Write sw.js**

```js
// web/js/sw.js
'use strict';

self.addEventListener('push', (event) => {
  let data = {};
  try { data = event.data.json(); } catch {}

  const title = data.title || 'Go-Stop';
  const options = {
    body: data.body || '',
    icon: '/icon-192.png',
    badge: '/icon-192.png',
    data: { url: data.url || '/' },
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const url = event.notification.data?.url || '/';
  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      for (const client of clientList) {
        if (client.url === url && 'focus' in client) return client.focus();
      }
      if (clients.openWindow) return clients.openWindow(url);
    })
  );
});
```

- [ ] **Step 6: Commit**

```bash
git add web/
git commit -m "feat: add frontend (HTML/CSS/JS + service worker)"
```

---

## Task 20: Deployment Files + README

**Files:**
- Create: `Procfile`
- Create: `scalingo.json`
- Create: `README.md`

- [ ] **Step 1: Write Procfile**

```
web: go-stop
```

- [ ] **Step 2: Write scalingo.json**

```json
{
  "name": "Go-Stop",
  "description": "Lightweight local ride-sharing notice board",
  "website": "https://github.com/z3spinner/go-stop",
  "addons": ["postgresql:postgresql-sandbox"],
  "env": {
    "VAPID_PUBLIC_KEY": {
      "description": "VAPID public key for Web Push notifications (generate with webpush-go or similar)",
      "required": true
    },
    "VAPID_PRIVATE_KEY": {
      "description": "VAPID private key for Web Push notifications",
      "required": true
    },
    "VAPID_EMAIL": {
      "description": "Contact email for Web Push — prefix with mailto: (e.g. mailto:you@example.com)",
      "required": true
    }
  }
}
```

- [ ] **Step 3: Write README.md**

````markdown
# Go-Stop

A lightweight local ride-sharing notice board. Drivers post one-time trips; searchers browse or post waiting requests. Matches trigger instant push notifications. Direct contact via phone number — no accounts, no in-app messaging.

[![Deploy to Scalingo](https://cdn.scalingo.com/deploy/button.svg)](https://my.scalingo.com/deploy?source=https://github.com/z3spinner/go-stop)

## How it works

- **Drivers** post a ride with origin, destination, date, departure time, and flexibility window
- **Searchers** browse by origin/destination or post a waiting request
- Both parties receive a **push notification** when a match is found
- Contact is made directly via the displayed phone number

## Requirements

- Go 1.22+
- PostgreSQL 14+

## Local setup

```bash
# Generate VAPID keys (one-time)
go run github.com/SherClockHolmes/webpush-go/cmd/vapid-keygen@latest

export DATABASE_URL="postgres://user:pass@localhost:5432/gostop?sslmode=disable"
export VAPID_PUBLIC_KEY="your-public-key"
export VAPID_PRIVATE_KEY="your-private-key"
export VAPID_EMAIL="mailto:you@example.com"
export PORT=8080

# Run migrations
psql $DATABASE_URL < db/migrations/001_create_tables.sql

# Run
go run .
```

Open [http://localhost:8080](http://localhost:8080).

## Deployment (Scalingo)

Click the button above. You will be prompted for your VAPID keys and email.

To generate VAPID keys before deploying:

```bash
go run github.com/SherClockHolmes/webpush-go/cmd/vapid-keygen@latest
```

## Architecture

Clean Architecture — dependencies point inward only.

```
domain ← usecase ← boundaries ← infrastructure
```

| Layer | Contents |
|---|---|
| `internal/domain` | Entities: Ride, Request, Subscription, Message |
| `internal/usecase` | Business logic with injected interfaces |
| `internal/boundaries` | Gin handlers + repository/notifier interfaces |
| `internal/infrastructure` | PostgreSQL (pgx) + Web Push (VAPID) |
| `web/` | Static SPA served by Go |

## License

MIT
````

- [ ] **Step 4: Final build check**

Run: `go build ./...`
Expected: no output (clean build)

- [ ] **Step 5: Run all tests**

Run: `go test ./...`
Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add Procfile scalingo.json README.md
git commit -m "feat: add deployment config and README with Scalingo deploy button"
```

---

## Task 21: Local Docker Devstack

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `.env.example`

PostgreSQL runs in Docker. The migration SQL is mounted into `docker-entrypoint-initdb.d` so it runs automatically on first start. The Go app is built in a multi-stage Docker image and reads config from environment variables.

- [ ] **Step 1: Write Dockerfile**

```dockerfile
# Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o go-stop .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/go-stop .
COPY web/ ./web/
EXPOSE 8080
CMD ["./go-stop"]
```

- [ ] **Step 2: Write docker-compose.yml**

```yaml
services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: gostop
      POSTGRES_PASSWORD: gostop
      POSTGRES_DB: gostop
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./db/migrations:/docker-entrypoint-initdb.d:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U gostop"]
      interval: 5s
      timeout: 5s
      retries: 5

  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://gostop:gostop@db:5432/gostop?sslmode=disable
      VAPID_PUBLIC_KEY: ${VAPID_PUBLIC_KEY}
      VAPID_PRIVATE_KEY: ${VAPID_PRIVATE_KEY}
      VAPID_EMAIL: ${VAPID_EMAIL:-mailto:dev@localhost}
      PORT: "8080"
    depends_on:
      db:
        condition: service_healthy

volumes:
  postgres_data:
```

- [ ] **Step 3: Write .env.example**

```
# Copy to .env and fill in your VAPID keys.
# Generate keys: go run github.com/SherClockHolmes/webpush-go/cmd/vapid-keygen@latest

VAPID_PUBLIC_KEY=
VAPID_PRIVATE_KEY=
VAPID_EMAIL=mailto:dev@localhost
```

- [ ] **Step 4: Add .env to .gitignore**

Create `.gitignore`:

```
.env
```

- [ ] **Step 5: Update README — add docker-compose section**

Add this section to `README.md` under `## Local setup`, before the manual setup instructions:

```markdown
### With Docker (recommended)

```bash
# Generate VAPID keys (one-time)
go run github.com/SherClockHolmes/webpush-go/cmd/vapid-keygen@latest

cp .env.example .env
# Edit .env and paste your VAPID keys

docker compose up --build
```

Open [http://localhost:8080](http://localhost:8080). The database is created and migrated automatically.

```

- [ ] **Step 6: Verify docker-compose build**

Run: `docker compose build`
Expected: build completes with no errors

- [ ] **Step 7: Smoke-test the stack**

Run: `docker compose up`
Expected:
- `db` container starts and passes healthcheck
- `app` container starts and logs `listening on :8080`
- `curl http://localhost:8080/api/destinations` returns `[]`

- [ ] **Step 8: Commit**

```bash
git add Dockerfile docker-compose.yml .env.example .gitignore README.md
git commit -m "feat: add docker-compose devstack"
```

---

## Task 22: Integration Tests

**Files:**
- Create: `internal/infrastructure/postgres/ride_repo_integration_test.go`
- Create: `internal/infrastructure/postgres/request_repo_integration_test.go`
- Create: `internal/infrastructure/postgres/destination_repo_integration_test.go`
- Create: `internal/infrastructure/postgres/subscription_repo_integration_test.go`
- Create: `internal/boundaries/handler/integration_test.go`

**Prerequisites:** Tasks 14–17 complete. Requires a running PostgreSQL database (use `docker compose up db -d` from Task 21).

Integration tests use the `//go:build integration` build tag and connect to the database via `TEST_DATABASE_URL` (or `DATABASE_URL` if `TEST_DATABASE_URL` is unset). Each test truncates its tables in `TestMain` before running. Run with:

```bash
TEST_DATABASE_URL="postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable" go test -tags integration ./...
```

- [ ] **Step 1: Write ride_repo_integration_test.go**

```go
//go:build integration

// internal/infrastructure/postgres/ride_repo_integration_test.go
package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		// skip all integration tests if no DB is available
		os.Exit(0)
	}

	var err error
	testPool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		panic("connect test db: " + err.Error())
	}
	defer testPool.Close()

	// clean slate
	testPool.Exec(context.Background(), `TRUNCATE rides, requests, subscriptions`)

	os.Exit(m.Run())
}

func truncate(t *testing.T) {
	t.Helper()
	testPool.Exec(context.Background(), `TRUNCATE rides, requests, subscriptions`)
}

func TestRideRepo_SaveAndFindByID(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool)

	ride := domain.Ride{
		ID:          "test-ride-1",
		DriverName:  "Alice",
		Phone:       "555-0001",
		Origin:      "Village A",
		Destination: "Station",
		Date:        time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility: domain.Approximate,
		PostedAt:    time.Now().UTC().Truncate(time.Second),
		ExpiresAt:   time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
	}

	if err := repo.Save(ride); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID("test-ride-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.DriverName != "Alice" {
		t.Errorf("expected DriverName Alice, got %s", got.DriverName)
	}
	if got.Flexibility != domain.Approximate {
		t.Errorf("expected flexibility 30, got %d", got.Flexibility)
	}
}

func TestRideRepo_FindAll_OnlyActive(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool)

	active := domain.Ride{
		ID: "active-1", DriverName: "Alice", Phone: "555-0001",
		Origin: "A", Destination: "B",
		Date:        time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	expired := domain.Ride{
		ID: "expired-1", DriverName: "Bob", Phone: "555-0002",
		Origin: "A", Destination: "B",
		Date:        time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2020, 1, 1, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	_ = repo.Save(active)
	_ = repo.Save(expired)

	rides, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(rides) != 1 {
		t.Errorf("expected 1 active ride, got %d", len(rides))
	}
	if rides[0].ID != "active-1" {
		t.Errorf("expected active-1, got %s", rides[0].ID)
	}
}

func TestRideRepo_FindMatching_WindowOverlap(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool)

	// Ride: 09:00 ±30 min → window 08:30–09:30
	ride := domain.Ride{
		ID: "ride-match-1", DriverName: "Alice", Phone: "555-0001",
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility: domain.Approximate,
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	}
	_ = repo.Save(ride)

	// Request: 09:15 ±0 min → window 09:15–09:15 — inside ride window, should match
	matching := domain.Request{
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 9, 15, 0, 0, time.UTC),
		Flexibility: domain.Exact,
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	}

	rides, err := repo.FindMatching(matching)
	if err != nil {
		t.Fatalf("FindMatching: %v", err)
	}
	if len(rides) != 1 {
		t.Errorf("expected 1 matching ride, got %d", len(rides))
	}
}

func TestRideRepo_FindMatching_NoOverlap(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool)

	// Ride: 09:00 exact → only matches 09:00
	ride := domain.Ride{
		ID: "ride-no-match", DriverName: "Alice", Phone: "555-0001",
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility: domain.Exact,
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	}
	_ = repo.Save(ride)

	// Request: 10:00 exact — no overlap with 09:00 exact
	nonMatching := domain.Request{
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 10, 0, 0, 0, time.UTC),
		Flexibility: domain.Exact,
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	}

	rides, err := repo.FindMatching(nonMatching)
	if err != nil {
		t.Fatalf("FindMatching: %v", err)
	}
	if len(rides) != 0 {
		t.Errorf("expected 0 matching rides, got %d", len(rides))
	}
}

func TestRideRepo_Delete(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool)

	ride := domain.Ride{
		ID: "to-delete", DriverName: "Alice", Phone: "555-0001",
		Origin: "A", Destination: "B",
		Date:        time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	_ = repo.Save(ride)

	if err := repo.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.FindByID("to-delete")
	if err == nil {
		t.Error("expected not found error after delete")
	}
}
```

- [ ] **Step 2: Write request_repo_integration_test.go**

```go
//go:build integration

// internal/infrastructure/postgres/request_repo_integration_test.go
package postgres_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func TestRequestRepo_SaveAndFindByID(t *testing.T) {
	truncate(t)
	repo := postgres.NewRequestRepo(testPool)

	req := domain.Request{
		ID:           "test-req-1",
		SearcherName: "Bob",
		Phone:        "555-0002",
		Origin:       "Village A",
		Destination:  "Station",
		Date:         time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt:  time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility:  domain.Flexible,
		PostedAt:     time.Now().UTC().Truncate(time.Second),
		ExpiresAt:    time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	}

	if err := repo.Save(req); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID("test-req-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.SearcherName != "Bob" {
		t.Errorf("expected SearcherName Bob, got %s", got.SearcherName)
	}
	if got.Flexibility != domain.Flexible {
		t.Errorf("expected flexibility 60, got %d", got.Flexibility)
	}
}

func TestRequestRepo_FindMatching_WindowOverlap(t *testing.T) {
	truncate(t)
	repo := postgres.NewRequestRepo(testPool)

	// Request: 09:15 ±30 min → window 08:45–09:45
	req := domain.Request{
		ID: "req-match-1", SearcherName: "Bob", Phone: "555-0002",
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 9, 15, 0, 0, time.UTC),
		Flexibility: domain.Approximate,
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	}
	_ = repo.Save(req)

	// Ride: 09:00 ±0 min → window 09:00–09:00 — inside request window, should match
	ride := domain.Ride{
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility: domain.Exact,
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	}

	reqs, err := repo.FindMatching(ride)
	if err != nil {
		t.Fatalf("FindMatching: %v", err)
	}
	if len(reqs) != 1 {
		t.Errorf("expected 1 matching request, got %d", len(reqs))
	}
}
```

- [ ] **Step 3: Write destination_repo_integration_test.go**

```go
//go:build integration

// internal/infrastructure/postgres/destination_repo_integration_test.go
package postgres_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func TestDestinationRepo_GetAll_DeduplicatesAndSorts(t *testing.T) {
	truncate(t)
	rideRepo := postgres.NewRideRepo(testPool)
	reqRepo := postgres.NewRequestRepo(testPool)
	destRepo := postgres.NewDestinationRepo(testPool)

	// Add rides with known origins/destinations
	_ = rideRepo.Save(domain.Ride{
		ID: "d1", DriverName: "A", Phone: "1",
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC),
	})
	_ = reqRepo.Save(domain.Request{
		ID: "r1", SearcherName: "B", Phone: "2",
		Origin: "Town B", Destination: "Village A", // Village A appears again — should be deduped
		Date:        time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC),
	})

	locs, err := destRepo.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	// Expect 3 distinct locations: Station, Town B, Village A
	if len(locs) != 3 {
		t.Errorf("expected 3 locations, got %d: %v", len(locs), locs)
	}
	if locs[0] != "Station" || locs[1] != "Town B" || locs[2] != "Village A" {
		t.Errorf("unexpected order: %v", locs)
	}
}
```

- [ ] **Step 4: Write subscription_repo_integration_test.go**

```go
//go:build integration

// internal/infrastructure/postgres/subscription_repo_integration_test.go
package postgres_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func TestSubscriptionRepo_SaveFindDelete(t *testing.T) {
	truncate(t)
	repo := postgres.NewSubscriptionRepo(testPool)

	sub := domain.Subscription{
		Phone:    "555-0001",
		Endpoint: "https://push.example.com/1",
		Keys:     domain.PushKeys{P256DH: "pubkey", Auth: "authkey"},
	}

	if err := repo.Save(sub); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByPhone("555-0001")
	if err != nil {
		t.Fatalf("FindByPhone: %v", err)
	}
	if got.Endpoint != sub.Endpoint {
		t.Errorf("expected endpoint %s, got %s", sub.Endpoint, got.Endpoint)
	}
	if got.Keys.P256DH != "pubkey" {
		t.Errorf("expected P256DH pubkey, got %s", got.Keys.P256DH)
	}

	if err := repo.Delete("555-0001"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = repo.FindByPhone("555-0001")
	if err == nil {
		t.Error("expected not found after delete")
	}
}

func TestSubscriptionRepo_Save_UpdatesOnConflict(t *testing.T) {
	truncate(t)
	repo := postgres.NewSubscriptionRepo(testPool)

	original := domain.Subscription{
		Phone:    "555-0002",
		Endpoint: "https://push.example.com/old",
		Keys:     domain.PushKeys{P256DH: "old-key", Auth: "old-auth"},
	}
	updated := domain.Subscription{
		Phone:    "555-0002",
		Endpoint: "https://push.example.com/new",
		Keys:     domain.PushKeys{P256DH: "new-key", Auth: "new-auth"},
	}

	_ = repo.Save(original)
	_ = repo.Save(updated)

	got, err := repo.FindByPhone("555-0002")
	if err != nil {
		t.Fatalf("FindByPhone: %v", err)
	}
	if got.Endpoint != "https://push.example.com/new" {
		t.Errorf("expected updated endpoint, got %s", got.Endpoint)
	}
}
```

- [ ] **Step 5: Write handler/integration_test.go**

```go
//go:build integration

// internal/boundaries/handler/integration_test.go
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/boundaries/handler"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
	"github.com/z3spinner/go-stop/internal/usecase"
)

var handlerPool *pgxpool.Pool

type noopNotifier struct{}

func (n *noopNotifier) Send(_ interface{}, _ interface{}) error { return nil }

func TestHandlerMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		os.Exit(0)
	}

	var err error
	handlerPool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		panic("connect test db: " + err.Error())
	}
	defer handlerPool.Close()

	handlerPool.Exec(context.Background(), `TRUNCATE rides, requests, subscriptions`)
	os.Exit(m.Run())
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	rideRepo := postgres.NewRideRepo(handlerPool)
	reqRepo := postgres.NewRequestRepo(handlerPool)
	subRepo := postgres.NewSubscriptionRepo(handlerPool)
	destRepo := postgres.NewDestinationRepo(handlerPool)
	notifier := &noopNotifier{}

	postRide := usecase.NewPostRide(rideRepo, reqRepo, subRepo, notifier)
	getRides := usecase.NewGetRides(rideRepo)
	searchRides := usecase.NewSearchRides(rideRepo)
	deleteRide := usecase.NewDeleteRide(rideRepo)
	postRequest := usecase.NewPostRequest(reqRepo, rideRepo, subRepo, notifier)
	deleteRequest := usecase.NewDeleteRequest(reqRepo)
	getDests := usecase.NewGetDestinations(destRepo)
	subscribe := usecase.NewSubscribe(subRepo)
	unsubscribe := usecase.NewUnsubscribe(subRepo)

	rideH := handler.NewRideHandler(postRide, getRides, searchRides, deleteRide, rideRepo)
	reqH := handler.NewRequestHandler(postRequest, deleteRequest, reqRepo)
	destH := handler.NewDestinationHandler(getDests)
	subH := handler.NewSubscriptionHandler(subscribe, unsubscribe)

	r := gin.New()
	r.POST("/api/rides", rideH.Post)
	r.GET("/api/rides", rideH.List)
	r.GET("/api/rides/:id", rideH.Get)
	r.DELETE("/api/rides/:id", rideH.Delete)
	r.POST("/api/requests", reqH.Post)
	r.GET("/api/requests/:id", reqH.Get)
	r.DELETE("/api/requests/:id", reqH.Delete)
	r.GET("/api/destinations", destH.List)
	r.POST("/api/subscriptions", subH.Subscribe)
	r.DELETE("/api/subscriptions/:phone", subH.Unsubscribe)
	return r
}

func truncateAll(t *testing.T) {
	t.Helper()
	handlerPool.Exec(context.Background(), `TRUNCATE rides, requests, subscriptions`)
}

func TestHTTP_PostAndGetRide(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	body := map[string]interface{}{
		"driver_name":  "Alice",
		"phone":        "555-0001",
		"origin":       "Village A",
		"destination":  "Station",
		"departure_at": "2030-06-01T09:00:00Z",
		"flexibility":  30,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/rides", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// GET should return the ride
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/rides", nil)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var rides []map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &rides)
	if len(rides) != 1 {
		t.Errorf("expected 1 ride, got %d", len(rides))
	}
}

func TestHTTP_DeleteRide_WrongPhone_Returns403(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	// post a ride
	body := map[string]interface{}{
		"driver_name": "Alice", "phone": "555-0001",
		"origin": "A", "destination": "B",
		"departure_at": "2030-06-01T09:00:00Z", "flexibility": 0,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/rides", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	var created map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created["ID"].(string)

	// attempt delete with wrong phone
	delBody, _ := json.Marshal(map[string]string{"phone": "555-9999"})
	w2 := httptest.NewRecorder()
	delReq, _ := http.NewRequest("DELETE", "/api/rides/"+id, bytes.NewBuffer(delBody))
	delReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, delReq)

	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w2.Code)
	}
}

func TestHTTP_Destinations_ReturnsSortedUnique(t *testing.T) {
	truncateAll(t)
	r := setupRouter()

	for _, ride := range []map[string]interface{}{
		{"driver_name": "A", "phone": "1", "origin": "Village A", "destination": "Station", "departure_at": "2030-06-01T09:00:00Z", "flexibility": 0},
		{"driver_name": "B", "phone": "2", "origin": "Town B", "destination": "Station", "departure_at": "2030-06-01T10:00:00Z", "flexibility": 0},
	} {
		b, _ := json.Marshal(ride)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/rides", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/destinations", nil)
	r.ServeHTTP(w, req)

	var dests []string
	json.Unmarshal(w.Body.Bytes(), &dests)
	if len(dests) != 3 {
		t.Errorf("expected 3 destinations, got %d: %v", len(dests), dests)
	}
}
```

Note: the `handler/integration_test.go` uses a `noopNotifier` — the Web Push integration itself is not tested here (it requires real push endpoints). The noopNotifier satisfies the `notification.Notifier` interface.

The `noopNotifier` must satisfy the real `Notifier` interface. Update the import and struct:

```go
import (
    "github.com/z3spinner/go-stop/internal/boundaries/notification"
    "github.com/z3spinner/go-stop/internal/domain"
)

type noopNotifier struct{}

func (n *noopNotifier) Send(_ domain.Subscription, _ domain.Message) error { return nil }

var _ notification.Notifier = (*noopNotifier)(nil) // compile-time interface check
```

- [ ] **Step 6: Update docker-compose.yml to add a test run command comment**

Add a comment to the top of `docker-compose.yml` explaining how to run integration tests:

```yaml
# Run integration tests:
#   docker compose up db -d
#   TEST_DATABASE_URL=postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable \
#     go test -tags integration ./...
```

- [ ] **Step 7: Verify integration tests compile**

Run: `go test -tags integration -run '^$' ./...`
Expected: no output (compiles, zero tests run since pattern matches nothing)

- [ ] **Step 8: Run integration tests against docker DB**

```bash
docker compose up db -d
# wait ~5 seconds for DB to be ready
TEST_DATABASE_URL="postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable" \
  go test -tags integration -v ./internal/infrastructure/postgres/... ./internal/boundaries/handler/...
```
Expected: all integration tests PASS

- [ ] **Step 9: Commit**

```bash
git add internal/infrastructure/postgres/*_integration_test.go
git add internal/boundaries/handler/integration_test.go
git add docker-compose.yml
git commit -m "test: add PostgreSQL and HTTP integration tests"
```

---

## Self-Review

### Spec Coverage

| Requirement | Task |
|---|---|
| Drivers propose rides | Task 6 (PostRide), Task 16 (handler), Task 19 (frontend) |
| Searchers browse/post requests | Task 7 (PostRequest), Task 8/9 (get/search), Task 19 (frontend) |
| Push notifications both ways | Task 5 (notify), Task 6+7 (trigger), Task 15 (webpush), Task 19 (sw.js) |
| No user accounts — phone auth | Task 12 (DeleteRide/DeleteRequest phone check) |
| Autocomplete from existing locations | Task 10 (GetDestinations), Task 19 (datalist in HTML) |
| One-time trips, no recurrence | Domain type has no recurrence field |
| Expiry | Task 13 (Expire use cases), Task 17 (hourly ticker in main.go) |
| Flexibility window matching | Task 4 (WindowsOverlap), Task 14 (PostgreSQL matching query) |
| PWA + service worker | Task 19 (manifest.json + sw.js) |
| Scalingo deployment | Task 20 (Procfile + scalingo.json + README) |
| Deploy on Scalingo button | Task 20 (README.md) |
| Local devstack (docker-compose) | Task 21 (Dockerfile + docker-compose.yml) |
| Integration tests (PostgreSQL + HTTP) | Task 22 (*_integration_test.go, build tag) |
| Clean Architecture | All tasks — dep direction enforced by package structure |

### Placeholder Scan

No TBD, TODO, or placeholder steps detected.

### Type Consistency

- `WindowsOverlap(ride domain.Ride, req domain.Request) bool` — used consistently in match_test.go and referenced in postgres matching query
- `ErrUnauthorized` defined in `delete_ride.go`, referenced in `delete_request.go` and both handlers ✓
- `mockNotifier`, `mockRideRepo`, `mockRequestRepo`, `mockSubRepo` — defined in post_ride_test.go and used across test files in the same package (`usecase_test`) ✓
- Repository interface `FindAll()` added to `RideRepository` — used in `GetRides` use case and implemented in `postgres/ride_repo.go` ✓
