// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

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
		sendToAll(ride.Phone, domain.Message{
			Title: "Votre trajet est-il parti avec des passagers ?",
			Body:  fmt.Sprintf("%s → %s", ride.Origin, ride.Destination),
			// Open a dedicated, single-purpose feedback screen rather than burying
			// the question at the bottom of the My Rides list.
			URL:         "/rides/" + ride.ID + "/feedback",
			ContactName: ride.DriverName,
			Phone:       ride.Phone,
			Origin:      ride.Origin,
			Destination: ride.Destination,
			DepartureAt: ride.DepartureAt,
		}, uc.subs, uc.notifier)
	}
	return nil
}
