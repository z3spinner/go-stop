package domain

import "time"

type Ride struct {
	ID          string
	DriverName  string
	Phone       string
	Origin      string
	Destination string
	Date        time.Time
	DepartureAt time.Time
	Flexibility Flexibility
	PostedAt    time.Time
	ExpiresAt   time.Time
}
