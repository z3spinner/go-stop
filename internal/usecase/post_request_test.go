package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type mockRideRepoWithMatch struct {
	matchResult []domain.Ride
}

func (m *mockRideRepoWithMatch) Save(domain.Ride) error                              { return nil }
func (m *mockRideRepoWithMatch) FindByID(string) (domain.Ride, error)               { return domain.Ride{}, errors.New("not found") }
func (m *mockRideRepoWithMatch) FindAll() ([]domain.Ride, error)                     { return nil, nil }
func (m *mockRideRepoWithMatch) FindByOriginAndDestination(string, string) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoWithMatch) FindMatching(domain.Request) ([]domain.Ride, error) { return m.matchResult, nil }
func (m *mockRideRepoWithMatch) Delete(string) error                                 { return nil }
func (m *mockRideRepoWithMatch) DeleteExpired() error                                { return nil }

func TestPostRequest_SavesRequest(t *testing.T) {
	reqs := &mockRequestRepo{}
	n := &mockNotifier{}

	uc := usecase.NewPostRequest(reqs, &mockRideRepoWithMatch{}, &mockSubRepo{}, n)
	err := uc.Execute(domain.Request{
		SearcherName: "Bob", Phone: "555-0002",
		Origin: "Village A", Destination: "Station",
		DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs.saved) != 1 {
		t.Errorf("expected 1 saved request, got %d", len(reqs.saved))
	}
	if reqs.saved[0].ID == "" {
		t.Error("expected request to have an ID assigned")
	}
}

func TestPostRequest_NotifiesMatchingDrivers(t *testing.T) {
	reqs := &mockRequestRepo{}
	ridesWithMatch := &mockRideRepoWithMatch{
		matchResult: []domain.Ride{{ID: "ride-1", DriverName: "Alice", Phone: "555-0001"}},
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
	reqs := &mockRequestRepo{saveErr: errors.New("db error")}
	uc := usecase.NewPostRequest(reqs, &mockRideRepoWithMatch{}, &mockSubRepo{}, &mockNotifier{})
	if err := uc.Execute(domain.Request{DepartureAt: time.Now()}); err == nil {
		t.Error("expected error when save fails")
	}
}

func TestPostRequest_SkipsNotificationIfNoSubscription(t *testing.T) {
	reqs := &mockRequestRepo{}
	ridesWithMatch := &mockRideRepoWithMatch{
		matchResult: []domain.Ride{{Phone: "555-no-sub"}},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewPostRequest(reqs, ridesWithMatch, subs, n)
	err := uc.Execute(domain.Request{DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send notification when driver has no subscription")
	}
}
