package postgres

import (
	"context"
	"errors"
	"time"

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

func (r *RideRepo) Save(ride domain.Ride) error {
	return r.q.InsertRide(context.Background(), queries.InsertRideParams{
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
		Lower:        origin,
		Lower_2:      destination,
		GraceMinutes: r.graceMins,
	})
	if err != nil {
		return nil, err
	}
	return ridesFromRows(rows), nil
}

func (r *RideRepo) FindByOriginDestinationAndDate(origin, destination string, date time.Time) ([]domain.Ride, error) {
	rows, err := r.q.SearchRidesByDate(context.Background(), queries.SearchRidesByDateParams{
		Lower:        origin,
		Lower_2:      destination,
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
		Lower:                  origin,
		Lower_2:                destination,
		Column3:                tsFrom(searchTime),
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
		Lower:                  origin,
		Lower_2:                destination,
		Date:                   dateFrom(date),
		Column4:                tsFrom(departureAt),
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
				Lower: req.Origin, Lower_2: req.Destination,
			})
	case req.Date.IsZero() && !req.DepartureAt.IsZero():
		rows, err = r.q.FindRidesMatchingDailyRequest(context.Background(),
			queries.FindRidesMatchingDailyRequestParams{
				Lower: req.Origin, Lower_2: req.Destination,
				Column3: tsFrom(req.DepartureAt),
				Column4: int32(req.Flexibility),
			})
	case req.DepartureAt.IsZero():
		rows, err = r.q.FindRidesMatchingDayRequest(context.Background(),
			queries.FindRidesMatchingDayRequestParams{
				Lower: req.Origin, Lower_2: req.Destination,
				Date: dateFrom(req.Date),
			})
	default:
		rows, err = r.q.FindRidesMatchingTimeRequest(context.Background(),
			queries.FindRidesMatchingTimeRequestParams{
				Lower: req.Origin, Lower_2: req.Destination,
				Date:    dateFrom(req.Date),
				Column4: tsFrom(req.DepartureAt),
				Column5: int32(req.Flexibility),
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

func (r *RideRepo) FindPendingFeedback() ([]domain.Ride, error) {
	rows, err := r.q.ListRidesPendingFeedback(context.Background())
	if err != nil {
		return nil, err
	}
	return ridesFromRows(rows), nil
}

func (r *RideRepo) SetFeedbackGiven(id string) error {
	return r.q.SetRideFeedbackGiven(context.Background(), uuidFrom(id))
}
