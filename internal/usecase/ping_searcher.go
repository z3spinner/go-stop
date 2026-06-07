// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

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

// Execute returns whether this ping newly established a connection: true only
// when it actually created a new driver_shared interest. A repeat ping (an
// interest already exists for this ride+searcher) re-notifies but reports
// connected=false, so the connection is not counted twice in the statistics.
func (uc *PingSearcher) Execute(requestID, rideID, driverPhone string) (connected bool, err error) {
	req, err := uc.requests.FindByID(requestID)
	if err != nil {
		return false, errors.New("request not found")
	}
	ride, err := uc.rides.FindByID(rideID)
	if err != nil {
		return false, errors.New("ride not found")
	}
	if ride.Phone != driverPhone {
		return false, ErrUnauthorized
	}

	// Create a "driver_shared" interest: driver consents to share their number.
	// Status "driver_shared" means only the SEARCHER can retrieve the driver's phone
	// via GetInterestContact — the driver cannot get the searcher's phone this way
	// (that requires the full mutual-consent flow).
	interest, err := uc.interests.FindByRideAndSearcher(rideID, req.Phone)
	if err != nil {
		// No existing interest — create a driver_shared one. This is the moment a
		// connection is made: the driver is sharing their contact with the searcher.
		interest = domain.Interest{
			ID:            uuid.New().String(),
			RideID:        rideID,
			SearcherPhone: req.Phone,
			SearcherName:  req.SearcherName,
			Status:        "driver_shared",
		}
		if err := uc.interests.Save(interest); err != nil {
			return false, fmt.Errorf("create interest: %w", err)
		}
		connected = true
	}
	// If interest already exists (pending/accepted/driver_shared), just re-notify.

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
	return connected, nil
}
