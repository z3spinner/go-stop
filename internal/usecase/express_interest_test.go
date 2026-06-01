package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// mockInterestRepo — shared across Tasks 5, 6, 7 test files
type mockInterestRepo struct {
	saved        []domain.Interest
	byID         map[string]domain.Interest
	saveErr      error
	acceptCalled []string
}

func (m *mockInterestRepo) Save(i domain.Interest) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if m.byID == nil {
		m.byID = make(map[string]domain.Interest)
	}
	m.saved = append(m.saved, i)
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

// ExpressInterest tests

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
	interest, err := uc.Execute("ride-1", "555-searcher", "Bob")

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
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewExpressInterest(rides, interests, subs, n)
	_, err := uc.Execute("ride-1", "555-searcher", "Bob")

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
	_, err := uc.Execute("ride-1", "555-same", "Self")

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
	_, err := uc.Execute("nonexistent", "555-searcher", "")

	if err == nil {
		t.Error("expected error for missing ride")
	}
}
