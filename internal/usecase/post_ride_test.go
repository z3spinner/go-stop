// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// ── Shared mocks (used across test files in this package) ────────────────────

type mockRideRepo struct {
	saved   []domain.Ride
	updated []domain.Ride
	byID    map[string]domain.Ride
	saveErr error
	// updateErr, when set, makes UpdateByID fail (e.g. repository.ErrDuplicateRide).
	updateErr error
	// dup, when set, makes Save report the ride as a pre-existing duplicate:
	// it returns dup with created=false and does not record a new save.
	dup *domain.Ride
}

func (m *mockRideRepo) Save(r domain.Ride) (domain.Ride, bool, error) {
	if m.saveErr != nil {
		return domain.Ride{}, false, m.saveErr
	}
	if m.dup != nil {
		return *m.dup, false, nil
	}
	m.saved = append(m.saved, r)
	return r, true, nil
}
func (m *mockRideRepo) UpdateByID(r domain.Ride) (domain.Ride, error) {
	if m.updateErr != nil {
		return domain.Ride{}, m.updateErr
	}
	m.updated = append(m.updated, r)
	if m.byID != nil {
		m.byID[r.ID] = r
	}
	return r, nil
}
func (m *mockRideRepo) FindByID(id string) (domain.Ride, error) {
	r, ok := m.byID[id]
	if !ok {
		return domain.Ride{}, errors.New("not found")
	}
	return r, nil
}
func (m *mockRideRepo) FindAll() ([]domain.Ride, error)           { return m.saved, nil }
func (m *mockRideRepo) FindByPhone(string) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepo) FindByOriginAndDestination(o, d string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepo) FindByOriginDestinationAndDate(string, string, time.Time) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepo) FindByOriginDestinationDateTime(string, string, time.Time, int) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepo) FindByOriginAndTime(string, string, time.Time, int) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepo) FindByOriginAndDestinationFuzzy(string, string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepo) FindMatching(domain.Request) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepo) Delete(string) error                                { return nil }
func (m *mockRideRepo) DeleteExpired() error                               { return nil }
func (m *mockRideRepo) ClaimFeedback(string) (bool, error)                 { return true, nil }

type mockRequestRepo struct {
	saved    []domain.Request
	matching []domain.Request
	saveErr  error
}

func (m *mockRequestRepo) Save(r domain.Request) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, r)
	return nil
}
func (m *mockRequestRepo) FindByPhone(string) ([]domain.Request, error) { return nil, nil }
func (m *mockRequestRepo) FindAllActive() ([]domain.Request, error)     { return m.matching, nil }
func (m *mockRequestRepo) FindByID(string) (domain.Request, error) {
	return domain.Request{}, errors.New("not found")
}
func (m *mockRequestRepo) FindMatching(domain.Ride) ([]domain.Request, error) {
	return m.matching, nil
}
func (m *mockRequestRepo) Delete(string) error  { return nil }
func (m *mockRequestRepo) DeleteExpired() error { return nil }

