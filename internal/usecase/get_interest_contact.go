package usecase

import (
	"errors"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

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

func (uc *GetInterestContact) Execute(interestID, requesterPhone string) (string, error) {
	interest, err := uc.interests.FindByID(interestID)
	if err != nil {
		return "", err
	}
	if interest.Status != "accepted" {
		return "", errors.New("interest not yet accepted")
	}
	ride, err := uc.rides.FindByID(interest.RideID)
	if err != nil {
		return "", err
	}
	switch requesterPhone {
	case ride.Phone:
		return interest.SearcherPhone, nil
	case interest.SearcherPhone:
		return ride.Phone, nil
	default:
		return "", ErrUnauthorized
	}
}
