package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type InterestRepo struct{ q *queries.Queries }

func NewInterestRepo(pool *pgxpool.Pool) *InterestRepo {
	return &InterestRepo{q: queries.New(pool)}
}

func (r *InterestRepo) Save(i domain.Interest) error {
	return r.q.InsertInterest(context.Background(), queries.InsertInterestParams{
		ID:            uuidFrom(i.ID),
		RideID:        uuidFrom(i.RideID),
		SearcherPhone: i.SearcherPhone,
		SearcherName:  i.SearcherName,
		Status:        i.Status,
	})
}

func (r *InterestRepo) FindByID(id string) (domain.Interest, error) {
	row, err := r.q.GetInterestByID(context.Background(), uuidFrom(id))
	if err != nil {
		return domain.Interest{}, errors.New("interest not found")
	}
	return interestFromRow(row), nil
}

func (r *InterestRepo) FindByRideAndSearcher(rideID, searcherPhone string) (domain.Interest, error) {
	row, err := r.q.GetInterestByRideAndSearcher(context.Background(), queries.GetInterestByRideAndSearcherParams{
		RideID:        uuidFrom(rideID),
		SearcherPhone: searcherPhone,
	})
	if err != nil {
		return domain.Interest{}, errors.New("interest not found")
	}
	return interestFromRow(row), nil
}

func (r *InterestRepo) FindByRide(rideID string) ([]domain.Interest, error) {
	rows, err := r.q.ListInterestsByRide(context.Background(), uuidFrom(rideID))
	if err != nil {
		return nil, err
	}
	return interestsFromRows(rows), nil
}

func (r *InterestRepo) Accept(id string) error {
	return r.q.AcceptInterest(context.Background(), uuidFrom(id))
}

func (r *InterestRepo) FindBySearcherPhone(phone string) ([]domain.InterestWithRide, error) {
	rows, err := r.q.ListInterestsBySearcher(context.Background(), phone)
	if err != nil {
		return nil, err
	}
	out := make([]domain.InterestWithRide, len(rows))
	for i, row := range rows {
		out[i] = domain.InterestWithRide{
			Interest: domain.Interest{
				ID:            uuidTo(row.ID),
				RideID:        uuidTo(row.RideID),
				SearcherPhone: row.SearcherPhone,
				SearcherName:  row.SearcherName,
				Status:        row.Status,
				CreatedAt:     tsTo(row.CreatedAt),
			},
			Origin:      row.Origin,
			Destination: row.Destination,
			DepartureAt: tsTo(row.DepartureAt),
			DriverName:  row.DriverName,
		}
	}
	return out, nil
}

func (r *InterestRepo) CountByRides(rideIDs []string) (map[string]int, error) {
	uuids := make([]pgtype.UUID, len(rideIDs))
	for i, id := range rideIDs {
		uuids[i] = uuidFrom(id)
	}
	rows, err := r.q.CountInterestsByRides(context.Background(), uuids)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[uuidTo(row.RideID)] = int(row.Count)
	}
	return counts, nil
}
