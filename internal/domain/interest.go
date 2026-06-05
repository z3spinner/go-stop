// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "time"

type Interest struct {
	ID            string
	RideID        string
	SearcherPhone string
	SearcherName  string
	Status        string // "pending" | "accepted"
	CreatedAt     time.Time
}

// InterestWithRide combines an interest with the ride details for display.
type InterestWithRide struct {
	Interest
	Origin      string
	Destination string
	DepartureAt time.Time
	DriverName  string
}
