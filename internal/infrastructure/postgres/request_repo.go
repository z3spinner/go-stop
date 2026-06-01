package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type RequestRepo struct {
	q      *queries.Queries
	crypto *PhoneCrypto
}

func NewRequestRepo(pool *pgxpool.Pool, crypto *PhoneCrypto) *RequestRepo {
	return &RequestRepo{q: queries.New(pool), crypto: crypto}
}

func (r *RequestRepo) decRequest(row queries.Request) (domain.Request, error) {
	req := requestFromRow(row)
	plain, err := r.crypto.Decrypt(req.Phone)
	if err != nil {
		return domain.Request{}, err
	}
	req.Phone = plain
	return req, nil
}

func (r *RequestRepo) decRequests(rows []queries.Request) ([]domain.Request, error) {
	out := make([]domain.Request, len(rows))
	for i, row := range rows {
		req, err := r.decRequest(row)
		if err != nil {
			return nil, err
		}
		out[i] = req
	}
	return out, nil
}

func (r *RequestRepo) Save(req domain.Request) error {
	enc, err := r.crypto.Encrypt(req.Phone)
	if err != nil {
		return err
	}
	return r.q.InsertRequest(context.Background(), queries.InsertRequestParams{
		ID:           uuidFrom(req.ID),
		SearcherName: req.SearcherName,
		Phone:        enc,
		Origin:       req.Origin,
		Destination:  req.Destination,
		Date:         dateFrom(req.Date),
		DepartureAt:  tsFrom(req.DepartureAt),
		Flexibility:  int32(req.Flexibility),
		PostedAt:     tsFrom(req.PostedAt),
		ExpiresAt:    tsFrom(req.ExpiresAt),
	})
}

func (r *RequestRepo) FindByID(id string) (domain.Request, error) {
	row, err := r.q.GetRequestByID(context.Background(), uuidFrom(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Request{}, errors.New("request not found")
		}
		return domain.Request{}, err
	}
	return r.decRequest(row)
}

func (r *RequestRepo) FindByPhone(phone string) ([]domain.Request, error) {
	enc, err := r.crypto.Encrypt(phone)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListRequestsByPhone(context.Background(), enc)
	if err != nil {
		return nil, err
	}
	return r.decRequests(rows)
}

func (r *RequestRepo) FindMatching(ride domain.Ride) ([]domain.Request, error) {
	// Ride phone must be encrypted to match stored values
	encPhone, err := r.crypto.Encrypt(ride.Phone)
	if err != nil {
		return nil, err
	}
	rideForQuery := ride
	rideForQuery.Phone = encPhone // not used by FindRequestsMatchingRide SQL but passed for clarity

	rows, err := r.q.FindRequestsMatchingRide(context.Background(), queries.FindRequestsMatchingRideParams{
		Lower:   ride.Origin,
		Lower_2: ride.Destination,
		Date:    dateFrom(ride.Date),
		Column4: tsFrom(ride.DepartureAt),
		Column5: int32(ride.Flexibility),
	})
	if err != nil {
		return nil, err
	}
	return r.decRequests(rows)
}

func (r *RequestRepo) Delete(id string) error {
	return r.q.DeleteRequest(context.Background(), uuidFrom(id))
}

func (r *RequestRepo) DeleteExpired() error {
	return r.q.DeleteExpiredRequests(context.Background())
}

// scanRequest and related helpers live in the same package (request_repo_helpers.go)
// but we need the nullable time scanner — reuse the package-level scanRequest.
var _ = time.Time{} // keep import
