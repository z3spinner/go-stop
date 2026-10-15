// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package repository

import (
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
)

// InboxRepository persists each user's notifications (the in-app inbox).
type InboxRepository interface {
	// Add records an item. An item whose (Phone, DedupKey) already exists is
	// silently ignored; an empty DedupKey is never deduplicated.
	Add(item domain.InboxItem) error

	// ListByPhone returns the newest items first, at most limit of them.
	ListByPhone(phone string, limit int) ([]domain.InboxItem, error)

	// CountUnread counts unread items among phone's newest limit items (the
	// same window ListByPhone returns).
	CountUnread(phone string, limit int) (int, error)

	// MarkRead marks the given items of phone as read. Ids belonging to another
	// phone or that are malformed are ignored; an empty list does nothing.
	MarkRead(phone string, ids []string) error

	// MarkAllRead marks every item of phone as read.
	MarkAllRead(phone string) error

	// Dismiss hides one of phone's items from the inbox (the user deleted it)
	// and marks it read. Unknown ids, other phones' ids and already-dismissed
	// items are ignored. The row is kept as a tombstone so a repeat of the same
	// event (same dedup key) doesn't bring it back.
	Dismiss(phone, id string) error

	// Restore un-dismisses phone's item if it was dismissed after
	// dismissedAfter (the undo window). Reports whether anything was restored.
	Restore(phone, id string, dismissedAfter time.Time) (bool, error)

	// ScrubDismissed blanks the content of items dismissed before cutoff and
	// returns how many were scrubbed.
	ScrubDismissed(cutoff time.Time) (int64, error)

	// DeleteForRide / DeleteForRequest remove every item about that post, in
	// anyone's inbox (used when the owner deletes the post).
	DeleteForRide(rideID string) error
	DeleteForRequest(requestID string) error

	// DeleteForGonePosts removes items about rides/requests that no longer
	// exist (expired or deleted), except feedback reminders, and returns how
	// many.
	DeleteForGonePosts() (int64, error)

	// DeleteOlderThan removes items created before cutoff and returns how many.
	DeleteOlderThan(cutoff time.Time) (int64, error)
}
