// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package postgres_test

import (
	"context"
	"testing"
)

// pg_dump/pg_restore run with an empty search_path, and CREATE TABLE inlines
// route_norm() while validating the generated *_norm columns. So route_norm
// must resolve its helpers without relying on search_path — otherwise a restore
// (e.g. Scalingo's Postgres major upgrade) fails to recreate rides/requests.
func TestRouteNorm_ResolvesWithEmptySearchPath(t *testing.T) {
	ctx := context.Background()
	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SET LOCAL search_path = ''`); err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	var got string
	if err := tx.QueryRow(ctx, `SELECT public.route_norm($1)`, "  Évry   Sud ").Scan(&got); err != nil {
		t.Fatalf("route_norm with empty search_path: %v", err)
	}
	if got != "evry sud" {
		t.Errorf("route_norm = %q, want %q", got, "evry sud")
	}
}
