// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type FeedbackQueueRepo struct{ q *queries.Queries }

func NewFeedbackQueueRepo(pool *pgxpool.Pool) *FeedbackQueueRepo {
	return &FeedbackQueueRepo{q: queries.New(pool)}
}

func (r *FeedbackQueueRepo) EnqueueStartedRides(windowStartAfter time.Time) error {
	return r.q.EnqueueStartedRides(context.Background(), tsFrom(windowStartAfter))
}

func (r *FeedbackQueueRepo) FindDue(retryAfter time.Time, maxRetries int) ([]domain.FeedbackTask, error) {
	rows, err := r.q.FindDueFeedback(context.Background(), queries.FindDueFeedbackParams{
		MaxRetries:  int32(maxRetries),
		RetryBefore: tsFrom(retryAfter),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.FeedbackTask, len(rows))
	for i, row := range rows {
		out[i] = feedbackTaskFromRow(row)
	}
	return out, nil
}

func (r *FeedbackQueueRepo) FindByRideID(rideID string) (domain.FeedbackTask, error) {
	row, err := r.q.GetFeedbackByRideID(context.Background(), uuidFrom(rideID))
	if err != nil {
		return domain.FeedbackTask{}, err
	}
	return feedbackTaskFromRow(row), nil
}

func (r *FeedbackQueueRepo) MarkSent(id string) error {
	return r.q.MarkFeedbackSent(context.Background(), uuidFrom(id))
}

func (r *FeedbackQueueRepo) DeleteByRideID(rideID string) (bool, error) {
	n, err := r.q.DeleteFeedbackByRideID(context.Background(), uuidFrom(rideID))
	return n > 0, err
}

func (r *FeedbackQueueRepo) DeleteExhausted(maxRetries int, ttl time.Duration) error {
	return r.q.DeleteExhaustedFeedback(context.Background(), queries.DeleteExhaustedFeedbackParams{
		MaxRetries: int32(maxRetries),
		TtlBefore:  tsFrom(time.Now().Add(-ttl)),
	})
}
