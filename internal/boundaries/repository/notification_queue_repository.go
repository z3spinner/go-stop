package repository

import (
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

type NotificationQueueRepository interface {
	// Enqueue adds a ride↔request pair. Ignores duplicates (ON CONFLICT DO NOTHING).
	Enqueue(rideID, requestID, searcherPhone string) error

	// FindPending returns entries due for (re-)notification:
	//   - sent_count < maxRetries
	//   - last_sent_at IS NULL or last_sent_at < retryAfter
	//   - the ride and request still exist and haven't expired
	//   - no interest has been expressed by this searcher for this ride
	FindPending(retryAfter time.Time, maxRetries int) ([]domain.NotificationQueueEntry, error)

	// MarkSent increments sent_count and sets last_sent_at for the given entry ID.
	MarkSent(id string) error
	// MarkSentByRideAndRequest updates sent_count for a specific ride+request pair.
	MarkSentByRideAndRequest(rideID, requestID string) error

	// DeleteForRide removes all entries for a ride (called on ride deletion).
	DeleteForRide(rideID string) error

	// DeleteExpired removes entries whose ride or request has expired.
	DeleteExpired() error

	// ListForSearcher returns active notification queue entries for a phone (for UI).
	ListForSearcher(searcherPhone string) ([]domain.NotificationQueueEntry, error)
}
