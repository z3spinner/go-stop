// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"fmt"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type AcceptInterest struct {
	interests repository.InterestRepository
	rides     repository.RideRepository
	subs      repository.SubscriptionRepository
	notifier  notification.Notifier
}

func NewAcceptInterest(
	interests repository.InterestRepository,
	rides repository.RideRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *AcceptInterest {
	return &AcceptInterest{interests: interests, rides: rides, subs: subs, notifier: notifier}
}

// Execute returns the searcher's phone and whether this call newly established a
// connection (the interest actually transitioned pending → accepted). A repeat
// accept of an already-accepted or driver_shared interest reports connected=false
// so the connection is not counted twice in the statistics.
func (uc *AcceptInterest) Execute(interestID, driverPhone string) (searcherPhone string, connected bool, err error) {
	interest, err := uc.interests.FindByID(interestID)
	if err != nil {
		return "", false, err
	}
	ride, err := uc.rides.FindByID(interest.RideID)
	if err != nil {
		return "", false, err
	}
	if ride.Phone != driverPhone {
		return "", false, ErrUnauthorized
	}
	if err := uc.interests.Accept(interestID); err != nil {
		return "", false, err
	}
	// A connection is "made" only when a still-pending interest is accepted; a
	// driver_shared interest already counted when the driver pinged the searcher.
	connected = interest.Status == "pending"
	// Notify searcher on all their devices (best-effort)
	sendToAll(interest.SearcherPhone, domain.Message{
		Title:       "Le conducteur accepte le contact",
		Body:        fmt.Sprintf("%s → %s", ride.Origin, ride.Destination),
		URL:         "/interests/" + interestID,
		Origin:      ride.Origin,
		Destination: ride.Destination,
		DepartureAt: ride.DepartureAt,
	}, uc.subs, uc.notifier)
	return interest.SearcherPhone, connected, nil
}
