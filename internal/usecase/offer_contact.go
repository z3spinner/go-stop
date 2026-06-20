// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// OfferContact lets anyone share their phone number with a searcher, even
// without a posted ride. The offerer consents by initiating, so the searcher
// immediately receives the contact details via push notification.
type OfferContact struct {
	requests     repository.RequestRepository
	contactOffers repository.ContactOfferRepository
	subs         repository.SubscriptionRepository
	notifier     notification.Notifier
}

func NewOfferContact(
	requests repository.RequestRepository,
	contactOffers repository.ContactOfferRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *OfferContact {
	return &OfferContact{requests: requests, contactOffers: contactOffers, subs: subs, notifier: notifier}
}

// Execute returns whether a new offer was created (true) or one already existed
// for this request+offerer pair (false). Either way, the searcher is notified.
func (uc *OfferContact) Execute(requestID, offererPhone, offererName string) (created bool, err error) {
	if strings.TrimSpace(offererName) == "" {
		return false, ErrNameRequired
	}

	req, err := uc.requests.FindByID(requestID)
	if err != nil {
		return false, errors.New("request not found")
	}
	if req.Phone == offererPhone {
		return false, ErrUnauthorized // searcher cannot offer contact to themselves
	}

	offer, findErr := uc.contactOffers.FindByRequestAndOfferer(requestID, offererPhone)
	if findErr != nil {
		// No existing offer — create a new one.
		offer = domain.ContactOffer{
			ID:           uuid.New().String(),
			RequestID:    requestID,
			OffererPhone: offererPhone,
			OffererName:  offererName,
		}
		if saveErr := uc.contactOffers.Save(offer); saveErr != nil {
			return false, fmt.Errorf("save contact offer: %w", saveErr)
		}
		created = true
	}

	// Notify the searcher on all their devices (best-effort).
	sendToAll(req.Phone, domain.Message{
		Title:       fmt.Sprintf("%s peut vous aider", offererName),
		Body:        fmt.Sprintf("%s → %s — partagez votre contact", req.Origin, req.Destination),
		URL:         "/my-searches",
		ContactName: offererName,
		Phone:       offererPhone,
		Origin:      req.Origin,
		Destination: req.Destination,
	}, uc.subs, uc.notifier)

	return created, nil
}
