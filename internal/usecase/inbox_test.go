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

// ── helpers ───────────────────────────────────────────────────────────────────

// assertInbox checks n recorded exactly one message, for phone, matching the
// inbox-relevant fields of want (Kind, DedupKey, ContactName, RideID, RequestID).
func assertInbox(t *testing.T, n *mockNotifier, phone string, want domain.Message) {
	t.Helper()
	if len(n.recorded) != 1 {
		t.Fatalf("expected 1 inbox record, got %d: %+v", len(n.recorded), n.recorded)
	}
	assertInboxMsg(t, n.recorded[0], phone, want)
}

func assertInboxMsg(t *testing.T, got recordedMsg, phone string, want domain.Message) {
	t.Helper()
	m := got.msg
	if got.phone != phone || m.Kind != want.Kind || m.DedupKey != want.DedupKey ||
		m.ContactName != want.ContactName || m.RideID != want.RideID || m.RequestID != want.RequestID {
		t.Errorf("inbox record for %s = {kind:%q dedup:%q name:%q ride:%q request:%q}, want for %s {kind:%q dedup:%q name:%q ride:%q request:%q}",
			got.phone, m.Kind, m.DedupKey, m.ContactName, m.RideID, m.RequestID,
			phone, want.Kind, want.DedupKey, want.ContactName, want.RideID, want.RequestID)
	}
}

// ── mock InboxRepository ──────────────────────────────────────────────────────

type mockInboxRepo struct {
	added            []domain.InboxItem
	addErr           error
	list             []domain.InboxItem
	unread           int
	listedPhone      string
	listedLimit      int
	countLimit       int
	markedPhone      string
	markedIDs        []string
	markCalled       bool
	markAllPhone     string
	deletedRides     []string
	deletedRequests  []string
	deleteCutoff     time.Time
	deleted          int64
	goneCalled       bool
	deleteForRideErr error
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
func (m *mockInboxRepo) CountUnread(_ string, limit int) (int, error) {
	m.countLimit = limit
	return m.unread, nil
}
func (m *mockInboxRepo) MarkRead(phone string, ids []string) error {
	m.markCalled, m.markedPhone, m.markedIDs = true, phone, ids
	return nil
}
func (m *mockInboxRepo) MarkAllRead(phone string) error {
	m.markAllPhone = phone
	return nil
}
func (m *mockInboxRepo) DeleteForRide(id string) error {
	m.deletedRides = append(m.deletedRides, id)
	return m.deleteForRideErr
}
func (m *mockInboxRepo) DeleteForRequest(id string) error {
	m.deletedRequests = append(m.deletedRequests, id)
	return nil
}
func (m *mockInboxRepo) DeleteForGonePosts() (int64, error) {
	m.goneCalled = true
	return 0, nil
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
		Kind: domain.KindRideMatch, DedupKey: "ride_match:r1", RideID: "r1",
	})

	if len(inbox.added) != 1 {
		t.Fatalf("expected 1 inbox item, got %d", len(inbox.added))
	}
	want := domain.InboxItem{
		Phone: "0611111111", Kind: domain.KindRideMatch, ContactName: "Alice",
		Origin: "Saillans", Destination: "Crest", DepartureAt: dep,
		URL: "/rides/r1", Title: "t", Body: "b", DedupKey: "ride_match:r1", RideID: "r1",
	}
	if got := inbox.added[0]; got != want {
		t.Errorf("recorded %+v, want %+v", got, want)
	}
}

// Daily alerts carry a 1970-01-01 time-of-day sentinel; it must not be stored
// as a departure (the inbox would flag the brand-new item as a past trip).
func TestInboxNotifier_DropsDailyAlertSentinelDeparture(t *testing.T) {
	inbox := &mockInboxRepo{}
	n := usecase.NewInboxNotifier(&mockNotifier{}, inbox)
	n.Record("0611111111", domain.Message{
		Kind: domain.KindRequestMatch, ContactName: "Bob",
		DepartureAt: time.Date(1970, 1, 1, 8, 30, 0, 0, time.UTC),
	})
	if len(inbox.added) != 1 || !inbox.added[0].DepartureAt.IsZero() {
		t.Errorf("expected no departure for a daily alert, got %+v", inbox.added)
	}
}

