package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type SubscriptionRepo struct{ q *queries.Queries }

func NewSubscriptionRepo(pool *pgxpool.Pool) *SubscriptionRepo {
	return &SubscriptionRepo{q: queries.New(pool)}
}

func (r *SubscriptionRepo) Save(sub domain.Subscription) error {
	return r.q.UpsertSubscription(context.Background(), queries.UpsertSubscriptionParams{
		Phone:    sub.Phone,
		Endpoint: sub.Endpoint,
		P256dh:   sub.Keys.P256DH,
		Auth:     sub.Keys.Auth,
	})
}

func (r *SubscriptionRepo) FindByPhone(phone string) ([]domain.Subscription, error) {
	rows, err := r.q.ListSubscriptionsByPhone(context.Background(), phone)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("subscription not found")
	}
	out := make([]domain.Subscription, len(rows))
	for i, row := range rows {
		out[i] = subscriptionFromRow(row)
	}
	return out, nil
}

func (r *SubscriptionRepo) Delete(phone string) error {
	return r.q.DeleteSubscription(context.Background(), phone)
}

func (r *SubscriptionRepo) DeleteByEndpoint(endpoint string) error {
	return r.q.DeleteSubscriptionByEndpoint(context.Background(), endpoint)
}
