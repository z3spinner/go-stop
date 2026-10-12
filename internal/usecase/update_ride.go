// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type UpdateRide struct {
	rides      repository.RideRepository
	requests   repository.RequestRepository
	subs       repository.SubscriptionRepository
	notifQueue repository.NotificationQueueRepository
	notifier   notification.Notifier
	graceMins  int
}

func NewUpdateRide(
	rides repository.RideRepository,
	requests repository.RequestRepository,
	subs repository.SubscriptionRepository,
	notifQueue repository.NotificationQueueRepository,
	notifier notification.Notifier,
	graceMins int,
) *UpdateRide {
	return &UpdateRide{rides: rides, requests: requests, subs: subs, notifQueue: notifQueue, notifier: notifier, graceMins: graceMins}
}

// Execute edits a ride the caller owns in place. Only origin, destination,
// departure time and flexibility change; driver name and phone are fixed and the
// ride id is preserved, so interests survive. Derived date/expiry are recomputed
// from the new departure time. After the change, matching re-runs and only
// newly-matching searchers are notified (those already pinged for this ride are
// skipped). Returns ErrNotFound, ErrUnauthorized, or repository.ErrDuplicateRide.
func (uc *UpdateRide) Execute(id, phone, origin, destination string, departureAt time.Time, flexibility domain.Flexibility) (domain.Ride, error) {
	ride, err := uc.rides.FindByID(id)
	if err != nil {
		return domain.Ride{}, ErrNotFound
	}
	if ride.Phone != phone {
		return domain.Ride{}, ErrUnauthorized
	}

	ride.Origin = origin
	ride.Destination = destination
	ride.DepartureAt = departureAt
	ride.Flexibility = flexibility
	ride.Date = rideDate(departureAt)
	ride.ExpiresAt = rideExpiry(departureAt, flexibility, uc.graceMins)

	updated, err := uc.rides.UpdateByID(ride)
	if err != nil {
		return domain.Ride{}, err
	}

	matching, err := uc.requests.FindMatching(updated)
	if err != nil {
		return domain.Ride{}, err
	}
	enqueueAndNotifyMatches(updated, matching, uc.notifQueue, uc.subs, uc.notifier)
	return updated, nil
}
