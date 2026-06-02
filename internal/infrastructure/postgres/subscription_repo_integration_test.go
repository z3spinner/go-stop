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

	subs, err := repo.FindByPhone("555-0001")
	if err != nil {
		t.Fatalf("FindByPhone: %v", err)
	}
	got := subs[0]
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

func TestSubscriptionRepo_MultipleDevicesSamePhone(t *testing.T) {
	truncate(t)
	repo := postgres.NewSubscriptionRepo(testPool)

	// Two different devices for the same phone
	_ = repo.Save(domain.Subscription{
		Phone: "555-0002", Endpoint: "https://push.example.com/device1",
		Keys: domain.PushKeys{P256DH: "key1", Auth: "auth1"},
	})
	_ = repo.Save(domain.Subscription{
		Phone: "555-0002", Endpoint: "https://push.example.com/device2",
		Keys: domain.PushKeys{P256DH: "key2", Auth: "auth2"},
	})

	subs, err := repo.FindByPhone("555-0002")
	if err != nil {
		t.Fatalf("FindByPhone: %v", err)
	}
	if len(subs) != 2 {
		t.Errorf("expected 2 subscriptions (one per device), got %d", len(subs))
	}

	// Same device re-subscribing updates keys, not adds a row
	_ = repo.Save(domain.Subscription{
		Phone: "555-0002", Endpoint: "https://push.example.com/device1",
		Keys: domain.PushKeys{P256DH: "new-key1", Auth: "new-auth1"},
	})
	subs2, _ := repo.FindByPhone("555-0002")
	if len(subs2) != 2 {
		t.Errorf("expected still 2 subscriptions after re-subscribe, got %d", len(subs2))
	}
}