func TestInboxNotifier_SkipsMessagesWithoutKind(t *testing.T) {
	inbox := &mockInboxRepo{}
	n := usecase.NewInboxNotifier(&mockNotifier{}, inbox)
	n.Record("0611111111", domain.Message{Title: "Qui ne tente rien…", Body: "quote", URL: "/"})
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
		t.Error("Send should be delegated to the wrapped pusher")
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
	n := &mockNotifier{}

	uc := usecase.NewAcceptInterest(interests, rides, &mockSubRepo{subs: map[string]domain.Subscription{}}, n)
	if _, _, err := uc.Execute("int-1", "555-driver"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.called {
		t.Error("no subscription: nothing should be pushed")
	}
	assertInbox(t, n, "555-searcher", domain.Message{
		Kind: domain.KindInterestAccepted, DedupKey: "interest_accepted:int-1",
		ContactName: "Bob", RideID: "ride-1",
	})
}

func TestNotifySearcher_RecordsRideMatchWithStableDedupKey(t *testing.T) {
	n := &mockNotifier{}
	ride := domain.Ride{ID: "r1", DriverName: "Bob", Origin: "A", Destination: "B"}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{}}

	usecase.NotifySearcher("0611111111", ride, subs, n)
	usecase.NotifySearcher("0611111111", ride, subs, n) // a retry

	if len(n.recorded) != 2 || n.recorded[0].msg.DedupKey != n.recorded[1].msg.DedupKey {
		t.Fatalf("retries must share a dedup key, got %+v", n.recorded)
	}
	assertInboxMsg(t, n.recorded[0], "0611111111", domain.Message{
		Kind: domain.KindRideMatch, DedupKey: "ride_match:r1", ContactName: "Bob", RideID: "r1",
	})
}

