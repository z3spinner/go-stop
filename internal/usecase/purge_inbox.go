// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"log"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

const (
	// DefaultInboxRetentionDays is how long inbox items are kept by default.
	DefaultInboxRetentionDays = 30
	// MaxInboxRetentionDays bounds the setting. Far larger values would
	// overflow time.Duration, put the purge cutoff in the future and delete
	// every notification.
	MaxInboxRetentionDays = 3650
)

// ClampInboxRetentionDays maps a configured retention to a safe value:
// <= 0 → the default, above the maximum → the maximum.
func ClampInboxRetentionDays(days int) int {
	switch {
	case days <= 0:
		return DefaultInboxRetentionDays
	case days > MaxInboxRetentionDays:
		return MaxInboxRetentionDays
	default:
		return days
	}
}

type PurgeInbox struct {
	inbox     repository.InboxRepository
	retention time.Duration
}

// NewPurgeInbox deletes inbox items older than retentionDays (clamped, see
// ClampInboxRetentionDays).
func NewPurgeInbox(inbox repository.InboxRepository, retentionDays int) *PurgeInbox {
	days := ClampInboxRetentionDays(retentionDays)
	return &PurgeInbox{inbox: inbox, retention: time.Duration(days) * 24 * time.Hour}
}

func (uc *PurgeInbox) Execute() error {
	n, err := uc.inbox.DeleteOlderThan(time.Now().Add(-uc.retention))
	if err != nil {
		return err
	}
	if n > 0 {
		log.Printf("purge inbox: %d old notifications deleted", n)
	}
	return nil
}
