// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"testing"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestSendTestPush_SendsLocalizedQuote(t *testing.T) {
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-1": {Phone: "555-1", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	count, err := usecase.NewSendTestPush(subs, n).Execute("555-1", "fr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 device, got %d", count)
	}
	if !n.called {
		t.Error("expected a push to be sent")
	}
	// The content is chosen server-side (never client-supplied).
	if n.lastMsg.Title != "Citation du jour" {
		t.Errorf("expected fr title, got %q", n.lastMsg.Title)
	}
	if n.lastMsg.Body == "" {
		t.Error("expected a non-empty quote body")
	}
}

func TestSendTestPush_ZeroWhenNoSubscription(t *testing.T) {
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}
	n := &mockNotifier{}

	count, err := usecase.NewSendTestPush(subs, n).Execute("555-x", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 devices, got %d", count)
	}
	if n.called {
		t.Error("should not push when there is no subscription")
	}
}
