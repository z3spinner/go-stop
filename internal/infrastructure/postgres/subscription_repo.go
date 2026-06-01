package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type SubscriptionRepo struct {
	q      *queries.Queries
	crypto *PhoneCrypto
}

func NewSubscriptionRepo(pool *pgxpool.Pool, crypto *PhoneCrypto) *SubscriptionRepo {
	return &SubscriptionRepo{q: queries.New(pool), crypto: crypto}
}

func (r *SubscriptionRepo) Save(sub domain.Subscription) error {
	enc, err := r.crypto.Encrypt(sub.Phone)
	if err != nil {
		return err
	}
	return r.q.UpsertSubscription(context.Background(), queries.UpsertSubscriptionParams{
		Phone:    enc,
		Endpoint: sub.Endpoint,
		P256dh:   sub.Keys.P256DH,
		Auth:     sub.Keys.Auth,
	})
}

func (r *SubscriptionRepo) FindByPhone(phone string) (domain.Subscription, error) {
	enc, err := r.crypto.Encrypt(phone)
	if err != nil {
		return domain.Subscription{}, err
	}
	row, err := r.q.GetSubscriptionByPhone(context.Background(), enc)
	if err != nil {
		return domain.Subscription{}, errors.New("subscription not found")
	}
	sub := subscriptionFromRow(row)
	plain, err := r.crypto.Decrypt(sub.Phone)
	if err != nil {
		return domain.Subscription{}, err
	}
	sub.Phone = plain
	return sub, nil
}

func (r *SubscriptionRepo) Delete(phone string) error {
	enc, err := r.crypto.Encrypt(phone)
	if err != nil {
		return err
	}
	return r.q.DeleteSubscription(context.Background(), enc)
}
