// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"log"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

type RecordFeedback struct {
	rides repository.RideRepository
	stats repository.StatRepository
	queue repository.FeedbackQueueRepository
}

func NewRecordFeedback(
	rides repository.RideRepository,
	stats repository.StatRepository,
	queue repository.FeedbackQueueRepository,
) *RecordFeedback {
	return &RecordFeedback{rides: rides, stats: stats, queue: queue}
}

// Execute records the driver's answer to "did someone come along?". It is
// idempotent and ride-independent: ownership and route/date come from the live
// ride if it still exists, otherwise from the queued task — so the in-app
// prompt, the push reminder (ride may be gone), and the delete flow all converge.
// The "claim" (conditional feedback_given flip, or queue-row delete) happens
// before the stat write, so only one concurrent caller records a stat.
func (uc *RecordFeedback) Execute(rideID, phone string, taken bool) error {
	if ride, err := uc.rides.FindByID(rideID); err == nil {
		// Ride still exists: verify ownership, then claim by flipping feedback_given.
		if ride.Phone != phone {
			return ErrUnauthorized
		}
		claimed, err := uc.rides.ClaimFeedback(rideID)
		if err != nil {
			return err
		}
		// Cancel any pending push reminder (idempotent; safe even if we lost the claim).
		_, _ = uc.queue.DeleteByRideID(rideID)
		if !claimed {
			return nil // already answered — idempotent no-op
		}
		return uc.record(rideID, ride.Origin, ride.Destination, ride.Date, taken)
	}

	// Ride gone (deleted/expired): fall back to the queued task.
	task, err := uc.queue.FindByRideID(rideID)
	if err != nil {
		return ErrNotFound
	}
	if task.Phone != phone {
		return ErrUnauthorized
	}
	// Claim by deleting the queue row; only the caller that deletes it records.
	claimed, err := uc.queue.DeleteByRideID(rideID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil // already answered — idempotent no-op
	}
	return uc.record(rideID, task.Origin, task.Destination, task.RideDate, taken)
}

func (uc *RecordFeedback) record(rideID, origin, destination string, date time.Time, taken bool) error {
	if err := uc.stats.Save(origin, destination, date, taken); err != nil {
		return err
	}
	// Logged explicitly so yes/no is visible in the logs — the HTTP access log
	// only shows the path, and `taken` travels in the request body.
	outcome := "drove_alone"
	if taken {
		outcome = "shared"
	}
	log.Printf("ride feedback ride=%s outcome=%s route=%q->%q", rideID, outcome, origin, destination)
	return nil
}
