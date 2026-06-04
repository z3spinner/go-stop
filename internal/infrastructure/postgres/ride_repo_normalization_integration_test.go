//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

// saveActiveRide stores a far-future ride on the given route so it stays active.
func saveActiveRide(t *testing.T, repo *postgres.RideRepo, origin, destination string) string {
	t.Helper()
	id := uuid.New().String()
	if err := repo.Save(domain.Ride{
		ID: id, DriverName: "Alice", Phone: "555-0001",
		Origin: origin, Destination: destination,
		Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
		DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
		Flexibility: domain.Approximate,
		PostedAt:    time.Now().UTC(),
		ExpiresAt:   time.Date(2030, 6, 2, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("Save(%q→%q): %v", origin, destination, err)
	}
	return id
}

// A ride stored as "Saillans"→"Crest" must be found regardless of the caller's
// case, accents, or surrounding whitespace — the original question behind this.
func TestRideRepo_Search_NormalizesCaseAccentWhitespace(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool, 60)
	saveActiveRide(t, repo, "Saillans", "Crest")

	matches := []struct {
		name                string
		origin, destination string
	}{
		{"exact", "Saillans", "Crest"},
		{"lowercase", "saillans", "crest"},
		{"uppercase", "SAILLANS", "CREST"},
		{"accented origin", "Sàillans", "Crest"},
		{"accented destination", "Saillans", "Crèst"},
		{"surrounding whitespace", "  Saillans ", "Crest"},
		{"collapsed inner whitespace", "Saillans", "Crest"},
	}
	for _, m := range matches {
		t.Run(m.name, func(t *testing.T) {
			rides, err := repo.FindByOriginAndDestination(m.origin, m.destination)
			if err != nil {
				t.Fatalf("FindByOriginAndDestination: %v", err)
			}
			if len(rides) != 1 {
				t.Errorf("%q→%q: expected 1 ride, got %d", m.origin, m.destination, len(rides))
			}
		})
	}
}

// Normalization must not collapse genuinely different place names together.
func TestRideRepo_Search_DoesNotMatchDifferentTown(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool, 60)
	saveActiveRide(t, repo, "Saillans", "Crest")

	rides, err := repo.FindByOriginAndDestination("Die", "Crest")
	if err != nil {
		t.Fatalf("FindByOriginAndDestination: %v", err)
	}
	if len(rides) != 0 {
		t.Errorf("expected no match for a different origin, got %d", len(rides))
	}
}

// The pg_trgm fuzzy fallback should catch a near-miss typo on the route.
func TestRideRepo_SearchFuzzy_MatchesTypo(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool, 60)
	saveActiveRide(t, repo, "Saillans", "Crest")

	rides, err := repo.FindByOriginAndDestinationFuzzy("Saillan", "Crest")
	if err != nil {
		t.Fatalf("FindByOriginAndDestinationFuzzy: %v", err)
	}
	if len(rides) != 1 {
		t.Fatalf("expected the typo 'Saillan' to fuzzy-match 'Saillans', got %d rides", len(rides))
	}
	if rides[0].Origin != "Saillans" {
		t.Errorf("expected matched ride origin 'Saillans', got %q", rides[0].Origin)
	}
}

// Fuzzy matching must still discriminate: an unrelated destination shouldn't
// drag in rides for a completely different route.
func TestRideRepo_SearchFuzzy_RejectsUnrelatedRoute(t *testing.T) {
	truncate(t)
	repo := postgres.NewRideRepo(testPool, 60)
	saveActiveRide(t, repo, "Saillans", "Crest")

	rides, err := repo.FindByOriginAndDestinationFuzzy("Saillans", "Marseille")
	if err != nil {
		t.Fatalf("FindByOriginAndDestinationFuzzy: %v", err)
	}
	if len(rides) != 0 {
		t.Errorf("expected no fuzzy match for unrelated destination, got %d", len(rides))
	}
}
