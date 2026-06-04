//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func clearSettings(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), `DELETE FROM app_settings`); err != nil {
		t.Fatalf("clear app_settings: %v", err)
	}
}

func TestSettingsRepo_GetMissingReturnsNotFound(t *testing.T) {
	clearSettings(t)
	repo := postgres.NewSettingsRepo(testPool)

	_, found, err := repo.Get(context.Background(), "nope")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Fatal("expected found=false for missing key")
	}
}

func TestSettingsRepo_InsertIfAbsentThenGet(t *testing.T) {
	clearSettings(t)
	repo := postgres.NewSettingsRepo(testPool)
	ctx := context.Background()

	if err := repo.InsertIfAbsent(ctx, "k", "first"); err != nil {
		t.Fatalf("InsertIfAbsent: %v", err)
	}
	// Second insert must NOT overwrite.
	if err := repo.InsertIfAbsent(ctx, "k", "second"); err != nil {
		t.Fatalf("InsertIfAbsent (2nd): %v", err)
	}

	v, found, err := repo.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || v != "first" {
		t.Fatalf("got (%q, %v), want (\"first\", true)", v, found)
	}
}
