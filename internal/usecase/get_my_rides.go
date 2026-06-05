// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type GetMyRides struct {
	rides repository.RideRepository
}

func NewGetMyRides(rides repository.RideRepository) *GetMyRides {
	return &GetMyRides{rides: rides}
}

func (uc *GetMyRides) Execute(phone string) ([]domain.Ride, error) {
	result, err := uc.rides.FindByPhone(phone)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []domain.Ride{}, nil
	}
	return result, nil
}
