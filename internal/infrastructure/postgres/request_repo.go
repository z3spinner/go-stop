package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
)

type RequestRepo struct{ pool *pgxpool.Pool }

func NewRequestRepo(pool *pgxpool.Pool) *RequestRepo { return &RequestRepo{pool: pool} }

func (r *RequestRepo) Save(req domain.Request) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO requests (id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		req.ID, req.SearcherName, req.Phone, req.Origin, req.Destination,
		req.Date, req.DepartureAt, int(req.Flexibility), req.PostedAt, req.ExpiresAt,
	)
	return err
}

func (r *RequestRepo) FindByID(id string) (domain.Request, error) {
	row := r.pool.QueryRow(context.Background(),
		`SELECT id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM requests WHERE id = $1`, id)
	var req domain.Request
	var flex int
	err := row.Scan(&req.ID, &req.SearcherName, &req.Phone, &req.Origin, &req.Destination,
		&req.Date, &req.DepartureAt, &flex, &req.PostedAt, &req.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Request{}, errors.New("request not found")
		}
		return domain.Request{}, err
	}
	req.Flexibility = domain.Flexibility(flex)
	return req, nil
}

func (r *RequestRepo) FindByPhone(phone string) ([]domain.Request, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM requests WHERE phone = $1 AND expires_at > NOW() ORDER BY departure_at ASC`,
		phone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reqs []domain.Request
	for rows.Next() {
		var req domain.Request
		var flex int
		if err := rows.Scan(&req.ID, &req.SearcherName, &req.Phone, &req.Origin, &req.Destination,
			&req.Date, &req.DepartureAt, &flex, &req.PostedAt, &req.ExpiresAt); err != nil {
			return nil, err
		}
		req.Flexibility = domain.Flexibility(flex)
		reqs = append(reqs, req)
	}
	if reqs == nil {
		reqs = []domain.Request{}
	}
	return reqs, rows.Err()
}

func (r *RequestRepo) FindMatching(ride domain.Ride) ([]domain.Request, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, searcher_name, phone, origin, destination, date, departure_at, flexibility, posted_at, expires_at
		 FROM requests
		 WHERE LOWER(origin) = LOWER($1) AND LOWER(destination) = LOWER($2) AND date = $3 AND expires_at > NOW()
		   AND (departure_at - (flexibility * interval '1 minute')) <= ($4::timestamptz + ($5 * interval '1 minute'))
		   AND (departure_at + (flexibility * interval '1 minute')) >= ($4::timestamptz - ($5 * interval '1 minute'))`,
		ride.Origin, ride.Destination, ride.Date, ride.DepartureAt, int(ride.Flexibility))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reqs []domain.Request
	for rows.Next() {
		var req domain.Request
		var flex int
		if err := rows.Scan(&req.ID, &req.SearcherName, &req.Phone, &req.Origin, &req.Destination,
			&req.Date, &req.DepartureAt, &flex, &req.PostedAt, &req.ExpiresAt); err != nil {
			return nil, err
		}
		req.Flexibility = domain.Flexibility(flex)
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
