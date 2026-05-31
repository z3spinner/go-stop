package usecase

import (
	"fmt"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

type SendFeedbackReminders struct {
	rides    repository.RideRepository
	subs     repository.SubscriptionRepository
	notifier notification.Notifier
}

func NewSendFeedbackReminders(
	rides repository.RideRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
) *SendFeedbackReminders {
	return &SendFeedbackReminders{rides: rides, subs: subs, notifier: notifier}
}

func (uc *SendFeedbackReminders) Execute() error {
	pending, err := uc.rides.FindPendingFeedback()
	if err != nil {
		return err
	}
	for _, ride := range pending {
		sub, err := uc.subs.FindByPhone(ride.Phone)
		if err != nil {
			continue
		}
		msg := domain.Message{
			Title:       "Votre trajet est-il parti avec des passagers ?",
			Body:        fmt.Sprintf("%s → %s", ride.Origin, ride.Destination),
			URL:         "/my-rides",
			ContactName: ride.DriverName,
			Phone:       ride.Phone,
			Origin:      ride.Origin,
			Destination: ride.Destination,
			DepartureAt: ride.DepartureAt,
		}
		_ = uc.notifier.Send(sub, msg)
	}
	return nil
}
