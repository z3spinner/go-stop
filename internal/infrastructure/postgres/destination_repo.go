package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DestinationRepo struct{ pool *pgxpool.Pool }

func NewDestinationRepo(pool *pgxpool.Pool) *DestinationRepo { return &DestinationRepo{pool: pool} }

func (r *DestinationRepo) GetAll() ([]string, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT DISTINCT origin AS location FROM rides
		 UNION SELECT DISTINCT destination FROM rides
		 UNION SELECT DISTINCT origin FROM requests
		 UNION SELECT DISTINCT destination FROM requests
		 ORDER BY location`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var locations []string
	for rows.Next() {
		var loc string
		if err := rows.Scan(&loc); err != nil {
			return nil, err
		}
		locations = append(locations, loc)
	}
	if locations == nil {
		locations = []string{}
	}
	return locations, rows.Err()
}
