// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type mockRideRepoSearch struct {
	resultsByRoute map[string][]domain.Ride
	fuzzyByRoute   map[string][]domain.Ride
	fuzzyCalled    bool
}

func (m *mockRideRepoSearch) Save(domain.Ride) error                    { return nil }
func (m *mockRideRepoSearch) FindByID(string) (domain.Ride, error)      { return domain.Ride{}, nil }
func (m *mockRideRepoSearch) FindAll() ([]domain.Ride, error)           { return nil, nil }
func (m *mockRideRepoSearch) FindByPhone(string) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoSearch) FindByOriginAndDestination(o, d string) ([]domain.Ride, error) {
	return m.resultsByRoute[o+"|"+d], nil
}
func (m *mockRideRepoSearch) FindByOriginAndDestinationFuzzy(o, d string) ([]domain.Ride, error) {
	m.fuzzyCalled = true
	return m.fuzzyByRoute[o+"|"+d], nil
}
func (m *mockRideRepoSearch) FindByOriginDestinationAndDate(string, string, time.Time) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoSearch) FindByOriginDestinationDateTime(string, string, time.Time, int) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoSearch) FindByOriginAndTime(string, string, time.Time, int) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoSearch) FindMatching(domain.Request) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoSearch) Delete(string) error                                { return nil }
func (m *mockRideRepoSearch) DeleteExpired() error                               { return nil }
func (m *mockRideRepoSearch) ClaimFeedback(string) (bool, error)                 { return true, nil }

func TestSearchRides_FiltersByOriginAndDestination(t *testing.T) {
	rides := &mockRideRepoSearch{
		resultsByRoute: map[string][]domain.Ride{
			"Village A|Station": {{ID: "1", Origin: "Village A", Destination: "Station"}},
		},
	}
	uc := usecase.NewSearchRides(rides)
	result, err := uc.Execute("Village A", "Station", time.Time{}, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 ride, got %d", len(result))
	}
}

func TestSearchRides_ReturnsEmptySliceWhenNoneFound(t *testing.T) {
	uc := usecase.NewSearchRides(&mockRideRepoSearch{resultsByRoute: map[string][]domain.Ride{}})
	result, err := uc.Execute("A", "B", time.Time{}, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected empty slice, not nil")
	}
}

func TestSearchRides_FallsBackToFuzzyWhenExactEmpty(t *testing.T) {
	rides := &mockRideRepoSearch{
		resultsByRoute: map[string][]domain.Ride{}, // exact match finds nothing
		fuzzyByRoute: map[string][]domain.Ride{
			"Saillan|Crest": {{ID: "1", Origin: "Saillans", Destination: "Crest"}},
		},
	}
	uc := usecase.NewSearchRides(rides)
	result, err := uc.Execute("Saillan", "Crest", time.Time{}, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rides.fuzzyCalled {
		t.Error("expected fuzzy fallback to be invoked when exact search was empty")
	}
	if len(result) != 1 || result[0].Origin != "Saillans" {
		t.Errorf("expected the fuzzy match (Saillans), got %#v", result)
	}
}

func TestSearchRides_SkipsFuzzyWhenExactMatches(t *testing.T) {
	rides := &mockRideRepoSearch{
		resultsByRoute: map[string][]domain.Ride{
			"Saillans|Crest": {{ID: "1", Origin: "Saillans", Destination: "Crest"}},
		},
		fuzzyByRoute: map[string][]domain.Ride{}, // would be empty if consulted
	}
	uc := usecase.NewSearchRides(rides)
	result, err := uc.Execute("Saillans", "Crest", time.Time{}, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rides.fuzzyCalled {
		t.Error("fuzzy fallback should not run when the exact search already matched")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 ride from the exact search, got %d", len(result))
	}
}

func TestSearchRides_FuzzyOnlyForRouteOnlySearch(t *testing.T) {
	// A date/time-scoped search must not trigger the date-blind fuzzy fallback.
	rides := &mockRideRepoSearch{resultsByRoute: map[string][]domain.Ride{}}
	uc := usecase.NewSearchRides(rides)
	_, err := uc.Execute("Saillan", "Crest", time.Now(), time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rides.fuzzyCalled {
		t.Error("fuzzy fallback should not run for a date-scoped search")
	}
}
