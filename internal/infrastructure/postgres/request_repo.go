package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
)

type RequestRepo struct{ pool *pgxpool.Pool }

func NewRequestRepo(pool *pgxpool.Pool) *RequestRepo { return &RequestRepo{pool: pool} }

func scanRequest(scan func(...any) error) (domain.Request, error) {
	var req domain.Request
	var flex int
	var date, deptAt *time.Time
	err := scan(&req.ID, &req.SearcherName, &req.Phone, &req.Origin, &req.Destination,
		&date, &deptAt, &flex, &req.PostedAt, &req.ExpiresAt)
	if err != nil {
		return domain.Request{}, err
	}
	if date != nil {
		req.Date = *date
	}
	if deptAt != nil {
		req.DepartureAt = *deptAt
	}
	req.Flexibility = domain.Flexibility(flex)
	return req, nil
}

func (r *RequestRepo) Save(req domain.Request) error {
	var date, deptAt interface{}
	if !req.Date.IsZero() {
		date = req.Date
	}
	if !req.DepartureAt.IsZero() {
		deptAt = req.DepartureAt
	}
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO requests (id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		req.ID, req.SearcherName, req.Phone, req.Origin, req.Destination,
		date, deptAt, int(req.Flexibility), req.PostedAt, req.ExpiresAt,
	)
	return err
}

func (r *RequestRepo) FindByID(id string) (domain.Request, error) {
	row := r.pool.QueryRow(context.Background(),
		`SELECT id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM requests WHERE id = $1`, id)
	req, err := scanRequest(row.Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Request{}, errors.New("request not found")
		}
		return domain.Request{}, err
	}
	return req, nil
}

func (r *RequestRepo) FindByPhone(phone string) ([]domain.Request, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM requests WHERE phone = $1 AND expires_at > NOW()
		 ORDER BY COALESCE(departure_at, date, expires_at) ASC`,
		phone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reqs []domain.Request
	for rows.Next() {
		req, err := scanRequest(rows.Scan)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if reqs == nil {
		reqs = []domain.Request{}
	}
	return reqs, rows.Err()
}

func (r *RequestRepo) FindMatching(ride domain.Ride) ([]domain.Request, error) {
	// Three alert modes inferred from NULL state of date/departure_at:
	//   anytime:  date IS NULL (matches any ride on this route)
	//   day:      date set, departure_at IS NULL (matches any time on ride's date)
	//   time:     both set (matches overlapping time window)
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM requests
		 WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2)
		   AND expires_at > NOW()
		   AND (
		     (date IS NULL AND departure_at IS NULL)
		     OR (date IS NULL AND departure_at IS NOT NULL
		         AND (departure_at::time - (flexibility * interval '1 minute')) <= ($4::timestamptz::time + ($5 * interval '1 minute'))
		         AND (departure_at::time + (flexibility * interval '1 minute')) >= ($4::timestamptz::time - ($5 * interval '1 minute')))
		     OR (date = $3 AND departure_at IS NULL)
		     OR (date = $3
		         AND (departure_at - (flexibility * interval '1 minute')) <= ($4::timestamptz + ($5 * interval '1 minute'))
		         AND (departure_at + (flexibility * interval '1 minute')) >= ($4::timestamptz - ($5 * interval '1 minute')))
		   )`,
		ride.Origin, ride.Destination, ride.Date, ride.DepartureAt, int(ride.Flexibility))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reqs []domain.Request
	for rows.Next() {
		req, err := scanRequest(rows.Scan)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if reqs == nil {
		reqs = []domain.Request{}
	}
	return reqs, rows.Err()
}

func (r *RequestRepo) Delete(id string) error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM requests WHERE id = $1`, id)
	return err
}

func (r *RequestRepo) DeleteExpired() error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM requests WHERE expires_at < NOW()`)
	return err
}
