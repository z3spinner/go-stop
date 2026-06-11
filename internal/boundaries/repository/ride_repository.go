// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

import (
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

type RideRepository interface {
	// Save upserts the ride on its dedup key (phone + normalized driver name +
	// normalized route + exact departure instant). A new ride is inserted and
	// returned with created=true. A re-post of an existing ride refreshes its
	// mutable non-key fields (driver name, route display, flexibility) and
	// returns the canonical row with created=false; id, posted_at and
	// feedback_given are preserved.
	Save(ride domain.Ride) (saved domain.Ride, created bool, err error)
	// UpdateByID edits a ride in place by its id (route, departure time,
	// derived date/expiry, flexibility), preserving driver_name, phone,
	// posted_at and feedback_given. Returns ErrDuplicateRide if the edit
	// collides with another of the driver's rides on the dedup key.
	UpdateByID(ride domain.Ride) (domain.Ride, error)
	FindByID(id string) (domain.Ride, error)
	FindAll() ([]domain.Ride, error)
	FindByPhone(phone string) ([]domain.Ride, error)
	FindByOriginAndDestination(origin, destination string) ([]domain.Ride, error)
	// FindByOriginAndDestinationFuzzy is a trigram-based fallback for typos and
	// spelling variants; used only when an exact route search returns nothing.
	FindByOriginAndDestinationFuzzy(origin, destination string) ([]domain.Ride, error)
	FindByOriginDestinationAndDate(origin, destination string, date time.Time) ([]domain.Ride, error)
	FindByOriginDestinationDateTime(origin, destination string, departureAt time.Time, toleranceMins int) ([]domain.Ride, error)
	FindByOriginAndTime(origin, destination string, searchTime time.Time, toleranceMins int) ([]domain.Ride, error)
	FindMatching(request domain.Request) ([]domain.Ride, error)
	Delete(id string) error
	DeleteExpired() error
	// ClaimFeedback flips feedback_given false→true and reports whether this call
	// performed the flip (true = caller won the claim and should record the stat).
	ClaimFeedback(id string) (bool, error)
}
