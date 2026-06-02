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

func (uc *AcceptInterest) Execute(interestID, driverPhone string) (string, error) {
	interest, err := uc.interests.FindByID(interestID)
	if err != nil {
		return "", err
	}
	ride, err := uc.rides.FindByID(interest.RideID)
	if err != nil {
		return "", err
	}
	if ride.Phone != driverPhone {
		return "", ErrUnauthorized
	}
	if err := uc.interests.Accept(interestID); err != nil {
		return "", err
	}
	// Notify searcher on all their devices (best-effort)
	sendToAll(interest.SearcherPhone, domain.Message{
		Title:       "Le conducteur accepte le contact",
		Body:        fmt.Sprintf("%s → %s", ride.Origin, ride.Destination),
		URL:         "/interests/" + interestID,
		Origin:      ride.Origin,
		Destination: ride.Destination,
		DepartureAt: ride.DepartureAt,
	}, uc.subs, uc.notifier)
	return interest.SearcherPhone, nil
}
