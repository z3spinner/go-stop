package usecase

import (
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type SearchRides struct {
	rides repository.RideRepository
}

func NewSearchRides(rides repository.RideRepository) *SearchRides {
	return &SearchRides{rides: rides}
}

// Execute returns rides for a route. When departureAt is non-zero the results
// are filtered to that calendar date only.
func (uc *SearchRides) Execute(origin, destination string, departureAt time.Time) ([]domain.Ride, error) {
	var (
		result []domain.Ride
		err    error
	)
	if departureAt.IsZero() {
		result, err = uc.rides.FindByOriginAndDestination(origin, destination)
	} else {
		date := time.Date(departureAt.Year(), departureAt.Month(), departureAt.Day(), 0, 0, 0, 0, departureAt.Location())
		result, err = uc.rides.FindByOriginDestinationAndDate(origin, destination, date)
	}
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []domain.Ride{}, nil
	}
	return result, nil
}
