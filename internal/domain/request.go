package domain

import "time"

type Request struct {
	ID           string
	SearcherName string
	Phone        string
	Origin       string
	Destination  string
	Date         time.Time
	DepartureAt  time.Time
	Flexibility  Flexibility
	PostedAt     time.Time
	ExpiresAt    time.Time
}
