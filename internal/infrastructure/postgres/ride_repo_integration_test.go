// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package postgres_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func TestRideRepo_SaveAndFindByID(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool, 60)

	testID := uuid.New().String()
	ride := domain.Ride{
		ID:          testID,
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

	if _, _, err := repo.Save(ride); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID(testID)
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
	repo := postgres.NewRideRepo(testPool, 60)

	activeID := uuid.New().String()
	_, _, _ = repo.Save(domain.Ride{
		ID: activeID, DriverName: "Alice", Phone: "1",
		Origin: "A", Destination: "B",
		Date:        time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC),
	})
	_, _, _ = repo.Save(domain.Ride{
		ID: uuid.New().String(), DriverName: "Bob", Phone: "2",
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
	if rides[0].ID != activeID {
		t.Errorf("expected %s, got %s", activeID, rides[0].ID)
	}
}

func TestRideRepo_FindMatching_WindowOverlap(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool, 60)

	// Ride: 09:00 ±30 min → window 08:30–09:30
	_, _, _ = repo.Save(domain.Ride{
		ID: uuid.New().String(), DriverName: "Alice", Phone: "1",
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
	repo := postgres.NewRideRepo(testPool, 60)

	// Ride: 09:00 exact
	_, _, _ = repo.Save(domain.Ride{
		ID: uuid.New().String(), DriverName: "Alice", Phone: "1",
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

func TestRideRepo_Save_UpsertsOnDuplicate(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool, 60)

	originalID := uuid.New().String()
	originalPosted := time.Now().UTC().Truncate(time.Second)
	saved, created, err := repo.Save(domain.Ride{
		ID: originalID, DriverName: "Alice", Phone: "0612345678",
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility: domain.Approximate, // 30
		PostedAt:    originalPosted,
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("first Save: %v", err)
	}
	if !created || saved.ID != originalID {
		t.Fatalf("first Save should create ride %q (created=%v, id=%q)", originalID, created, saved.ID)
	}

	// Same dedup key (normalized name/route differences, a fresh ID, a later
	// posted_at) but a changed flexibility: no new row — the existing ride is
	// upserted and returned, keeping its id and posted_at.
	saved2, created2, err := repo.Save(domain.Ride{
		ID: uuid.New().String(), DriverName: " alice ", Phone: "0612345678",
		Origin: "village a", Destination: "STATION",
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility: domain.Exact, // 0 — different from the original 30
		PostedAt:    originalPosted.Add(time.Hour),
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("duplicate Save: %v", err)
	}
	if created2 {
		t.Error("duplicate Save should report created=false")
	}
	if saved2.ID != originalID {
		t.Errorf("duplicate Save should return original ride %q, got %q", originalID, saved2.ID)
	}
	if saved2.Flexibility != domain.Exact {
		t.Errorf("duplicate Save should refresh flexibility to %d, got %d", domain.Exact, saved2.Flexibility)
	}
	if !saved2.PostedAt.Equal(originalPosted) {
		t.Errorf("duplicate Save should preserve posted_at %v, got %v", originalPosted, saved2.PostedAt)
	}

	// Exactly one row, and the persisted flexibility reflects the upsert.
	all, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected exactly 1 ride after duplicate Save, got %d", len(all))
	}
	if all[0].Flexibility != domain.Exact {
		t.Errorf("persisted ride should have upserted flexibility %d, got %d", domain.Exact, all[0].Flexibility)
	}
}

// Concurrent identical submits (the double-tap / retry case) must insert exactly
// one row — the unique index, not a TOCTOU check, is what guarantees this.
func TestRideRepo_Save_ConcurrentDuplicatesInsertOnce(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool, 60)

	const n = 8
	var createdCount int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, created, err := repo.Save(domain.Ride{
				ID: uuid.New().String(), DriverName: "Alice", Phone: "0612345678",
				Origin: "Village A", Destination: "Station",
				Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
				DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
				Flexibility: domain.Approximate,
				PostedAt:    time.Now().UTC(),
				ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
			})
			if err != nil {
				t.Errorf("concurrent Save: %v", err)
				return
			}
			if created {
				atomic.AddInt32(&createdCount, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if createdCount != 1 {
		t.Errorf("expected exactly 1 concurrent Save to create the ride, got %d", createdCount)
	}
	all, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected exactly 1 ride row after %d concurrent submits, got %d", n, len(all))
	}
}

func TestRideRepo_Delete(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool, 60)

	deleteID := uuid.New().String()
	_, _, _ = repo.Save(domain.Ride{
		ID: deleteID, DriverName: "Alice", Phone: "1",
		Origin: "A", Destination: "B",
		Date:        time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC),
	})

	if err := repo.Delete(deleteID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := repo.FindByID(deleteID)
	if err == nil {
		t.Error("expected not found after delete")
	}
}
