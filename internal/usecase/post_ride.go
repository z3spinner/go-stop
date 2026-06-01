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

func (uc *PostRide) Execute(ride domain.Ride) (domain.Ride, error) {
	ride.ID = uuid.New().String()
	ride.PostedAt = time.Now()
	ride.Date = time.Date(ride.DepartureAt.Year(), ride.DepartureAt.Month(), ride.DepartureAt.Day(), 0, 0, 0, 0, ride.DepartureAt.Location())
	ride.ExpiresAt = time.Date(ride.DepartureAt.Year(), ride.DepartureAt.Month(), ride.DepartureAt.Day()+1, 0, 0, 0, 0, ride.DepartureAt.Location())

	if err := uc.rides.Save(ride); err != nil {
		return domain.Ride{}, err
	}

	matching, err := uc.requests.FindMatching(ride)
	if err != nil {
		return domain.Ride{}, err
	}

	for _, req := range matching {
		// Enqueue for retry tracking regardless of subscription state
		_ = uc.notifQueue.Enqueue(ride.ID, req.ID, req.Phone)
		NotifySearcher(req.Phone, ride, uc.subs, uc.notifier)
		_ = uc.notifQueue.MarkSentByRideAndRequest(ride.ID, req.ID)
	}
	return ride, nil
}
