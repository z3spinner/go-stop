//go:build integration

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
		t.Errorf("expected Flexibility 60, got %d", got.Flexibility)
	}
}

func TestRequestRepo_FindMatching_WindowOverlap(t *testing.T) {
	truncate(t)
	repo := postgres.NewRequestRepo(testPool)

	// Request: 09:15 ±30 min → window 08:45–09:45
	_ = repo.Save(domain.Request{
		ID: "req-1", SearcherName: "Bob", Phone: "2",
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 9, 15, 0, 0, time.UTC),
		Flexibility: domain.Approximate,
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	})

	// Ride: 09:00 exact — inside request window
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
