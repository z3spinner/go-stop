// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import "github.com/z3spinner/go-stop/internal/boundaries/repository"

// GetRequestContactOfferStatus reports whether a driver has already offered
// their contact on a given request.
type GetRequestContactOfferStatus struct {
	requests      repository.RequestRepository
	contactOffers repository.ContactOfferRepository
}

func NewGetRequestContactOfferStatus(
	requests repository.RequestRepository,
	contactOffers repository.ContactOfferRepository,
) *GetRequestContactOfferStatus {
	return &GetRequestContactOfferStatus{requests: requests, contactOffers: contactOffers}
}

func (uc *GetRequestContactOfferStatus) Execute(requestID, offererPhone string) (bool, error) {
	req, err := uc.requests.FindByID(requestID)
	if err != nil {
		return false, ErrNotFound
	}
	if req.Phone == offererPhone {
		return false, ErrUnauthorized
	}
	_, err = uc.contactOffers.FindByRequestAndOfferer(requestID, offererPhone)
	return err == nil, nil
}
