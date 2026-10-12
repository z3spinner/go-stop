// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type InboxRepo struct{ q *queries.Queries }

func NewInboxRepo(pool *pgxpool.Pool) *InboxRepo {
	return &InboxRepo{q: queries.New(pool)}
}

func (r *InboxRepo) Add(item domain.InboxItem) error {
	return r.q.AddNotification(context.Background(), queries.AddNotificationParams{
		Phone:       item.Phone,
		Kind:        item.Kind,
		ContactName: item.ContactName,
		Origin:      item.Origin,
		Destination: item.Destination,
		DepartureAt: tsFrom(item.DepartureAt),
		Url:         item.URL,
		Title:       item.Title,
		Body:        item.Body,
		RideID:      uuidFrom(item.RideID), // "" → NULL
		RequestID:   uuidFrom(item.RequestID),
		DedupKey:    pgtype.Text{String: item.DedupKey, Valid: item.DedupKey != ""},
	})
}

func (r *InboxRepo) ListByPhone(phone string, limit int) ([]domain.InboxItem, error) {
	rows, err := r.q.ListNotificationsByPhone(context.Background(), queries.ListNotificationsByPhoneParams{
		Phone:    phone,
		MaxItems: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.InboxItem, len(rows))
	for i, row := range rows {
		out[i] = inboxItemFromRow(row)
	}
	return out, nil
}

func (r *InboxRepo) CountUnread(phone string, limit int) (int, error) {
	n, err := r.q.CountUnreadNotifications(context.Background(), queries.CountUnreadNotificationsParams{
		Phone:    phone,
		MaxItems: int32(limit),
	})
	return int(n), err
}

func (r *InboxRepo) MarkAllRead(phone string) error {
	return r.q.MarkAllNotificationsRead(context.Background(), phone)
}

// MarkRead marks only the given ids. An empty or all-malformed list does
// nothing — it never falls back to "mark all".
func (r *InboxRepo) MarkRead(phone string, ids []string) error {
	uuids := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		if u := uuidFrom(id); u.Valid {
			uuids = append(uuids, u)
		}
	}
	if len(uuids) == 0 {
		return nil
	}
	return r.q.MarkNotificationsRead(context.Background(), queries.MarkNotificationsReadParams{
		Phone: phone,
		Ids:   uuids,
	})
}

func (r *InboxRepo) DeleteForRide(rideID string) error {
	u := uuidFrom(rideID)
	if !u.Valid {
		return nil
	}
	return r.q.DeleteInboxForRide(context.Background(), u)
}

func (r *InboxRepo) DeleteForRequest(requestID string) error {
	u := uuidFrom(requestID)
	if !u.Valid {
		return nil
	}
	return r.q.DeleteInboxForRequest(context.Background(), u)
}

func (r *InboxRepo) DeleteForGonePosts() (int64, error) {
	return r.q.DeleteInboxForGonePosts(context.Background())
}

func (r *InboxRepo) DeleteOlderThan(cutoff time.Time) (int64, error) {
	return r.q.DeleteNotificationsOlderThan(context.Background(), tsFrom(cutoff))
}
