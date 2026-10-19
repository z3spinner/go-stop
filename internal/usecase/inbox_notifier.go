// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"log"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// InboxNotifier is the production notification.Notifier: it delivers pushes
// through the wrapped Pusher and keeps every message that has a Kind in the
// recipient's in-app inbox.
type InboxNotifier struct {
	notification.Pusher
	inbox repository.InboxRepository
}

var _ notification.Notifier = (*InboxNotifier)(nil)

func NewInboxNotifier(push notification.Pusher, inbox repository.InboxRepository) *InboxNotifier {
	return &InboxNotifier{Pusher: push, inbox: inbox}
}

// Record stores msg for phone. Messages without a Kind are push-only. The other
// party's phone number (msg.Phone) is deliberately not stored. Best-effort: a
// storage failure is logged and never blocks the push.
func (n *InboxNotifier) Record(phone string, msg domain.Message) string {
	if msg.Kind == "" || phone == "" {
		return ""
	}
	departure := msg.DepartureAt
	if departure.Year() == 1970 {
		// Daily alerts carry a 1970-01-01 time-of-day sentinel, not a real
		// departure; storing it would make the item look like a past trip.
		departure = time.Time{}
	}
	id, err := n.inbox.Add(domain.InboxItem{
		Phone:       phone,
		Kind:        msg.Kind,
		ContactName: msg.ContactName,
		Origin:      msg.Origin,
		Destination: msg.Destination,
		DepartureAt: departure,
		URL:         msg.URL,
		Title:       msg.Title,
		Body:        msg.Body,
		DedupKey:    msg.DedupKey,
		RideID:      msg.RideID,
		RequestID:   msg.RequestID,
	})
	if err != nil {
		log.Printf("inbox record error phone=***%s kind=%s: %v", lastN(phone, 3), msg.Kind, err)
		return ""
	}
	return id
}

// Acknowledge marks phone's items with these dedup keys read. Best-effort.
func (n *InboxNotifier) Acknowledge(phone string, dedupKeys ...string) {
	if phone == "" || len(dedupKeys) == 0 {
		return
	}
	if err := n.inbox.MarkReadByDedupKeys(phone, dedupKeys); err != nil {
		log.Printf("inbox acknowledge error phone=***%s: %v", lastN(phone, 3), err)
	}
}
