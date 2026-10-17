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
	// RideID / RequestID name the post this message is about, so the inbox
	// copies can be deleted when its owner deletes the post. Server-side only.
	RideID    string `json:"-"`
	RequestID string `json:"-"`
	// InboxID is the inbox item this push is about (set by sendToAll after
	// recording, and by reminders). The service worker uses it to mark the item
	// read when the push is tapped and to replace an earlier push for the same
	// item instead of stacking a new one.
	InboxID string `json:"inbox_id,omitempty"`
}
