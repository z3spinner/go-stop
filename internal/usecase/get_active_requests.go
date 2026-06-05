// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// GetActiveRequests lists every non-expired ride request (all searchers), for
// the public requests feed on the home page.
type GetActiveRequests struct {
	requests repository.RequestRepository
}

func NewGetActiveRequests(requests repository.RequestRepository) *GetActiveRequests {
	return &GetActiveRequests{requests: requests}
}

func (uc *GetActiveRequests) Execute() ([]domain.Request, error) {
	result, err := uc.requests.FindAllActive()
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []domain.Request{}, nil
	}
	return result, nil
}
