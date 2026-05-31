package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
)

type SubscriptionRepo struct{ pool *pgxpool.Pool }

func NewSubscriptionRepo(pool *pgxpool.Pool) *SubscriptionRepo {
	return &SubscriptionRepo{pool: pool}
}

func (r *SubscriptionRepo) Save(sub domain.Subscription) error {
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO subscriptions (id, phone, endpoint, p256dh, auth)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4)
		 ON CONFLICT (phone) DO UPDATE SET endpoint = $2, p256dh = $3, auth = $4`,
		sub.Phone, sub.Endpoint, sub.Keys.P256DH, sub.Keys.Auth,
	)
	return err
}

func (r *SubscriptionRepo) FindByPhone(phone string) (domain.Subscription, error) {
	row := r.pool.QueryRow(context.Background(),
		`SELECT id, phone, endpoint, p256dh, auth FROM subscriptions WHERE phone = $1`, phone)
	var sub domain.Subscription
	err := row.Scan(&sub.ID, &sub.Phone, &sub.Endpoint, &sub.Keys.P256DH, &sub.Keys.Auth)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Subscription{}, errors.New("subscription not found")
		}
		return domain.Subscription{}, err
	}
	return sub, nil
}

func (r *SubscriptionRepo) Delete(phone string) error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM subscriptions WHERE phone = $1`, phone)
	return err
}
