package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func cancelDeps(driverPhone string) (*mockRideRepo, *mockSubRepo, *mockNotifier) {
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{
			"ride-1": {
				ID: "ride-1", Phone: driverPhone,
				Origin: "Saillans", Destination: "Crest",
				DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		driverPhone: {Phone: driverPhone, Endpoint: "https://push.example.com"},
	}}
	return rides, subs, &mockNotifier{}
}

func TestCancelInterest_DeletesPendingInterestAndNotifiesDriver(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", SearcherName: "Bob", Status: "pending",
	}
	interests := &mockInterestRepo{byID: map[string]domain.Interest{"int-1": interest}}
	rides, subs, n := cancelDeps("555-driver")

	uc := usecase.NewCancelInterest(interests, rides, subs, n)
	if err := uc.Execute("int-1", "555-searcher"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(interests.deleteCalled) != 1 || interests.deleteCalled[0] != "int-1" {
		t.Errorf("expected Delete(int-1) to be called, got %v", interests.deleteCalled)
	}
	if !n.called {
		t.Error("expected the driver to be notified of the cancellation")
	}
}

func TestCancelInterest_RejectsNonOwner(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "pending",
	}
	interests := &mockInterestRepo{byID: map[string]domain.Interest{"int-1": interest}}
	rides, subs, n := cancelDeps("555-driver")

	uc := usecase.NewCancelInterest(interests, rides, subs, n)
	err := uc.Execute("int-1", "555-stranger")
	if !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
	if len(interests.deleteCalled) != 0 {
		t.Errorf("expected no delete, got %v", interests.deleteCalled)
	}
	if n.called {
		t.Error("expected no notification on unauthorized cancel")
	}
}

func TestCancelInterest_RejectsAlreadyAccepted(t *testing.T) {
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "accepted",
	}
	interests := &mockInterestRepo{byID: map[string]domain.Interest{"int-1": interest}}
	rides, subs, n := cancelDeps("555-driver")

	uc := usecase.NewCancelInterest(interests, rides, subs, n)
	err := uc.Execute("int-1", "555-searcher")
	if !errors.Is(err, usecase.ErrNotPending) {
		t.Errorf("expected ErrNotPending, got %v", err)
	}
	if len(interests.deleteCalled) != 0 {
		t.Errorf("expected no delete, got %v", interests.deleteCalled)
	}
	if n.called {
		t.Error("expected no notification when the interest is not pending")
	}
}

func TestCancelInterest_ReturnsErrorIfNotFound(t *testing.T) {
	interests := &mockInterestRepo{byID: map[string]domain.Interest{}}
	rides, subs, n := cancelDeps("555-driver")

	uc := usecase.NewCancelInterest(interests, rides, subs, n)
	if err := uc.Execute("missing", "555-searcher"); err == nil {
		t.Error("expected error for missing interest")
	}
}
