package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
)

type StatRepo struct{ pool *pgxpool.Pool }

func NewStatRepo(pool *pgxpool.Pool) *StatRepo { return &StatRepo{pool: pool} }

func (r *StatRepo) Save(origin, destination string, rideDate time.Time, taken bool) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO ride_stats (origin, destination, ride_date, taken) VALUES ($1, $2, $3, $4)`,
		origin, destination, rideDate, taken)
	return err
}

func (r *StatRepo) GetStats() (domain.Stats, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT origin, destination, COUNT(*) AS count
		 FROM ride_stats
		 WHERE taken = true
		   AND recorded_at >= DATE_TRUNC('week', NOW())
		 GROUP BY origin, destination
		 ORDER BY count DESC
		 LIMIT 5`)
	if err != nil {
		return domain.Stats{}, err
	}
	defer rows.Close()

	var topRoutes []domain.RouteStat
	for rows.Next() {
		var rs domain.RouteStat
		if err := rows.Scan(&rs.Origin, &rs.Destination, &rs.Count); err != nil {
			return domain.Stats{}, err
		}
		topRoutes = append(topRoutes, rs)
	}
	if err := rows.Err(); err != nil {
		return domain.Stats{}, err
	}
	if topRoutes == nil {
		topRoutes = []domain.RouteStat{}
	}

	var totalConfirmed, totalThisWeek int
	err = r.pool.QueryRow(context.Background(),
		`SELECT
		   COUNT(*) FILTER (WHERE taken = true) AS total_confirmed,
		   COUNT(*) FILTER (WHERE taken = true AND recorded_at >= DATE_TRUNC('week', NOW())) AS total_this_week
		 FROM ride_stats`).Scan(&totalConfirmed, &totalThisWeek)
	if err != nil {
		return domain.Stats{}, err
	}

	return domain.Stats{
		TopRoutes:      topRoutes,
		TotalConfirmed: totalConfirmed,
		TotalThisWeek:  totalThisWeek,
	}, nil
}