func TestPingSearcher_RecordsDriverSharedOncePerInterest(t *testing.T) {
	reqs := &mockRequestRepoByID{byID: map[string]domain.Request{
		"req-1": {ID: "req-1", Phone: "555-searcher"},
	}}
	rides := &mockRideRepo{byID: map[string]domain.Ride{
		"ride-1": {ID: "ride-1", Phone: "555-driver", DriverName: "Bob"},
	}}
	interests := &mockInterestRepo{}
	n := &mockNotifier{}
	uc := usecase.NewPingSearcher(reqs, rides, interests, &mockSubRepo{subs: map[string]domain.Subscription{}}, n)

	for i := 0; i < 2; i++ { // second ping = repeat
		if _, err := uc.Execute("req-1", "ride-1", "555-driver"); err != nil {
			t.Fatalf("ping %d: %v", i, err)
		}
	}
	if len(interests.saved) != 1 {
		t.Fatalf("expected one interest, got %d", len(interests.saved))
	}
	id := interests.saved[0].ID
	if len(n.recorded) != 2 || n.recorded[0].msg.DedupKey != n.recorded[1].msg.DedupKey {
		t.Fatalf("repeat pings must share a dedup key, got %+v", n.recorded)
	}
	assertInboxMsg(t, n.recorded[0], "555-searcher", domain.Message{
		Kind: domain.KindDriverShared, DedupKey: "driver_shared:" + id, ContactName: "Bob", RideID: "ride-1",
	})
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

// ── Deleting a post removes its copies from other people's inboxes ────────────

func TestDeleteRide_RemovesInboxCopies(t *testing.T) {
	rides := &mockRideRepoDelete{rides: map[string]domain.Ride{"ride-1": {ID: "ride-1", Phone: "555-0001"}}}
	inbox := &mockInboxRepo{}
	uc := usecase.NewDeleteRide(rides, &noopNotifQueue{}, &mockFeedbackQueueDelete{}, inbox)
	if err := uc.Execute("ride-1", "555-0001"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inbox.deletedRides) != 1 || inbox.deletedRides[0] != "ride-1" {
		t.Errorf("expected inbox copies of ride-1 removed, got %v", inbox.deletedRides)
	}
}

func TestDeleteRide_InboxCleanupFailureStillSucceeds(t *testing.T) {
	rides := &mockRideRepoDelete{rides: map[string]domain.Ride{"ride-1": {ID: "ride-1", Phone: "555-0001"}}}
	inbox := &mockInboxRepo{deleteForRideErr: errors.New("db down")}
	uc := usecase.NewDeleteRide(rides, &noopNotifQueue{}, &mockFeedbackQueueDelete{}, inbox)
	if err := uc.Execute("ride-1", "555-0001"); err != nil {
		t.Fatalf("cleanup is best-effort; got %v", err)
	}
}

func TestDeleteRide_WrongPhoneLeavesInbox(t *testing.T) {
	rides := &mockRideRepoDelete{rides: map[string]domain.Ride{"ride-1": {ID: "ride-1", Phone: "555-0001"}}}
	inbox := &mockInboxRepo{}
	uc := usecase.NewDeleteRide(rides, &noopNotifQueue{}, &mockFeedbackQueueDelete{}, inbox)
	_ = uc.Execute("ride-1", "555-9999")
	if len(inbox.deletedRides) != 0 {
		t.Errorf("unauthorized delete must not touch inboxes, got %v", inbox.deletedRides)
	}
}

func TestDeleteRequest_RemovesInboxCopies(t *testing.T) {
	reqs := &mockRequestRepoDelete{requests: map[string]domain.Request{"req-1": {ID: "req-1", Phone: "555-0002"}}}
	inbox := &mockInboxRepo{}
	uc := usecase.NewDeleteRequest(reqs, inbox)
	if err := uc.Execute("req-1", "555-0002"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqs.deleted) != 1 || len(inbox.deletedRequests) != 1 || inbox.deletedRequests[0] != "req-1" {
		t.Errorf("expected request and its inbox copies removed, got req=%v inbox=%v", reqs.deleted, inbox.deletedRequests)
	}
}

func TestDeleteRequest_WrongPhoneLeavesInbox(t *testing.T) {
	reqs := &mockRequestRepoDelete{requests: map[string]domain.Request{"req-1": {ID: "req-1", Phone: "555-0002"}}}
	inbox := &mockInboxRepo{}
	err := usecase.NewDeleteRequest(reqs, inbox).Execute("req-1", "555-9999")
	if !errors.Is(err, usecase.ErrUnauthorized) || len(reqs.deleted) != 0 || len(inbox.deletedRequests) != 0 {
		t.Errorf("expected ErrUnauthorized and nothing deleted, got err=%v req=%v inbox=%v", err, reqs.deleted, inbox.deletedRequests)
	}
}

// ── GetInbox / MarkInboxRead / PurgeInbox ─────────────────────────────────────

func TestGetInbox_ReturnsItemsAndUnreadOverSameWindow(t *testing.T) {
	inbox := &mockInboxRepo{list: []domain.InboxItem{{ID: "n1"}, {ID: "n2"}}, unread: 1}
	got, err := usecase.NewGetInbox(inbox).Execute("0611111111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Items) != 2 || got.Unread != 1 {
		t.Errorf("got %d items / %d unread, want 2 / 1", len(got.Items), got.Unread)
	}
	if inbox.listedPhone != "0611111111" || inbox.listedLimit != usecase.InboxListLimit || inbox.countLimit != usecase.InboxListLimit {
		t.Errorf("listed phone=%q limit=%d countLimit=%d", inbox.listedPhone, inbox.listedLimit, inbox.countLimit)
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
	if err := usecase.NewMarkInboxRead(inbox).Execute("0611111111", []string{"n1"}, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inbox.markedPhone != "0611111111" || len(inbox.markedIDs) != 1 || inbox.markedIDs[0] != "n1" || inbox.markAllPhone != "" {
		t.Errorf("marked phone=%q ids=%v all=%q", inbox.markedPhone, inbox.markedIDs, inbox.markAllPhone)
	}
}

// Empty ids must never mean "everything".
func TestMarkInboxRead_EmptyIDsIsNoop(t *testing.T) {
	inbox := &mockInboxRepo{}
	if err := usecase.NewMarkInboxRead(inbox).Execute("0611111111", nil, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inbox.markCalled || inbox.markAllPhone != "" {
		t.Errorf("empty ids should mark nothing, got ids-call=%v all=%q", inbox.markCalled, inbox.markAllPhone)
	}
}

func TestMarkInboxRead_AllFlag(t *testing.T) {
	inbox := &mockInboxRepo{}
	if err := usecase.NewMarkInboxRead(inbox).Execute("0611111111", nil, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inbox.markAllPhone != "0611111111" || inbox.markCalled {
		t.Errorf("expected mark-all for the phone only, got all=%q ids-call=%v", inbox.markAllPhone, inbox.markCalled)
	}
}

func TestMarkInboxRead_RequiresPhone(t *testing.T) {
	inbox := &mockInboxRepo{}
	err := usecase.NewMarkInboxRead(inbox).Execute("", nil, true)
	if !errors.Is(err, usecase.ErrUnauthorized) || inbox.markCalled || inbox.markAllPhone != "" {
		t.Errorf("expected ErrUnauthorized and no repo call, got err=%v", err)
	}
}

func TestClampInboxRetentionDays(t *testing.T) {
	for _, tc := range []struct{ in, want int }{
		{0, usecase.DefaultInboxRetentionDays},
		{-5, usecase.DefaultInboxRetentionDays},
		{7, 7},
		{usecase.MaxInboxRetentionDays, usecase.MaxInboxRetentionDays},
		{200000, usecase.MaxInboxRetentionDays}, // would overflow time.Duration
	} {
		if got := usecase.ClampInboxRetentionDays(tc.in); got != tc.want {
			t.Errorf("Clamp(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestPurgeInbox_DeletesOlderThanRetention(t *testing.T) {
	for _, tc := range []struct{ days, wantDays int }{
		{0, usecase.DefaultInboxRetentionDays},
		{7, 7},
		{200000, usecase.MaxInboxRetentionDays},
	} {
		inbox := &mockInboxRepo{}
		before := time.Now()
		if err := usecase.NewPurgeInbox(inbox, tc.days).Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := before.Add(-time.Duration(tc.wantDays) * 24 * time.Hour)
		if d := inbox.deleteCutoff.Sub(want); d < 0 || d > time.Minute {
			t.Errorf("days=%d: cutoff %v, want ≈ %v", tc.days, inbox.deleteCutoff, want)
		}
		if !inbox.deleteCutoff.Before(before) {
			t.Errorf("days=%d: cutoff must be in the past", tc.days)
		}
		if !inbox.goneCalled {
			t.Errorf("days=%d: items about expired/deleted posts must be purged too", tc.days)
		}
	}
}
