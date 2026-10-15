// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

// RestoreInboxItem undoes a recent DeleteInboxItem (within InboxUndoWindow).
type RestoreInboxItem struct {
	inbox repository.InboxRepository
	now   func() time.Time
}

func NewRestoreInboxItem(inbox repository.InboxRepository) *RestoreInboxItem {
	return &RestoreInboxItem{inbox: inbox, now: time.Now}
}

// Execute returns ErrNotFound when there is nothing to restore: unknown id,
// someone else's item, not deleted, or deleted too long ago.
func (uc *RestoreInboxItem) Execute(phone, id string) error {
	if phone == "" {
		return ErrUnauthorized
	}
	ok, err := uc.inbox.Restore(phone, id, uc.now().Add(-InboxUndoWindow))
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}
