package usecase

import (
	"log"

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
	// Logged explicitly so yes/no is visible in the logs — the HTTP access log
	// only shows the path, and `taken` travels in the request body.
	outcome := "drove_alone"
	if taken {
		outcome = "shared"
	}
	log.Printf("ride feedback ride=%s outcome=%s route=%q->%q", rideID, outcome, ride.Origin, ride.Destination)
	return uc.rides.SetFeedbackGiven(rideID)
}
