// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func TestDestinationRepo_GetAll_DeduplicatesAndSorts(t *testing.T) {
	truncate(t)
	rideRepo := postgres.NewRideRepo(testPool, 60)
	reqRepo := postgres.NewRequestRepo(testPool, 60)
	destRepo := postgres.NewDestinationRepo(testPool)

	_, _, _ = rideRepo.Save(domain.Ride{
		ID: uuid.New().String(), DriverName: "A", Phone: "1",
		Origin: "Village A", Destination: "Station",
		Date:        time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC),
	})
	_ = reqRepo.Save(domain.Request{
		ID: uuid.New().String(), SearcherName: "B", Phone: "2",
		Origin: "Town B", Destination: "Village A",
		Date:        time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC),
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 1, 2, 0, 0, 0, 0, time.UTC),
	})

	locs, err := destRepo.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	// Expect 3 distinct locations. Village A appears in both rides and requests
	// so it ranks first by popularity; Station and Town B each appear once.
	if len(locs) != 3 {
		t.Errorf("expected 3 locations, got %d: %v", len(locs), locs)
	}
	if locs[0] != "Village A" || locs[1] != "Station" || locs[2] != "Town B" {
		t.Errorf("unexpected order or values: %v", locs)
	}
}
