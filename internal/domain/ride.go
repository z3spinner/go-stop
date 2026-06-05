// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "time"

type Ride struct {
	ID            string
	DriverName    string
	Phone         string
	Origin        string
	Destination   string
	Date          time.Time
	DepartureAt   time.Time
	Flexibility   Flexibility
	PostedAt      time.Time
	ExpiresAt     time.Time
	FeedbackGiven bool
}
