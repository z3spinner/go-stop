// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "time"

// Notification kinds. A Message with a Kind is also kept in the recipient's
// in-app inbox; a Message without one (test push, engagement reminder) is
// push-only. The frontend renders each kind in the reader's language, so these
// values are part of the API contract — don't rename them.
const (
	KindRideMatch         = "ride_match"         // a ride matches the searcher's request
	KindRequestMatch      = "request_match"      // a request matches the driver's ride
	KindInterestReceived  = "interest_received"  // a searcher asked for the driver's contact
	KindInterestCancelled = "interest_cancelled" // the searcher withdrew that request
	KindInterestAccepted  = "interest_accepted"  // the driver accepted the searcher's request
	KindDriverShared      = "driver_shared"      // a driver shared their number with a searcher
	KindContactOffered    = "contact_offered"    // someone offered their contact on a request
	KindFeedbackReminder  = "feedback_reminder"  // "did your ride leave with passengers?"
)

// InboxItem is one persisted notification in a user's in-app inbox.
// It never carries the other party's phone number.
type InboxItem struct {
	ID          string
	Phone       string // recipient
	Kind        string
	ContactName string
	Origin      string
	Destination string
	DepartureAt time.Time // zero = unknown
	URL         string
	Title       string // server-rendered fallback for clients that don't know Kind
	Body        string
	DedupKey    string // "" = never deduplicated
	RideID      string // post this item is about ("" = none); deleting it deletes the item
	RequestID   string
	CreatedAt   time.Time
	ReadAt      time.Time // zero = unread
}
