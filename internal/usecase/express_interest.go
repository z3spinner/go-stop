package usecase

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type ExpressInterest struct {
	rides     repository.RideRepository
	interests repository.InterestRepository
	subs      repository.SubscriptionRepository
	notifier  notification.Notifier
}

func NewExpressInterest(
	rides repository.RideRepository,
	interests repository.InterestRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *ExpressInterest {
	return &ExpressInterest{rides: rides, interests: interests, subs: subs, notifier: notifier}
}

func (uc *ExpressInterest) Execute(rideID, searcherPhone, searcherName string) (domain.Interest, error) {
	ride, err := uc.rides.FindByID(rideID)
	if err != nil {
		return domain.Interest{}, err
	}
	if ride.Phone == searcherPhone {
		return domain.Interest{}, errors.New("searcher cannot be the driver")
	}

	interest := domain.Interest{
		ID:            uuid.New().String(),
		RideID:        rideID,
		SearcherPhone: searcherPhone,
		SearcherName:  searcherName,
		Status:        "pending",
	}
	if err := uc.interests.Save(interest); err != nil {
		return domain.Interest{}, err
	}

	// Re-fetch in case a duplicate already existed (Save uses ON CONFLICT DO NOTHING)
	if existing, err := uc.interests.FindByRideAndSearcher(rideID, searcherPhone); err == nil {
		interest = existing
	}

	// Notify driver (best-effort)
	if sub, err := uc.subs.FindByPhone(ride.Phone); err == nil {
		msg := domain.Message{
			Title:       fmt.Sprintf("%s est intéressé(e) par votre trajet", interest.SearcherName),
			Body:        fmt.Sprintf("%s → %s", ride.Origin, ride.Destination),
			URL:         "/my-rides",
			Origin:      ride.Origin,
			Destination: ride.Destination,
			DepartureAt: ride.DepartureAt,
		}
		_ = uc.notifier.Send(sub, msg)
	}

	return interest, nil
}
