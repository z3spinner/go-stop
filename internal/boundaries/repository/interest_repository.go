package repository

import "github.com/z3spinner/go-stop/internal/domain"

type InterestRepository interface {
	Save(interest domain.Interest) error
	FindByID(id string) (domain.Interest, error)
	FindByRideAndSearcher(rideID, searcherPhone string) (domain.Interest, error)
	FindByRide(rideID string) ([]domain.Interest, error)
	Accept(id string) error
}
