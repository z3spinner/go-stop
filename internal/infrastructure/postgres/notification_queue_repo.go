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

type NotificationQueueRepo struct{ q *queries.Queries }

func NewNotificationQueueRepo(pool *pgxpool.Pool) *NotificationQueueRepo {
	return &NotificationQueueRepo{q: queries.New(pool)}
}

func (r *NotificationQueueRepo) Enqueue(rideID, requestID, searcherPhone string) error {
	return r.q.EnqueueNotification(context.Background(), queries.EnqueueNotificationParams{
		RideID:        uuidFrom(rideID),
		RequestID:     uuidFrom(requestID),
		SearcherPhone: searcherPhone,
	})
}

func (r *NotificationQueueRepo) FindPending(retryAfter time.Time, maxRetries int) ([]domain.NotificationQueueEntry, error) {
	rows, err := r.q.FindPendingNotifications(context.Background(), queries.FindPendingNotificationsParams{
		MaxRetries:  int32(maxRetries),
		RetryBefore: tsFrom(retryAfter),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.NotificationQueueEntry, len(rows))
	for i, row := range rows {
		out[i] = domain.NotificationQueueEntry{
			ID:            uuidTo(row.ID),
			RideID:        uuidTo(row.RideID),
			RequestID:     uuidTo(row.RequestID),
			SearcherPhone: row.SearcherPhone,
			SentCount:     int(row.SentCount),
			LastSentAt:    tsTo(row.LastSentAt),
			CreatedAt:     tsTo(row.CreatedAt),
		}
	}
	return out, nil
}

func (r *NotificationQueueRepo) MarkSent(id string) error {
	return r.q.MarkNotificationSent(context.Background(), uuidFrom(id))
}

func (r *NotificationQueueRepo) MarkSentByRideAndRequest(rideID, requestID string) error {
	return r.q.MarkNotificationSentByRideAndRequest(context.Background(),
		queries.MarkNotificationSentByRideAndRequestParams{
			RideID:    uuidFrom(rideID),
			RequestID: uuidFrom(requestID),
		})
}

func (r *NotificationQueueRepo) DeleteExpired() error {
	return r.q.DeleteExpiredNotifications(context.Background())
}

func (r *NotificationQueueRepo) DeleteForRide(rideID string) error {
	return r.q.DeleteNotificationsForRide(context.Background(), uuidFrom(rideID))
}

func (r *NotificationQueueRepo) ListForSearcher(searcherPhone string) ([]domain.NotificationQueueEntry, error) {
	rows, err := r.q.ListNotificationsForSearcher(context.Background(), searcherPhone)
	if err != nil {
		return nil, err
	}
	out := make([]domain.NotificationQueueEntry, len(rows))
	for i, row := range rows {
		out[i] = domain.NotificationQueueEntry{
			ID:            uuidTo(row.ID),
			RideID:        uuidTo(row.RideID),
			RequestID:     uuidTo(row.RequestID),
			SearcherPhone: row.SearcherPhone,
			SentCount:     int(row.SentCount),
			LastSentAt:    tsTo(row.LastSentAt),
			CreatedAt:     tsTo(row.CreatedAt),
		}
	}
	return out, nil
}
