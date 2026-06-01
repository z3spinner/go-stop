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

func (r *SubscriptionRepo) FindByPhone(phone string) (domain.Subscription, error) {
	row, err := r.q.GetSubscriptionByPhone(context.Background(), phone)
	if err != nil {
		return domain.Subscription{}, errors.New("subscription not found")
	}
	return subscriptionFromRow(row), nil
}

func (r *SubscriptionRepo) Delete(phone string) error {
	return r.q.DeleteSubscription(context.Background(), phone)
}
