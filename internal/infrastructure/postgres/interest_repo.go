package postgres

import (
	"context"
	"errors"

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
