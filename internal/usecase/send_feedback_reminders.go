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

// feedbackTTL bounds how long an unanswered task lingers before cleanup.
const feedbackTTL = 7 * 24 * time.Hour

type SendFeedbackReminders struct {
	queue      repository.FeedbackQueueRepository
	subs       repository.SubscriptionRepository
	notifier   notification.Notifier
	interval   time.Duration
	maxRetries int
}

func NewSendFeedbackReminders(
	queue repository.FeedbackQueueRepository,
	subs repository.SubscriptionRepository,
	notifier notification.Notifier,
	intervalHours, maxRetries int,
) *SendFeedbackReminders {
	if intervalHours <= 0 {
		intervalHours = DefaultRetryIntervalHours
	}
	if maxRetries <= 0 {
		maxRetries = DefaultMaxRetries
	}
	return &SendFeedbackReminders{
		queue:      queue,
		subs:       subs,
		notifier:   notifier,
		interval:   time.Duration(intervalHours) * time.Hour,
		maxRetries: maxRetries,
	}
}

func (uc *SendFeedbackReminders) Execute() error {
	retryAfter := time.Now().Add(-uc.interval)
	due, err := uc.queue.FindDue(retryAfter, uc.maxRetries)
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
		}, uc.subs, uc.notifier)
		_ = uc.queue.MarkSent(task.ID)
	}
	_ = uc.queue.DeleteExhausted(uc.maxRetries, feedbackTTL)
	return nil
}
