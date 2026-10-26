// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package postgres_test

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

// Time-of-day matches (time-only search, daily alerts) compare wall-clock times
// in the service timezone, so they hold across a DST change: a 09:10 ride in
// summer is still "around 09:30" when searched from winter.

func paris(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatalf("load Europe/Paris: %v", err)
	}
	return loc
}

// saveRideDeparting stores an active ride A→B leaving at dep, give or take flex.
func saveRideDeparting(t *testing.T, repo *postgres.RideRepo, dep time.Time, flex domain.Flexibility) string {
	t.Helper()
	id := uuid.New().String()
	if _, _, err := repo.Save(domain.Ride{
		ID: id, DriverName: "Alice", Phone: "555-0001",
		Origin: "A", Destination: "B",
		Date:        time.Date(dep.Year(), dep.Month(), dep.Day(), 0, 0, 0, 0, time.UTC),
		DepartureAt: dep,
		Flexibility: flex,
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   dep.AddDate(0, 0, 1),
	}); err != nil {
		t.Fatalf("Save ride at %v: %v", dep, err)
	}
	return id
}

// dailyAlertAt is a daily A→B alert for the given local HH:MM, stored the way
// the request handler stores it: on the 1970-01-01 UTC sentinel.
func dailyAlertAt(hour, minute int) domain.Request {
	return domain.Request{
		ID: uuid.New().String(), SearcherName: "Bob", Phone: "555-0002",
		Origin: "A", Destination: "B",
		DepartureAt: time.Date(1970, 1, 1, hour, minute, 0, 0, time.UTC),
		Flexibility: domain.Exact,
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Now().AddDate(1, 0, 0),
	}
}

func TestRideRepo_FindByOriginAndTime_UsesServiceWallClock(t *testing.T) {
	truncate(t)
	loc := paris(t)
	repo := postgres.NewRideRepo(testPool, 60, loc)
	near := saveRideDeparting(t, repo, time.Date(2031, 9, 1, 9, 10, 0, 0, loc), domain.Exact) // CEST
	far := saveRideDeparting(t, repo, time.Date(2031, 9, 1, 15, 0, 0, 0, loc), domain.Exact)

	// Searched from winter (CET), the handler's "09:30 today" instant.
	rides, err := repo.FindByOriginAndTime("A", "B", time.Date(2031, 1, 15, 9, 30, 0, 0, loc), 60)
	if err != nil {
		t.Fatalf("FindByOriginAndTime: %v", err)
	}
	var ids []string
	for _, r := range rides {
		ids = append(ids, r.ID)
	}
	if !slices.Contains(ids, near) || slices.Contains(ids, far) {
		t.Errorf("got %v, want only the 09:10 ride %s", ids, near)
	}
}

func TestRequestRepo_FindMatching_DailyAlertUsesServiceWallClock(t *testing.T) {
	truncate(t)
	loc := paris(t)
	repo := postgres.NewRequestRepo(testPool, 60, loc)
	alert := dailyAlertAt(8, 0)
	if err := repo.Save(alert); err != nil {
		t.Fatalf("Save alert: %v", err)
	}

	cases := []struct {
		name  string
		dep   time.Time
		match bool
	}{
		{"winter 08:00", time.Date(2031, 1, 15, 8, 0, 0, 0, loc), true},
		{"summer 08:00", time.Date(2031, 7, 15, 8, 0, 0, 0, loc), true},
		{"winter 09:00", time.Date(2031, 1, 15, 9, 0, 0, 0, loc), false},
		{"summer 10:00", time.Date(2031, 7, 15, 10, 0, 0, 0, loc), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := repo.FindMatching(domain.Ride{
				Origin: "A", Destination: "B",
				Date:        time.Date(c.dep.Year(), c.dep.Month(), c.dep.Day(), 0, 0, 0, 0, time.UTC),
				DepartureAt: c.dep,
				Flexibility: domain.Exact,
			})
			if err != nil {
				t.Fatalf("FindMatching: %v", err)
			}
			if matched := len(got) == 1 && got[0].ID == alert.ID; matched != c.match {
				t.Errorf("matched = %v, want %v", matched, c.match)
			}
		})
	}
}

