package usecase_test

import (
	"time"
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

type mockRideRepoSearch struct {
	resultsByRoute map[string][]domain.Ride
}

func (m *mockRideRepoSearch) Save(domain.Ride) error                              { return nil }
func (m *mockRideRepoSearch) FindByID(string) (domain.Ride, error)               { return domain.Ride{}, nil }
func (m *mockRideRepoSearch) FindAll() ([]domain.Ride, error)                     { return nil, nil }
func (m *mockRideRepoSearch) FindByPhone(string) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoSearch) FindByOriginAndDestination(o, d string) ([]domain.Ride, error) {
	return m.resultsByRoute[o+"|"+d], nil
}
func (m *mockRideRepoSearch) FindByOriginDestinationAndDate(string, string, time.Time) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoSearch) FindByOriginDestinationDateTime(string, string, time.Time, int) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoSearch) FindMatching(domain.Request) ([]domain.Ride, error) { return nil, nil }
func (m *mockRideRepoSearch) Delete(string) error                                 { return nil }
func (m *mockRideRepoSearch) DeleteExpired() error                                { return nil }
func (m *mockRideRepoSearch) FindPendingFeedback() ([]domain.Ride, error)        { return nil, nil }
func (m *mockRideRepoSearch) SetFeedbackGiven(string) error                      { return nil }

func TestSearchRides_FiltersByOriginAndDestination(t *testing.T) {
	rides := &mockRideRepoSearch{
		resultsByRoute: map[string][]domain.Ride{
			"Village A|Station": {{ID: "1", Origin: "Village A", Destination: "Station"}},
		},
	}
	uc := usecase.NewSearchRides(rides)
	result, err := uc.Execute("Village A", "Station", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 ride, got %d", len(result))
	}
}

func TestSearchRides_ReturnsEmptySliceWhenNoneFound(t *testing.T) {
	uc := usecase.NewSearchRides(&mockRideRepoSearch{resultsByRoute: map[string][]domain.Ride{}})
	result, err := uc.Execute("A", "B", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected empty slice, not nil")
	}
}
