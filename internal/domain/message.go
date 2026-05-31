package domain

import "time"

type Message struct {
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	URL         string    `json:"url"`
	ContactName string    `json:"contact_name"`
	Phone       string    `json:"phone"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	DepartureAt time.Time `json:"departure_at"`
}
