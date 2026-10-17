// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"errors"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
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
}

func NewGetInterestContact(
	interests repository.InterestRepository,
	rides repository.RideRepository,
) *GetInterestContact {
	return &GetInterestContact{interests: interests, rides: rides}
}

func (uc *GetInterestContact) Execute(interestID, requesterPhone string) (ContactInfo, error) {
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
