package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
)

type InterestRepo struct{ pool *pgxpool.Pool }

func NewInterestRepo(pool *pgxpool.Pool) *InterestRepo { return &InterestRepo{pool: pool} }

func (r *InterestRepo) Save(i domain.Interest) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO interests (id, ride_id, searcher_phone, searcher_name, status)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (ride_id, searcher_phone) DO NOTHING`,
		i.ID, i.RideID, i.SearcherPhone, i.SearcherName, i.Status)
	return err
}

func (r *InterestRepo) FindByID(id string) (domain.Interest, error) {
	row := r.pool.QueryRow(context.Background(),
		`SELECT id, ride_id, searcher_phone, searcher_name, status, created_at FROM interests WHERE id = $1`, id)
	return scanInterest(row)
}

func (r *InterestRepo) FindByRideAndSearcher(rideID, searcherPhone string) (domain.Interest, error) {
	row := r.pool.QueryRow(context.Background(),
		`SELECT id, ride_id, searcher_phone, searcher_name, status, created_at
		 FROM interests WHERE ride_id = $1 AND searcher_phone = $2`,
		rideID, searcherPhone)
	return scanInterest(row)
}

func (r *InterestRepo) FindByRide(rideID string) ([]domain.Interest, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, ride_id, searcher_phone, searcher_name, status, created_at
		 FROM interests WHERE ride_id = $1 ORDER BY created_at ASC`, rideID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var interests []domain.Interest
	for rows.Next() {
		i, err := scanInterestRow(rows)
		if err != nil {
			return nil, err
		}
		interests = append(interests, i)
	}
	if interests == nil {
		interests = []domain.Interest{}
	}
	return interests, rows.Err()
}

func (r *InterestRepo) Accept(id string) error {
	_, err := r.pool.Exec(context.Background(),
		`UPDATE interests SET status = 'accepted' WHERE id = $1`, id)
	return err
}

func scanInterest(row pgx.Row) (domain.Interest, error) {
	var i domain.Interest
	err := row.Scan(&i.ID, &i.RideID, &i.SearcherPhone, &i.SearcherName, &i.Status, &i.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Interest{}, errors.New("interest not found")
		}
		return domain.Interest{}, err
	}
	return i, nil
}

func scanInterestRow(rows pgx.Rows) (domain.Interest, error) {
	var i domain.Interest
	err := rows.Scan(&i.ID, &i.RideID, &i.SearcherPhone, &i.SearcherName, &i.Status, &i.CreatedAt)
	return i, err
}
