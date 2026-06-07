// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "time"

// FeedbackTask is a queued post-ride feedback request. It is self-contained:
// it carries the owner's phone (for the ownership check) and origin/destination/
// ride_date (to write the stat) so feedback can be solicited and recorded even
// after the ride row has been deleted or expired.
type FeedbackTask struct {
	ID          string
	RideID      string
	Phone       string
	Origin      string
	Destination string
	RideDate    time.Time
	DepartureAt time.Time
	SendAfter   time.Time
	SentCount   int
	LastSentAt  time.Time // zero = never sent
	CreatedAt   time.Time
}
