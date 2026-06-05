// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres/sqlc/queries"
)

type DestinationRepo struct{ q *queries.Queries }

func NewDestinationRepo(pool *pgxpool.Pool) *DestinationRepo {
	return &DestinationRepo{q: queries.New(pool)}
}

func (r *DestinationRepo) GetAll() ([]string, error) {
	locs, err := r.q.ListDestinations(context.Background())
	if err != nil {
		return nil, err
	}
	if locs == nil {
		return []string{}, nil
	}
	return locs, nil
}
