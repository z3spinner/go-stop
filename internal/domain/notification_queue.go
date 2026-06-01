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
