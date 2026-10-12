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

	// CountUnread returns how many of phone's items have no ReadAt.
	CountUnread(phone string) (int, error)

	// MarkRead marks phone's items as read: the given ids, or all of them when
	// ids is empty. Ids belonging to another phone are ignored.
	MarkRead(phone string, ids []string) error

	// DeleteOlderThan removes items created before cutoff and returns how many.
	DeleteOlderThan(cutoff time.Time) (int64, error)
}
