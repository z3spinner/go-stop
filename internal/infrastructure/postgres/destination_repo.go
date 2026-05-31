package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DestinationRepo struct{ pool *pgxpool.Pool }

func NewDestinationRepo(pool *pgxpool.Pool) *DestinationRepo { return &DestinationRepo{pool: pool} }

// GetAll returns known locations sorted by popularity (most-used first).
// Combines active rides/requests with historical ride_stats so locations
// persist after rides expire and popular routes surface at the top of dropdowns.
func (r *DestinationRepo) GetAll() ([]string, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT location FROM (
		   SELECT origin      AS location FROM rides
		   UNION ALL SELECT destination FROM rides
		   UNION ALL SELECT origin      FROM requests
		   UNION ALL SELECT destination FROM requests
		   UNION ALL SELECT origin      FROM ride_stats
		   UNION ALL SELECT destination FROM ride_stats
		 ) all_locs
		 GROUP BY location
		 ORDER BY COUNT(*) DESC, location ASC`)
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
