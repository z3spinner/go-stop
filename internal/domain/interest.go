package domain

import "time"

type Interest struct {
	ID            string
	RideID        string
	SearcherPhone string
	Status        string // "pending" | "accepted"
	CreatedAt     time.Time
}
