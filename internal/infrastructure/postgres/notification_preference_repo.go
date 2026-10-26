// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

// NotificationPreferenceRepo stores per-phone notification preferences; a phone
// without a row has the defaults.
type NotificationPreferenceRepo struct{ q *queries.Queries }

func NewNotificationPreferenceRepo(pool *pgxpool.Pool) *NotificationPreferenceRepo {
	return &NotificationPreferenceRepo{q: queries.New(pool)}
}

func (r *NotificationPreferenceRepo) Get(phone string) (domain.NotificationPreferences, error) {
	row, err := r.q.GetNotificationPreferences(context.Background(), phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DefaultNotificationPreferences(), nil
	}
	if err != nil {
		return domain.NotificationPreferences{}, err
	}
	return domain.NotificationPreferences{MorningDigest: row.MorningDigest}, nil
}

func (r *NotificationPreferenceRepo) SetMorningDigest(phone string, on bool) error {
	return r.q.UpsertMorningDigestPreference(context.Background(), queries.UpsertMorningDigestPreferenceParams{
		Phone:         phone,
		MorningDigest: on,
	})
}

func (r *NotificationPreferenceRepo) MorningDigestOptOuts() ([]string, error) {
	return r.q.ListMorningDigestOptOuts(context.Background())
}
