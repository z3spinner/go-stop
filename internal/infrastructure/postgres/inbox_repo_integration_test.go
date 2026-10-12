// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func TestInboxRepo_AddListDedupAndRead(t *testing.T) {
	truncate(t)
	repo := postgres.NewInboxRepo(testPool)
	dep := time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC)

	add := func(it domain.InboxItem) {
		t.Helper()
		if err := repo.Add(it); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	add(domain.InboxItem{Phone: "0611", Kind: domain.KindRideMatch, ContactName: "Bob",
		Origin: "A", Destination: "B", DepartureAt: dep, URL: "/rides/r1", DedupKey: "ride_match:r1"})
	add(domain.InboxItem{Phone: "0611", Kind: domain.KindRideMatch, DedupKey: "ride_match:r1"}) // retry: ignored
	add(domain.InboxItem{Phone: "0611", Kind: domain.KindInterestReceived})                     // no key
	add(domain.InboxItem{Phone: "0611", Kind: domain.KindInterestReceived})                     // no key: kept
	add(domain.InboxItem{Phone: "0622", Kind: domain.KindRideMatch, DedupKey: "ride_match:r1"}) // other phone

	items, err := repo.ListByPhone("0611", 10)
	if err != nil {
		t.Fatalf("ListByPhone: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items (retry deduped), got %d", len(items))
	}
	first := items[len(items)-1] // oldest last
	if first.ContactName != "Bob" || !first.DepartureAt.Equal(dep) || first.URL != "/rides/r1" ||
		first.ID == "" || first.CreatedAt.IsZero() || !first.ReadAt.IsZero() {
		t.Errorf("unexpected stored item: %+v", first)
	}
	if items[0].DepartureAt != (time.Time{}) {
		t.Errorf("missing departure should round-trip as zero time, got %v", items[0].DepartureAt)
	}
	if limited, _ := repo.ListByPhone("0611", 2); len(limited) != 2 {
		t.Errorf("limit not applied: got %d", len(limited))
	}

	if n, _ := repo.CountUnread("0611", 100); n != 3 {
		t.Errorf("expected 3 unread, got %d", n)
	}
	if err := repo.MarkRead("0611", []string{first.ID, "not-a-uuid"}); err != nil {
		t.Fatalf("MarkRead ids: %v", err)
	}
	if n, _ := repo.CountUnread("0611", 100); n != 2 {
		t.Errorf("expected 2 unread after marking one, got %d", n)
	}
	if err := repo.MarkRead("0611", []string{"not-a-uuid"}); err != nil {
		t.Fatalf("MarkRead malformed: %v", err)
	}
	if n, _ := repo.CountUnread("0611", 100); n != 2 {
		t.Errorf("malformed ids must not mark all; got %d unread", n)
	}
	if err := repo.MarkRead("0611", nil); err != nil {
		t.Fatalf("MarkRead empty: %v", err)
	}
	if n, _ := repo.CountUnread("0611", 100); n != 2 {
		t.Errorf("empty ids must not mark all; got %d unread", n)
	}
	if err := repo.MarkAllRead("0611"); err != nil {
		t.Fatalf("MarkAllRead: %v", err)
	}
	if n, _ := repo.CountUnread("0611", 100); n != 0 {
		t.Errorf("expected 0 unread after mark-all, got %d", n)
	}
	if n, _ := repo.CountUnread("0622", 100); n != 1 {
		t.Errorf("other phone untouched: expected 1 unread, got %d", n)
	}
}

func TestInboxRepo_DeleteOlderThan(t *testing.T) {
	truncate(t)
	repo := postgres.NewInboxRepo(testPool)
	if err := repo.Add(domain.InboxItem{Phone: "0611", Kind: domain.KindRideMatch, DedupKey: "old"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Add(domain.InboxItem{Phone: "0611", Kind: domain.KindRideMatch, DedupKey: "new"}); err != nil {
		t.Fatal(err)
	}
	if _, err := testPool.Exec(context.Background(),
		`UPDATE notifications SET created_at = NOW() - interval '31 days' WHERE dedup_key = 'old'`); err != nil {
		t.Fatal(err)
	}

	n, err := repo.DeleteOlderThan(time.Now().Add(-30 * 24 * time.Hour))
	if err != nil {
		t.Fatalf("DeleteOlderThan: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 deleted, got %d", n)
	}
	items, _ := repo.ListByPhone("0611", 10)
	if len(items) != 1 || items[0].DedupKey != "new" {
		t.Errorf("expected only the recent item to remain, got %+v", items)
	}
}

func TestInboxRepo_UnreadCountUsesListWindow(t *testing.T) {
	truncate(t)
	repo := postgres.NewInboxRepo(testPool)
	for i := 0; i < 3; i++ {
		if err := repo.Add(domain.InboxItem{Phone: "0611", Kind: domain.KindRideMatch}); err != nil {
			t.Fatal(err)
		}
	}
	// Make the oldest one clearly oldest, then only it stays unread.
	if _, err := testPool.Exec(context.Background(), `UPDATE notifications SET created_at = NOW() - interval '1 day'
		WHERE id = (SELECT id FROM notifications ORDER BY created_at LIMIT 1)`); err != nil {
		t.Fatal(err)
	}
	newest, _ := repo.ListByPhone("0611", 2)
	if err := repo.MarkRead("0611", []string{newest[0].ID, newest[1].ID}); err != nil {
		t.Fatal(err)
	}
	if n, _ := repo.CountUnread("0611", 2); n != 0 {
		t.Errorf("unread outside the newest-2 window must not count; got %d", n)
	}
	if n, _ := repo.CountUnread("0611", 3); n != 1 {
		t.Errorf("expected 1 unread in the newest-3 window, got %d", n)
	}
}

func TestInboxRepo_DeleteForRideAndRequest(t *testing.T) {
	truncate(t)
	repo := postgres.NewInboxRepo(testPool)
	const rideA, rideB = "11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"
	const reqA = "33333333-3333-3333-3333-333333333333"
	for _, it := range []domain.InboxItem{
		{Phone: "0611", Kind: domain.KindRideMatch, RideID: rideA},
		{Phone: "0622", Kind: domain.KindDriverShared, RideID: rideA},
		{Phone: "0611", Kind: domain.KindRideMatch, RideID: rideB},
		{Phone: "0633", Kind: domain.KindContactOffered, RequestID: reqA},
		{Phone: "0633", Kind: domain.KindInterestReceived}, // linked to nothing
	} {
		if err := repo.Add(it); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.DeleteForRide(rideA); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteForRequest(reqA); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteForRide(""); err != nil { // no id: no-op, never "delete unlinked"
		t.Fatal(err)
	}
	var left []string
	rows, _ := testPool.Query(context.Background(), `SELECT kind || ':' || COALESCE(ride_id::text, request_id::text, '-') FROM notifications ORDER BY 1`)
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		left = append(left, s)
	}
	rows.Close()
	want := []string{"interest_received:-", "ride_match:" + rideB}
	if len(left) != 2 || left[0] != want[0] || left[1] != want[1] {
		t.Errorf("remaining = %v, want %v", left, want)
	}
	items, _ := repo.ListByPhone("0611", 10)
	if len(items) != 1 || items[0].RideID != rideB || items[0].RequestID != "" {
		t.Errorf("round-trip of ride/request ids failed: %+v", items)
	}
}
