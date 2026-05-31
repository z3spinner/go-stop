package repository

import "github.com/z3spinner/go-stop/internal/domain"

type SubscriptionRepository interface {
	Save(subscription domain.Subscription) error
	FindByPhone(phone string) (domain.Subscription, error)
	Delete(phone string) error
}
