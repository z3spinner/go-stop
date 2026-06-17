// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"time"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// rideDate is the calendar day a ride departs (midnight in the departure's own
// location), used as the ride's grouping date.
func rideDate(departureAt time.Time) time.Time {
	return time.Date(departureAt.Year(), departureAt.Month(), departureAt.Day(), 0, 0, 0, 0, departureAt.Location())
}

// rideExpiry is when a ride retires: midnight after its departure day, but never
// before its grace window ends (departure + flexibility + grace). Without the
// grace floor a ride departing shortly before midnight would expire minutes
// later — and be cleaned up / filtered out of search while still inside its
// grace window. max() keeps the original next-midnight retention for the common
// daytime case and only ever extends it for late-night departures.
func rideExpiry(departureAt time.Time, flexibility domain.Flexibility, graceMins int) time.Time {
	nextMidnight := time.Date(departureAt.Year(), departureAt.Month(), departureAt.Day()+1, 0, 0, 0, 0, departureAt.Location())
	graceEnd := departureAt.Add(time.Duration(int(flexibility)+graceMins) * time.Minute)
	if graceEnd.After(nextMidnight) {
		return graceEnd
	}
	return nextMidnight
}

type PostRide struct {
	rides      repository.RideRepository
	requests   repository.RequestRepository
	subs       repository.SubscriptionRepository
	notifQueue repository.NotificationQueueRepository
	notifier   notification.Notifier
	graceMins  int
}

func NewPostRide(
	rides repository.RideRepository,
	requests repository.RequestRepository,
	subs repository.SubscriptionRepository,
	notifQueue repository.NotificationQueueRepository,
	notifier notification.Notifier,
	graceMins int,
) *PostRide {
	return &PostRide{rides: rides, requests: requests, subs: subs, notifQueue: notifQueue, notifier: notifier, graceMins: graceMins}
}

// Execute posts a ride idempotently. A re-post of an identical ride (same
// phone + name + route + exact departure time) upserts the existing ride's
// mutable fields, returns it with created=false, and does not re-notify
// searchers, who were already notified when the ride was first posted.
func (uc *PostRide) Execute(ride domain.Ride) (saved domain.Ride, created bool, err error) {
	ride.ID = uuid.New().String()
	ride.PostedAt = time.Now()
	ride.Date = rideDate(ride.DepartureAt)
	ride.ExpiresAt = rideExpiry(ride.DepartureAt, ride.Flexibility, uc.graceMins)

	saved, created, err = uc.rides.Save(ride)
	if err != nil {
		return domain.Ride{}, false, err
	}
	if !created {
		// Duplicate post: nothing new to match or notify.
		return saved, false, nil
	}

	matching, err := uc.requests.FindMatching(saved)
	if err != nil {
		return domain.Ride{}, false, err
	}
	enqueueAndNotifyMatches(saved, matching, uc.notifQueue, uc.subs, uc.notifier)
	return saved, true, nil
}
