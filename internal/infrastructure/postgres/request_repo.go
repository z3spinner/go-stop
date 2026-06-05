// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type RequestRepo struct {
	q         *queries.Queries
	graceMins int32
}

func NewRequestRepo(pool *pgxpool.Pool, graceMins int) *RequestRepo {
	return &RequestRepo{q: queries.New(pool), graceMins: int32(graceMins)}
}

func (r *RequestRepo) Save(req domain.Request) error {
	return r.q.InsertRequest(context.Background(), queries.InsertRequestParams{
		ID:           uuidFrom(req.ID),
		SearcherName: req.SearcherName,
		Phone:        req.Phone,
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
		return domain.Request{}, errors.New("request not found")
	}
	return requestFromRow(row), nil
}

func (r *RequestRepo) FindByPhone(phone string) ([]domain.Request, error) {
	rows, err := r.q.ListRequestsByPhone(context.Background(), queries.ListRequestsByPhoneParams{
		Phone:        phone,
		GraceMinutes: r.graceMins,
	})
	if err != nil {
		return nil, err
	}
	return requestsFromRows(rows), nil
}

func (r *RequestRepo) FindAllActive() ([]domain.Request, error) {
	rows, err := r.q.ListActiveRequests(context.Background(), r.graceMins)
	if err != nil {
		return nil, err
	}
	return requestsFromRows(rows), nil
}

func (r *RequestRepo) FindMatching(ride domain.Ride) ([]domain.Request, error) {
	rows, err := r.q.FindRequestsMatchingRide(context.Background(), queries.FindRequestsMatchingRideParams{
		Origin:        ride.Origin,
		Destination:   ride.Destination,
		Date:          dateFrom(ride.Date),
		DepartureAt:   tsFrom(ride.DepartureAt),
		WindowMinutes: int32(ride.Flexibility),
	})
	if err != nil {
		return nil, err
	}
	return requestsFromRows(rows), nil
}

func (r *RequestRepo) Delete(id string) error {
	return r.q.DeleteRequest(context.Background(), uuidFrom(id))
}

func (r *RequestRepo) DeleteExpired() error {
	return r.q.DeleteExpiredRequests(context.Background())
}
