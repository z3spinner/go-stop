// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

// ── mock InboxRepository ──────────────────────────────────────────────────────

type mockInboxRepo struct {
	added        []domain.InboxItem
	addErr       error
	list         []domain.InboxItem
	unread       int
	listedPhone  string
	listedLimit  int
	markedPhone  string
	markedIDs    []string
	markCalled   bool
	deleteCutoff time.Time
	deleted      int64
}

func (m *mockInboxRepo) Add(item domain.InboxItem) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.added = append(m.added, item)
	return nil
}
func (m *mockInboxRepo) ListByPhone(phone string, limit int) ([]domain.InboxItem, error) {
	m.listedPhone, m.listedLimit = phone, limit
	return m.list, nil
}
func (m *mockInboxRepo) CountUnread(string) (int, error) { return m.unread, nil }
func (m *mockInboxRepo) MarkRead(phone string, ids []string) error {
	m.markCalled, m.markedPhone, m.markedIDs = true, phone, ids
	return nil
}
func (m *mockInboxRepo) DeleteOlderThan(cutoff time.Time) (int64, error) {
	m.deleteCutoff = cutoff
	return m.deleted, nil
}

// ── InboxNotifier ─────────────────────────────────────────────────────────────

func TestInboxNotifier_RecordsKindedMessageWithoutOtherPartysPhone(t *testing.T) {
	inbox := &mockInboxRepo{}
	n := usecase.NewInboxNotifier(&mockNotifier{}, inbox)
	dep := time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC)

	n.Record("0611111111", domain.Message{
		Title: "t", Body: "b", URL: "/rides/r1",
		ContactName: "Alice", Phone: "0699999999",
		Origin: "Saillans", Destination: "Crest", DepartureAt: dep,
		Kind: domain.KindRideMatch, DedupKey: "ride_match:r1",
	})

	if len(inbox.added) != 1 {
		t.Fatalf("expected 1 inbox item, got %d", len(inbox.added))
	}
	got := inbox.added[0]
	want := domain.InboxItem{
		Phone: "0611111111", Kind: domain.KindRideMatch, ContactName: "Alice",
		Origin: "Saillans", Destination: "Crest", DepartureAt: dep,
		URL: "/rides/r1", Title: "t", Body: "b", DedupKey: "ride_match:r1",
	}
	if got != want {
		t.Errorf("recorded %+v, want %+v", got, want)
	}
}

func TestInboxNotifier_SkipsMessagesWithoutKind(t *testing.T) {
	inbox := &mockInboxRepo{}
	n := usecase.NewInboxNotifier(&mockNotifier{}, inbox)
	n.Record("0611111111", domain.Message{Title: "Test", Body: "quote", URL: "/"})
	if len(inbox.added) != 0 {
		t.Errorf("kind-less message should be push-only, got %d inbox items", len(inbox.added))
	}
}

