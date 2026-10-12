// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase

import (
	"time"

	"github.com/z3spinner/go-stop/internal/boundaries/repository"
)

// FeedbackEnqueueBound limits enqueue to rides whose window started within the
// last day, so old rides are not back-filled on first deploy.
const FeedbackEnqueueBound = 24 * time.Hour

type EnqueueFeedback struct {
	queue repository.FeedbackQueueRepository
}

func NewEnqueueFeedback(queue repository.FeedbackQueueRepository) *EnqueueFeedback {
	return &EnqueueFeedback{queue: queue}
}

func (uc *EnqueueFeedback) Execute() error {
	return uc.queue.EnqueueStartedRides(time.Now().Add(-FeedbackEnqueueBound))
}
