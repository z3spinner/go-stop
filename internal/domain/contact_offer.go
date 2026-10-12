// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "time"

// ContactOffer represents a person voluntarily sharing their contact details
// with a searcher, without necessarily having a posted ride.
type ContactOffer struct {
	ID           string
	RequestID    string
	OffererPhone string
	OffererName  string
	CreatedAt    time.Time
}
