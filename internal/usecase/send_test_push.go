package usecase

import (
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// SendTestPush lets a user verify their notifications end-to-end by pushing the
// quote of the day to all of their registered devices. The content is chosen
// server-side from a fixed set (the client only picks a language), so a caller
// can never author the notification a device receives.
type SendTestPush struct {
	subs     repository.SubscriptionRepository
	notifier notification.Notifier
}

func NewSendTestPush(subs repository.SubscriptionRepository, notifier notification.Notifier) *SendTestPush {
	return &SendTestPush{subs: subs, notifier: notifier}
}

// Execute pushes the day's quote (in lang) to every subscription for the phone
// and returns how many it reached (0 = nothing registered for this phone).
func (uc *SendTestPush) Execute(phone, lang string) (int, error) {
	subList, err := uc.subs.FindByPhone(phone)
	// FindByPhone reports "not found" when none are stored (same convention as
	// sendToAll); treat that as zero devices rather than an error.
	if err != nil || len(subList) == 0 {
		return 0, nil
	}
	title, body := quoteOfTheDay(lang, time.Now())
	sendToAll(phone, domain.Message{Title: title, Body: body, URL: "/"}, uc.subs, uc.notifier)
	return len(subList), nil
}
