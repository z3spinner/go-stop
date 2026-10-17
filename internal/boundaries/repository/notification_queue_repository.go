// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

import "github.com/z3spinner/go-stop/internal/domain"

type NotificationQueueRepository interface {
	// Enqueue adds a ride↔request pair, ignoring duplicates (ON CONFLICT DO
	// NOTHING). Returns true when the pair was newly inserted, false when it
	// already existed — callers push only newly-matched searchers.
	Enqueue(rideID, requestID, searcherPhone string) (inserted bool, err error)

	// MarkSentByRideAndRequest updates sent_count for a specific ride+request pair.
	MarkSentByRideAndRequest(rideID, requestID string) error

	// DeleteForRide removes all entries for a ride (called on ride deletion).
	DeleteForRide(rideID string) error

	// DeleteExpired removes entries whose ride or request has expired.
	DeleteExpired() error

	// ListForSearcher returns active notification queue entries for a phone (for UI).
	ListForSearcher(searcherPhone string) ([]domain.NotificationQueueEntry, error)
}
