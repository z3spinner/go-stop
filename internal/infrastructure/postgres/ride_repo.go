// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type RideRepo struct {
	q         *queries.Queries
	graceMins int32
}

func NewRideRepo(pool *pgxpool.Pool, graceMins int) *RideRepo {
	return &RideRepo{q: queries.New(pool), graceMins: int32(graceMins)}
}

// Save inserts the ride idempotently. If an identical ride already exists — same
// phone, normalized driver name, normalized route and exact departure instant
// (the uq_rides_dedup index) — no row is inserted; the existing ride is returned
// with created=false so the caller can skip re-notifying searchers.
func (r *RideRepo) Save(ride domain.Ride) (domain.Ride, bool, error) {
	_, err := r.q.InsertRide(context.Background(), queries.InsertRideParams{
		ID:          uuidFrom(ride.ID),
		DriverName:  ride.DriverName,
		Phone:       ride.Phone,
		Origin:      ride.Origin,
		Destination: ride.Destination,
		Date:        dateFrom(ride.Date),
		DepartureAt: tsFrom(ride.DepartureAt),
		Flexibility: int32(ride.Flexibility),
		PostedAt:    tsFrom(ride.PostedAt),
		ExpiresAt:   tsFrom(ride.ExpiresAt),
	})
	if err == nil {
		return ride, true, nil
	}
	// ON CONFLICT DO NOTHING returns no rows on a duplicate; anything else is a
	// real error.
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Ride{}, false, err
	}
	existing, err := r.q.GetRideByDedupKey(context.Background(), queries.GetRideByDedupKeyParams{
		Phone:       ride.Phone,
		DriverName:  ride.DriverName,
		Origin:      ride.Origin,
		Destination: ride.Destination,
		DepartureAt: tsFrom(ride.DepartureAt),
	})
	if err != nil {
		return domain.Ride{}, false, err
	}
	return rideFromRow(existing), false, nil
}

func (r *RideRepo) FindByID(id string) (domain.Ride, error) {
	row, err := r.q.GetRideByID(context.Background(), uuidFrom(id))
	if err != nil {
		return domain.Ride{}, errors.New("ride not found")
	}
	return rideFromRow(row), nil
}

func (r *RideRepo) FindAll() ([]domain.Ride, error) {
	rows, err := r.q.ListRidesActive(context.Background(), r.graceMins)
	if err != nil {
		return nil, err
	}
	return ridesFromRows(rows), nil
}

func (r *RideRepo) FindByPhone(phone string) ([]domain.Ride, error) {
	rows, err := r.q.ListRidesByPhone(context.Background(), phone)
	if err != nil {
		return nil, err
	}
	return ridesFromRows(rows), nil
}

func (r *RideRepo) FindByOriginAndDestination(origin, destination string) ([]domain.Ride, error) {
	rows, err := r.q.SearchRides(context.Background(), queries.SearchRidesParams{
		Origin:       origin,
		Destination:  destination,
		GraceMinutes: r.graceMins,
	})
	if err != nil {
		return nil, err
	}
	return ridesFromRows(rows), nil
}

func (r *RideRepo) FindByOriginAndDestinationFuzzy(origin, destination string) ([]domain.Ride, error) {
	rows, err := r.q.SearchRidesFuzzy(context.Background(), queries.SearchRidesFuzzyParams{
		Origin:       origin,
		Destination:  destination,
		GraceMinutes: r.graceMins,
	})
	if err != nil {
		return nil, err
	}
	return ridesFromRows(rows), nil
}

func (r *RideRepo) FindByOriginDestinationAndDate(origin, destination string, date time.Time) ([]domain.Ride, error) {
	rows, err := r.q.SearchRidesByDate(context.Background(), queries.SearchRidesByDateParams{
		Origin:       origin,
		Destination:  destination,
		Date:         dateFrom(date),
		GraceMinutes: r.graceMins,
	})
	if err != nil {
		return nil, err
	}
	return ridesFromRows(rows), nil
}

func (r *RideRepo) FindByOriginAndTime(origin, destination string, searchTime time.Time, toleranceMins int) ([]domain.Ride, error) {
	rows, err := r.q.SearchRidesByTime(context.Background(), queries.SearchRidesByTimeParams{
		Origin:                 origin,
		Destination:            destination,
		SearchTime:             tsFrom(searchTime),
		GraceMinutes:           r.graceMins,
		SearchToleranceMinutes: int32(toleranceMins),
	})
	if err != nil {
		return nil, err
	}
	return ridesFromRows(rows), nil
}

func (r *RideRepo) FindByOriginDestinationDateTime(origin, destination string, departureAt time.Time, toleranceMins int) ([]domain.Ride, error) {
	date := time.Date(departureAt.Year(), departureAt.Month(), departureAt.Day(), 0, 0, 0, 0, departureAt.Location())
	rows, err := r.q.SearchRidesByDateTime(context.Background(), queries.SearchRidesByDateTimeParams{
		Origin:                 origin,
		Destination:            destination,
		Date:                   dateFrom(date),
		SearchTime:             tsFrom(departureAt),
		GraceMinutes:           r.graceMins,
		SearchToleranceMinutes: int32(toleranceMins),
	})
	if err != nil {
		return nil, err
	}
	return ridesFromRows(rows), nil
}

func (r *RideRepo) FindMatching(req domain.Request) ([]domain.Ride, error) {
	var rows []queries.Ride
	var err error
	switch {
	case req.Date.IsZero() && req.DepartureAt.IsZero():
		rows, err = r.q.FindRidesMatchingAnytimeRequest(context.Background(),
			queries.FindRidesMatchingAnytimeRequestParams{
				Origin: req.Origin, Destination: req.Destination,
			})
	case req.Date.IsZero() && !req.DepartureAt.IsZero():
		rows, err = r.q.FindRidesMatchingDailyRequest(context.Background(),
			queries.FindRidesMatchingDailyRequestParams{
				Origin: req.Origin, Destination: req.Destination,
				DepartureAt:   tsFrom(req.DepartureAt),
				WindowMinutes: int32(req.Flexibility),
			})
	case req.DepartureAt.IsZero():
		rows, err = r.q.FindRidesMatchingDayRequest(context.Background(),
			queries.FindRidesMatchingDayRequestParams{
				Origin: req.Origin, Destination: req.Destination,
				Date: dateFrom(req.Date),
			})
	default:
		rows, err = r.q.FindRidesMatchingTimeRequest(context.Background(),
			queries.FindRidesMatchingTimeRequestParams{
				Origin: req.Origin, Destination: req.Destination,
				Date:          dateFrom(req.Date),
				DepartureAt:   tsFrom(req.DepartureAt),
				WindowMinutes: int32(req.Flexibility),
			})
	}
	if err != nil {
		return nil, err
	}
	return ridesFromRows(rows), nil
}

func (r *RideRepo) Delete(id string) error {
	return r.q.DeleteRide(context.Background(), uuidFrom(id))
}

func (r *RideRepo) DeleteExpired() error {
	return r.q.DeleteExpiredRides(context.Background())
}

func (r *RideRepo) ClaimFeedback(id string) (bool, error) {
	n, err := r.q.ClaimRideFeedback(context.Background(), uuidFrom(id))
	return n == 1, err
}
