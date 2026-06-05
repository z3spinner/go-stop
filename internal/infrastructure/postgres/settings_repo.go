// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

// SettingsRepo is a generic key/value store for runtime-provisioned config.
type SettingsRepo struct{ q *queries.Queries }

func NewSettingsRepo(pool *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{q: queries.New(pool)}
}

// Get returns the value for key. found is false (with nil error) when the key
// is absent.
func (r *SettingsRepo) Get(ctx context.Context, key string) (string, bool, error) {
	v, err := r.q.GetSetting(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// InsertIfAbsent writes key=value only if key does not already exist. A losing
// writer in a concurrent race is a no-op, not an error.
func (r *SettingsRepo) InsertIfAbsent(ctx context.Context, key, value string) error {
	return r.q.InsertSettingIfAbsent(ctx, queries.InsertSettingIfAbsentParams{
		Key:   key,
		Value: value,
	})
}
