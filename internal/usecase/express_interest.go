// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"errors"
	"fmt"
	"strings"

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
	if strings.TrimSpace(searcherName) == "" {
		return domain.Interest{}, ErrNameRequired
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

	// Notify driver on all their devices (best-effort)
	sendToAll(ride.Phone, domain.Message{
		Title:       "Quelqu'un demande votre contact",
		Body:        fmt.Sprintf("%s cherche un trajet %s → %s", searcherName, ride.Origin, ride.Destination),
		URL:         "/my-rides",
		Origin:      ride.Origin,
		Destination: ride.Destination,
		DepartureAt: ride.DepartureAt,
	}, uc.subs, uc.notifier)

	return interest, nil
}
