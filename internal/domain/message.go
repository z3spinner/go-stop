// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

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
	// Kind identifies the notification type (see inbox.go). When set, the
	// message is also kept in the recipient's in-app inbox.
	Kind string `json:"kind,omitempty"`
	// DedupKey identifies one logical event so retries of the same push don't
	// create duplicate inbox entries. Server-side only.
	DedupKey string `json:"-"`
}
