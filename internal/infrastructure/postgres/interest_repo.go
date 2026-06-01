package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type InterestRepo struct {
	q      *queries.Queries
	crypto *PhoneCrypto
}

func NewInterestRepo(pool *pgxpool.Pool, crypto *PhoneCrypto) *InterestRepo {
	return &InterestRepo{q: queries.New(pool), crypto: crypto}
}

func (r *InterestRepo) decInterest(row queries.Interest) (domain.Interest, error) {
	i := interestFromRow(row)
	plain, err := r.crypto.Decrypt(i.SearcherPhone)
	if err != nil {
		return domain.Interest{}, err
	}
	i.SearcherPhone = plain
	return i, nil
}

func (r *InterestRepo) Save(i domain.Interest) error {
	enc, err := r.crypto.Encrypt(i.SearcherPhone)
	if err != nil {
		return err
	}
	return r.q.InsertInterest(context.Background(), queries.InsertInterestParams{
		ID:            uuidFrom(i.ID),
		RideID:        uuidFrom(i.RideID),
		SearcherPhone: enc,
		SearcherName:  i.SearcherName,
		Status:        i.Status,
	})
}

func (r *InterestRepo) FindByID(id string) (domain.Interest, error) {
	row, err := r.q.GetInterestByID(context.Background(), uuidFrom(id))
	if err != nil {
		return domain.Interest{}, errors.New("interest not found")
	}
	return r.decInterest(row)
}

func (r *InterestRepo) FindByRideAndSearcher(rideID, searcherPhone string) (domain.Interest, error) {
	enc, err := r.crypto.Encrypt(searcherPhone)
	if err != nil {
		return domain.Interest{}, err
	}
	row, err := r.q.GetInterestByRideAndSearcher(context.Background(), queries.GetInterestByRideAndSearcherParams{
		RideID:        uuidFrom(rideID),
		SearcherPhone: enc,
	})
	if err != nil {
		return domain.Interest{}, errors.New("interest not found")
	}
	return r.decInterest(row)
}

func (r *InterestRepo) FindByRide(rideID string) ([]domain.Interest, error) {
	rows, err := r.q.ListInterestsByRide(context.Background(), uuidFrom(rideID))
	if err != nil {
		return nil, err
	}
	out := make([]domain.Interest, len(rows))
	for i, row := range rows {
		interest, err := r.decInterest(row)
		if err != nil {
			return nil, err
		}
		out[i] = interest
	}
	return out, nil
}

func (r *InterestRepo) Accept(id string) error {
	return r.q.AcceptInterest(context.Background(), uuidFrom(id))
}
