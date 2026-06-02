package usecase

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// PingSearcher lets a driver proactively notify a matching searcher.
// It creates a pre-accepted interest so the searcher can immediately
// see the driver's phone — the driver is consenting by initiating.
type PingSearcher struct {
	requests  repository.RequestRepository
	rides     repository.RideRepository
	interests repository.InterestRepository
	subs      repository.SubscriptionRepository
	notifier  notification.Notifier
}

func NewPingSearcher(
	requests repository.RequestRepository,
	rides repository.RideRepository,
	interests repository.InterestRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *PingSearcher {
	return &PingSearcher{requests: requests, rides: rides, interests: interests, subs: subs, notifier: notifier}
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

	// Create or accept an interest on behalf of the searcher.
	// The driver is consenting to share their number by pinging.
	interest, err := uc.interests.FindByRideAndSearcher(rideID, req.Phone)
	if err != nil {
		// No existing interest — create a pre-accepted one
		interest = domain.Interest{
			ID:            uuid.New().String(),
			RideID:        rideID,
			SearcherPhone: req.Phone,
			SearcherName:  req.SearcherName,
			Status:        "accepted",
		}
		if err := uc.interests.Save(interest); err != nil {
			return fmt.Errorf("create interest: %w", err)
		}
	} else if interest.Status == "pending" {
		// Existing pending interest — accept it
		if err := uc.interests.Accept(interest.ID); err != nil {
			return fmt.Errorf("accept interest: %w", err)
		}
	}
	// If already accepted, nothing to do — just re-notify.

	msg := domain.Message{
		Title:       fmt.Sprintf("%s peut vous emmener", ride.DriverName),
		Body:        fmt.Sprintf("%s → %s — le conducteur partage son numéro avec vous", ride.Origin, ride.Destination),
		URL:         "/interests/" + interest.ID,
		ContactName: ride.DriverName,
		Phone:       ride.Phone,
		Origin:      ride.Origin,
		Destination: ride.Destination,
		DepartureAt: ride.DepartureAt,
	}
	sendToAll(req.Phone, msg, uc.subs, uc.notifier)
	return nil
}
