// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "time"

// Notification kinds. A Message with a Kind is also kept in the recipient's
// in-app inbox; a Message without one (the engagement reminder broadcast) is
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
	KindTest              = "test"               // the user's own "send a test notification"
)

// ReminderRule says which kinds get periodic reminder pushes while unread
// (see usecase.SendInboxReminders) and at most how many (0 = no limit: until
// read, deleted or expired).
type ReminderRule struct {
	Kinds        []string
	MaxReminders int
}

// FeedbackPromptMaxReminders caps the "did your ride leave with passengers?"
// prompt at 2 reminders (3 pushes in all): it outlives the ride on purpose,
// and isn't worth more nagging.
const FeedbackPromptMaxReminders = 2

// RideMatchMaxReminders caps "a ride matches your request" at 2 reminders
// (3 pushes in all, as the old match retries did): a match is an opportunity,
// not someone waiting on an answer.
const RideMatchMaxReminders = 2

// ReminderRules: notifications about another person's action are reminded
// until handled; ride matches and the feedback prompt a limited number of
// times. A match for a driver (request_match) is informational, a cancelled
// request and test pushes need no follow-up.
var ReminderRules = []ReminderRule{
	{Kinds: []string{KindInterestReceived, KindInterestAccepted, KindDriverShared, KindContactOffered}},
	{Kinds: []string{KindRideMatch}, MaxReminders: RideMatchMaxReminders},
	{Kinds: []string{KindFeedbackReminder}, MaxReminders: FeedbackPromptMaxReminders},
}

// InboxKey is the dedup key of the inbox item of the given kind about id
// (an interest, offer, ride or request id), e.g. "interest_received:<id>".
func InboxKey(kind, id string) string { return kind + ":" + id }

// InboxItem is one persisted notification in a user's in-app inbox.
// It never carries the other party's phone number.
type InboxItem struct {
	ID            string
	Phone         string // recipient
	Kind          string
	ContactName   string
	Origin        string
	Destination   string
	DepartureAt   time.Time // zero = unknown
	URL           string
	Title         string // server-rendered fallback for clients that don't know Kind
	Body          string
	DedupKey      string // "" = never deduplicated
	RideID        string // post this item is about ("" = none); deleting it deletes the item
	RequestID     string
	CreatedAt     time.Time
	ReadAt        time.Time // zero = unread
	RemindedAt    time.Time // last reminder push; zero = none yet
	ReminderCount int
}
