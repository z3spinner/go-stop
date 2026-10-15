// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

// InboxUndoWindow is how long a deleted inbox item can be restored ("Undo").
// After it, PurgeInbox blanks the item's content.
const InboxUndoWindow = 10 * time.Minute

// DeleteInboxItem removes one notification from the caller's inbox (swipe or
// the delete button). Idempotent: deleting an unknown or already-deleted item,
// or someone else's, does nothing.
type DeleteInboxItem struct {
	inbox repository.InboxRepository
}

func NewDeleteInboxItem(inbox repository.InboxRepository) *DeleteInboxItem {
	return &DeleteInboxItem{inbox: inbox}
}

func (uc *DeleteInboxItem) Execute(phone, id string) error {
	if phone == "" {
		return ErrUnauthorized
	}
	return uc.inbox.Dismiss(phone, id)
}
