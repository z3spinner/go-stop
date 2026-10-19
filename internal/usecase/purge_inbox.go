// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"log"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
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

// NewPurgeInbox deletes inbox items about rides/requests that have expired or
// been deleted, anything older than retentionDays (clamped, see
// ClampInboxRetentionDays) and the weekly engagement reminder after a week, and
// blanks the content of items the user deleted more than InboxUndoWindow ago. Runs in the hourly cron after the expiry jobs.
func NewPurgeInbox(inbox repository.InboxRepository, retentionDays int) *PurgeInbox {
	days := ClampInboxRetentionDays(retentionDays)
	return &PurgeInbox{inbox: inbox, retention: time.Duration(days) * 24 * time.Hour}
}

func (uc *PurgeInbox) Execute() error {
	gone, err := uc.inbox.DeleteForGonePosts()
	if err != nil {
		return err
	}
	now := time.Now()
	old, err := uc.inbox.DeleteOlderThan(now.Add(-uc.retention))
	if err != nil {
		return err
	}
	stale, err := uc.inbox.DeleteKindOlderThan(domain.KindEngagementReminder, now.Add(-domain.EngagementReminderLifetime))
	if err != nil {
		return err
	}
	old += stale
	scrubbed, err := uc.inbox.ScrubDismissed(now.Add(-InboxUndoWindow))
	if err != nil {
		return err
	}
	if gone > 0 || old > 0 || scrubbed > 0 {
		log.Printf("purge inbox: %d about expired/deleted posts, %d past retention, %d deleted by users scrubbed", gone, old, scrubbed)
	}
	return nil
}
