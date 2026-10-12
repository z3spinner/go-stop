// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import "github.com/z3spinner/go-stop/internal/boundaries/repository"

type MarkInboxRead struct {
	inbox repository.InboxRepository
}

func NewMarkInboxRead(inbox repository.InboxRepository) *MarkInboxRead {
	return &MarkInboxRead{inbox: inbox}
}

// Execute marks phone's items read: every item when all is true, otherwise
// only the given ids. An empty id list with all=false does nothing, so a client
// that filters its ids down to nothing can never clear the whole inbox.
func (uc *MarkInboxRead) Execute(phone string, ids []string, all bool) error {
	if phone == "" {
		return ErrUnauthorized
	}
	if all {
		return uc.inbox.MarkAllRead(phone)
	}
	if len(ids) == 0 {
		return nil
	}
	return uc.inbox.MarkRead(phone, ids)
}
