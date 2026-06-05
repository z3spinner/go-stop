// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"fmt"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type CancelInterest struct {
	interests repository.InterestRepository
	rides     repository.RideRepository
	subs      repository.SubscriptionRepository
	notifier  notification.Notifier
}

func NewCancelInterest(
	interests repository.InterestRepository,
	rides repository.RideRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *CancelInterest {
	return &CancelInterest{interests: interests, rides: rides, subs: subs, notifier: notifier}
}

// Execute lets a searcher withdraw their own contact request.
//
// Only the searcher who created the interest may cancel it (ErrUnauthorized
// otherwise), and only while it is still pending (ErrNotPending otherwise) —
// once a driver has accepted, contact has been exchanged and there is nothing
// to withdraw. A missing interest surfaces the repository's not-found error.
//
// On success the driver — who was notified when the interest was expressed and
// may be weighing whether to accept — is notified that it has been withdrawn.
func (uc *CancelInterest) Execute(interestID, searcherPhone string) error {
	interest, err := uc.interests.FindByID(interestID)
	if err != nil {
		return err
	}
	if interest.SearcherPhone != searcherPhone {
		return ErrUnauthorized
	}
	if interest.Status != "pending" {
		return ErrNotPending
	}
	if err := uc.interests.Delete(interestID); err != nil {
		return err
	}

	// Notify the driver on all their devices that the request was withdrawn
	// (best-effort; cancellation already succeeded).
	if ride, err := uc.rides.FindByID(interest.RideID); err == nil {
		sendToAll(ride.Phone, domain.Message{
			Title:       "Demande de contact annulée",
			Body:        fmt.Sprintf("%s a annulé sa demande (%s → %s)", interest.SearcherName, ride.Origin, ride.Destination),
			URL:         "/my-rides",
			Origin:      ride.Origin,
			Destination: ride.Destination,
			DepartureAt: ride.DepartureAt,
		}, uc.subs, uc.notifier)
	}
	return nil
}
