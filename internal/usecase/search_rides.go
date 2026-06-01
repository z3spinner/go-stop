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

const searchToleranceMins = 60 // rides within ±60 min of the search time are shown

// Execute returns rides for a route.
//   - No departureAt → all active rides on the route
//   - Date only (time = 00:00:00 UTC) → all rides on that calendar date
//   - Date + time → rides whose departure window overlaps search time ±60 min
func (uc *SearchRides) Execute(origin, destination string, departureAt time.Time) ([]domain.Ride, error) {
	var (
		result []domain.Ride
		err    error
	)
	switch {
	case departureAt.IsZero():
		result, err = uc.rides.FindByOriginAndDestination(origin, destination)
	case departureAt.UTC().Hour() == 0 && departureAt.UTC().Minute() == 0:
		// Date-only search (frontend sends midnight UTC when no time is set)
		date := time.Date(departureAt.Year(), departureAt.Month(), departureAt.Day(), 0, 0, 0, 0, departureAt.Location())
		result, err = uc.rides.FindByOriginDestinationAndDate(origin, destination, date)
	default:
		// Date + time search
		result, err = uc.rides.FindByOriginDestinationDateTime(origin, destination, departureAt, searchToleranceMins)
	}
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []domain.Ride{}, nil
	}
	return result, nil
}
