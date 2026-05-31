package notification

import "github.com/z3spinner/go-stop/internal/domain"

type Notifier interface {
	Send(subscription domain.Subscription, message domain.Message) error
}
