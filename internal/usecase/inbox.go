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

const (
	// DefaultInboxRetentionDays is how long inbox items are kept.
	DefaultInboxRetentionDays = 30
	// InboxListLimit caps how many items GetInbox returns.
	InboxListLimit = 100
)

// InboxNotifier wraps a push Notifier so that every message with a Kind is also
// kept in the recipient's inbox (it implements notification.Recorder, which
// sendToAll calls once per recipient). Wiring it in main.go is the only change
// needed for all existing use cases to start filling the inbox.
type InboxNotifier struct {
	notification.Notifier
	inbox repository.InboxRepository
}

func NewInboxNotifier(push notification.Notifier, inbox repository.InboxRepository) *InboxNotifier {
	return &InboxNotifier{Notifier: push, inbox: inbox}
}

// Record stores msg for phone. Messages without a Kind are push-only. The other
// party's phone number (msg.Phone) is deliberately not stored. Best-effort: a
// storage failure is logged and never blocks the push.
func (n *InboxNotifier) Record(phone string, msg domain.Message) {
	if msg.Kind == "" || phone == "" {
		return
	}
	err := n.inbox.Add(domain.InboxItem{
		Phone:       phone,
		Kind:        msg.Kind,
		ContactName: msg.ContactName,
		Origin:      msg.Origin,
		Destination: msg.Destination,
		DepartureAt: msg.DepartureAt,
		URL:         msg.URL,
		Title:       msg.Title,
		Body:        msg.Body,
		DedupKey:    msg.DedupKey,
	})
	if err != nil {
		log.Printf("inbox record error phone=***%s kind=%s: %v", lastN(phone, 3), msg.Kind, err)
	}
}

// Inbox is the result of GetInbox.
type Inbox struct {
	Items  []domain.InboxItem
	Unread int
}

type GetInbox struct {
	inbox repository.InboxRepository
}

func NewGetInbox(inbox repository.InboxRepository) *GetInbox {
	return &GetInbox{inbox: inbox}
}

// Execute returns phone's newest inbox items and its unread count.
func (uc *GetInbox) Execute(phone string) (Inbox, error) {
	if phone == "" {
		return Inbox{}, ErrUnauthorized
	}
	items, err := uc.inbox.ListByPhone(phone, InboxListLimit)
	if err != nil {
		return Inbox{}, err
	}
	unread, err := uc.inbox.CountUnread(phone)
	if err != nil {
		return Inbox{}, err
	}
	return Inbox{Items: items, Unread: unread}, nil
}

type MarkInboxRead struct {
	inbox repository.InboxRepository
}

func NewMarkInboxRead(inbox repository.InboxRepository) *MarkInboxRead {
	return &MarkInboxRead{inbox: inbox}
}

// Execute marks the given items (or all, when ids is empty) as read for phone.
func (uc *MarkInboxRead) Execute(phone string, ids []string) error {
	if phone == "" {
		return ErrUnauthorized
	}
	return uc.inbox.MarkRead(phone, ids)
}

type PurgeInbox struct {
	inbox     repository.InboxRepository
	retention time.Duration
	now       func() time.Time
}

// NewPurgeInbox deletes inbox items older than retentionDays (<= 0 → default).
func NewPurgeInbox(inbox repository.InboxRepository, retentionDays int) *PurgeInbox {
	if retentionDays <= 0 {
		retentionDays = DefaultInboxRetentionDays
	}
	return &PurgeInbox{
		inbox:     inbox,
		retention: time.Duration(retentionDays) * 24 * time.Hour,
		now:       time.Now,
	}
}

func (uc *PurgeInbox) Execute() error {
	n, err := uc.inbox.DeleteOlderThan(uc.now().Add(-uc.retention))
	if err != nil {
		return err
	}
	if n > 0 {
		log.Printf("purge inbox: %d old notifications deleted", n)
	}
	return nil
}
