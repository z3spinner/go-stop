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

func TestRequestRepo_SaveAndFindByID(t *testing.T) {
	truncate(t)
	repo := postgres.NewRequestRepo(testPool, 60)

	testID := uuid.New().String()
	req := domain.Request{
		ID:           testID,
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

	got, err := repo.FindByID(testID)
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

func TestRequestRepo_FindAllActive_Ordering(t *testing.T) {
	truncate(t)
	repo := postgres.NewRequestRepo(testPool, 60)

	zero := time.Time{}
	expires := time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)
	save := func(name string, date, dep time.Time) {
		if err := repo.Save(domain.Request{
			ID: uuid.New().String(), SearcherName: name, Phone: "1",
			Origin: "A", Destination: "B",
			Date: date, DepartureAt: dep,
			PostedAt: time.Now().UTC(), ExpiresAt: expires,
		}); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
	}

	// The Date/DepartureAt zero-ness mirrors how the handler stores each mode
	// (zero → NULL). Inserted scrambled to prove it's the query that orders.
	save("anytime", zero, zero)                                        // both NULL
	save("daily", zero, time.Date(1970, 1, 1, 17, 0, 0, 0, time.UTC))  // 1970 sentinel
	save("day9", time.Date(2030, 6, 9, 0, 0, 0, 0, time.UTC), zero)    // date only, later
	save("time6", zero, time.Date(2030, 6, 6, 7, 0, 0, 0, time.UTC))   // one-off, mid
	save("day5", time.Date(2030, 6, 5, 0, 0, 0, 0, time.UTC), zero)    // date only, earlier
	save("time5am", zero, time.Date(2030, 6, 5, 4, 5, 0, 0, time.UTC)) // one-off, soonest

	got, err := repo.FindAllActive()
	if err != nil {
		t.Fatalf("FindAllActive: %v", err)
	}
	order := make([]string, len(got))
	for i, r := range got {
		order[i] = r.SearcherName
	}

	// Dated entries are interleaved chronologically; a date-only alert sorts at
	// the END of its day (below a same-day date+time, above any later day). Then
	// daily recurring, then anytime last.
	want := []string{"time5am", "day5", "time6", "day9", "daily", "anytime"}
	if len(order) != len(want) {
		t.Fatalf("got %d requests, want %d: %v", len(order), len(want), order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("ordering mismatch:\n got  %v\n want %v", order, want)
		}
	}
}

func TestRequestRepo_FindMatching_WindowOverlap(t *testing.T) {
	truncate(t)
	repo := postgres.NewRequestRepo(testPool, 60)

	// Request: 09:15 ±30 min → window 08:45–09:45
	_ = repo.Save(domain.Request{
		ID: uuid.New().String(), SearcherName: "Bob", Phone: "2",
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

func TestRequestRepo_FindAllActive_HidesPastGraceWindow(t *testing.T) {
	truncate(t)
	const grace = 30
	repo := postgres.NewRequestRepo(testPool, grace)

	now := time.Now().UTC()
	zero := time.Time{}
	farExp := now.AddDate(0, 1, 0)
	save := func(name string, date, dep time.Time, flex domain.Flexibility) {
		if err := repo.Save(domain.Request{
			ID: uuid.New().String(), SearcherName: name, Phone: "1",
			Origin: "A", Destination: "B", Date: date, DepartureAt: dep, Flexibility: flex,
			PostedAt: now, ExpiresAt: farExp,
		}); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
	}
	dayOf := func(ts time.Time) time.Time {
		return time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, time.UTC)
	}

	// time alerts (Exact, so window = the instant itself):
	pastDep := now.Add(-40 * time.Minute)   // ended 40m ago > 30 grace → hidden
	recentDep := now.Add(-10 * time.Minute) // ended 10m ago < 30 grace → shown
	futureDep := now.Add(2 * time.Hour)     // not yet → shown
	save("timePast", dayOf(pastDep), pastDep, domain.Exact)
	save("timeRecent", dayOf(recentDep), recentDep, domain.Exact)
	save("timeFuture", dayOf(futureDep), futureDep, domain.Exact)
	// date-only alerts:
	save("dayPast", dayOf(now.AddDate(0, 0, -2)), zero, domain.Exact) // 2 days ago → hidden
	save("dayToday", dayOf(now), zero, domain.Exact)                  // today → shown
	// no-moment alerts always shown:
	save("anytime", zero, zero, domain.Exact)

	got, err := repo.FindAllActive()
	if err != nil {
		t.Fatalf("FindAllActive: %v", err)
	}
	shown := map[string]bool{}
	for _, r := range got {
		shown[r.SearcherName] = true
	}
	for name, want := range map[string]bool{
		"timePast": false, "timeRecent": true, "timeFuture": true,
		"dayPast": false, "dayToday": true, "anytime": true,
	} {
		if shown[name] != want {
			t.Errorf("%s: shown=%v, want %v", name, shown[name], want)
		}
	}
}
