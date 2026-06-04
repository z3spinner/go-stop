package usecase

import "github.com/z3spinner/go-stop/internal/boundaries/repository"

type DeleteRide struct {
	rides      repository.RideRepository
	notifQueue repository.NotificationQueueRepository
}

func NewDeleteRide(rides repository.RideRepository, notifQueue repository.NotificationQueueRepository) *DeleteRide {
	return &DeleteRide{rides: rides, notifQueue: notifQueue}
}

func (uc *DeleteRide) Execute(id, phone string) error {
	ride, err := uc.rides.FindByID(id)
	if err != nil {
		return err
	}
	if ride.Phone != phone {
		return ErrUnauthorized
	}
	if err := uc.rides.Delete(id); err != nil {
		return err
	}
	_ = uc.notifQueue.DeleteForRide(id) // best-effort cleanup
	return nil
}
