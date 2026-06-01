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
	if interest.Status != "accepted" {
		return ContactInfo{}, errors.New("interest not yet accepted")
	}
	ride, err := uc.rides.FindByID(interest.RideID)
	if err != nil {
		return ContactInfo{}, err
	}
	switch requesterPhone {
	case ride.Phone:
		// Driver is asking — return searcher's info
		return ContactInfo{
			Phone:       interest.SearcherPhone,
			Name:        interest.SearcherName,
			Role:        "searcher",
			Origin:      ride.Origin,
			Destination: ride.Destination,
			DepartureAt: ride.DepartureAt,
		}, nil
	case interest.SearcherPhone:
		// Searcher is asking — return driver's info
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
}
