package usecase

import "github.com/z3spinner/go-stop/internal/boundaries/repository"

type GetDestinations struct {
	destinations repository.DestinationRepository
}

func NewGetDestinations(destinations repository.DestinationRepository) *GetDestinations {
	return &GetDestinations{destinations: destinations}
}

func (uc *GetDestinations) Execute() ([]string, error) {
	return uc.destinations.GetAll()
}
