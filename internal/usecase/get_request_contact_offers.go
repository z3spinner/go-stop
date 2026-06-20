// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// GetRequestContactOffers returns the contact offers made for a given request.
// Only the request owner (matching phone) may retrieve this list.
type GetRequestContactOffers struct {
	requests      repository.RequestRepository
	contactOffers repository.ContactOfferRepository
}

func NewGetRequestContactOffers(
	requests repository.RequestRepository,
	contactOffers repository.ContactOfferRepository,
) *GetRequestContactOffers {
	return &GetRequestContactOffers{requests: requests, contactOffers: contactOffers}
}

func (uc *GetRequestContactOffers) Execute(requestID, requesterPhone string) ([]domain.ContactOffer, error) {
	req, err := uc.requests.FindByID(requestID)
	if err != nil {
		return nil, ErrNotFound
	}
	if req.Phone != requesterPhone {
		return nil, ErrUnauthorized
	}
	return uc.contactOffers.ListByRequest(requestID)
}
