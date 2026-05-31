package usecase

import "github.com/z3spinner/go-stop/internal/boundaries/repository"

type DeleteRide struct {
	rides repository.RideRepository
}

func NewDeleteRide(rides repository.RideRepository) *DeleteRide {
	return &DeleteRide{rides: rides}
}

func (uc *DeleteRide) Execute(id, phone string) error {
	ride, err := uc.rides.FindByID(id)
	if err != nil {
		return err
	}
	if ride.Phone != phone {
		return ErrUnauthorized
	}
	return uc.rides.Delete(id)
}
