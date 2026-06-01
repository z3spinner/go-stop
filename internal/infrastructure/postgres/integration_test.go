//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		os.Exit(0)
	}

	var err error
	testPool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		panic("connect test db: " + err.Error())
	}
	defer testPool.Close()

	// Wait for schema to be ready — pg_isready passes before migrations finish on
	// a fresh tmpfs container, so we poll until the tables actually exist.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		_, err = testPool.Exec(context.Background(), `TRUNCATE rides, requests, subscriptions`)
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		panic("schema not ready after 30s: " + err.Error())
	}

	os.Exit(m.Run())
}

func truncate(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), `TRUNCATE rides, requests, subscriptions, ride_stats, interests, search_events, ride_events`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
