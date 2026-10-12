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

	if n, _ := repo.CountUnread("0611"); n != 3 {
		t.Errorf("expected 3 unread, got %d", n)
	}
	if err := repo.MarkRead("0611", []string{first.ID, "not-a-uuid"}); err != nil {
		t.Fatalf("MarkRead ids: %v", err)
	}
	if n, _ := repo.CountUnread("0611"); n != 2 {
		t.Errorf("expected 2 unread after marking one, got %d", n)
	}
	if err := repo.MarkRead("0611", []string{"not-a-uuid"}); err != nil {
		t.Fatalf("MarkRead malformed: %v", err)
	}
	if n, _ := repo.CountUnread("0611"); n != 2 {
		t.Errorf("malformed ids must not mark all; got %d unread", n)
	}
	if err := repo.MarkRead("0611", nil); err != nil {
		t.Fatalf("MarkRead all: %v", err)
	}
	if n, _ := repo.CountUnread("0611"); n != 0 {
		t.Errorf("expected 0 unread after mark-all, got %d", n)
	}
	if n, _ := repo.CountUnread("0622"); n != 1 {
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
