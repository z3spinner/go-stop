package usecase

import "github.com/z3spinner/go-stop/internal/boundaries/repository"

type DeleteRequest struct {
	requests repository.RequestRepository
}

func NewDeleteRequest(requests repository.RequestRepository) *DeleteRequest {
	return &DeleteRequest{requests: requests}
}

func (uc *DeleteRequest) Execute(id, phone string) error {
	req, err := uc.requests.FindByID(id)
	if err != nil {
		return err
	}
	if req.Phone != phone {
		return ErrUnauthorized
	}
	return uc.requests.Delete(id)
}
