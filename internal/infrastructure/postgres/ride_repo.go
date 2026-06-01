package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
)

type RideRepo struct {
	pool      *pgxpool.Pool
	graceMins int // rides hidden from public listings once their flex window + this many minutes has passed
}

func NewRideRepo(pool *pgxpool.Pool, graceMins int) *RideRepo {
	return &RideRepo{pool: pool, graceMins: graceMins}
}

// graceClause returns a SQL fragment that hides rides whose departure window
// ended more than graceMins ago. Safe to embed: graceMins is a server-side integer.
func (r *RideRepo) graceClause() string {
	return fmt.Sprintf(
		"AND departure_at + (flexibility * interval '1 minute') + interval '%d minutes' > NOW()",
		r.graceMins,
	)
}

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
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
		 FROM rides WHERE id = $1`, id)
	return scanRide(row)
}

func (r *RideRepo) FindAll() ([]domain.Ride, error) {
	rows, err := r.pool.Query(context.Background(),
fmt.Sprintf(`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
		 FROM rides
		 WHERE expires_at > NOW()
		   %s
		 ORDER BY departure_at ASC`, r.graceClause()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRides(rows)
}

func (r *RideRepo) FindByPhone(phone string) ([]domain.Ride, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
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
fmt.Sprintf(`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
		 FROM rides WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2) AND expires_at > NOW()
		   %s ORDER BY departure_at ASC`, r.graceClause()),
		origin, destination)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRides(rows)
}

func (r *RideRepo) FindMatching(req domain.Request) ([]domain.Ride, error) {
	var rows pgx.Rows
	var err error
	switch {
	case req.Date.IsZero(): // anytime — match any active ride on this route
		rows, err = r.pool.Query(context.Background(),
			`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
			 FROM rides
			 WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2) AND expires_at > NOW()`,
			req.Origin, req.Destination)
	case req.DepartureAt.IsZero(): // day — match any ride on the given date
		rows, err = r.pool.Query(context.Background(),
			`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
			 FROM rides
			 WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2) AND date = $3 AND expires_at > NOW()`,
			req.Origin, req.Destination, req.Date)
	default: // specific time window
		rows, err = r.pool.Query(context.Background(),
			`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
			 FROM rides
			 WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2) AND date = $3 AND expires_at > NOW()
			   AND (departure_at - (flexibility * interval '1 minute')) <= ($4::timestamptz + ($5 * interval '1 minute'))
			   AND (departure_at + (flexibility * interval '1 minute')) >= ($4::timestamptz - ($5 * interval '1 minute'))`,
			req.Origin, req.Destination, req.Date, req.DepartureAt, int(req.Flexibility))
	}
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

func (r *RideRepo) FindPendingFeedback() ([]domain.Ride, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, driver_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at, feedback_given
		 FROM rides
		 WHERE departure_at BETWEEN (NOW() - INTERVAL '23 hours') AND (NOW() - INTERVAL '30 minutes')
		   AND feedback_given = false
		   AND expires_at > NOW()
		 ORDER BY departure_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRides(rows)
}

func (r *RideRepo) SetFeedbackGiven(id string) error {
	_, err := r.pool.Exec(context.Background(),
		`UPDATE rides SET feedback_given = true WHERE id = $1`, id)
	return err
}

func scanRide(row pgx.Row) (domain.Ride, error) {
	var ride domain.Ride
	var flex int
	err := row.Scan(&ride.ID, &ride.DriverName, &ride.Phone, &ride.Origin, &ride.Destination,
		&ride.Date, &ride.DepartureAt, &flex, &ride.PostedAt, &ride.ExpiresAt, &ride.FeedbackGiven)
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
			&ride.Date, &ride.DepartureAt, &flex, &ride.PostedAt, &ride.ExpiresAt, &ride.FeedbackGiven); err != nil {
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
