//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

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
		t.Errorf("expected Flexibility 30, got %d", got.Flexibility)
	}
}

func TestRideRepo_FindAll_OnlyReturnsActive(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool)

	_ = repo.Save(domain.Ride{
		ID: "active-1", DriverName: "Alice", Phone: "1",
		Origin: "A", Destination: "B",
		Date:        time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC),
	})
	_ = repo.Save(domain.Ride{
		ID: "expired-1", DriverName: "Bob", Phone: "2",
		Origin: "A", Destination: "B",
		Date:        time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2020, 1, 1, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
	})

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
	_ = repo.Save(domain.Ride{
		ID: "ride-1", DriverName: "Alice", Phone: "1",
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility: domain.Approximate,
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	})

	// Request: 09:15 exact — inside ride window
	req := domain.Request{
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 9, 15, 0, 0, time.UTC),
		Flexibility: domain.Exact,
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	}

	rides, err := repo.FindMatching(req)
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

	// Ride: 09:00 exact
	_ = repo.Save(domain.Ride{
		ID: "ride-2", DriverName: "Alice", Phone: "1",
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility: domain.Exact,
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	})

	// Request: 10:00 exact — no overlap
	req := domain.Request{
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 10, 0, 0, 0, time.UTC),
		Flexibility: domain.Exact,
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	}

	rides, err := repo.FindMatching(req)
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

	_ = repo.Save(domain.Ride{
		ID: "to-delete", DriverName: "Alice", Phone: "1",
		Origin: "A", Destination: "B",
		Date:        time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC),
	})

	if err := repo.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := repo.FindByID("to-delete")
	if err == nil {
		t.Error("expected not found after delete")
	}
}