func TestInboxNotifier_DelegatesSend(t *testing.T) {
	push := &mockNotifier{}
	n := usecase.NewInboxNotifier(push, &mockInboxRepo{})
	if err := n.Send(domain.Subscription{}, domain.Message{Title: "x"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !push.called || push.lastMsg.Title != "x" {
		t.Error("Send should be delegated to the wrapped push notifier")
	}
}

// The point of the inbox: a recipient with no push subscription still gets it.
func TestAcceptInterest_RecordsInboxEvenWithoutSubscription(t *testing.T) {
	interest := domain.Interest{ID: "int-1", RideID: "ride-1", SearcherPhone: "555-searcher", Status: "pending"}
	interests := &mockInterestRepo{
		byID:  map[string]domain.Interest{"int-1": interest},
		saved: []domain.Interest{interest},
	}
	rides := &mockRideRepo{byID: map[string]domain.Ride{
		"ride-1": {ID: "ride-1", Phone: "555-driver", DriverName: "Bob", Origin: "Saillans", Destination: "Crest"},
	}}
	push := &mockNotifier{}
	inbox := &mockInboxRepo{}

	uc := usecase.NewAcceptInterest(interests, rides, &mockSubRepo{subs: map[string]domain.Subscription{}},
		usecase.NewInboxNotifier(push, inbox))
	if _, _, err := uc.Execute("int-1", "555-driver"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if push.called {
		t.Error("no subscription: nothing should be pushed")
	}
	if len(inbox.added) != 1 {
		t.Fatalf("expected 1 inbox item, got %d", len(inbox.added))
	}
	got := inbox.added[0]
	if got.Phone != "555-searcher" || got.Kind != domain.KindInterestAccepted ||
		got.ContactName != "Bob" || got.URL != "/interests/int-1" ||
		got.DedupKey != "interest_accepted:int-1" {
		t.Errorf("unexpected inbox item: %+v", got)
	}
}

func TestNotifySearcher_DedupKeyIsStablePerRide(t *testing.T) {
	inbox := &mockInboxRepo{}
	n := usecase.NewInboxNotifier(&mockNotifier{}, inbox)
	ride := domain.Ride{ID: "r1", DriverName: "Bob", Origin: "A", Destination: "B"}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}

	usecase.NotifySearcher("0611111111", ride, subs, n)
	usecase.NotifySearcher("0611111111", ride, subs, n) // a retry

	if len(inbox.added) != 2 {
		t.Fatalf("expected both calls to reach the repo, got %d", len(inbox.added))
	}
	if inbox.added[0].DedupKey != "ride_match:r1" || inbox.added[0].DedupKey != inbox.added[1].DedupKey {
		t.Errorf("retries must share a dedup key, got %q and %q", inbox.added[0].DedupKey, inbox.added[1].DedupKey)
	}
}

func TestInboxNotifier_StorageErrorDoesNotBlockPush(t *testing.T) {
	push := &mockNotifier{}
	n := usecase.NewInboxNotifier(push, &mockInboxRepo{addErr: errors.New("db down")})
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"0611111111": {Phone: "0611111111", Endpoint: "https://push.example.com"},
	}}
	usecase.NotifySearcher("0611111111", domain.Ride{ID: "r1"}, subs, n)
	if !push.called {
		t.Error("push must still be sent when the inbox write fails")
	}
}

// ── GetInbox / MarkInboxRead / PurgeInbox ─────────────────────────────────────

func TestGetInbox_ReturnsItemsAndUnread(t *testing.T) {
	inbox := &mockInboxRepo{list: []domain.InboxItem{{ID: "n1"}, {ID: "n2"}}, unread: 1}
	got, err := usecase.NewGetInbox(inbox).Execute("0611111111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Items) != 2 || got.Unread != 1 {
		t.Errorf("got %d items / %d unread, want 2 / 1", len(got.Items), got.Unread)
	}
	if inbox.listedPhone != "0611111111" || inbox.listedLimit != usecase.InboxListLimit {
		t.Errorf("listed phone=%q limit=%d", inbox.listedPhone, inbox.listedLimit)
	}
}

func TestGetInbox_RequiresPhone(t *testing.T) {
	_, err := usecase.NewGetInbox(&mockInboxRepo{}).Execute("")
	if !errors.Is(err, usecase.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestMarkInboxRead_PassesIDs(t *testing.T) {
	inbox := &mockInboxRepo{}
	if err := usecase.NewMarkInboxRead(inbox).Execute("0611111111", []string{"n1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inbox.markedPhone != "0611111111" || len(inbox.markedIDs) != 1 || inbox.markedIDs[0] != "n1" {
		t.Errorf("marked phone=%q ids=%v", inbox.markedPhone, inbox.markedIDs)
	}
}

func TestMarkInboxRead_RequiresPhone(t *testing.T) {
	inbox := &mockInboxRepo{}
	err := usecase.NewMarkInboxRead(inbox).Execute("", nil)
	if !errors.Is(err, usecase.ErrUnauthorized) || inbox.markCalled {
		t.Errorf("expected ErrUnauthorized and no repo call, got err=%v called=%v", err, inbox.markCalled)
	}
}

func TestPurgeInbox_DeletesOlderThanRetention(t *testing.T) {
	for _, tc := range []struct{ days, wantDays int }{{0, usecase.DefaultInboxRetentionDays}, {7, 7}} {
		inbox := &mockInboxRepo{}
		before := time.Now()
		if err := usecase.NewPurgeInbox(inbox, tc.days).Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := before.Add(-time.Duration(tc.wantDays) * 24 * time.Hour)
		if d := inbox.deleteCutoff.Sub(want); d < 0 || d > time.Minute {
			t.Errorf("days=%d: cutoff %v, want ≈ %v", tc.days, inbox.deleteCutoff, want)
		}
	}
}
