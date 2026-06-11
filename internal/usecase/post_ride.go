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

type PostRide struct {
	rides      repository.RideRepository
	requests   repository.RequestRepository
	subs       repository.SubscriptionRepository
	notifQueue repository.NotificationQueueRepository
	notifier   notification.Notifier
}

func NewPostRide(
	rides repository.RideRepository,
	requests repository.RequestRepository,
	subs repository.SubscriptionRepository,
	notifQueue repository.NotificationQueueRepository,
	notifier notification.Notifier,
) *PostRide {
	return &PostRide{rides: rides, requests: requests, subs: subs, notifQueue: notifQueue, notifier: notifier}
}

// Execute posts a ride idempotently. A re-post of an identical ride (same
// phone + name + route + exact departure time) returns the existing ride with
// created=false and does not re-notify searchers, who were already notified when
// the ride was first posted.
func (uc *PostRide) Execute(ride domain.Ride) (saved domain.Ride, created bool, err error) {
	ride.ID = uuid.New().String()
	ride.PostedAt = time.Now()
	ride.Date = time.Date(ride.DepartureAt.Year(), ride.DepartureAt.Month(), ride.DepartureAt.Day(), 0, 0, 0, 0, ride.DepartureAt.Location())
	ride.ExpiresAt = time.Date(ride.DepartureAt.Year(), ride.DepartureAt.Month(), ride.DepartureAt.Day()+1, 0, 0, 0, 0, ride.DepartureAt.Location())

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

	for _, req := range matching {
		// Enqueue for retry tracking regardless of subscription state
		_ = uc.notifQueue.Enqueue(saved.ID, req.ID, req.Phone)
		NotifySearcher(req.Phone, saved, uc.subs, uc.notifier)
		_ = uc.notifQueue.MarkSentByRideAndRequest(saved.ID, req.ID)
	}
	return saved, true, nil
}
