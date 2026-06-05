// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

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
//   - all zero → all active rides
//   - searchDate only → rides on that calendar date
//   - departureAt (date+time) → rides within ±searchToleranceMins of that datetime
//   - searchTime (time only, searchDate zero) → rides on any date whose departure window overlaps the time ±tolerance
func (uc *SearchRides) Execute(origin, destination string, searchDate, departureAt, searchTime time.Time) ([]domain.Ride, error) {
	var (
		result []domain.Ride
		err    error
	)
	switch {
	case !departureAt.IsZero():
		result, err = uc.rides.FindByOriginDestinationDateTime(origin, destination, departureAt, searchToleranceMins)
	case !searchDate.IsZero():
		result, err = uc.rides.FindByOriginDestinationAndDate(origin, destination, searchDate)
	case !searchTime.IsZero():
		result, err = uc.rides.FindByOriginAndTime(origin, destination, searchTime, searchToleranceMins)
	default:
		result, err = uc.rides.FindByOriginAndDestination(origin, destination)
		// Fall back to trigram fuzzy matching when an exact route search finds
		// nothing, so typos/spelling variants ("Saillan" → "Saillans") still hit.
		if err == nil && len(result) == 0 {
			result, err = uc.rides.FindByOriginAndDestinationFuzzy(origin, destination)
		}
	}
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []domain.Ride{}, nil
	}
	return result, nil
}
