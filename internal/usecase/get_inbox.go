// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// InboxListLimit caps how many items GetInbox returns. The unread count is
// computed over the same window, so the badge never counts items the inbox
// page can't show (and therefore can't mark read).
const InboxListLimit = 100

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

// Execute returns phone's newest inbox items and how many of them are unread.
func (uc *GetInbox) Execute(phone string) (Inbox, error) {
	if phone == "" {
		return Inbox{}, ErrUnauthorized
	}
	items, err := uc.inbox.ListByPhone(phone, InboxListLimit)
	if err != nil {
		return Inbox{}, err
	}
	unread, err := uc.inbox.CountUnread(phone, InboxListLimit)
	if err != nil {
		return Inbox{}, err
	}
	return Inbox{Items: items, Unread: unread}, nil
}
