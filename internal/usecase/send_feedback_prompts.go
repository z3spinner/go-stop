// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"fmt"
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/notification"
	"github.com/z3spinner/go-stop/internal/boundaries/repository"
	"github.com/z3spinner/go-stop/internal/domain"
)

// feedbackTTL bounds how long an unanswered task lingers before cleanup. Until
// then the driver can still answer, even after the ride itself has expired.
const feedbackTTL = 7 * 24 * time.Hour

// SendFeedbackPrompts pushes "did your ride leave with passengers?" once per
// ride, when its feedback task falls due (EnqueueFeedback). Repeats are inbox
// reminders (domain.ReminderRules, at most FeedbackPromptMaxReminders), which
// stop when the driver answers, taps the notification or deletes it. Nothing is
// sent during quiet hours; due prompts go out at the end of them.
type SendFeedbackPrompts struct {
	queue    repository.FeedbackQueueRepository
	subs     repository.SubscriptionRepository
	notifier notification.Notifier
	quiet    QuietHours
	tz       *time.Location
	now      func() time.Time
}

func NewSendFeedbackPrompts(
	queue repository.FeedbackQueueRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
	quiet QuietHours,
	tz *time.Location,
) *SendFeedbackPrompts {
	if tz == nil {
		tz = time.UTC
	}
	return &SendFeedbackPrompts{queue: queue, subs: subs, notifier: notifier, quiet: quiet, tz: tz, now: time.Now}
}

func (uc *SendFeedbackPrompts) Execute() error {
	_ = uc.queue.DeleteStale(feedbackTTL)
	if uc.quiet.ContainsTime(uc.now(), uc.tz) {
		return nil
	}
	due, err := uc.queue.FindUnsent()
	if err != nil {
		return err
	}
	for _, task := range due {
		sendToAll(task.Phone, domain.Message{
			Title:       "Votre trajet est-il parti avec des passagers ?",
			Body:        fmt.Sprintf("%s → %s", task.Origin, task.Destination),
			URL:         "/rides/" + task.RideID + "/feedback",
			Phone:       task.Phone,
			Origin:      task.Origin,
			Destination: task.Destination,
			DepartureAt: task.DepartureAt,
			Kind:        domain.KindFeedbackReminder,
			DedupKey:    domain.InboxKey(domain.KindFeedbackReminder, task.RideID),
			RideID:      task.RideID,
		}, uc.subs, uc.notifier)
		_ = uc.queue.MarkSent(task.ID)
	}
	return nil
}
