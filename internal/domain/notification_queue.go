// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "time"

type NotificationQueueEntry struct {
	ID            string
	RideID        string
	RequestID     string
	SearcherPhone string
	SentCount     int
	LastSentAt    time.Time
	CreatedAt     time.Time
}
