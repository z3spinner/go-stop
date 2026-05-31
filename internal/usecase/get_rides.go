package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type GetRides struct {
	rides repository.RideRepository
}

func NewGetRides(rides repository.RideRepository) *GetRides {
	return &GetRides{rides: rides}
}

func (uc *GetRides) Execute() ([]domain.Ride, error) {
	result, err := uc.rides.FindAll()
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []domain.Ride{}, nil
	}
	return result, nil
}
