// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/infrastructure/postgres"
)

func TestInboxRepo_AddListDedupAndRead(t *testing.T) {
	truncate(t)
	repo := postgres.NewInboxRepo(testPool)
	dep := time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC)

	add := func(it domain.InboxItem) {
		t.Helper()
		if _, err := repo.Add(it); err != nil {
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
	if _, err := repo.Add(domain.InboxItem{Phone: "0611", Kind: domain.KindRideMatch, DedupKey: "old"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(domain.InboxItem{Phone: "0611", Kind: domain.KindRideMatch, DedupKey: "new"}); err != nil {
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
		if _, err := repo.Add(domain.InboxItem{Phone: "0611", Kind: domain.KindRideMatch}); err != nil {
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
		if _, err := repo.Add(it); err != nil {
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

// Items about a ride/request disappear once it expires (the expiry job deletes
// the row), except the driver's own feedback prompt while it can still be
// answered (its feedback task exists). Live posts and unlinked items are
// untouched.
func TestInboxRepo_DeleteForGonePosts(t *testing.T) {
	truncate(t)
	inbox := postgres.NewInboxRepo(testPool)
	rides := postgres.NewRideRepo(testPool, 60)
	requests := postgres.NewRequestRepo(testPool, 60)

	past := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	future := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	expiredRide, liveRide := uuid.New().String(), uuid.New().String()
	expiredReq, liveReq := uuid.New().String(), uuid.New().String()
	for _, rd := range []domain.Ride{
		{ID: expiredRide, DriverName: "Old", Phone: "0601", Origin: "A", Destination: "B", Date: past, DepartureAt: past, PostedAt: past, ExpiresAt: past},
		{ID: liveRide, DriverName: "New", Phone: "0602", Origin: "A", Destination: "C", Date: future, DepartureAt: future, PostedAt: past, ExpiresAt: future},
	} {
		if _, _, err := rides.Save(rd); err != nil {
			t.Fatalf("save ride: %v", err)
		}
	}
	for _, rq := range []domain.Request{
		{ID: expiredReq, SearcherName: "Old", Phone: "0603", Origin: "A", Destination: "B", Date: past, DepartureAt: past, PostedAt: past, ExpiresAt: past},
		{ID: liveReq, SearcherName: "New", Phone: "0604", Origin: "A", Destination: "C", Date: future, DepartureAt: future, PostedAt: past, ExpiresAt: future},
	} {
		if err := requests.Save(rq); err != nil {
			t.Fatalf("save request: %v", err)
		}
	}
	for _, it := range []domain.InboxItem{
		{Phone: "0611", Kind: domain.KindRideMatch, RideID: expiredRide, DedupKey: "gone-ride"},
		{Phone: "0601", Kind: domain.KindFeedbackReminder, RideID: expiredRide, DedupKey: "keep-feedback"},
		{Phone: "0611", Kind: domain.KindRideMatch, RideID: liveRide, DedupKey: "keep-ride"},
		{Phone: "0612", Kind: domain.KindContactOffered, RequestID: expiredReq, DedupKey: "gone-request"},
		{Phone: "0612", Kind: domain.KindRequestMatch, RequestID: liveReq, DedupKey: "keep-request"},
		{Phone: "0613", Kind: domain.KindInterestReceived, DedupKey: "keep-unlinked"},
	} {
		if _, err := inbox.Add(it); err != nil {
			t.Fatal(err)
		}
	}

	// The expired ride's feedback prompt can still be answered: its task exists.
	if _, err := testPool.Exec(context.Background(), `TRUNCATE feedback_queue`); err != nil {
		t.Fatal(err)
	}
	if _, err := testPool.Exec(context.Background(),
		`INSERT INTO feedback_queue (ride_id, phone, origin, destination, ride_date, departure_at, send_after)
		 VALUES ($1, '0601', 'A', 'B', CURRENT_DATE, NOW() - interval '48 hours', NOW() - interval '47 hours')`, expiredRide); err != nil {
		t.Fatal(err)
	}

	// What the hourly cron does: expiry jobs first, then the inbox purge.
	if err := rides.DeleteExpired(); err != nil {
		t.Fatal(err)
	}
	if err := requests.DeleteExpired(); err != nil {
		t.Fatal(err)
	}
	n, err := inbox.DeleteForGonePosts()
	if err != nil {
		t.Fatalf("DeleteForGonePosts: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 deleted, got %d", n)
	}
	var left []string
	rows, _ := testPool.Query(context.Background(), `SELECT dedup_key FROM notifications ORDER BY dedup_key`)
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		left = append(left, k)
	}
	rows.Close()
	want := []string{"keep-feedback", "keep-request", "keep-ride", "keep-unlinked"}
	if strings.Join(left, ",") != strings.Join(want, ",") {
		t.Errorf("remaining = %v, want %v", left, want)
	}
}

// A deleted (dismissed) item disappears from the list and the badge, a repeat
// of the same event doesn't bring it back, undo works only inside the window,
// and the content is blanked once the window has passed.
func TestInboxRepo_DismissRestoreScrub(t *testing.T) {
	truncate(t)
	repo := postgres.NewInboxRepo(testPool)
	dep := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	match := domain.InboxItem{Phone: "0611", Kind: domain.KindRideMatch, ContactName: "Alice",
		Origin: "Saillans", Destination: "Crest", DepartureAt: dep, URL: "/rides/r1", DedupKey: "ride_match:r1"}
	if _, err := repo.Add(match); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(domain.InboxItem{Phone: "0611", Kind: domain.KindInterestReceived, DedupKey: "other"}); err != nil {
		t.Fatal(err)
	}
	items, _ := repo.ListByPhone("0611", 10)
	var id string
	for _, it := range items {
		if it.DedupKey == "ride_match:r1" {
			id = it.ID
		}
	}

	// Another phone can't delete it; malformed ids are ignored.
	if err := repo.Dismiss("0622", id); err != nil {
		t.Fatal(err)
	}
	if err := repo.Dismiss("0611", "not-a-uuid"); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.ListByPhone("0611", 10); len(got) != 2 {
		t.Fatalf("other phone / malformed id must not delete; got %d items", len(got))
	}

	if err := repo.Dismiss("0611", id); err != nil {
		t.Fatalf("Dismiss: %v", err)
	}
	if got, _ := repo.ListByPhone("0611", 10); len(got) != 1 || got[0].DedupKey != "other" {
		t.Errorf("dismissed item must be hidden, got %+v", got)
	}
	if n, _ := repo.CountUnread("0611", 10); n != 1 {
		t.Errorf("dismissed item must not count as unread; got %d", n)
	}
	// A push retry of the same match must not resurrect it.
	if _, err := repo.Add(match); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.ListByPhone("0611", 10); len(got) != 1 {
		t.Errorf("retry resurrected a deleted item: %+v", got)
	}

	// Undo: not by another phone, not outside the window, yes inside it.
	if ok, _ := repo.Restore("0622", id, time.Now().Add(-10*time.Minute)); ok {
		t.Error("another phone must not restore")
	}
	if ok, _ := repo.Restore("0611", id, time.Now().Add(time.Minute)); ok {
		t.Error("restore must respect the window")
	}
	if ok, err := repo.Restore("0611", id, time.Now().Add(-10*time.Minute)); err != nil || !ok {
		t.Fatalf("Restore: ok=%v err=%v", ok, err)
	}
	got, _ := repo.ListByPhone("0611", 10)
	if len(got) != 2 {
		t.Fatalf("restored item should be back, got %d", len(got))
	}
	if ok, _ := repo.Restore("0611", id, time.Now().Add(-10*time.Minute)); ok {
		t.Error("restoring a live item must report nothing restored")
	}

	// Delete again, let the undo window lapse, scrub.
	if err := repo.Dismiss("0611", id); err != nil {
		t.Fatal(err)
	}
	if _, err := testPool.Exec(context.Background(), `UPDATE notifications SET dismissed_at = NOW() - interval '1 hour' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if n, err := repo.ScrubDismissed(time.Now().Add(-10 * time.Minute)); err != nil || n != 1 {
		t.Fatalf("ScrubDismissed: n=%d err=%v", n, err)
	}
	var name, origin, title, url, key string
	var departure *time.Time
	if err := testPool.QueryRow(context.Background(),
		`SELECT contact_name, origin, title, url, departure_at, dedup_key FROM notifications WHERE id = $1`, id).
		Scan(&name, &origin, &title, &url, &departure, &key); err != nil {
		t.Fatal(err)
	}
	if name != "" || origin != "" || title != "" || url != "/" || departure != nil || key != "ride_match:r1" {
		t.Errorf("scrub should blank content but keep the tombstone: name=%q origin=%q title=%q url=%q dep=%v key=%q",
			name, origin, title, url, departure, key)
	}
	if ok, _ := repo.Restore("0611", id, time.Now().Add(-10*time.Minute)); ok {
		t.Error("a scrubbed item must not be restorable")
	}
	if n, _ := repo.ScrubDismissed(time.Now().Add(-10 * time.Minute)); n != 0 {
		t.Errorf("scrub should be idempotent, got %d", n)
	}
}

// Add returns the item's id, and the existing id for a repeat of the event.
func TestInboxRepo_AddReturnsID(t *testing.T) {
	truncate(t)
	repo := postgres.NewInboxRepo(testPool)
	id1, err := repo.Add(domain.InboxItem{Phone: "0611", Kind: domain.KindRideMatch, Title: "first", DedupKey: "k"})
	if err != nil || id1 == "" {
		t.Fatalf("Add: id=%q err=%v", id1, err)
	}
	id2, err := repo.Add(domain.InboxItem{Phone: "0611", Kind: domain.KindRideMatch, Title: "second", DedupKey: "k"})
	if err != nil || id2 != id1 {
		t.Fatalf("repeat should return the existing id %q, got %q (err %v)", id1, id2, err)
	}
	items, _ := repo.ListByPhone("0611", 10)
	if len(items) != 1 || items[0].Title != "first" {
		t.Errorf("repeat must leave the row untouched, got %+v", items)
	}
	id3, _ := repo.Add(domain.InboxItem{Phone: "0611", Kind: domain.KindTest})
	if id3 == "" || id3 == id1 {
		t.Errorf("an item without dedup key gets its own id, got %q", id3)
	}
}

func TestInboxRepo_MarkReadByDedupKeys(t *testing.T) {
	truncate(t)
	repo := postgres.NewInboxRepo(testPool)
	for _, it := range []domain.InboxItem{
		{Phone: "0611", Kind: domain.KindInterestReceived, DedupKey: "interest_received:a"},
		{Phone: "0611", Kind: domain.KindInterestReceived, DedupKey: "interest_received:b"},
		{Phone: "0622", Kind: domain.KindInterestReceived, DedupKey: "interest_received:a"},
	} {
		if _, err := repo.Add(it); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.MarkReadByDedupKeys("0611", []string{"interest_received:a"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkReadByDedupKeys("0611", nil); err != nil { // no keys: no-op
		t.Fatal(err)
	}
	if n, _ := repo.CountUnread("0611", 10); n != 1 {
		t.Errorf("0611 unread = %d, want 1", n)
	}
	if n, _ := repo.CountUnread("0622", 10); n != 1 {
		t.Errorf("another phone's item with the same key must stay unread, got %d", n)
	}
}

// Due = unread, not deleted, a reminder kind, last push at or before cutoff.
func TestInboxRepo_FindRemindersDueAndMarkReminded(t *testing.T) {
	truncate(t)
	repo := postgres.NewInboxRepo(testPool)
	add := func(it domain.InboxItem, age time.Duration) string {
		t.Helper()
		id, err := repo.Add(it)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := testPool.Exec(context.Background(), `UPDATE notifications SET created_at = NOW() - $2::interval WHERE id = $1`,
			id, fmt.Sprintf("%d seconds", int(age.Seconds()))); err != nil {
			t.Fatal(err)
		}
		return id
	}
	kinds := []string{domain.KindInterestReceived, domain.KindContactOffered}
	due := add(domain.InboxItem{Phone: "0611", Kind: domain.KindInterestReceived, Title: "t", DedupKey: "due"}, 4*time.Hour)
	add(domain.InboxItem{Phone: "0611", Kind: domain.KindInterestReceived, DedupKey: "too-recent"}, time.Hour)
	add(domain.InboxItem{Phone: "0611", Kind: domain.KindRideMatch, DedupKey: "other-kind"}, 4*time.Hour)
	read := add(domain.InboxItem{Phone: "0611", Kind: domain.KindContactOffered, DedupKey: "read"}, 4*time.Hour)
	gone := add(domain.InboxItem{Phone: "0611", Kind: domain.KindContactOffered, DedupKey: "deleted"}, 4*time.Hour)
	if err := repo.MarkRead("0611", []string{read}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Dismiss("0611", gone); err != nil {
		t.Fatal(err)
	}

	cutoff := time.Now().Add(-3 * time.Hour)
	got, err := repo.FindRemindersDue(kinds, 0, cutoff, 10)
	if err != nil {
		t.Fatalf("FindRemindersDue: %v", err)
	}
	if len(got) != 1 || got[0].ID != due || got[0].Title != "t" || got[0].Phone != "0611" {
		t.Fatalf("expected only the due item, got %+v", got)
	}

	// After a reminder it isn't due again until another interval passes.
	if err := repo.MarkReminded(due); err != nil {
		t.Fatal(err)
	}
	if again, _ := repo.FindRemindersDue(kinds, 0, cutoff, 10); len(again) != 0 {
		t.Errorf("just reminded: expected nothing due, got %d", len(again))
	}
	if _, err := testPool.Exec(context.Background(), `UPDATE notifications SET reminded_at = NOW() - interval '4 hours' WHERE id = $1`, due); err != nil {
		t.Fatal(err)
	}
	later, _ := repo.FindRemindersDue(kinds, 0, cutoff, 10)
	if len(later) != 1 || later[0].ReminderCount != 1 || later[0].RemindedAt.IsZero() {
		t.Errorf("expected it due again with count 1, got %+v", later)
	}
	// Reading it stops the reminders.
	if err := repo.MarkRead("0611", []string{due}); err != nil {
		t.Fatal(err)
	}
	if after, _ := repo.FindRemindersDue(kinds, 0, cutoff, 10); len(after) != 0 {
		t.Errorf("read items must not be reminded, got %d", len(after))
	}
}

// A capped kind stops being due once it has had its reminders.
func TestInboxRepo_FindRemindersDueRespectsTheCap(t *testing.T) {
	truncate(t)
	repo := postgres.NewInboxRepo(testPool)
	id, err := repo.Add(domain.InboxItem{Phone: "0611", Kind: domain.KindFeedbackReminder, DedupKey: "feedback_reminder:r1"})
	if err != nil {
		t.Fatal(err)
	}
	kinds := []string{domain.KindFeedbackReminder}
	age := func() {
		t.Helper()
		if _, err := testPool.Exec(context.Background(),
			`UPDATE notifications SET created_at = NOW() - interval '10 hours', reminded_at = CASE WHEN reminded_at IS NULL THEN NULL ELSE NOW() - interval '4 hours' END WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
	}
	cutoff := time.Now().Add(-3 * time.Hour)
	for i := 0; i < 2; i++ {
		age()
		got, _ := repo.FindRemindersDue(kinds, 2, cutoff, 10)
		if len(got) != 1 {
			t.Fatalf("reminder %d: expected due, got %d", i+1, len(got))
		}
		if err := repo.MarkReminded(id); err != nil {
			t.Fatal(err)
		}
	}
	age()
	if got, _ := repo.FindRemindersDue(kinds, 2, cutoff, 10); len(got) != 0 {
		t.Errorf("after 2 reminders the capped item must not be due, got %d", len(got))
	}
	if got, _ := repo.FindRemindersDue(kinds, 0, cutoff, 10); len(got) != 1 {
		t.Errorf("without a cap it would still be due, got %d", len(got))
	}
}

// The feedback prompt outlives its ride while it can still be answered (its
// feedback task exists) and goes with the task.
func TestInboxRepo_FeedbackPromptGoesWithItsTask(t *testing.T) {
	truncate(t)
	if _, err := testPool.Exec(context.Background(), `TRUNCATE feedback_queue`); err != nil {
		t.Fatal(err)
	}
	repo := postgres.NewInboxRepo(testPool)
	withTask, withoutTask := uuid.New().String(), uuid.New().String()
	for _, rid := range []string{withTask, withoutTask} {
		if _, err := repo.Add(domain.InboxItem{Phone: "0611", Kind: domain.KindFeedbackReminder, RideID: rid, DedupKey: "feedback_reminder:" + rid}); err != nil {
			t.Fatal(err)
		}
	}
	// Neither ride exists any more; only one still has its feedback task.
	if _, err := testPool.Exec(context.Background(),
		`INSERT INTO feedback_queue (ride_id, phone, origin, destination, ride_date, departure_at, send_after)
		 VALUES ($1, '0611', 'A', 'B', CURRENT_DATE, NOW() - interval '3 hours', NOW() - interval '1 hour')`, withTask); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.DeleteForGonePosts(); err != nil {
		t.Fatal(err)
	}
	items, _ := repo.ListByPhone("0611", 10)
	if len(items) != 1 || items[0].RideID != withTask {
		t.Errorf("expected only the prompt with a task to remain, got %+v", items)
	}
}
