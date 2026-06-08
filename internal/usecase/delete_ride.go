// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import "github.com/z3spinner/go-stop/internal/boundaries/repository"

type DeleteRide struct {
	rides         repository.RideRepository
	notifQueue    repository.NotificationQueueRepository
	feedbackQueue repository.FeedbackQueueRepository
}

func NewDeleteRide(rides repository.RideRepository, notifQueue repository.NotificationQueueRepository, feedbackQueue repository.FeedbackQueueRepository) *DeleteRide {
	return &DeleteRide{rides: rides, notifQueue: notifQueue, feedbackQueue: feedbackQueue}
}

func (uc *DeleteRide) Execute(id, phone string) error {
	ride, err := uc.rides.FindByID(id)
	if err != nil {
		return err
	}
	if ride.Phone != phone {
		return ErrUnauthorized
	}
	if err := uc.rides.Delete(id); err != nil {
		return err
	}
	_ = uc.notifQueue.DeleteForRide(id) // best-effort cleanup
	// Drop any pending feedback task: the driver explicitly deleted this ride, so
	// they should not keep getting "did someone come along?" reminders for it
	// (which also 404 once the ride row is gone). Expired rides keep their task —
	// that post-ride feedback is still wanted. Best-effort.
	_, _ = uc.feedbackQueue.DeleteByRideID(id)
	return nil
}
