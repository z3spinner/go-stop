// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import "github.com/z3spinner/go-stop/internal/boundaries/repository"

type GetDestinations struct {
	destinations repository.DestinationRepository
}

func NewGetDestinations(destinations repository.DestinationRepository) *GetDestinations {
	return &GetDestinations{destinations: destinations}
}

func (uc *GetDestinations) Execute() ([]string, error) {
	return uc.destinations.GetAll()
}
