// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestSubscribe_SavesSubscription(t *testing.T) {
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	uc := usecase.NewSubscribe(subs)
	err := uc.Execute(domain.Subscription{
		Phone: "555-0001", Endpoint: "https://push.example.com",
		Keys: domain.PushKeys{P256DH: "key1", Auth: "auth1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs.saved) != 1 {
		t.Errorf("expected 1 saved subscription, got %d", len(subs.saved))
	}
}

func TestUnsubscribe_DeletesSubscription(t *testing.T) {
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0001": {Phone: "555-0001"},
	}}
	uc := usecase.NewUnsubscribe(subs)
	if err := uc.Execute("555-0001"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := subs.subs["555-0001"]; ok {
		t.Error("subscription should have been deleted")
	}
}
