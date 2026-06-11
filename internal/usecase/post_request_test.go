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

type mockRideRepoWithMatch struct {
	matchResult []domain.Ride
}

func (m *mockRideRepoWithMatch) Save(rd domain.Ride) (domain.Ride, bool, error) {
	return rd, true, nil
}
func (m *mockRideRepoWithMatch) UpdateByID(rd domain.Ride) (domain.Ride, error) { return rd, nil }
func (m *mockRideRepoWithMatch) FindByID(string) (domain.Ride, error) {
	return domain.Ride{}, errors.New("not found")
}
func (m *mockRideRepoWithMatch) FindAll() ([]domain.Ride, error)           { return nil, nil }
func (m *mockRideRepoWithMatch) FindByPhone(string) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoWithMatch) FindByOriginAndDestination(string, string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoWithMatch) FindByOriginDestinationAndDate(string, string, time.Time) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoWithMatch) FindByOriginDestinationDateTime(string, string, time.Time, int) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoWithMatch) FindByOriginAndTime(string, string, time.Time, int) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoWithMatch) FindByOriginAndDestinationFuzzy(string, string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoWithMatch) FindMatching(domain.Request) ([]domain.Ride, error) {
	return m.matchResult, nil
}
func (m *mockRideRepoWithMatch) Delete(string) error                { return nil }
func (m *mockRideRepoWithMatch) DeleteExpired() error               { return nil }
func (m *mockRideRepoWithMatch) ClaimFeedback(string) (bool, error) { return true, nil }

func TestPostRequest_SavesRequest(t *testing.T) {
	reqs := &mockRequestRepo{}
	n := &mockNotifier{}

	uc := usecase.NewPostRequest(reqs, &mockRideRepoWithMatch{}, &mockSubRepo{}, n)
	_, err := uc.Execute(domain.Request{
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
	_, err := uc.Execute(domain.Request{
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
	if _, err := uc.Execute(domain.Request{DepartureAt: time.Now()}); err == nil {
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
	_, err := uc.Execute(domain.Request{DepartureAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("should not send notification when driver has no subscription")
	}
}

func TestPostRequest_TimeAlertExpiresDayAfterNotAYear(t *testing.T) {
	reqs := &mockRequestRepo{}
	uc := usecase.NewPostRequest(reqs, &mockRideRepoWithMatch{}, &mockSubRepo{}, &mockNotifier{})

	// A one-off time alert (date NULL from the handler, a real future departure).
	_, err := uc.Execute(domain.Request{
		Origin: "A", Destination: "B",
		DepartureAt: time.Date(2030, 6, 5, 8, 0, 0, 0, time.UTC),
		Flexibility: domain.Approximate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	saved := reqs.saved[0]
	// Date is derived from the departure day, and it expires the day after (like a
	// ride) — not a year out from being mistaken for a daily alert.
	if want := time.Date(2030, 6, 5, 0, 0, 0, 0, time.UTC); !saved.Date.Equal(want) {
		t.Errorf("Date = %v, want %v", saved.Date, want)
	}
	if want := time.Date(2030, 6, 6, 0, 0, 0, 0, time.UTC); !saved.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", saved.ExpiresAt, want)
	}
}

func TestPostRequest_DailyAlertExpiresInAYear(t *testing.T) {
	reqs := &mockRequestRepo{}
	uc := usecase.NewPostRequest(reqs, &mockRideRepoWithMatch{}, &mockSubRepo{}, &mockNotifier{})

	// Daily alert: the 1970-01-01 sentinel time, no date.
	_, err := uc.Execute(domain.Request{
		Origin: "A", Destination: "B",
		DepartureAt: time.Date(1970, 1, 1, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	saved := reqs.saved[0]
	if !saved.Date.IsZero() {
		t.Errorf("daily Date should stay zero (NULL), got %v", saved.Date)
	}
	if saved.ExpiresAt.Before(time.Now().AddDate(0, 11, 0)) {
		t.Errorf("daily alert should expire ~1 year out, got %v", saved.ExpiresAt)
	}
}