func TestRideRepo_FindMatching_DailyAlertUsesServiceWallClock(t *testing.T) {
	truncate(t)
	loc := paris(t)
	repo := postgres.NewRideRepo(testPool, 60, loc)
	winter8 := saveRideDeparting(t, repo, time.Date(2031, 1, 15, 8, 0, 0, 0, loc), domain.Exact)
	summer8 := saveRideDeparting(t, repo, time.Date(2031, 7, 15, 8, 0, 0, 0, loc), domain.Exact)
	saveRideDeparting(t, repo, time.Date(2031, 1, 15, 9, 0, 0, 0, loc), domain.Exact)
	saveRideDeparting(t, repo, time.Date(2031, 7, 15, 10, 0, 0, 0, loc), domain.Exact)

	rides, err := repo.FindMatching(dailyAlertAt(8, 0))
	if err != nil {
		t.Fatalf("FindMatching: %v", err)
	}
	var ids []string
	for _, r := range rides {
		ids = append(ids, r.ID)
	}
	slices.Sort(ids)
	want := []string{winter8, summer8}
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Errorf("got %v, want the two 08:00 rides %v", ids, want)
	}
}

// Windows that touch midnight wrap round the clock: 00:10 is 40 minutes after
// 23:30, and a 00:10 ride give or take 30 minutes starts at 23:40.

func rideIDs(rides []domain.Ride) []string {
	var ids []string
	for _, r := range rides {
		ids = append(ids, r.ID)
	}
	return ids
}

func TestRideRepo_FindByOriginAndTime_WindowWrapsMidnight(t *testing.T) {
	loc := paris(t)
	cases := []struct {
		name   string
		ride   time.Time
		flex   domain.Flexibility
		search time.Time
		match  bool
	}{
		{"search 23:30 finds 00:10", time.Date(2031, 1, 16, 0, 10, 0, 0, loc), domain.Exact, time.Date(2031, 1, 15, 23, 30, 0, 0, loc), true},
		{"search 00:20 finds 00:10±30", time.Date(2031, 1, 16, 0, 10, 0, 0, loc), domain.Approximate, time.Date(2031, 1, 15, 0, 20, 0, 0, loc), true},
		{"search 00:30 finds 23:50", time.Date(2031, 1, 15, 23, 50, 0, 0, loc), domain.Exact, time.Date(2031, 1, 15, 0, 30, 0, 0, loc), true},
		{"search 23:30 skips 01:00", time.Date(2031, 1, 16, 1, 0, 0, 0, loc), domain.Exact, time.Date(2031, 1, 15, 23, 30, 0, 0, loc), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			truncate(t)
			repo := postgres.NewRideRepo(testPool, 60, loc)
			id := saveRideDeparting(t, repo, c.ride, c.flex)
			rides, err := repo.FindByOriginAndTime("A", "B", c.search, 60)
			if err != nil {
				t.Fatalf("FindByOriginAndTime: %v", err)
			}
			if got := slices.Contains(rideIDs(rides), id); got != c.match {
				t.Errorf("matched = %v, want %v", got, c.match)
			}
		})
	}
}

func TestDailyAlertMatching_WindowWrapsMidnight(t *testing.T) {
	truncate(t)
	loc := paris(t)
	rides := postgres.NewRideRepo(testPool, 60, loc)
	requests := postgres.NewRequestRepo(testPool, 60, loc)
	alert := dailyAlertAt(23, 50)
	alert.Flexibility = domain.Approximate
	if err := requests.Save(alert); err != nil {
		t.Fatalf("Save alert: %v", err)
	}
	dep := time.Date(2031, 1, 16, 0, 5, 0, 0, loc) // 15 minutes after 23:50
	rideID := saveRideDeparting(t, rides, dep, domain.Exact)

	matched, err := requests.FindMatching(domain.Ride{
		Origin: "A", Destination: "B",
		Date:        time.Date(2031, 1, 16, 0, 0, 0, 0, time.UTC),
		DepartureAt: dep, Flexibility: domain.Exact,
	})
	if err != nil {
		t.Fatalf("RequestRepo.FindMatching: %v", err)
	}
	if len(matched) != 1 || matched[0].ID != alert.ID {
		t.Errorf("RequestRepo.FindMatching = %v, want the 23:50 alert", matched)
	}

	found, err := rides.FindMatching(alert)
	if err != nil {
		t.Fatalf("RideRepo.FindMatching: %v", err)
	}
	if ids := rideIDs(found); !slices.Equal(ids, []string{rideID}) {
		t.Errorf("RideRepo.FindMatching = %v, want the 00:05 ride %s", ids, rideID)
	}
}
