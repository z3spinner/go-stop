package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// mockRideRepoFeedback — minimal ride repo for feedback tests
type mockRideRepoFeedback struct {
	rides       map[string]domain.Ride
	feedbackSet []string
}

func (m *mockRideRepoFeedback) Save(domain.Ride) error { return nil }
func (m *mockRideRepoFeedback) FindByID(id string) (domain.Ride, error) {
	r, ok := m.rides[id]
	if !ok {
		return domain.Ride{}, errors.New("not found")
	}
	return r, nil
}
func (m *mockRideRepoFeedback) FindAll() ([]domain.Ride, error)           { return nil, nil }
func (m *mockRideRepoFeedback) FindByPhone(string) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoFeedback) FindByOriginAndDestination(string, string) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoFeedback) FindByOriginDestinationAndDate(string, string, time.Time) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoFeedback) FindByOriginDestinationDateTime(string, string, time.Time, int) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoFeedback) FindByOriginAndTime(string, string, time.Time, int) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoFeedback) FindMatching(domain.Request) ([]domain.Ride, error) {
	return nil, nil
}
func (m *mockRideRepoFeedback) FindPendingFeedback() ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoFeedback) Delete(string) error                         { return nil }
func (m *mockRideRepoFeedback) DeleteExpired() error                        { return nil }
func (m *mockRideRepoFeedback) SetFeedbackGiven(id string) error {
	m.feedbackSet = append(m.feedbackSet, id)
	return nil
}

// mockStatRepo
type mockStatRepo struct {
	saved   []savedStat
	saveErr error
	stats   domain.Stats
}

type savedStat struct {
	origin, destination string
	rideDate            time.Time
	taken               bool
}

func (m *mockStatRepo) Save(origin, destination string, rideDate time.Time, taken bool) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = append(m.saved, savedStat{origin, destination, rideDate, taken})
	return nil
}
func (m *mockStatRepo) RecordSearch(string, string) error { return nil }
func (m *mockStatRepo) RecordRide(string, string) error   { return nil }
func (m *mockStatRepo) GetStats() (domain.Stats, error)   { return m.stats, nil }

func TestRecordFeedback_SavesStatAndMarksFeedbackGiven(t *testing.T) {
	rides := &mockRideRepoFeedback{
		rides: map[string]domain.Ride{
			"ride-1": {
				ID: "ride-1", Phone: "555-0001",
				Origin: "Saillans", Destination: "Crest",
				Date:        time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC),
				DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	stats := &mockStatRepo{}

	uc := usecase.NewRecordFeedback(rides, stats)
	err := uc.Execute("ride-1", "555-0001", true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats.saved) != 1 {
		t.Errorf("expected 1 stat saved, got %d", len(stats.saved))
	}
	if !stats.saved[0].taken {
		t.Error("expected taken=true")
	}
	if stats.saved[0].origin != "Saillans" {
		t.Errorf("expected origin Saillans, got %s", stats.saved[0].origin)
	}
	if len(rides.feedbackSet) != 1 || rides.feedbackSet[0] != "ride-1" {
		t.Error("expected feedback_given set on ride-1")
	}
}

func TestRecordFeedback_SavesNegativeFeedback(t *testing.T) {
	rides := &mockRideRepoFeedback{
		rides: map[string]domain.Ride{
			"ride-2": {ID: "ride-2", Phone: "555-0001", Origin: "A", Destination: "B",
				Date: time.Now()},
		},
	}
	stats := &mockStatRepo{}

	uc := usecase.NewRecordFeedback(rides, stats)
	err := uc.Execute("ride-2", "555-0001", false)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.saved[0].taken {
		t.Error("expected taken=false")
	}
}

func TestRecordFeedback_RejectsWrongPhone(t *testing.T) {
	rides := &mockRideRepoFeedback{
		rides: map[string]domain.Ride{
			"ride-1": {ID: "ride-1", Phone: "555-0001"},
		},
	}
	stats := &mockStatRepo{}

	uc := usecase.NewRecordFeedback(rides, stats)
	err := uc.Execute("ride-1", "555-9999", true)

	if !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
	if len(stats.saved) != 0 {
		t.Error("should not save stat on unauthorized")
	}
}

func TestRecordFeedback_ReturnsErrorIfRideNotFound(t *testing.T) {
	rides := &mockRideRepoFeedback{rides: map[string]domain.Ride{}}
	stats := &mockStatRepo{}

	uc := usecase.NewRecordFeedback(rides, stats)
	err := uc.Execute("nonexistent", "555-0001", true)

	if err == nil {
		t.Error("expected not found error")
	}
}
