// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"errors"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type ContactInfo struct {
	Phone       string
	Name        string
	Role        string // "driver" or "searcher"
	Origin      string
	Destination string
	DepartureAt time.Time
}

type GetInterestContact struct {
	interests repository.InterestRepository
	rides     repository.RideRepository
	notifier  notification.Notifier
}

func NewGetInterestContact(
	interests repository.InterestRepository,
	rides repository.RideRepository,
	notifier notification.Notifier,
) *GetInterestContact {
	return &GetInterestContact{interests: interests, rides: rides, notifier: notifier}
}

// Execute returns the other party's contact. When the searcher opens the
// driver's contact, the notifications that announced it count as read (no
// more reminders).
func (uc *GetInterestContact) Execute(interestID, requesterPhone string) (ContactInfo, error) {
	info, err := uc.contact(interestID, requesterPhone)
	if err == nil && info.Role == "driver" {
		uc.notifier.Acknowledge(requesterPhone,
			domain.InboxKey(domain.KindInterestAccepted, interestID),
			domain.InboxKey(domain.KindDriverShared, interestID))
	}
	return info, err
}

func (uc *GetInterestContact) contact(interestID, requesterPhone string) (ContactInfo, error) {
	interest, err := uc.interests.FindByID(interestID)
	if err != nil {
		return ContactInfo{}, err
	}
	ride, err := uc.rides.FindByID(interest.RideID)
	if err != nil {
		return ContactInfo{}, err
	}

	switch interest.Status {
	case "driver_shared":
		// Driver unilaterally shared their number (ping flow).
		// Only the searcher may retrieve the driver's phone — the driver
		// cannot use this to look up the searcher's number.
		if requesterPhone != interest.SearcherPhone {
			return ContactInfo{}, ErrUnauthorized
		}
		return ContactInfo{
			Phone:       ride.Phone,
			Name:        ride.DriverName,
			Role:        "driver",
			Origin:      ride.Origin,
			Destination: ride.Destination,
			DepartureAt: ride.DepartureAt,
		}, nil

	case "accepted":
		// Mutual consent — both parties may retrieve each other's info.
		switch requesterPhone {
		case ride.Phone:
			return ContactInfo{
				Phone:       interest.SearcherPhone,
				Name:        interest.SearcherName,
				Role:        "searcher",
				Origin:      ride.Origin,
				Destination: ride.Destination,
				DepartureAt: ride.DepartureAt,
			}, nil
		case interest.SearcherPhone:
			return ContactInfo{
				Phone:       ride.Phone,
				Name:        ride.DriverName,
				Role:        "driver",
				Origin:      ride.Origin,
				Destination: ride.Destination,
				DepartureAt: ride.DepartureAt,
			}, nil
		default:
			return ContactInfo{}, ErrUnauthorized
		}

	default:
		return ContactInfo{}, errors.New("interest not yet accepted")
	}
}
