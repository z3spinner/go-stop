package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type Subscribe struct {
	subs repository.SubscriptionRepository
}

func NewSubscribe(subs repository.SubscriptionRepository) *Subscribe {
	return &Subscribe{subs: subs}
}

func (uc *Subscribe) Execute(sub domain.Subscription) error {
	return uc.subs.Save(sub)
}

type Unsubscribe struct {
	subs repository.SubscriptionRepository
}

func NewUnsubscribe(subs repository.SubscriptionRepository) *Unsubscribe {
	return &Unsubscribe{subs: subs}
}

func (uc *Unsubscribe) Execute(phone string) error {
	return uc.subs.Delete(phone)
}
