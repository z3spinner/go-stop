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
	saved    []domain.Ride
	byID     map[string]domain.Ride
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
	r, ok := m.byID[id]
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
func (m *mockRideRepo) Delete(string) error                                 { return nil }
func (m *mockRideRepo) DeleteExpired() error                                { return nil }

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
func (m *mockSubRepo) FindByPhone(phone string) (domain.Subscription, error) {
	s, ok := m.subs[phone]
	if !ok {
		return domain.Subscription{}, errors.New("not found")
	}
	return s, nil
}
func (m *mockSubRepo) Delete(phone string) error { delete(m.subs, phone); return nil }

// ── PostRide tests ────────────────────────────────────────────────────────────

func TestPostRide_SavesRide(t *testing.T) {
	rides := &mockRideRepo{}
	reqs := &mockRequestRepo{}
	subs := &mockSubRepo{}
	n := &mockNotifier{}

	uc := usecase.NewPostRide(rides, reqs, subs, n)
	_, err := uc.Execute(domain.Ride{
		DriverName: "Alice", Phone: "555-0001",
		Origin: "Village A", Destination: "Station",
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility: domain.Approximate,
	})

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
		matching: []domain.Request{{ID: "req-1", SearcherName: "Bob", Phone: "555-0002"}},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0002": {Phone: "555-0002", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewPostRide(rides, reqs, subs, n)
	_, err := uc.Execute(domain.Ride{
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

func TestPostRide_SkipsNotificationIfNoSubscription(t *testing.T) {
	rides := &mockRideRepo{}
	reqs := &mockRequestRepo{matching: []domain.Request{{Phone: "555-0003"}}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewPostRide(rides, reqs, subs, n)
	_, err := uc.Execute(domain.Ride{DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)})

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
	if _, err := uc.Execute(domain.Ride{DepartureAt: time.Now()}); err == nil {
		t.Error("expected error when save fails")
	}
}
