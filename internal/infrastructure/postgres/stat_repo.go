package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type StatRepo struct{ q *queries.Queries }

func NewStatRepo(pool *pgxpool.Pool) *StatRepo { return &StatRepo{q: queries.New(pool)} }

func (r *StatRepo) Save(origin, destination string, rideDate time.Time, taken bool) error {
	return r.q.InsertRideStat(context.Background(), queries.InsertRideStatParams{
		Origin:      origin,
		Destination: destination,
		RideDate:    dateFrom(rideDate),
		Taken:       taken,
	})
}

func (r *StatRepo) RecordSearch(origin, destination string) error {
	return r.q.InsertSearchEvent(context.Background(), queries.InsertSearchEventParams{
		Origin:      origin,
		Destination: destination,
	})
}

func (r *StatRepo) RecordRide(origin, destination string) error {
	return r.q.InsertRideEvent(context.Background(), queries.InsertRideEventParams{
		Origin:      origin,
		Destination: destination,
	})
}

func (r *StatRepo) GetStats() (domain.Stats, error) {
	ctx := context.Background()

	topRows, err := r.q.GetTopRoutes(ctx)
	if err != nil {
		return domain.Stats{}, err
	}
	topRoutes := make([]domain.RouteStat, len(topRows))
	for i, row := range topRows {
		topRoutes[i] = domain.RouteStat{Origin: row.Origin, Destination: row.Destination, Count: int(row.Count)}
	}

	totals, err := r.q.GetRideStatsTotals(ctx)
	if err != nil {
		return domain.Stats{}, err
	}

	searchCounts, err := r.q.GetSearchEventCounts(ctx)
	if err != nil {
		return domain.Stats{}, err
	}

	rideCounts, err := r.q.GetRideEventCounts(ctx)
	if err != nil {
		return domain.Stats{}, err
	}

	return domain.Stats{
		TopRoutes:      topRoutes,
		TotalConfirmed: int(totals.TotalConfirmed),
		TotalThisWeek:  int(totals.TotalThisWeek),
		Searches: domain.ActivityCounts{
			AllTime:   int(searchCounts.AllTime),
			ThisYear:  int(searchCounts.ThisYear),
			ThisMonth: int(searchCounts.ThisMonth),
		},
		RidesPosted: domain.ActivityCounts{
			AllTime:   int(rideCounts.AllTime),
			ThisYear:  int(rideCounts.ThisYear),
			ThisMonth: int(rideCounts.ThisMonth),
		},
	}, nil
}
