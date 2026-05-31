//go:build integration

package postgres_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func TestSubscriptionRepo_SaveFindDelete(t *testing.T) {
	truncate(t)
	repo := postgres.NewSubscriptionRepo(testPool)

	sub := domain.Subscription{
		Phone:    "555-0001",
		Endpoint: "https://push.example.com/1",
		Keys:     domain.PushKeys{P256DH: "pubkey", Auth: "authkey"},
	}

	if err := repo.Save(sub); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByPhone("555-0001")
	if err != nil {
		t.Fatalf("FindByPhone: %v", err)
	}
	if got.Endpoint != sub.Endpoint {
		t.Errorf("expected endpoint %s, got %s", sub.Endpoint, got.Endpoint)
	}
	if got.Keys.P256DH != "pubkey" {
		t.Errorf("expected P256DH pubkey, got %s", got.Keys.P256DH)
	}

	if err := repo.Delete("555-0001"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.FindByPhone("555-0001")
	if err == nil {
		t.Error("expected not found after delete")
	}
}

func TestSubscriptionRepo_Save_UpdatesOnConflict(t *testing.T) {
	truncate(t)
	repo := postgres.NewSubscriptionRepo(testPool)

	_ = repo.Save(domain.Subscription{
		Phone: "555-0002", Endpoint: "https://push.example.com/old",
		Keys: domain.PushKeys{P256DH: "old-key", Auth: "old-auth"},
	})
	_ = repo.Save(domain.Subscription{
		Phone: "555-0002", Endpoint: "https://push.example.com/new",
		Keys: domain.PushKeys{P256DH: "new-key", Auth: "new-auth"},
	})

	got, err := repo.FindByPhone("555-0002")
	if err != nil {
		t.Fatalf("FindByPhone: %v", err)
	}
	if got.Endpoint != "https://push.example.com/new" {
		t.Errorf("expected updated endpoint, got %s", got.Endpoint)
	}
}