type mockSubRepo struct {
	subs    map[string]domain.Subscription
	saved   []domain.Subscription
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
func (m *mockSubRepo) FindByPhone(phone string) ([]domain.Subscription, error) {
	if s, ok := m.subs[phone]; ok {
		return []domain.Subscription{s}, nil
	}
	return nil, errors.New("not found")
}
func (m *mockSubRepo) DeleteByEndpoint(string) error { return nil }
func (m *mockSubRepo) Delete(phone string) error     { delete(m.subs, phone); return nil }

// ── PostRide tests ────────────────────────────────────────────────────────────

func TestPostRide_SavesRide(t *testing.T) {
	rides := &mockRideRepo{}
	reqs := &mockRequestRepo{}
	subs := &mockSubRepo{}
	n := &mockNotifier{}

	uc := usecase.NewPostRide(rides, reqs, subs, &noopNotifQueue{}, n, 60)
	saved, created, err := uc.Execute(domain.Ride{
		DriverName: "Alice", Phone: "555-0001",
		Origin: "Village A", Destination: "Station",
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility: domain.Approximate,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected a new ride to report created=true")
	}
	if saved.ID == "" {
		t.Error("expected returned ride to have an ID")
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

// A ride departing shortly before midnight must stay alive (and so searchable)
// for its whole grace window. The naive expiry of "midnight after the departure
// day" would retire it minutes after departure — before the grace window ends —
// so the row gets cleaned up / filtered out mid-grace. ExpiresAt must therefore
// be at least departure + flexibility + grace.
func TestPostRide_ExpiryCoversGraceWindowPastMidnight(t *testing.T) {
	rides := &mockRideRepo{}
	uc := usecase.NewPostRide(rides, &mockRequestRepo{}, &mockSubRepo{}, &noopNotifQueue{}, &mockNotifier{}, 60)

	// Departs 23:50 with 30-min flexibility; grace is 60 min.
	// Grace window ends at 23:50 + 30 + 60 = 01:00 the next day — well past
	// the 00:00 midnight that the old logic would have used.
	dep := time.Date(2026, 6, 1, 23, 50, 0, 0, time.UTC)
	if _, _, err := uc.Execute(domain.Ride{
		DriverName: "Alice", Phone: "555-0001",
		Origin: "Village A", Destination: "Station",
		DepartureAt: dep, Flexibility: domain.Flexibility(30),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := dep.Add(90 * time.Minute) // departure + flexibility + grace
	got := rides.saved[0].ExpiresAt
	if got.Before(want) {
		t.Errorf("ExpiresAt %v is before the grace-window end %v; ride would vanish mid-grace", got, want)
	}
}

// A normal daytime ride should keep the original "midnight after departure"
// retention — the grace floor only ever extends expiry, never shortens it.
func TestPostRide_ExpiryStaysAtNextMidnightForDaytimeRide(t *testing.T) {
	rides := &mockRideRepo{}
	uc := usecase.NewPostRide(rides, &mockRequestRepo{}, &mockSubRepo{}, &noopNotifQueue{}, &mockNotifier{}, 60)

	dep := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	if _, _, err := uc.Execute(domain.Ride{
		DriverName: "Alice", Phone: "555-0001",
		Origin: "Village A", Destination: "Station",
		DepartureAt: dep, Flexibility: domain.Flexibility(30),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	if got := rides.saved[0].ExpiresAt; !got.Equal(want) {
		t.Errorf("ExpiresAt = %v, want next midnight %v", got, want)
	}
}

func TestPostRide_NotifiesMatchingSearchers(t *testing.T) {
	rides := &mockRideRepo{}
	reqs := &mockRequestRepo{
		matching: []domain.Request{{ID: "req-1", SearcherName: "Bob", Phone: "555-0002"}},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0002": {Phone: "555-0002", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewPostRide(rides, reqs, subs, &noopNotifQueue{}, n, 60)
	_, _, err := uc.Execute(domain.Ride{
		DriverName: "Alice", Phone: "555-0001",
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !n.called {
		t.Error("expected notification to be sent")
	}
}

// A duplicate re-post must return the existing ride without re-running the
// match/notify path — searchers were already pinged for the original.
func TestPostRide_DuplicateReturnsExistingAndSkipsNotify(t *testing.T) {
	existing := domain.Ride{
		ID: "ride-original", DriverName: "Alice", Phone: "555-0001",
		Origin: "Village A", Destination: "Station",
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	}
	rides := &mockRideRepo{dup: &existing}
	reqs := &mockRequestRepo{
		matching: []domain.Request{{ID: "req-1", SearcherName: "Bob", Phone: "555-0002"}},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0002": {Phone: "555-0002", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewPostRide(rides, reqs, subs, &noopNotifQueue{}, n, 60)
	saved, created, err := uc.Execute(domain.Ride{
		DriverName: "Alice", Phone: "555-0001",
		Origin: "Village A", Destination: "Station",
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("expected a duplicate to report created=false")
	}
	if saved.ID != "ride-original" {
		t.Errorf("expected the existing ride to be returned, got ID %q", saved.ID)
	}
	if n.called {
		t.Error("should not notify searchers for a duplicate re-post")
	}
}

func TestPostRide_SkipsNotificationIfNoSubscription(t *testing.T) {
	rides := &mockRideRepo{}
	reqs := &mockRequestRepo{matching: []domain.Request{{Phone: "555-0003"}}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewPostRide(rides, reqs, subs, &noopNotifQueue{}, n, 60)
	_, _, err := uc.Execute(domain.Ride{DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send notification when searcher has no subscription")
	}
}

func TestPostRide_ReturnsErrorIfSaveFails(t *testing.T) {
	rides := &mockRideRepo{saveErr: errors.New("db error")}
	uc := usecase.NewPostRide(rides, &mockRequestRepo{}, &mockSubRepo{}, &noopNotifQueue{}, &mockNotifier{}, 60)
	if _, _, err := uc.Execute(domain.Ride{DepartureAt: time.Now()}); err == nil {
		t.Error("expected error when save fails")
	}
}
