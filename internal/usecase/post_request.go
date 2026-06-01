package usecase

import (
	"time"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type PostRequest struct {
	requests repository.RequestRepository
	rides    repository.RideRepository
	subs     repository.SubscriptionRepository
	notifier notification.Notifier
}

func NewPostRequest(
	requests repository.RequestRepository,
	rides repository.RideRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *PostRequest {
	return &PostRequest{requests: requests, rides: rides, subs: subs, notifier: notifier}
}

func (uc *PostRequest) Execute(req domain.Request) (domain.Request, error) {
	req.ID = uuid.New().String()
	req.PostedAt = time.Now()
	switch {
	case req.Date.IsZero() && req.DepartureAt.IsZero(): // anytime
		req.ExpiresAt = time.Now().AddDate(1, 0, 0)
	case req.Date.IsZero() && !req.DepartureAt.IsZero(): // daily — time only, any date
		req.ExpiresAt = time.Now().AddDate(1, 0, 0)
	case !req.Date.IsZero() && req.DepartureAt.IsZero(): // day — date set, any time
		req.ExpiresAt = req.Date.AddDate(0, 0, 1)
	default: // specific time window
		req.Date = time.Date(req.DepartureAt.Year(), req.DepartureAt.Month(), req.DepartureAt.Day(), 0, 0, 0, 0, req.DepartureAt.Location())
		req.ExpiresAt = req.Date.AddDate(0, 0, 1)
	}

	if err := uc.requests.Save(req); err != nil {
		return domain.Request{}, err
	}

	matching, err := uc.rides.FindMatching(req)
	if err != nil {
		return domain.Request{}, err
	}

	for _, ride := range matching {
		sub, err := uc.subs.FindByPhone(ride.Phone)
		if err != nil {
			continue
		}
		_ = NotifyDriver(sub, req, uc.notifier)
	}
	return req, nil
}
