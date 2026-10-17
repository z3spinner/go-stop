// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

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
