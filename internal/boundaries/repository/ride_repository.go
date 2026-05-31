package repository

import "github.com/z3spinner/go-stop/internal/domain"

type RideRepository interface {
	Save(ride domain.Ride) error
	FindByID(id string) (domain.Ride, error)
	FindAll() ([]domain.Ride, error)
	FindByPhone(phone string) ([]domain.Ride, error)
	FindByOriginAndDestination(origin, destination string) ([]domain.Ride, error)
	FindMatching(request domain.Request) ([]domain.Ride, error)
	Delete(id string) error
	DeleteExpired() error
}
