package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
)

type RideRepo struct{ pool *pgxpool.Pool }

func NewRideRepo(pool *pgxpool.Pool) *RideRepo { return &RideRepo{pool: pool} }

func (r *RideRepo) Save(ride domain.Ride) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO rides (id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		ride.ID, ride.DriverName, ride.Phone, ride.Origin, ride.Destination,
		ride.Date, ride.DepartureAt, int(ride.Flexibility), ride.PostedAt, ride.ExpiresAt,
	)
	return err
}

func (r *RideRepo) FindByID(id string) (domain.Ride, error) {
	row := r.pool.QueryRow(context.Background(),
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM rides WHERE id = $1`, id)
	return scanRide(row)
}

func (r *RideRepo) FindAll() ([]domain.Ride, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM rides WHERE expires_at > NOW() ORDER BY departure_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRides(rows)
}

func (r *RideRepo) FindByPhone(phone string) ([]domain.Ride, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM rides WHERE phone = $1 AND expires_at > NOW() ORDER BY departure_at ASC`,
		phone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRides(rows)
}

func (r *RideRepo) FindByOriginAndDestination(origin, destination string) ([]domain.Ride, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM rides WHERE origin = $1 AND destination = $2 AND expires_at > NOW() ORDER BY departure_at ASC`,
		origin, destination)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRides(rows)
}

func (r *RideRepo) FindMatching(req domain.Request) ([]domain.Ride, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM rides
		 WHERE origin = $1 AND destination = $2 AND date = $3 AND expires_at > NOW()
		   AND (departure_at - (flexibility * interval '1 minute')) <= ($4::timestamptz + ($5 * interval '1 minute'))
		   AND (departure_at + (flexibility * interval '1 minute')) >= ($4::timestamptz - ($5 * interval '1 minute'))`,
		req.Origin, req.Destination, req.Date, req.DepartureAt, int(req.Flexibility))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRides(rows)
}

func (r *RideRepo) Delete(id string) error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM rides WHERE id = $1`, id)
	return err
}

func (r *RideRepo) DeleteExpired() error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM rides WHERE expires_at < NOW()`)
	return err
}

func scanRide(row pgx.Row) (domain.Ride, error) {
	var ride domain.Ride
	var flex int
	err := row.Scan(&ride.ID, &ride.DriverName, &ride.Phone, &ride.Origin, &ride.Destination,
		&ride.Date, &ride.DepartureAt, &flex, &ride.PostedAt, &ride.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Ride{}, errors.New("ride not found")
		}
		return domain.Ride{}, err
	}
	ride.Flexibility = domain.Flexibility(flex)
	return ride, nil
}

func collectRides(rows pgx.Rows) ([]domain.Ride, error) {
	var rides []domain.Ride
	for rows.Next() {
		var ride domain.Ride
		var flex int
		if err := rows.Scan(&ride.ID, &ride.DriverName, &ride.Phone, &ride.Origin, &ride.Destination,
			&ride.Date, &ride.DepartureAt, &flex, &ride.PostedAt, &ride.ExpiresAt); err != nil {
			return nil, err
		}
		ride.Flexibility = domain.Flexibility(flex)
		rides = append(rides, ride)
	}
	if rides == nil {
		rides = []domain.Ride{}
	}
	return rides, rows.Err()
}
