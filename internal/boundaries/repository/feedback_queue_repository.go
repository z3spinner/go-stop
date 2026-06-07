// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

import (
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

type FeedbackQueueRepository interface {
	// EnqueueStartedRides inserts a task for every ride whose window has started
	// (departure_at in the past but after windowStartAfter), is not yet answered,
	// and is not already queued. Idempotent via ride_id UNIQUE.
	EnqueueStartedRides(windowStartAfter time.Time) error

	// FindDue returns tasks past send_after that are still retry-eligible
	// (sent_count < maxRetries AND (last_sent_at IS NULL OR last_sent_at < retryAfter)).
	FindDue(retryAfter time.Time, maxRetries int) ([]domain.FeedbackTask, error)

	// FindByRideID returns the task for a ride; a non-nil error means absent.
	FindByRideID(rideID string) (domain.FeedbackTask, error)

	// MarkSent increments sent_count and sets last_sent_at.
	MarkSent(id string) error

	// DeleteByRideID removes the task for a ride and reports whether a row was
	// actually deleted. The bool is the concurrency "claim": when feedback is
	// recorded for a ride whose row is gone, only the caller that deletes the
	// task (returns true) writes the stat; concurrent callers get false and no-op.
	DeleteByRideID(rideID string) (bool, error)

	// DeleteExhausted removes tasks that hit maxRetries or are older than ttl.
	DeleteExhausted(maxRetries int, ttl time.Duration) error
}
