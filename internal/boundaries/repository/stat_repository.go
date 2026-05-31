package repository

import (
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

type StatRepository interface {
	Save(origin, destination string, rideDate time.Time, taken bool) error
	GetStats() (domain.Stats, error)
}
