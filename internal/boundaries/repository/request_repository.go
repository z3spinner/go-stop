package repository

import "github.com/z3spinner/go-stop/internal/domain"

type RequestRepository interface {
	Save(request domain.Request) error
	FindByID(id string) (domain.Request, error)
	FindByPhone(phone string) ([]domain.Request, error)
	FindMatching(ride domain.Ride) ([]domain.Request, error)
	Delete(id string) error
	DeleteExpired() error
}
