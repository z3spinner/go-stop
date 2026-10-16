// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// GetRequestContactOffers returns the contact offers made for a given request.
// Only the request owner (matching phone) may retrieve this list. Seeing the
// offers counts as reading their notifications (no more reminders).
type GetRequestContactOffers struct {
	requests      repository.RequestRepository
	contactOffers repository.ContactOfferRepository
	notifier      notification.Notifier
}

func NewGetRequestContactOffers(
	requests repository.RequestRepository,
	contactOffers repository.ContactOfferRepository,
	notifier notification.Notifier,
) *GetRequestContactOffers {
	return &GetRequestContactOffers{requests: requests, contactOffers: contactOffers, notifier: notifier}
}

func (uc *GetRequestContactOffers) Execute(requestID, requesterPhone string) ([]domain.ContactOffer, error) {
	req, err := uc.requests.FindByID(requestID)
	if err != nil {
		return nil, ErrNotFound
	}
	if req.Phone != requesterPhone {
		return nil, ErrUnauthorized
	}
	offers, err := uc.contactOffers.ListByRequest(requestID)
	if err != nil {
		return nil, err
	}
	if len(offers) > 0 {
		keys := make([]string, len(offers))
		for i, o := range offers {
			keys[i] = domain.InboxKey(domain.KindContactOffered, o.ID)
		}
		uc.notifier.Acknowledge(requesterPhone, keys...)
	}
	return offers, nil
}
