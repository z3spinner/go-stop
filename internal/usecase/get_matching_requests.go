// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// GetMatchingRequests finds active waiting requests whose departure window
// overlaps with a given ride — i.e. people who want this journey.
type GetMatchingRequests struct {
	rides    repository.RideRepository
	requests repository.RequestRepository
}

func NewGetMatchingRequests(rides repository.RideRepository, requests repository.RequestRepository) *GetMatchingRequests {
	return &GetMatchingRequests{rides: rides, requests: requests}
}

func (uc *GetMatchingRequests) Execute(rideID string) ([]domain.Request, error) {
	ride, err := uc.rides.FindByID(rideID)
	if err != nil {
		return nil, err
	}
	result, err := uc.requests.FindMatching(ride)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []domain.Request{}, nil
	}
	return result, nil
}
