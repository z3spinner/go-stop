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
