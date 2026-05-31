//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

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

	testPool.Exec(context.Background(), `TRUNCATE rides, requests, subscriptions`)
	os.Exit(m.Run())
}

func truncate(t *testing.T) {
	t.Helper()
	testPool.Exec(context.Background(), `TRUNCATE rides, requests, subscriptions`)
}
