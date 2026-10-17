// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package usecase_test

import (
	"testing"
	"time"

	"github.com/z3spinner/go-stop/internal/domain"
	"github.com/z3spinner/go-stop/internal/usecase"
)

func TestSendFeedbackPrompts_PushesOnceAndRecordsTheInboxItem(t *testing.T) {
	queue := &mockFeedbackQueue{due: []domain.FeedbackTask{
		{ID: "fq-1", RideID: "ride-1", Phone: "555-0001", Origin: "Saillans", Destination: "Crest",
			DepartureAt: time.Date(2030, 6, 1, 9, 0, 0, 0, time.UTC)},
	}}
	subs := &mockSubRepo{subs: map[string]domain.Subscription{
		"555-0001": {Phone: "555-0001", Endpoint: "https://push.example.com"},
	}}
	n := &mockNotifier{}

	uc := usecase.NewSendFeedbackPrompts(queue, subs, n, usecase.QuietHours{}, time.UTC)
	if err := uc.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !n.called {
		t.Error("expected push notification to be sent")
	}
	if n.lastMsg.URL != "/rides/ride-1/feedback" || n.lastMsg.InboxID == "" {
		t.Errorf("unexpected push: %+v", n.lastMsg)
	}
	if len(queue.marked) != 1 || queue.marked[0] != "fq-1" {
		t.Error("expected task fq-1 marked sent")
	}
	// Unanswered tasks are kept a week (answers still possible after expiry).
	if queue.deleteStaleTTL != 7*24*time.Hour {
		t.Errorf("stale cleanup TTL = %v, want 7 days", queue.deleteStaleTTL)
	}
	// Recorded in the inbox, where the reminders pick it up.
	assertInbox(t, n, "555-0001", domain.Message{
		Kind: domain.KindFeedbackReminder, DedupKey: "feedback_reminder:ride-1", RideID: "ride-1",
	})
}

func TestSendFeedbackPrompts_WaitsDuringQuietHours(t *testing.T) {
	h := time.Now().UTC().Hour()
	quiet := usecase.QuietHours{Start: h, End: (h + 1) % 24}
	queue := &mockFeedbackQueue{due: []domain.FeedbackTask{{ID: "fq-1", RideID: "ride-1", Phone: "555-0001"}}}
	n := &mockNotifier{}
	if err := usecase.NewSendFeedbackPrompts(queue, &mockSubRepo{subs: map[string]domain.Subscription{}}, n, quiet, time.UTC).Execute(); err != nil {
		t.Fatal(err)
	}
	if queue.findCalled || len(queue.marked) != 0 || len(n.recorded) != 0 {
		t.Errorf("quiet hours: nothing should be sent, got find=%v marked=%v records=%d", queue.findCalled, queue.marked, len(n.recorded))
	}
	if queue.deleteStaleTTL == 0 {
		t.Error("cleanup still runs during quiet hours")
	}
}

func TestSendFeedbackPrompts_NoSubscriptionStillReachesTheInbox(t *testing.T) {
	queue := &mockFeedbackQueue{due: []domain.FeedbackTask{{ID: "fq-1", RideID: "ride-1", Phone: "555-no-sub"}}}
	n := &mockNotifier{}
	if err := usecase.NewSendFeedbackPrompts(queue, &mockSubRepo{subs: map[string]domain.Subscription{}}, n, usecase.QuietHours{}, time.UTC).Execute(); err != nil {
		t.Fatal(err)
	}
	if n.called {
		t.Error("no device: nothing to push")
	}
	if len(n.recorded) != 1 || len(queue.marked) != 1 {
		t.Errorf("expected the prompt recorded and the task marked sent, got records=%d marked=%v", len(n.recorded), queue.marked)
	}
}

func TestSendFeedbackPrompts_NoDueTasks(t *testing.T) {
	n := &mockNotifier{}
	if err := usecase.NewSendFeedbackPrompts(&mockFeedbackQueue{}, &mockSubRepo{subs: map[string]domain.Subscription{}}, n, usecase.QuietHours{}, time.UTC).Execute(); err != nil {
		t.Fatal(err)
	}
	if n.called || len(n.recorded) != 0 {
		t.Error("nothing should be sent")
	}
}
