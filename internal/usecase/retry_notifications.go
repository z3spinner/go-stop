// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"log"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

const (
	DefaultRetryIntervalHours = 2
	DefaultMaxRetries         = 3
)

type RetryNotifications struct {
	queue      repository.NotificationQueueRepository
	rides      repository.RideRepository
	subs       repository.SubscriptionRepository
	notifier   notification.Notifier
	interval   time.Duration
	maxRetries int
}

func NewRetryNotifications(
	queue repository.NotificationQueueRepository,
	rides repository.RideRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
	intervalHours, maxRetries int,
) *RetryNotifications {
	if intervalHours <= 0 {
		intervalHours = DefaultRetryIntervalHours
	}
	if maxRetries <= 0 {
		maxRetries = DefaultMaxRetries
	}
	return &RetryNotifications{
		queue:      queue,
		rides:      rides,
		subs:       subs,
		notifier:   notifier,
		interval:   time.Duration(intervalHours) * time.Hour,
		maxRetries: maxRetries,
	}
}

func (uc *RetryNotifications) Execute() error {
	retryAfter := time.Now().Add(-uc.interval)
	pending, err := uc.queue.FindPending(retryAfter, uc.maxRetries)
	if err != nil {
		return err
	}

	sent := 0
	for _, entry := range pending {
		ride, err := uc.rides.FindByID(entry.RideID)
		if err != nil {
			continue
		}
		NotifySearcher(entry.SearcherPhone, ride, uc.subs, uc.notifier)
		_ = uc.queue.MarkSent(entry.ID)
		sent++
	}
	if len(pending) > 0 {
		log.Printf("retry notifications: %d of %d pending processed", sent, len(pending))
	}
	_ = uc.queue.DeleteExpired()
	return nil
}

// NotificationSummary is returned by GetPendingNotifications for UI display.
type NotificationSummary struct {
	Entry domain.NotificationQueueEntry
	Ride  domain.Ride
}

type GetPendingNotifications struct {
	queue repository.NotificationQueueRepository
	rides repository.RideRepository
}

func NewGetPendingNotifications(
	queue repository.NotificationQueueRepository,
	rides repository.RideRepository,
) *GetPendingNotifications {
	return &GetPendingNotifications{queue: queue, rides: rides}
}

func (uc *GetPendingNotifications) Execute(searcherPhone string) ([]NotificationSummary, error) {
	entries, err := uc.queue.ListForSearcher(searcherPhone)
	if err != nil {
		return nil, err
	}
	out := make([]NotificationSummary, 0, len(entries))
	for _, e := range entries {
		ride, err := uc.rides.FindByID(e.RideID)
		if err != nil {
			continue
		}
		out = append(out, NotificationSummary{Entry: e, Ride: ride})
	}
	return out, nil
}
