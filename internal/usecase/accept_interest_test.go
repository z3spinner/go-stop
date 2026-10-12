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
			"ride-1": {
				ID: "ride-1", Phone: "555-driver",
				Origin: "Saillans", Destination: "Crest",
				DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-searcher": {Phone: "555-searcher", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewAcceptInterest(interests, rides, subs, n)
	searcherPhone, connected, err := uc.Execute("int-1", "555-driver")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if searcherPhone != "555-searcher" {
		t.Errorf("expected searcher phone 555-searcher, got %s", searcherPhone)
	}
	if !connected {
		t.Error("expected connected=true when a pending interest is newly accepted")
	}
	if len(interests.acceptCalled) == 0 || interests.acceptCalled[0] != "int-1" {
		t.Error("expected Accept called on interest int-1")
	}
	if !n.called {
		t.Error("expected push notification sent to searcher")
	}
}

func TestAcceptInterest_RejectsWrongDriverPhone(t *testing.T) {
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
			"ride-1": {ID: "ride-1", Phone: "555-driver"},
		},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewAcceptInterest(interests, rides, subs, n)
	_, _, err := uc.Execute("int-1", "555-wrong")

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
	_, _, err := uc.Execute("nonexistent", "555-driver")

	if err == nil {
		t.Error("expected error for missing interest")
	}
}

func TestAcceptInterest_AlreadyAcceptedReportsNotNewlyConnected(t *testing.T) {
	// Re-accepting an already-accepted interest must not report a new connection,
	// otherwise a repeated click would double-count in the statistics.
	interest := domain.Interest{
		ID: "int-1", RideID: "ride-1",
		SearcherPhone: "555-searcher", Status: "accepted",
	}
	interests := &mockInterestRepo{
		byID:  map[string]domain.Interest{"int-1": interest},
		saved: []domain.Interest{interest},
	}
	rides := &mockRideRepo{
		byID: map[string]domain.Ride{"ride-1": {ID: "ride-1", Phone: "555-driver"}},
	}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	uc := usecase.NewAcceptInterest(interests, rides, subs, n)
	_, connected, err := uc.Execute("int-1", "555-driver")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if connected {
		t.Error("expected connected=false when re-accepting an already-accepted interest")
	}
}
