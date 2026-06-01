package repository

import (
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

type RideRepository interface {
	Save(ride domain.Ride) error
	FindByID(id string) (domain.Ride, error)
	FindAll() ([]domain.Ride, error)
	FindByPhone(phone string) ([]domain.Ride, error)
	FindByOriginAndDestination(origin, destination string) ([]domain.Ride, error)
	FindByOriginDestinationAndDate(origin, destination string, date time.Time) ([]domain.Ride, error)
	FindByOriginDestinationDateTime(origin, destination string, departureAt time.Time, toleranceMins int) ([]domain.Ride, error)
	FindMatching(request domain.Request) ([]domain.Ride, error)
	FindPendingFeedback() ([]domain.Ride, error)
	Delete(id string) error
	DeleteExpired() error
	SetFeedbackGiven(id string) error
}
