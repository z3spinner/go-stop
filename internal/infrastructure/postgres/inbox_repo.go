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

func (r *InboxRepo) CountUnread(phone string) (int, error) {
	n, err := r.q.CountUnreadNotifications(context.Background(), phone)
	return int(n), err
}

func (r *InboxRepo) MarkRead(phone string, ids []string) error {
	if len(ids) == 0 {
		return r.q.MarkAllNotificationsRead(context.Background(), phone)
	}
	uuids := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		if u := uuidFrom(id); u.Valid {
			uuids = append(uuids, u)
		}
	}
	if len(uuids) == 0 {
		return nil // only malformed ids: nothing to mark (never "mark all")
	}
	return r.q.MarkNotificationsRead(context.Background(), queries.MarkNotificationsReadParams{
		Phone: phone,
		Ids:   uuids,
	})
}

func (r *InboxRepo) DeleteOlderThan(cutoff time.Time) (int64, error) {
	return r.q.DeleteNotificationsOlderThan(context.Background(), tsFrom(cutoff))
}
