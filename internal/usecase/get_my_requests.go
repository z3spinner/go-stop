package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type GetMyRequests struct {
	requests repository.RequestRepository
}

func NewGetMyRequests(requests repository.RequestRepository) *GetMyRequests {
	return &GetMyRequests{requests: requests}
}

func (uc *GetMyRequests) Execute(phone string) ([]domain.Request, error) {
	result, err := uc.requests.FindByPhone(phone)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []domain.Request{}, nil
	}
	return result, nil
}
