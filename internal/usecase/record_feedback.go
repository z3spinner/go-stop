package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

type RecordFeedback struct {
	rides repository.RideRepository
	stats repository.StatRepository
}

func NewRecordFeedback(rides repository.RideRepository, stats repository.StatRepository) *RecordFeedback {
	return &RecordFeedback{rides: rides, stats: stats}
}

func (uc *RecordFeedback) Execute(rideID, phone string, taken bool) error {
	ride, err := uc.rides.FindByID(rideID)
	if err != nil {
		return err
	}
	if ride.Phone != phone {
		return ErrUnauthorized
	}
	if err := uc.stats.Save(ride.Origin, ride.Destination, ride.Date, taken); err != nil {
		return err
	}
	return uc.rides.SetFeedbackGiven(rideID)
}
