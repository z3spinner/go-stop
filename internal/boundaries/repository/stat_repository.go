package repository

import (
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

type StatRepository interface {
	Save(origin, destination string, rideDate time.Time, taken bool) error
	RecordSearch(origin, destination string) error
	RecordRide(origin, destination string) error
	GetStats() (domain.Stats, error)
}
