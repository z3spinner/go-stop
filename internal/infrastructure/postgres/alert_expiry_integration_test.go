// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package postgres_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func saveAlert(t *testing.T, repo *postgres.RequestRepo, name string, date, dep time.Time, expires time.Time) string {
	t.Helper()
	id := uuid.New().String()
	if err := repo.Save(domain.Request{
		ID: id, SearcherName: name, Phone: "555-0100", Origin: "Crest", Destination: "Saillans",
		Date: date, DepartureAt: dep, PostedAt: time.Now().Add(-27 * 24 * time.Hour), ExpiresAt: expires,
	}); err != nil {
		t.Fatalf("Save %s: %v", name, err)
	}
	return id
}

func names(reqs []domain.Request) []string {
	out := make([]string, len(reqs))
	for i, r := range reqs {
		out[i] = r.SearcherName
	}
	sort.Strings(out)
	return out
}

func TestRequestRepo_FindExpiringBefore_OnlyUnwarnedOpenEndedAlerts(t *testing.T) {
	truncate(t)
	repo := postgres.NewRequestRepo(testPool, 60)
	now := time.Now().UTC()
	soon := now.Add(2 * 24 * time.Hour)
	daily := time.Date(1970, 1, 1, 7, 35, 0, 0, time.UTC)
	day := time.Date(now.Year(), now.Month(), now.Day()+2, 0, 0, 0, 0, time.UTC)

	saveAlert(t, repo, "anytime-soon", time.Time{}, time.Time{}, soon)
	saveAlert(t, repo, "daily-soon", time.Time{}, daily, soon)
	saveAlert(t, repo, "anytime-later", time.Time{}, time.Time{}, now.Add(20*24*time.Hour))
	saveAlert(t, repo, "anytime-expired", time.Time{}, time.Time{}, now.Add(-time.Hour))
	saveAlert(t, repo, "dated-soon", day, time.Time{}, soon) // expires with its date: never warned

	before := now.Add(domain.AlertExpiryNotice)
	got, err := repo.FindExpiringBefore(before)
	if err != nil {
		t.Fatalf("FindExpiringBefore: %v", err)
	}
	if want := []string{"anytime-soon", "daily-soon"}; !equalStrings(names(got), want) {
		t.Fatalf("due = %v, want %v", names(got), want)
	}

	// Warned about this expiry → no longer due.
	for _, r := range got {
		if err := repo.MarkExpiryWarned(r.ID, r.ExpiresAt); err != nil {
			t.Fatalf("MarkExpiryWarned: %v", err)
		}
	}
	if got, _ := repo.FindExpiringBefore(before); len(got) != 0 {
		t.Fatalf("after warning, due = %v, want none", names(got))
	}

	// Renewed, then (a lifetime later) about to expire again → due again.
	id := got0ID(t, repo, "anytime-soon")
	if err := repo.SetExpiry(id, now.Add(time.Hour)); err != nil {
		t.Fatalf("SetExpiry: %v", err)
	}
	again, _ := repo.FindExpiringBefore(before)
	if want := []string{"anytime-soon"}; !equalStrings(names(again), want) {
		t.Fatalf("after a new expiry, due = %v, want %v", names(again), want)
	}
	if r, _ := repo.FindByID(id); r.ExpiresAt.Sub(now.Add(time.Hour)).Abs() > time.Second {
		t.Errorf("SetExpiry: ExpiresAt = %v", r.ExpiresAt)
	}
}

func TestRequestRepo_ExpiryWarningGoesWithTheRequest(t *testing.T) {
	truncate(t)
	repo := postgres.NewRequestRepo(testPool, 60)
	id := saveAlert(t, repo, "a", time.Time{}, time.Time{}, time.Now().Add(time.Hour))
	if err := repo.MarkExpiryWarned(id, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var n int
	if err := testPool.QueryRow(context.Background(), `SELECT COUNT(*) FROM request_expiry_warnings`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("warning rows left after delete: %d", n)
	}
}

func got0ID(t *testing.T, repo *postgres.RequestRepo, name string) string {
	t.Helper()
	var id string
	if err := testPool.QueryRow(context.Background(), `SELECT id::text FROM requests WHERE searcher_name = $1`, name).Scan(&id); err != nil {
		t.Fatalf("find %s: %v", name, err)
	}
	return id
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
