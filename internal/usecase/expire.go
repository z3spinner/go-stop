package usecase

import "github.com/z3spinner/go-stop/internal/boundaries/repository"

type ExpireRides struct {
	rides repository.RideRepository
}

func NewExpireRides(rides repository.RideRepository) *ExpireRides {
	return &ExpireRides{rides: rides}
}

func (uc *ExpireRides) Execute() error {
	return uc.rides.DeleteExpired()
}

type ExpireRequests struct {
	requests repository.RequestRepository
}

func NewExpireRequests(requests repository.RequestRepository) *ExpireRequests {
	return &ExpireRequests{requests: requests}
}

func (uc *ExpireRequests) Execute() error {
	return uc.requests.DeleteExpired()
}
