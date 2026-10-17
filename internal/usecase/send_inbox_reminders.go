// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

const (
	// DefaultInboxReminderIntervalHours is how long after the last push an
	// unread item gets a reminder.
	DefaultInboxReminderIntervalHours = 3
	// DefaultInboxReminderQuietHours: no reminders from 21:00 to 08:00 local.
	DefaultInboxReminderQuietHours = "21-8"
	// inboxReminderBatch bounds one cron cycle's work.
	inboxReminderBatch = 500
	// InboxReminderTitlePrefix starts every reminder's title.
	InboxReminderTitlePrefix = "Rappel : "
)

// QuietHours is a daily local-time window without reminders: [Start, End)
// hours, wrapping past midnight when Start > End. Start == End means none.
type QuietHours struct{ Start, End int }

// ParseQuietHours reads "21-8" style settings; "", "none" or "off" disable.
func ParseQuietHours(s string) (QuietHours, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "none" || s == "off" {
		return QuietHours{}, nil
	}
	a, b, ok := strings.Cut(s, "-")
	start, err1 := strconv.Atoi(strings.TrimSpace(a))
	end, err2 := strconv.Atoi(strings.TrimSpace(b))
	if !ok || err1 != nil || err2 != nil || start < 0 || start > 23 || end < 0 || end > 23 {
		return QuietHours{}, fmt.Errorf("quiet hours %q: want e.g. \"21-8\" (hours 0-23) or \"none\"", s)
	}
	return QuietHours{Start: start, End: end}, nil
}

// ContainsTime reports whether t, in tz, falls in the quiet window.
func (q QuietHours) ContainsTime(t time.Time, tz *time.Location) bool {
	if tz == nil {
		tz = time.UTC
	}
	return q.Contains(t.In(tz).Hour())
}

// Contains reports whether local hour h falls in the quiet window.
func (q QuietHours) Contains(h int) bool {
	switch {
	case q.Start == q.End:
		return false
	case q.Start < q.End:
		return h >= q.Start && h < q.End
	default: // wraps past midnight
		return h >= q.Start || h < q.End
	}
}

// SendInboxReminders re-pushes unread inbox items (domain.ReminderRules: those
// about another person's action, and the feedback prompt) every interval
// until they are read, deleted, gone (their ride/request expired or was
// deleted, which removes them) or, for capped kinds, reminded enough times.
// Reminders are push-only: they never add inbox items. Each one carries the
// item's id, so the device replaces the earlier push and tapping it marks
// the item read. Runs in the hourly cron.
type SendInboxReminders struct {
	inbox    repository.InboxRepository
	subs     repository.SubscriptionRepository
	pusher   notification.Pusher
	interval time.Duration
	quiet    QuietHours
	tz       *time.Location
	now      func() time.Time
}

func NewSendInboxReminders(
	inbox repository.InboxRepository,
	subs repository.SubscriptionRepository,
	pusher notification.Pusher,
	intervalHours int,
	quiet QuietHours,
	tz *time.Location,
) *SendInboxReminders {
	if intervalHours <= 0 {
		intervalHours = DefaultInboxReminderIntervalHours
	}
	if tz == nil {
		tz = time.UTC
	}
	return &SendInboxReminders{
		inbox: inbox, subs: subs, pusher: pusher,
		interval: time.Duration(intervalHours) * time.Hour,
		quiet:    quiet, tz: tz, now: time.Now,
	}
}

func (uc *SendInboxReminders) Execute() error {
	now := uc.now()
	if uc.quiet.ContainsTime(now, uc.tz) {
		return nil
	}
	var due []domain.InboxItem
	for _, rule := range domain.ReminderRules {
		items, err := uc.inbox.FindRemindersDue(rule.Kinds, rule.MaxReminders, now.Add(-uc.interval), inboxReminderBatch)
		if err != nil {
			return err
		}
		due = append(due, items...)
	}
	for _, it := range due {
		// No Kind: a reminder must not be recorded as a new inbox item.
		pushToDevices(it.Phone, domain.Message{
			Title:       InboxReminderTitlePrefix + it.Title,
			Body:        it.Body,
			URL:         it.URL,
			ContactName: it.ContactName,
			Origin:      it.Origin,
			Destination: it.Destination,
			DepartureAt: it.DepartureAt,
			InboxID:     it.ID,
		}, uc.subs, uc.pusher)
		if err := uc.inbox.MarkReminded(it.ID); err != nil {
			log.Printf("inbox reminder: mark reminded: %v", err)
		}
	}
	if len(due) > 0 {
		log.Printf("inbox reminders: %d sent", len(due))
	}
	return nil
}
