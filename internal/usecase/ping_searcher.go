package usecase

import (
	"errors"
	"fmt"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// PingSearcher lets a driver proactively notify a matching searcher that
// their ride is available — the searcher can then request contact.
type PingSearcher struct {
	requests repository.RequestRepository
	rides    repository.RideRepository
	subs     repository.SubscriptionRepository
	notifier notification.Notifier
}

func NewPingSearcher(
	requests repository.RequestRepository,
	rides repository.RideRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *PingSearcher {
	return &PingSearcher{requests: requests, rides: rides, subs: subs, notifier: notifier}
}

func (uc *PingSearcher) Execute(requestID, rideID, driverPhone string) error {
	req, err := uc.requests.FindByID(requestID)
	if err != nil {
		return errors.New("request not found")
	}
	ride, err := uc.rides.FindByID(rideID)
	if err != nil {
		return errors.New("ride not found")
	}
	if ride.Phone != driverPhone {
		return ErrUnauthorized
	}
	msg := domain.Message{
		Title:       fmt.Sprintf("%s peut vous emmener", ride.DriverName),
		Body:        fmt.Sprintf("Trajet %s → %s correspond à votre recherche. Demandez le contact !", ride.Origin, ride.Destination),
		URL:         "/rides/" + ride.ID,
		ContactName: ride.DriverName,
		Origin:      ride.Origin,
		Destination: ride.Destination,
		DepartureAt: ride.DepartureAt,
	}
	sendToAll(req.Phone, msg, uc.subs, uc.notifier)
	return nil
}
